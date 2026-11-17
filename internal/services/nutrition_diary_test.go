package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

// Fictional values only: "Test oats", 40 g and the like.

func diaryService(t *testing.T) (*Service, *time.Time) {
	t.Helper()
	s := testService(t)
	cur := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC) // 09:00 local
	s.Now = func() time.Time { return cur }
	return s, &cur
}

func catalogueFood(t *testing.T, s *Service, id, name string, kcal, protein, fiber *float64) models.NutritionFood {
	t.Helper()
	f, err := s.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
		Provider: "usda_fdc", ProviderID: id, Name: name, NormalizedName: NormalizeFoodName(name), Basis: "per_100g",
		Nutrients:      models.Nutrients{Kcal: kcal, ProteinG: protein, FiberG: fiber},
		DataProvenance: "database",
	}, "2026-11-16T15:00:00.000000000Z")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

var keyCounter int

func newKey() string {
	keyCounter++
	return fmt.Sprintf("%032x", keyCounter)
}

func addOats(t *testing.T, s *Service, food models.NutritionFood, grams float64) DiaryResult {
	t.Helper()
	res, err := s.AddDiaryItem(context.Background(), DiaryActor{}, models.NewDiaryItem{
		IdempotencyKey: newKey(), MealSlot: "breakfast", FoodID: food.ID, Unit: "g", Amount: grams,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestDiaryDayCountsGYSTAndKeepsAppleHealthAsReference(t *testing.T) {
	s, cur := diaryService(t)
	date := s.TodayString()
	// Apple Health synced 1800 kcal for today; no diary yet.
	if _, err := s.SyncHealth(models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{DietaryCalories: fp(1800), ProteinG: fp(90)}}); err != nil {
		t.Fatal(err)
	}
	in, err := s.DayIntake(date)
	if err != nil || in.Source != models.IntakeSourceAppleHealth || *in.Counted.Kcal != 1800 {
		t.Fatalf("no diary: %+v %v", in, err)
	}

	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), fp(13), nil)
	res := addOats(t, s, oats, 40) // 152 kcal, 5.2 g protein
	in, err = s.DayIntake(date)
	if err != nil {
		t.Fatal(err)
	}
	if in.Source != models.IntakeSourceGYST || in.Items != 1 || *in.Counted.Kcal != 152 {
		t.Fatalf("gyst day must count only the diary, never the sum: %+v", in)
	}
	if in.Reference == nil || *in.Reference.Kcal != 1800 {
		t.Fatalf("synced total must be the reference: %+v", in.Reference)
	}
	// The same rule drives HealthToday: eaten kcal and food remaining use the diary.
	cycle := cycleFixture()
	cycle.StartDate, cycle.EndDate, cycle.DailyCalorieTarget = "2026-11-01", "2027-01-31", 2000
	if err := s.CreateHealthCycle(&cycle); err != nil {
		t.Fatal(err)
	}
	today, err := s.HealthToday()
	if err != nil || today.Intake.Source != models.IntakeSourceGYST {
		t.Fatalf("health today: %+v %v", today.Intake, err)
	}
	if today.Observed.FoodRemaining == nil || *today.Observed.FoodRemaining != 2000-152 {
		t.Fatalf("food remaining = %v, want 1848 from the diary", today.Observed.FoodRemaining)
	}
	// The synced row itself is untouched (sync contract and upsert unchanged).
	if today.Snapshot == nil || *today.Snapshot.DietaryCalories != 1800 {
		t.Fatalf("snapshot changed: %+v", today.Snapshot)
	}

	// Remove: totals update at once and the day falls back to Apple Health.
	if err := s.ArchiveDiaryItem(context.Background(), DiaryActor{}, res.ItemID); err != nil {
		t.Fatal(err)
	}
	if in, _ = s.DayIntake(date); in.Source != models.IntakeSourceAppleHealth || *in.Counted.Kcal != 1800 {
		t.Fatalf("after remove: %+v", in)
	}
	// Undo within the window restores it.
	*cur = cur.Add(9 * time.Minute)
	if err := s.RestoreDiaryItem(context.Background(), DiaryActor{}, res.ItemID); err != nil {
		t.Fatal(err)
	}
	if in, _ = s.DayIntake(date); in.Source != models.IntakeSourceGYST || *in.Counted.Kcal != 152 {
		t.Fatalf("after restore: %+v", in)
	}
}

