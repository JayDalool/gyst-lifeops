package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// GYST-180 / GYST-192: an AI estimate holds nutrition for the grams the AI gave.
// The review script scales it to the grams as typed ((v*g)/estGrams, to 0.1);
// the server accepts the posted values as "unedited" only when they equal its
// own figure to the last digit, then stores the estimate scaled to the grams
// rounded to 0.1. testdata/camera_ai_rounding.json (fictional) is the one table
// both sides are checked against, so a rounding drift fails here and not on a
// phone as an item quietly saved as "user".

type aiRoundingRow struct {
	EstGrams    float64            `json:"est_grams"`
	Est         map[string]float64 `json:"est"`
	GramsTyped  string             `json:"grams_typed"`
	StoredGrams float64            `json:"stored_grams"`
	Posted      map[string]string  `json:"posted"`
	Stored      map[string]float64 `json:"stored"`
}

func loadAIRounding(t *testing.T) []aiRoundingRow {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "camera_ai_rounding.json"))
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Rows []aiRoundingRow `json:"rows"`
	}
	if err := json.Unmarshal(b, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) < 40 {
		t.Fatalf("golden table has %d rows", len(table.Rows))
	}
	return table.Rows
}

func TestCameraAIRoundingGoldenTableServerSide(t *testing.T) {
	h := testHandler(t)
	for i, row := range loadAIRounding(t) {
		grams, _ := strconv.ParseFloat(row.GramsTyped, 64)
		pick := func(k string) *float64 {
			v, err := strconv.ParseFloat(row.Posted[k], 64)
			if err != nil {
				t.Fatal(err)
			}
			return &v
		}
		f := func(k string) *float64 { v := row.Est[k]; return &v }
		est := row.EstGrams
		res, err := h.Service.AddCameraMeal(context.Background(), services.DiaryActor{}, services.CameraMeal{
			IdempotencyKey: fmt.Sprintf("%032x", i+1), MealSlot: "dinner",
			Items: []services.CameraItem{{Name: "Test stew", Grams: grams, AIEstGrams: &est,
				AIEstNutrients: &models.Nutrients{Kcal: f("kcal"), ProteinG: f("protein_g"), CarbsG: f("carbs_g"), FatG: f("fat_g")},
				Nutrients:      models.Nutrients{Kcal: pick("kcal"), ProteinG: pick("protein_g"), CarbsG: pick("carbs_g"), FatG: pick("fat_g")}}},
		})
		if err != nil {
			t.Fatalf("row %d (%v g of %v, typed %s): refused: %v", i, est, row.Est, row.GramsTyped, err)
		}
		items, _ := h.Service.ListDiaryItems(h.Service.TodayString())
		var it models.DiaryItem
		for _, c := range items {
			if c.MealID == res.MealID {
				it = c
			}
		}
		if it.NutritionProvenance != "ai_estimate" {
			t.Fatalf("row %d (%s g, est %v g): script's own figures saved as %q", i, row.GramsTyped, est, it.NutritionProvenance)
		}
		if *it.Grams != row.StoredGrams {
			t.Errorf("row %d: grams %v want %v", i, *it.Grams, row.StoredGrams)
		}
		for k, got := range map[string]*float64{"kcal": it.Kcal, "protein_g": it.ProteinG, "carbs_g": it.CarbsG, "fat_g": it.FatG} {
			if *got != row.Stored[k] {
				t.Errorf("row %d %s at %s g: stored %v, want %v", i, k, row.GramsTyped, *got, row.Stored[k])
			}
		}
		// Every row: the shared rule (models.EstimateHolds, behind the day total
		// and the diary rows) sees all four stored values as still the estimate.
		// A changed formula order makes some row of the table fail here.
		if e := it.Estimated(); !(e.Kcal && e.ProteinG && e.CarbsG && e.FatG) {
			t.Errorf("row %d (%s g, est %v g): EstimateHolds on the stored item: %+v", i, row.GramsTyped, est, e)
		}
		for k, got := range map[string]*float64{"kcal": it.Kcal, "protein_g": it.ProteinG, "carbs_g": it.CarbsG, "fat_g": it.FatG} {
			e := row.Est[k]
			if !models.EstimateHolds(got, &e, *it.Grams, est) {
				t.Errorf("row %d %s: EstimateHolds false on stored %v", i, k, *got)
			}
		}
		// The AI's own numbers are kept exactly as given.
		if *it.AIEstKcal != row.Est["kcal"] || *it.AIEstFatG != row.Est["fat_g"] {
			t.Errorf("row %d: kept estimate changed", i)
		}
	}
}

