package handlers

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// The review script rounds the macros it shows from the grams as typed; the
// server rounds the grams to 0.1 first and stores its own figure (GYST-179).
// testdata/camera_rounding.json (fictional foods) is the one table both sides
// are checked against: Go here, the script's maths under node below. If either
// side drifts, an old cached screen would meet a false refusal.

type roundingRow struct {
	Food        string              `json:"food"`
	Per100      map[string]*float64 `json:"per100"`
	GramsTyped  string              `json:"grams_typed"`
	StoredGrams float64             `json:"stored_grams"`
	Posted      map[string]string   `json:"posted"`
	Stored      map[string]*float64 `json:"stored"`
}

var roundingKeys = []string{"kcal", "protein_g", "carbs_g", "fat_g", "fiber_g"}

func loadRounding(t *testing.T) []roundingRow {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "camera_rounding.json"))
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Rows []roundingRow `json:"rows"`
	}
	if err := json.Unmarshal(b, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) < 40 {
		t.Fatalf("golden table has %d rows", len(table.Rows))
	}
	return table.Rows
}

func nutrientsFrom(get func(k string) *float64) models.Nutrients {
	return models.Nutrients{Kcal: get("kcal"), ProteinG: get("protein_g"), CarbsG: get("carbs_g"), FatG: get("fat_g"), FiberG: get("fiber_g")}
}