func TestDiaryRestoreOnlyInsideWindow(t *testing.T) {
	s, cur := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	res := addOats(t, s, oats, 40)
	ctx := context.Background()
	if err := s.ArchiveDiaryItem(ctx, DiaryActor{}, res.ItemID); err != nil {
		t.Fatal(err)
	}
	*cur = cur.Add(DiaryUndoWindow + time.Second)
	if err := s.RestoreDiaryItem(ctx, DiaryActor{}, res.ItemID); !errors.Is(err, ErrDiaryItemNotFound) {
		t.Fatalf("restore after the window: %v", err)
	}
	// A live item cannot be "restored" either.
	other := addOats(t, s, oats, 10)
	if err := s.RestoreDiaryItem(ctx, DiaryActor{}, other.ItemID); !errors.Is(err, ErrDiaryItemNotFound) {
		t.Fatalf("restore of a live item: %v", err)
	}
	// Archiving twice is refused, not silently repeated.
	if err := s.ArchiveDiaryItem(ctx, DiaryActor{}, res.ItemID); !errors.Is(err, ErrDiaryItemNotFound) {
		t.Fatalf("second archive: %v", err)
	}
}

func TestDiaryPastDaysAreRefused(t *testing.T) {
	s, cur := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	res := addOats(t, s, oats, 40)
	*cur = cur.Add(24 * time.Hour) // the next local day
	ctx := context.Background()
	var inv *HealthValidationError
	if err := s.UpdateDiaryItemAmount(ctx, DiaryActor{}, res.ItemID, 80); !errors.As(err, &inv) || !strings.Contains(err.Error(), "today") {
		t.Fatalf("update past day: %v", err)
	}
	if err := s.ArchiveDiaryItem(ctx, DiaryActor{}, res.ItemID); !errors.As(err, &inv) {
		t.Fatalf("archive past day: %v", err)
	}
	prev, _ := s.DayIntake("2026-11-16")
	if prev.Source != models.IntakeSourceGYST || *prev.Counted.Kcal != 152 {
		t.Fatalf("past day changed: %+v", prev)
	}
}

func TestDiaryEditingCatalogueDoesNotChangeLoggedTotals(t *testing.T) {
	s, _ := diaryService(t)
	date := s.TodayString()
	v1 := catalogueFood(t, s, "oats-1", "Test oats", fp(380), fp(13), nil)
	res := addOats(t, s, v1, 100)
	// The catalogue food is edited: a new version supersedes the old one.
	v2 := catalogueFood(t, s, "oats-1", "Test oats", fp(500), fp(20), nil)
	if v2.Version != 2 || v2.ID == v1.ID {
		t.Fatalf("expected version 2: %+v", v2)
	}
	in, _ := s.DayIntake(date)
	if *in.Counted.Kcal != 380 || *in.Counted.ProteinG != 13 {
		t.Fatalf("logged totals changed with the catalogue: %+v", in.Counted)
	}
	// Editing the amount rescales from the version that was logged.
	if err := s.UpdateDiaryItemAmount(context.Background(), DiaryActor{}, res.ItemID, 50); err != nil {
		t.Fatal(err)
	}
	in, _ = s.DayIntake(date)
	if *in.Counted.Kcal != 190 {
		t.Fatalf("rescaled from the wrong version: %v", *in.Counted.Kcal)
	}
	it, _ := s.Repo.GetDiaryItem(res.ItemID)
	if *it.FoodVersion != 1 || *it.FoodID != v1.ID {
		t.Fatalf("item must keep pointing at the logged version: %+v", it)
	}
	// A superseded version can no longer be logged.
	_, err := s.AddDiaryItem(context.Background(), DiaryActor{}, models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "lunch", FoodID: v1.ID, Unit: "g", Amount: 10})
	var inv *HealthValidationError
	if !errors.As(err, &inv) {
		t.Fatalf("superseded food accepted: %v", err)
	}
	// Re-upserting identical content makes no new version.
	if again := catalogueFood(t, s, "oats-1", "Test oats", fp(500), fp(20), nil); again.ID != v2.ID {
		t.Fatalf("unchanged food got a new version: %+v", again)
	}
}