func TestCameraAIRoundingGoldenTableScriptMaths(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the golden-table check of web/static/js/food-camera-math.js: %v", err)
	}
	runner := filepath.Join(t.TempDir(), "check.js")
	const js = `const M = require(process.argv[2]);
const rows = JSON.parse(require('fs').readFileSync(process.argv[3], 'utf8')).rows;
let bad = 0;
for (const r of rows) {
  const g = parseFloat(r.grams_typed.replace(',', '.'));
  for (const k of ['kcal', 'protein_g', 'carbs_g', 'fat_g']) {
    const got = M.prefillEst(r.est[k], r.est_grams, g);
    if (got !== r.posted[k]) { bad++; console.log('MISMATCH', r.est_grams, r.grams_typed, k, got, r.posted[k]); }
  }
}
// estimateHolds compares parsed numbers like the server (Codex round b).
const holds = [['350', 350, 200, 200, true], ['350.0', 350, 200, 200, true], ['350,0', 350, 200, 200, true], ['350.04', 350, 200, 200, false],
  ['175', 350, 200, 100, true], ['', 350, 200, 200, false], ['x', 350, 200, 200, false], ['0.1', 0.05, 100, 100, true], ['350', null, 200, 200, false], ['350', 350, 200, 0, false]];
for (const [t, v, e, g, want] of holds) { if (M.estimateHolds(t, v, e, g) !== want) { bad++; console.log('HOLDS', t, v, e, g, want); } }
// Codex round c: cleared or zero grams blank the AI-filled fields only.
const kept = M.followAI({kcal: true, protein_g: true}, {kcal: 350, protein_g: 14, carbs_g: 40}, 250, 125);
if (kept.kcal !== '175' || kept.protein_g !== '7' || 'carbs_g' in kept) { bad++; console.log('FOLLOW', JSON.stringify(kept)); }
for (const g of [NaN, 0, -5]) {
  const blank = M.followAI({kcal: true, protein_g: true}, {kcal: 350, protein_g: 14}, 250, g);
  if (blank.kcal !== '' || blank.protein_g !== '' || Object.keys(blank).length !== 2) { bad++; console.log('BLANK', g, JSON.stringify(blank)); }
}
if (Object.keys(M.followAI({}, {kcal: 350}, 250, NaN)).length !== 0) { bad++; console.log('owner fields touched'); }
// The script's rule agrees on every row: what the review shows for the typed
// grams is still "the estimate".
for (const r of rows) {
  const g = parseFloat(r.grams_typed.replace(',', '.'));
  for (const k of ['kcal', 'protein_g', 'carbs_g', 'fat_g']) {
    if (!M.estimateHolds(r.posted[k], r.est[k], r.est_grams, g)) { bad++; console.log('ROW HOLDS', r.est_grams, r.grams_typed, k, r.posted[k]); }
  }
}
// The day-total sentence and its whole-number rounding match the Go side
// (handlers.estimateNote, formatThousands).
const notes = [[0, 410, ''], [2, 410, 'Includes AI estimates: 2 items, about 410 kcal'], [1, 1234.5, 'Includes AI estimates: 1 item, about 1,235 kcal'], [3, null, 'Includes AI estimates: 3 items'], [2, 0.4, 'Includes AI estimates: 2 items, about 0 kcal']];
for (const [n, k, want] of notes) { const got = M.estimateNote(n, k); if (got !== want) { bad++; console.log('NOTE', n, k, got, want); } }
console.log(bad === 0 ? 'ok ' + rows.length : 'bad ' + bad);
process.exit(bad === 0 ? 0 : 1);
`
	if err := os.WriteFile(runner, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}
	math, _ := filepath.Abs(filepath.Join("..", "..", "web", "static", "js", "food-camera-math.js"))
	table, _ := filepath.Abs(filepath.Join("testdata", "camera_ai_rounding.json"))
	out, err := exec.Command(node, runner, math, table).CombinedOutput()
	if err != nil {
		t.Fatalf("node check failed: %v\n%s", err, out)
	}
	t.Logf("node: %s", out)
}
