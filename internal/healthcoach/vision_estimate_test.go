package healthcoach

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

// GYST-180: the photo AI also estimates calories, protein, carbs and fat. The
// request changes only in the instruction and the schema; a bad estimate is
// dropped and the item stays. Fictional values only.

var estFields = []string{"est_kcal", "est_protein_g", "est_carbs_g", "est_fat_g"}

// itemJSON builds one reply item with the given raw estimate values (JSON
// text, so a test can send text, null or nothing).
func itemJSON(grams string, kcal, p, c, f string) string {
	field := func(k, v string) string {
		if v == "" {
			return ""
		}
		return fmt.Sprintf(`"%s":%s,`, k, v)
	}
	return `{"name":"test stew","role":"main","est_grams":` + grams + `,` +
		field("est_kcal", kcal) + field("est_protein_g", p) + field("est_carbs_g", c) + field("est_fat_g", f) +
		`"confidence":0.8,"visible":true,"notes":""}`
}

func parseOne(t *testing.T, item string) FoodPhotoItem {
	t.Helper()
	res, err := ParseFoodPhoto(`{"items":[` + item + `],"uncertainties":[],"questions":[]}`)
	if err != nil {
		t.Fatalf("estimate problem must not fail the reply: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("item dropped with its estimate: %+v", res.Items)
	}
	return res.Items[0]
}

func TestParseFoodPhotoKeepsValidEstimate(t *testing.T) {
	it := parseOne(t, itemJSON("200", "320", "20", "30", "12")) // 4*20+4*30+9*12 = 308
	if it.Est == nil || *it.EstGrams != 200 || it.Est.Kcal != 320 || it.Est.ProteinG != 20 || it.Est.CarbsG != 30 || it.Est.FatG != 12 {
		t.Fatalf("%+v", it)
	}
	// Water and tea: zeros are real values, not missing.
	if z := parseOne(t, itemJSON("250", "0", "0", "0", "0")); z.Est == nil || z.Est.Kcal != 0 {
		t.Fatalf("zero estimate dropped: %+v", z)
	}
	// Unknown extras on the item are ignored.
	if e := parseOne(t, strings.Replace(itemJSON("200", "320", "20", "30", "12"), `"notes":""`, `"notes":"","est_fibre":3`, 1)); e.Est == nil {
		t.Fatal("unknown extra dropped the estimate")
	}
}

// Every way an estimate can be wrong drops the estimate, keeps the item and is
// never an error (c4).
func TestParseFoodPhotoDropsBadEstimates(t *testing.T) {
	cases := map[string]string{
		"negative kcal":      itemJSON("200", "-1", "20", "30", "12"),
		"negative protein":   itemJSON("200", "320", "-0.1", "30", "12"),
		"text kcal":          itemJSON("200", `"320"`, "20", "30", "12"),
		"text NaN":           itemJSON("200", `"NaN"`, "20", "30", "12"),
		"bool":               itemJSON("200", "true", "20", "30", "12"),
		"kcal too high":      itemJSON("200", "100001", "20", "30", "12"),
		"protein too high":   itemJSON("3000", "400", "10001", "0", "0"),
		"one null":           itemJSON("200", "320", "20", "null", "12"),
		"one missing":        itemJSON("200", "320", "20", "30", ""),
		"all null":           itemJSON("200", "null", "null", "null", "null"),
		"est_grams null":     itemJSON("null", "320", "20", "30", "12"),
		"macros outweigh":    itemJSON("50", "320", "20", "30", "12"), // 62 g of macros in 50 g
		"kcal far too high":  itemJSON("200", "900", "20", "30", "12"),
		"kcal far too low":   itemJSON("200", "10", "20", "30", "12"),
		"kcal with no macro": itemJSON("200", "500", "0", "0", "0"),
	}
	for name, item := range cases {
		it := parseOne(t, item)
		if it.Est != nil {
			t.Errorf("%s: estimate kept: %+v", name, it.Est)
		}
		if it.Name != "test stew" {
			t.Errorf("%s: item changed", name)
		}
	}
	// The edge of the kcal band is kept: macros give 308, so up to 1.5x + 25.
	if it := parseOne(t, itemJSON("200", "487", "20", "30", "12")); it.Est == nil {
		t.Error("kcal inside the band dropped")
	}
	if it := parseOne(t, itemJSON("200", "488", "20", "30", "12")); it.Est != nil {
		t.Error("kcal just outside the band kept")
	}
	// A macro total of grams+1 is kept, above it is dropped.
	if it := parseOne(t, itemJSON("61", "320", "20", "30", "12")); it.Est == nil {
		t.Error("p+c+f = grams+1 dropped")
	}
}

func TestPlausibleEstimateRejectsNonFinite(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if plausibleEstimate(FoodEstimate{Kcal: v}, 100) || plausibleEstimate(FoodEstimate{ProteinG: v}, 100) {
			t.Errorf("%v accepted", v)
		}
	}
}

// Both providers get the same instruction and schema, the new ones, with the
// image, the model and the user sentence exactly as the goldens captured them.
func TestRequestsCarryEstimateInstructionAndSchema(t *testing.T) {
	for _, want := range []string{"est_kcal", "est_protein_g", "est_carbs_g", "est_fat_g", "est_grams", "AI estimates"} {
		if !strings.Contains(foodPhotoPrompt, want) {
			t.Errorf("instruction lacks %q", want)
		}
	}
	if strings.Contains(foodPhotoPrompt, "Do not give calories") {
		t.Error("the old ban on nutrition values is still in the instruction")
	}
	schema := foodPhotoSchema()
	item := schema["properties"].(map[string]any)["items"].(map[string]any)["items"].(map[string]any)
	props := item["properties"].(map[string]any)
	required := item["required"].([]string)
	for _, k := range estFields {
		typ, _ := json.Marshal(props[k])
		if string(typ) != `{"type":["number","null"]}` {
			t.Errorf("%s schema %s", k, typ)
		}
		found := false
		for _, r := range required {
			found = found || r == k
		}
		if !found {
			t.Errorf("%s not required (strict mode needs every key listed)", k)
		}
	}
	a, fa := newFake(t, textMessage("end_turn", goodVision))
	if _, err := a.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	if got, _ := json.Marshal(fa.requests[0]["system"]); !strings.Contains(string(got), "est_kcal") {
		t.Error("Anthropic system lacks the estimate instruction")
	}
	o, fo := newOAIFake(t, jsonReply(200, visionReply(goodVision)))
	if _, err := o.AnalyzeFoodPhoto(context.Background(), fakeJPEG); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fo.requests[0]["instructions"].(string), "est_kcal") {
		t.Error("OpenAI instructions lack the estimate instruction")
	}
}