func TestDiaryNullNutrientsStayNullAndMarkPartial(t *testing.T) {
	s, _ := diaryService(t)
	date := s.TodayString()
	noFiber := catalogueFood(t, s, "a", "Test oats", fp(380), fp(13), nil)
	withFiber := catalogueFood(t, s, "b", "Test bran", fp(200), fp(10), fp(30))
	addOats(t, s, noFiber, 100)
	in, _ := s.DayIntake(date)
	if in.Counted.FiberG != nil || in.Missing.FiberG != 1 {
		t.Fatalf("all-NULL fiber must stay NULL, not 0: %+v %+v", in.Counted, in.Missing)
	}
	addOats(t, s, withFiber, 10)
	in, _ = s.DayIntake(date)
	if in.Counted.FiberG == nil || *in.Counted.FiberG != 3 || in.Missing.FiberG != 1 || in.Missing.Kcal != 0 {
		t.Fatalf("partial fiber: %+v %+v", in.Counted, in.Missing)
	}
}

func TestDiaryIdempotentSubmit(t *testing.T) {
	s, _ := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	ctx := context.Background()
	key := newKey()
	in := models.NewDiaryItem{IdempotencyKey: key, MealSlot: "breakfast", FoodID: oats.ID, Unit: "g", Amount: 40}
	first, err := s.AddDiaryItem(ctx, DiaryActor{}, in)
	if err != nil || !first.Created {
		t.Fatalf("first: %+v %v", first, err)
	}
	second, err := s.AddDiaryItem(ctx, DiaryActor{}, in)
	if err != nil || second.Created || second.MealID != first.MealID || second.ItemID != first.ItemID {
		t.Fatalf("double submit must return the same meal: %+v %v", second, err)
	}
	items, _ := s.ListDiaryItems(s.TodayString())
	if len(items) != 1 {
		t.Fatalf("double submit created %d items", len(items))
	}
	if n := countAudit(t, s, AuditActionNutritionMealCreate); n != 1 {
		t.Fatalf("duplicate submit wrote %d create audit rows, want 1", n)
	}
}

func TestDiaryIdempotencyUnderConcurrency(t *testing.T) {
	s, _ := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	key := newKey()
	var wg sync.WaitGroup
	results := make([]DiaryResult, 8)
	errs := make([]error, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = s.AddDiaryItem(context.Background(), DiaryActor{}, models.NewDiaryItem{IdempotencyKey: key, MealSlot: "breakfast", FoodID: oats.ID, Unit: "g", Amount: 40})
		}(i)
	}
	wg.Wait()
	created := 0
	for i, r := range results {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if r.MealID != results[0].MealID || r.ItemID != results[0].ItemID {
			t.Fatalf("different meals for one key: %+v vs %+v", r, results[0])
		}
		if r.Created {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("%d goroutines created the meal, want exactly 1", created)
	}
	if items, _ := s.ListDiaryItems(s.TodayString()); len(items) != 1 {
		t.Fatalf("%d items, want 1", len(items))
	}
	if n := countAudit(t, s, AuditActionNutritionMealCreate); n != 1 {
		t.Fatalf("%d create audit rows, want 1", n)
	}
}