func TestCameraRoundingGoldenTableServerSide(t *testing.T) {
	h := testHandler(t)
	foods := map[string]models.NutritionFood{}
	for _, row := range loadRounding(t) {
		f, ok := foods[row.Food]
		if !ok {
			var err error
			f, err = h.Service.Repo.UpsertNutritionFoodVersion(context.Background(), models.NutritionFood{
				Provider: "cnf", ProviderID: "golden-" + row.Food, Name: "Golden " + row.Food, NormalizedName: "golden " + row.Food, Basis: "per_100g",
				Nutrients: nutrientsFrom(func(k string) *float64 { return row.Per100[k] }), DataProvenance: "database", DatasetVersion: "test",
			}, clock.FormatInstant(h.Service.Now()))
			if err != nil {
				t.Fatal(err)
			}
			foods[row.Food] = f
		}
		grams, err := strconv.ParseFloat(row.GramsTyped, 64)
		if err != nil {
			t.Fatal(err)
		}
		// A request built exactly as the old script builds it: the macros the
		// script showed, parsed back from their text, and the grams as typed.
		posted := nutrientsFrom(func(k string) *float64 {
			if row.Posted[k] == "" {
				return nil
			}
			v, err := strconv.ParseFloat(row.Posted[k], 64)
			if err != nil {
				t.Fatal(err)
			}
			return &v
		})
		ci := services.CameraItem{Name: row.Food, Grams: grams, Nutrients: posted}
		if err := h.linkPrefill(&ci, "nf:"+strconv.FormatInt(f.ID, 10)); err != nil {
			t.Fatalf("%s at %s g: old-script request refused: %v", row.Food, row.GramsTyped, err)
		}
		if ci.FoodID == nil || ci.NutritionProv != "database" {
			t.Fatalf("%s at %s g: not linked", row.Food, row.GramsTyped)
		}
		if got := services.Round1(grams); got != row.StoredGrams {
			t.Errorf("%s: grams %v stored as %v, want %v", row.GramsTyped, grams, got, row.StoredGrams)
		}
		for _, p := range [][2]*float64{{ci.Nutrients.Kcal, row.Stored["kcal"]}, {ci.Nutrients.ProteinG, row.Stored["protein_g"]},
			{ci.Nutrients.CarbsG, row.Stored["carbs_g"]}, {ci.Nutrients.FatG, row.Stored["fat_g"]}, {ci.Nutrients.FiberG, row.Stored["fiber_g"]}} {
			if (p[0] == nil) != (p[1] == nil) || (p[0] != nil && *p[0] != *p[1]) {
				t.Errorf("%s at %s g: stored %v, want %v", row.Food, row.GramsTyped, deref(p[0]), deref(p[1]))
			}
		}
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// The script's own maths run under node against the same table. Missing node
// fails the test instead of skipping: a skipped check is how the two sides
// would drift unseen (CI installs node for the other scripts/check-*.cjs).
func TestCameraRoundingGoldenTableScriptMaths(t *testing.T) {
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
  for (const k of ['kcal', 'protein_g', 'carbs_g', 'fat_g', 'fiber_g']) {
    const got = M.prefill(r.per100[k], g);
    if (got !== r.posted[k]) { bad++; console.log('MISMATCH', r.food, r.grams_typed, k, got, r.posted[k]); }
  }
}
console.log(bad === 0 ? 'ok ' + rows.length : 'bad ' + bad);
process.exit(bad === 0 ? 0 : 1);
`
	if err := os.WriteFile(runner, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}
	math, _ := filepath.Abs(filepath.Join("..", "..", "web", "static", "js", "food-camera-math.js"))
	table, _ := filepath.Abs(filepath.Join("testdata", "camera_rounding.json"))
	out, err := exec.Command(node, runner, math, table).CombinedOutput()
	if err != nil {
		t.Fatalf("node check failed: %v\n%s", err, out)
	}
	t.Logf("node: %s", out)
}

// Codex round 2 on the review script (GYST-179): the pure logic the screen
// relies on, run under node like the golden table. The DOM wiring is covered by
// the browser pass recorded in the PR; these pin the rules.
func TestCameraScriptLogic(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node is required for the script-logic check: %v", err)
	}
	runner := filepath.Join(t.TempDir(), "logic.js")
	const js = `const assert = require('assert');
const M = require(process.argv[2]);

// 1. Typing a nutrient only notes a ratio; "0.05" stays exactly as typed
// because nothing computes a value to write back into the field.
assert.strictEqual(M.typedRatio('0.05', 100), 0.0005);
assert.strictEqual(M.typedRatio('15,5', 100), 0.155);   // a comma decimal
assert.strictEqual(M.typedRatio('abc', 100), null);
assert.strictEqual(M.typedRatio('', 100), null);
assert.strictEqual(M.typedRatio('12', NaN), null);        // no grams yet: nothing to note
assert.strictEqual(M.typedRatio('12', 0), null);
// Following only touches fields that have a ratio; the others are left alone.
const only = M.followValues({kcal: 2}, 110);
assert.deepStrictEqual(Object.keys(only), ['kcal']);
assert.strictEqual(only.kcal, '220');

// 2. Nutrients typed before grams are captured at the grams as they stand
// before a step or the slider, then scale. 200 kcal typed, grams 100 typed,
// stepper to 110: 220. The slider at x1.2 after that: base 110 -> 132 g.
let ratio = M.captureRatios({}, {kcal: '200', protein_g: '', carbs_g: 'x'}, 100);
assert.deepStrictEqual(Object.keys(ratio), ['kcal']);
assert.strictEqual(ratio.kcal, 2);
assert.strictEqual(M.followValues(ratio, 110).kcal, '220');
assert.strictEqual(M.followValues(ratio, 132).kcal, '264');
// An existing ratio is never replaced by what a field shows after rounding.
ratio = M.captureRatios({kcal: 2.0004}, {kcal: '200'}, 100);
assert.strictEqual(ratio.kcal, 2.0004);
// No grams: nothing is captured (grams typed digit by digit must not fix "1" of "150").
assert.deepStrictEqual(M.captureRatios({}, {kcal: '200'}, NaN), {});
// The input object is not changed.
const before = {kcal: 1}; M.captureRatios(before, {fat_g: '5'}, 10);
assert.deepStrictEqual(before, {kcal: 1});

// 3. Search answers: only the newest request id counts; a changed query, row
// or mode invalidates everything still in flight.
const req = M.latest();
const a = req.next();
assert.ok(req.isCurrent(a));
const b = req.next();
assert.ok(!req.isCurrent(a) && req.isCurrent(b));
req.invalidate();                       // chooser reopened for another row, or closed
assert.ok(!req.isCurrent(b));
const c = req.next();
assert.ok(req.isCurrent(c) && !req.isCurrent(a));
console.log('ok');
`
	if err := os.WriteFile(runner, []byte(js), 0o600); err != nil {
		t.Fatal(err)
	}
	math, _ := filepath.Abs(filepath.Join("..", "..", "web", "static", "js", "food-camera-math.js"))
	if out, err := exec.Command(node, runner, math).CombinedOutput(); err != nil {
		t.Fatalf("script logic check failed: %v\n%s", err, out)
	}
}
