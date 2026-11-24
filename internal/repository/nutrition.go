package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"lifeops-tracker/internal/models"
)

// Food diary storage (Issue #207). Timestamps are written by the service
// (clock.FormatInstant); nothing here reads the SQL clock.

// ErrDiaryItemState means an item was not in the state the change needs
// (already archived, not archived, or not found).
var ErrDiaryItemState = errors.New("diary item is not in the expected state")

// immediateTx runs fn inside BEGIN IMMEDIATE on one connection, so a
// read-then-write sequence can't lose a lock upgrade to another writer.
func (r *Repository) immediateTx(ctx context.Context, fn func(conn *sql.Conn) error) (err error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
		}
	}()
	if err = fn(conn); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

// NutritionDayTotals sums the live items (live item in a live meal) per local
// day from..to inclusive. A day with no live item is absent. Each sum is nil
// when no item has the nutrient; Gaps counts the items that lack it.
func (r *Repository) NutritionDayTotals(from, to string) (map[string]models.NutritionDayTotals, error) {
	rows, err := r.db.Query(`SELECT m.local_date, COUNT(*),
 SUM(i.kcal), COUNT(*) - COUNT(i.kcal),
 SUM(i.protein_g), COUNT(*) - COUNT(i.protein_g),
 SUM(i.carbs_g), COUNT(*) - COUNT(i.carbs_g),
 SUM(i.fat_g), COUNT(*) - COUNT(i.fat_g),
 SUM(i.fiber_g), COUNT(*) - COUNT(i.fiber_g)
 FROM nutrition_meal_items i JOIN nutrition_meals m ON m.id = i.meal_id
 WHERE i.archived_at IS NULL AND m.archived_at IS NULL AND m.local_date >= ? AND m.local_date <= ?
 GROUP BY m.local_date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.NutritionDayTotals{}
	for rows.Next() {
		var t models.NutritionDayTotals
		if err := rows.Scan(&t.Date, &t.Items,
			&t.Sum.Kcal, &t.Gaps.Kcal, &t.Sum.ProteinG, &t.Gaps.ProteinG, &t.Sum.CarbsG, &t.Gaps.CarbsG,
			&t.Sum.FatG, &t.Gaps.FatG, &t.Sum.FiberG, &t.Gaps.FiberG); err != nil {
			return nil, err
		}
		out[t.Date] = t
	}
	return out, rows.Err()
}

const nutritionFoodColumns = `id, provider, provider_id, version, name, brand, normalized_name, basis, serving_label, serving_grams,
 kcal, protein_g, carbs_g, fat_g, fiber_g, data_provenance, dataset_version, created_at, superseded_at, archived_at`

func scanNutritionFood(row scanner) (models.NutritionFood, error) {
	var f models.NutritionFood
	var superseded, archived sql.NullString
	err := row.Scan(&f.ID, &f.Provider, &f.ProviderID, &f.Version, &f.Name, &f.Brand, &f.NormalizedName, &f.Basis, &f.ServingLabel, &f.ServingGrams,
		&f.Kcal, &f.ProteinG, &f.CarbsG, &f.FatG, &f.FiberG, &f.DataProvenance, &f.DatasetVersion, &f.CreatedAt, &superseded, &archived)
	f.SupersededAt, f.ArchivedAt = superseded.String, archived.String
	return f, err
}

// GetNutritionFood returns one catalogue row (any version) by id.
func (r *Repository) GetNutritionFood(id int64) (models.NutritionFood, error) {
	return scanNutritionFood(r.db.QueryRow(`SELECT `+nutritionFoodColumns+` FROM nutrition_foods WHERE id = ?`, id))
}

// escapeLike escapes the LIKE wildcards and the escape character itself.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// SearchNutritionFoods finds current, not archived foods whose normalized
// name contains the (already normalized) query. Names that start with it
// come first. LIKE is the first implementation; full-text search (#208) can
// replace this one function.
func (r *Repository) SearchNutritionFoods(normalized string, limit int) ([]models.NutritionFood, error) {
	like := escapeLike(normalized)
	rows, err := r.db.Query(`SELECT `+nutritionFoodColumns+` FROM nutrition_foods
 WHERE superseded_at IS NULL AND archived_at IS NULL AND normalized_name LIKE ? ESCAPE '\'
 ORDER BY (normalized_name LIKE ? ESCAPE '\') DESC, length(normalized_name), name, id
 LIMIT ?`, "%"+like+"%", like+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.NutritionFood{}
	for rows.Next() {
		f, err := scanNutritionFood(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func sameFloat(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// sameNutritionFood compares everything a version stores except identity and
// timestamps.
func sameNutritionFood(a, b models.NutritionFood) bool {
	return a.Name == b.Name && a.Brand == b.Brand && a.NormalizedName == b.NormalizedName && a.Basis == b.Basis &&
		a.ServingLabel == b.ServingLabel && sameFloat(a.ServingGrams, b.ServingGrams) &&
		sameFloat(a.Kcal, b.Kcal) && sameFloat(a.ProteinG, b.ProteinG) && sameFloat(a.CarbsG, b.CarbsG) &&
		sameFloat(a.FatG, b.FatG) && sameFloat(a.FiberG, b.FiberG) &&
		a.DataProvenance == b.DataProvenance && a.DatasetVersion == b.DatasetVersion
}

// UpsertNutritionFoodVersion makes f the current version of its
// (provider, provider_id) food, in one transaction. If the current row already
// has the same content it is returned unchanged. Otherwise the current row is
// marked superseded at `at` and f is inserted as the next version. f.ID,
// Version and the timestamps are ignored.
func (r *Repository) UpsertNutritionFoodVersion(ctx context.Context, f models.NutritionFood, at string) (models.NutritionFood, error) {
	var out models.NutritionFood
	err := r.immediateTx(ctx, func(conn *sql.Conn) (err error) {
		out, err = upsertNutritionFoodTx(ctx, conn, f, at)
		return err
	})
	return out, err
}

func upsertNutritionFoodTx(ctx context.Context, conn *sql.Conn, f models.NutritionFood, at string) (models.NutritionFood, error) {
	cur, err := scanNutritionFood(conn.QueryRowContext(ctx, `SELECT `+nutritionFoodColumns+` FROM nutrition_foods
 WHERE provider = ? AND provider_id = ? AND superseded_at IS NULL`, f.Provider, f.ProviderID))
	version := int64(1)
	switch {
	case err == nil:
		if sameNutritionFood(cur, f) {
			return cur, nil
		}
		version = cur.Version + 1
		if _, err := conn.ExecContext(ctx, `UPDATE nutrition_foods SET superseded_at = ? WHERE id = ? AND superseded_at IS NULL`, at, cur.ID); err != nil {
			return cur, err
		}
	case errors.Is(err, sql.ErrNoRows):
	default:
		return cur, err
	}
	res, err := conn.ExecContext(ctx, `INSERT INTO nutrition_foods (provider, provider_id, version, name, brand, normalized_name, basis, serving_label, serving_grams,
 kcal, protein_g, carbs_g, fat_g, fiber_g, data_provenance, dataset_version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.Provider, f.ProviderID, version, f.Name, f.Brand, f.NormalizedName, f.Basis, f.ServingLabel, f.ServingGrams,
		f.Kcal, f.ProteinG, f.CarbsG, f.FatG, f.FiberG, f.DataProvenance, f.DatasetVersion, at)
	if err != nil {
		return cur, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return cur, err
	}
	return scanNutritionFood(conn.QueryRowContext(ctx, `SELECT `+nutritionFoodColumns+` FROM nutrition_foods WHERE id = ?`, id))
}

// DiaryItemInsert is one snapshotted item for CreateMealWithItems.
type DiaryItemInsert struct {
	FoodID, FoodVersion               *int64
	Name, Unit                        string
	Amount                            float64
	Grams                             *float64
	QuantityProvenance, NutritionProv string
	Nutrients                         models.Nutrients
	AIEstGrams                        *float64 // camera estimate snapshot (#211); nil otherwise
	// AIEst holds the AI's own nutrients at its estimated grams (GYST-179),
	// written once here and never by a later edit. nil when it gave none.
	AIEst *models.Nutrients
}

// DiaryMealInsert is the meal row for CreateMealWithItems.
type DiaryMealInsert struct {
	LocalDate, MealSlot, CreatedVia, IdempotencyKey, CreatedAt string
}

// CreateMealWithItems inserts a meal and its items in one transaction. A meal
// whose idempotency_key already exists is not inserted again: created is false,
// mealID is the existing meal and no item is written. itemIDs are the meal's
// item ids either way.
func (r *Repository) CreateMealWithItems(ctx context.Context, m DiaryMealInsert, items []DiaryItemInsert) (mealID int64, itemIDs []int64, created bool, err error) {
	err = r.immediateTx(ctx, func(conn *sql.Conn) error {
		e := conn.QueryRowContext(ctx, `INSERT INTO nutrition_meals (local_date, meal_slot, created_via, idempotency_key, created_at)
 VALUES (?, ?, ?, ?, ?) ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
			m.LocalDate, m.MealSlot, m.CreatedVia, m.IdempotencyKey, m.CreatedAt).Scan(&mealID)
		switch {
		case errors.Is(e, sql.ErrNoRows):
			if e := conn.QueryRowContext(ctx, `SELECT id FROM nutrition_meals WHERE idempotency_key = ?`, m.IdempotencyKey).Scan(&mealID); e != nil {
				return e
			}
			return collectItemIDs(ctx, conn, mealID, &itemIDs)
		case e != nil:
			return e
		}
		created = true
		for _, it := range items {
			var est models.Nutrients
			if it.AIEst != nil {
				est = *it.AIEst
			}
			if _, e := conn.ExecContext(ctx, `INSERT INTO nutrition_meal_items (meal_id, food_id, food_version, name, unit, amount, grams,
 quantity_provenance, nutrition_provenance, kcal, protein_g, carbs_g, fat_g, fiber_g, ai_est_grams,
 ai_est_kcal, ai_est_protein_g, ai_est_carbs_g, ai_est_fat_g, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				mealID, it.FoodID, it.FoodVersion, it.Name, it.Unit, it.Amount, it.Grams, it.QuantityProvenance, it.NutritionProv,
				it.Nutrients.Kcal, it.Nutrients.ProteinG, it.Nutrients.CarbsG, it.Nutrients.FatG, it.Nutrients.FiberG, it.AIEstGrams,
				est.Kcal, est.ProteinG, est.CarbsG, est.FatG, m.CreatedAt); e != nil {
				return e
			}
		}
		return collectItemIDs(ctx, conn, mealID, &itemIDs)
	})
	return mealID, itemIDs, created, err
}

// CreateMealWithMirroredFood is CreateMealWithItems for a My food: in the same
// transaction, and only when the meal is new, it makes mirror the current
// catalogue version (a new version only when the content changed) and inserts
// the one item that build returns for that row. A repeated idempotency key
// writes nothing. An error from build rolls everything back.
func (r *Repository) CreateMealWithMirroredFood(ctx context.Context, m DiaryMealInsert, mirror models.NutritionFood, at string,
	build func(models.NutritionFood) (DiaryItemInsert, error)) (mealID, itemID int64, created bool, err error) {
	err = r.immediateTx(ctx, func(conn *sql.Conn) error {
		e := conn.QueryRowContext(ctx, `INSERT INTO nutrition_meals (local_date, meal_slot, created_via, idempotency_key, created_at)
 VALUES (?, ?, ?, ?, ?) ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
			m.LocalDate, m.MealSlot, m.CreatedVia, m.IdempotencyKey, m.CreatedAt).Scan(&mealID)
		if errors.Is(e, sql.ErrNoRows) {
			if e := conn.QueryRowContext(ctx, `SELECT id FROM nutrition_meals WHERE idempotency_key = ?`, m.IdempotencyKey).Scan(&mealID); e != nil {
				return e
			}
			var ids []int64
			if e := collectItemIDs(ctx, conn, mealID, &ids); e != nil {
				return e
			}
			if len(ids) > 0 {
				itemID = ids[0]
			}
			return nil
		} else if e != nil {
			return e
		}
		created = true
		food, e := upsertNutritionFoodTx(ctx, conn, mirror, at)
		if e != nil {
			return e
		}
		it, e := build(food)
		if e != nil {
			return e
		}
		res, e := conn.ExecContext(ctx, `INSERT INTO nutrition_meal_items (meal_id, food_id, food_version, name, unit, amount, grams,
 quantity_provenance, nutrition_provenance, kcal, protein_g, carbs_g, fat_g, fiber_g, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mealID, it.FoodID, it.FoodVersion, it.Name, it.Unit, it.Amount, it.Grams, it.QuantityProvenance, it.NutritionProv,
			it.Nutrients.Kcal, it.Nutrients.ProteinG, it.Nutrients.CarbsG, it.Nutrients.FatG, it.Nutrients.FiberG, m.CreatedAt)
		if e != nil {
			return e
		}
		itemID, e = res.LastInsertId()
		return e
	})
	return mealID, itemID, created, err
}

func collectItemIDs(ctx context.Context, conn *sql.Conn, mealID int64, out *[]int64) error {
	rows, err := conn.QueryContext(ctx, `SELECT id FROM nutrition_meal_items WHERE meal_id = ? ORDER BY id`, mealID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		*out = append(*out, id)
	}
	return rows.Err()
}

const diaryItemSelect = `SELECT i.id, i.meal_id, m.meal_slot, m.local_date, i.food_id, i.food_version, i.name, i.unit, i.amount, i.grams,
 i.quantity_provenance, i.nutrition_provenance, i.kcal, i.protein_g, i.carbs_g, i.fat_g, i.fiber_g, i.created_at, i.archived_at, m.archived_at, i.ai_est_grams, i.ai_est_kcal, i.ai_est_protein_g, i.ai_est_carbs_g, i.ai_est_fat_g, m.created_via
 FROM nutrition_meal_items i JOIN nutrition_meals m ON m.id = i.meal_id`

func scanDiaryItem(row scanner) (models.DiaryItem, error) {
	var it models.DiaryItem
	var archived, mealArchived sql.NullString
	err := row.Scan(&it.ID, &it.MealID, &it.MealSlot, &it.LocalDate, &it.FoodID, &it.FoodVersion, &it.Name, &it.Unit, &it.Amount, &it.Grams,
		&it.QuantityProvenance, &it.NutritionProvenance, &it.Kcal, &it.ProteinG, &it.CarbsG, &it.FatG, &it.FiberG, &it.CreatedAt, &archived, &mealArchived, &it.AIEstGrams, &it.AIEstKcal, &it.AIEstProteinG, &it.AIEstCarbsG, &it.AIEstFatG, &it.CreatedVia)
	it.ArchivedAt = archived.String
	if it.ArchivedAt == "" {
		it.ArchivedAt = mealArchived.String // an item of an archived meal is not live
	}
	return it, err
}

// GetDiaryItem returns one item (live or archived) with its meal's slot and date.
func (r *Repository) GetDiaryItem(id int64) (models.DiaryItem, error) {
	return scanDiaryItem(r.db.QueryRow(diaryItemSelect+` WHERE i.id = ?`, id))
}

// ListDiaryItems returns the live items of a local day, oldest first.
func (r *Repository) ListDiaryItems(date string) ([]models.DiaryItem, error) {
	rows, err := r.db.Query(diaryItemSelect+` WHERE m.local_date = ? AND i.archived_at IS NULL AND m.archived_at IS NULL ORDER BY i.id`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.DiaryItem{}
	for rows.Next() {
		it, err := scanDiaryItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// UpdateDiaryItemAmount rewrites a live item's quantity and snapshot nutrients.
// A missing or archived item is ErrDiaryItemState.
func (r *Repository) UpdateDiaryItemAmount(id int64, amount float64, grams *float64, n models.Nutrients) error {
	res, err := r.db.Exec(`UPDATE nutrition_meal_items SET amount = ?, grams = ?, kcal = ?, protein_g = ?, carbs_g = ?, fat_g = ?, fiber_g = ?
 WHERE id = ? AND archived_at IS NULL`, amount, grams, n.Kcal, n.ProteinG, n.CarbsG, n.FatG, n.FiberG, id)
	return oneRow(res, err)
}

// ArchiveDiaryItem sets archived_at on a live item.
func (r *Repository) ArchiveDiaryItem(id int64, at string) error {
	res, err := r.db.Exec(`UPDATE nutrition_meal_items SET archived_at = ? WHERE id = ? AND archived_at IS NULL`, at, id)
	return oneRow(res, err)
}

// RestoreDiaryItem clears archived_at on an archived item.
func (r *Repository) RestoreDiaryItem(id int64) error {
	res, err := r.db.Exec(`UPDATE nutrition_meal_items SET archived_at = NULL WHERE id = ? AND archived_at IS NOT NULL`, id)
	return oneRow(res, err)
}

func oneRow(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return ErrDiaryItemState
	}
	return nil
}

// AIAccuracyByWeek reports, per Monday-start week of local dates in [from, to]
// (inclusive YYYY-MM-DD), how far the saved values of live camera items are from
// the AI's own estimate (GYST-179). The saved value is the item's current row, so
// removed items and archived meals drop out. An average is nil, not 0, when no
// item that week has both numbers; each measure carries its paired count.
func (r *Repository) AIAccuracyByWeek(from, to string) ([]models.AIAccuracyWeek, error) {
	measures := [...][2]string{{"i.grams", "i.ai_est_grams"}, {"i.kcal", "i.ai_est_kcal"}, {"i.protein_g", "i.ai_est_protein_g"},
		{"i.carbs_g", "i.ai_est_carbs_g"}, {"i.fat_g", "i.ai_est_fat_g"}}
	cols := ""
	for _, m := range measures {
		d := "(" + m[0] + " - " + m[1] + ")" // NULL when either side is NULL, so AVG and COUNT skip it
		cols += ", COUNT(" + d + "), AVG(" + d + "), AVG(ABS" + d + ")"
	}
	rows, err := r.db.Query(`SELECT date(m.local_date, 'weekday 0', '-6 days') AS week_start, COUNT(*)`+cols+`
 FROM nutrition_meal_items i JOIN nutrition_meals m ON m.id = i.meal_id
 WHERE i.archived_at IS NULL AND m.archived_at IS NULL AND m.created_via = 'camera' AND i.ai_est_grams IS NOT NULL
 AND m.local_date BETWEEN ? AND ?
 GROUP BY week_start ORDER BY week_start`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.AIAccuracyWeek{}
	for rows.Next() {
		var w models.AIAccuracyWeek
		var n [5]int
		var bias, mae [5]sql.NullFloat64
		if err := rows.Scan(&w.WeekStart, &w.Items, &n[0], &bias[0], &mae[0], &n[1], &bias[1], &mae[1], &n[2], &bias[2], &mae[2],
			&n[3], &bias[3], &mae[3], &n[4], &bias[4], &mae[4]); err != nil {
			return nil, err
		}
		stat := func(i int) models.AIAccuracyStat {
			s := models.AIAccuracyStat{Paired: n[i]}
			if bias[i].Valid {
				s.Bias = &bias[i].Float64
			}
			if mae[i].Valid {
				s.MAE = &mae[i].Float64
			}
			return s
		}
		w.Grams, w.Kcal, w.ProteinG, w.CarbsG, w.FatG = stat(0), stat(1), stat(2), stat(3), stat(4)
		out = append(out, w)
	}
	return out, rows.Err()
}