func TestDiaryValidation(t *testing.T) {
	s, _ := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	ctx := context.Background()
	base := func() models.NewDiaryItem {
		return models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "lunch", FoodID: oats.ID, Unit: "g", Amount: 40}
	}
	bad := map[string]func(*models.NewDiaryItem){
		"short key":                         func(i *models.NewDiaryItem) { i.IdempotencyKey = "abc" },
		"uppercase key":                     func(i *models.NewDiaryItem) { i.IdempotencyKey = strings.Repeat("A", 32) },
		"unknown slot":                      func(i *models.NewDiaryItem) { i.MealSlot = "brunch" },
		"unknown via":                       func(i *models.NewDiaryItem) { i.CreatedVia = "telepathy" },
		"unknown quantity":                  func(i *models.NewDiaryItem) { i.QuantityProvenance = "guess" },
		"unknown unit":                      func(i *models.NewDiaryItem) { i.Unit = "oz" },
		"zero amount":                       func(i *models.NewDiaryItem) { i.Amount = 0 },
		"negative amount":                   func(i *models.NewDiaryItem) { i.Amount = -5 },
		"NaN amount":                        func(i *models.NewDiaryItem) { i.Amount = math.NaN() },
		"huge amount":                       func(i *models.NewDiaryItem) { i.Amount = 1e9 },
		"unknown food":                      func(i *models.NewDiaryItem) { i.FoodID = 99999 },
		"servings without a serving weight": func(i *models.NewDiaryItem) { i.Unit = "serving" },
	}
	for name, mutate := range bad {
		in := base()
		mutate(&in)
		_, err := s.AddDiaryItem(ctx, DiaryActor{}, in)
		var inv *HealthValidationError
		if !errors.As(err, &inv) {
			t.Errorf("%s: err = %v, want a validation error", name, err)
		}
	}
	if items, _ := s.ListDiaryItems(s.TodayString()); len(items) != 0 {
		t.Fatalf("a refused add wrote %d items", len(items))
	}
	if n := countAudit(t, s, AuditActionNutritionMealCreate); n != 0 {
		t.Fatalf("a refused add wrote %d audit rows", n)
	}
	if err := s.UpdateDiaryItemAmount(ctx, DiaryActor{}, 99999, 5); !errors.Is(err, ErrDiaryItemNotFound) {
		t.Fatalf("update unknown item: %v", err)
	}
	if err := s.UpdateDiaryItemAmount(ctx, DiaryActor{}, 1, 0); err == nil {
		t.Fatal("zero amount accepted on update")
	}
}

func TestDiaryAuditHasDecisionsNeverData(t *testing.T) {
	s, cur := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), fp(13), nil)
	ctx := context.Background()
	actor := DiaryActor{Ref: "session:ab12cd34", Ingress: AuditIngressTailnet, RequestID: "abc123"}
	res, err := s.AddDiaryItem(ctx, actor, models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "breakfast", FoodID: oats.ID, Unit: "g", Amount: 41})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateDiaryItemAmount(ctx, actor, res.ItemID, 42); err != nil {
		t.Fatal(err)
	}
	if err := s.ArchiveDiaryItem(ctx, actor, res.ItemID); err != nil {
		t.Fatal(err)
	}
	*cur = cur.Add(time.Minute)
	if err := s.RestoreDiaryItem(ctx, actor, res.ItemID); err != nil {
		t.Fatal(err)
	}
	events, err := s.Repo.RecentAuditEvents(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]int{}
	for _, e := range events {
		actions[e.Action]++
		for _, forbidden := range []string{"oats", "Oats", "41", "42", "380"} {
			if strings.Contains(e.Detail, forbidden) || strings.Contains(e.TargetRef, forbidden) {
				t.Errorf("audit row leaks data %q: %+v", forbidden, e)
			}
		}
		if e.Ingress != AuditIngressTailnet || e.ActorRef != "session:ab12cd34" || e.Outcome != AuditOutcomeSuccess {
			t.Errorf("unexpected audit row: %+v", e)
		}
	}
	if actions[AuditActionNutritionMealCreate] != 1 || actions[AuditActionNutritionMealUpdate] != 2 || actions[AuditActionNutritionMealArchive] != 1 {
		t.Fatalf("audit actions = %v", actions)
	}
}

func countAudit(t *testing.T, s *Service, action string) int {
	t.Helper()
	events, err := s.Repo.RecentAuditEvents(context.Background(), 500)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range events {
		if e.Action == action {
			n++
		}
	}
	return n
}

func TestSearchFoodsEscapesWildcardsAndMirrorsMyFoods(t *testing.T) {
	s, _ := diaryService(t)
	ctx := context.Background()
	catalogueFood(t, s, "a", "100% test oats", fp(380), nil, nil)
	catalogueFood(t, s, "b", "Test rice", fp(130), nil, nil)
	got, err := s.SearchFoods(ctx, "  TEST   oats ")
	if err != nil || len(got) != 1 || got[0].Name != "100% test oats" {
		t.Fatalf("search: %+v %v", got, err)
	}
	// % and _ are literal characters, not wildcards.
	if got, _ := s.SearchFoods(ctx, "%"); len(got) != 1 || got[0].ProviderID != "a" {
		t.Fatalf("%% matched %+v", got)
	}
	if got, _ := s.SearchFoods(ctx, "_"); len(got) != 0 {
		t.Fatalf("_ matched %+v", got)
	}
	if got, _ := s.SearchFoods(ctx, `\`); len(got) != 0 {
		t.Fatalf(`\ matched %+v`, got)
	}
	var inv *HealthValidationError
	if _, err := s.SearchFoods(ctx, "   "); !errors.As(err, &inv) {
		t.Fatalf("blank search: %v", err)
	}
	if _, err := s.SearchFoods(ctx, strings.Repeat("x", 81)); !errors.As(err, &inv) {
		t.Fatalf("long search: %v", err)
	}

	// My foods are found by reading health_foods; a search writes nothing.
	mine := models.HealthFood{Name: "Test shake", Serving: "1 bottle", Kcal: fp(180), ProteinG: fp(30), Active: true}
	if err := s.Repo.CreateHealthFood(&mine); err != nil {
		t.Fatal(err)
	}
	mirrored := func() []models.NutritionFood {
		rows, err := s.Repo.SearchNutritionFoods("shake", 50)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	first, err := s.SearchFoods(ctx, "shake")
	if err != nil || len(first) != 1 || first[0].Provider != "user" || first[0].Basis != "per_serving" ||
		first[0].ProviderID != fmt.Sprintf("health_food:%d", mine.ID) || first[0].Ref != fmt.Sprintf("hf:%d", mine.ID) {
		t.Fatalf("my food result: %+v %v", first, err)
	}
	if _, err := s.SearchFoods(ctx, "shake"); err != nil || len(mirrored()) != 0 {
		t.Fatalf("a search must not mirror: %d catalogue rows", len(mirrored()))
	}
	// Adding mirrors it inside the write: one version until the food changes.
	add := func(key string) DiaryResult {
		res, err := s.AddDiaryItem(ctx, DiaryActor{}, models.NewDiaryItem{IdempotencyKey: key, MealSlot: "snack", HealthFoodID: mine.ID, Unit: "serving", Amount: 1})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	k1 := newKey()
	res := add(k1)
	if rows := mirrored(); len(rows) != 1 || rows[0].Version != 1 {
		t.Fatalf("after add: %+v", rows)
	}
	add(k1) // same key: nothing more is written
	add(newKey())
	if rows := mirrored(); len(rows) != 1 || rows[0].Version != 1 {
		t.Fatalf("an unchanged food must not get a new version: %+v", rows)
	}
	mine.Kcal = fp(200)
	if err := s.Repo.UpdateHealthFood(mine); err != nil {
		t.Fatal(err)
	}
	if rows := mirrored(); len(rows) != 1 || rows[0].Version != 1 {
		t.Fatal("editing My foods alone must not write the catalogue")
	}
	if changed, _ := s.SearchFoods(ctx, "shake"); len(changed) != 1 || *changed[0].Kcal != 200 {
		t.Fatalf("search must show the current My food values: %+v", changed)
	}
	res2 := add(newKey())
	if rows := mirrored(); len(rows) != 1 || rows[0].Version != 2 || *rows[0].Kcal != 200 {
		t.Fatalf("changed food: %+v", rows)
	}
	old, _ := s.Repo.GetDiaryItem(res.ItemID)
	it, _ := s.Repo.GetDiaryItem(res2.ItemID)
	if it.Grams != nil || *it.Kcal != 200 || *old.Kcal != 180 {
		t.Fatalf("snapshots: old %+v new %+v", old, it)
	}
	// An inactive My food is not offered and cannot be logged.
	mine.Active = false
	if err := s.Repo.UpdateHealthFood(mine); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.SearchFoods(ctx, "shake"); len(got) != 0 {
		t.Fatalf("inactive food offered: %+v", got)
	}
	var inv2 *HealthValidationError
	if _, err := s.AddDiaryItem(ctx, DiaryActor{}, models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "snack", HealthFoodID: mine.ID, Unit: "serving", Amount: 1}); !errors.As(err, &inv2) {
		t.Fatalf("inactive food logged: %v", err)
	}
}

func TestDiaryOversizedScaledValuesAreValidationErrors(t *testing.T) {
	s, _ := diaryService(t)
	ctx := context.Background()
	oats := catalogueFood(t, s, "oats-big", "Test big oats", fp(380), fp(13), nil)
	var inv *HealthValidationError
	// 99,999 g of a 380 kcal/100 g food is 379,996 kcal: over the 100,000 kcal limit.
	_, err := s.AddDiaryItem(ctx, DiaryActor{}, models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "lunch", FoodID: oats.ID, Unit: "g", Amount: 99999})
	if !errors.As(err, &inv) || !strings.Contains(inv.Error(), "too large") {
		t.Fatalf("huge add: %v", err)
	}
	if items, _ := s.Repo.ListDiaryItems(s.TodayString()); len(items) != 0 {
		t.Fatal("a refused add wrote an item")
	}
	res := addOats(t, s, oats, 100)
	if err := s.UpdateDiaryItemAmount(ctx, DiaryActor{}, res.ItemID, 99999); !errors.As(err, &inv) || !strings.Contains(inv.Error(), "too large") {
		t.Fatalf("huge edit: %v", err)
	}
	it, _ := s.Repo.GetDiaryItem(res.ItemID)
	if it.Amount != 100 {
		t.Fatalf("a refused edit changed the item: %+v", it)
	}
	// My foods: a per-serving food with big values hits the same check.
	mine := models.HealthFood{Name: "Test feast", Serving: "1 plate", Kcal: fp(9000), Active: true}
	if err := s.Repo.CreateHealthFood(&mine); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddDiaryItem(ctx, DiaryActor{}, models.NewDiaryItem{IdempotencyKey: newKey(), MealSlot: "dinner", HealthFoodID: mine.ID, Unit: "serving", Amount: 20}); !errors.As(err, &inv) {
		t.Fatalf("huge My food add: %v", err)
	}
	if rows, _ := s.Repo.SearchNutritionFoods("feast", 5); len(rows) != 0 {
		t.Fatal("a refused My food add mirrored the food")
	}
}

func TestDiaryFractionalGramsAreStoredAsTyped(t *testing.T) {
	s, _ := diaryService(t)
	oats := catalogueFood(t, s, "oats-1", "Test oats", fp(380), nil, nil)
	res := addOats(t, s, oats, 40.123)
	it, err := s.Repo.GetDiaryItem(res.ItemID)
	if err != nil || it.Grams == nil || *it.Grams != 40.123 || it.Amount != 40.123 {
		t.Fatalf("fractional grams: %+v %v", it, err)
	}
}
