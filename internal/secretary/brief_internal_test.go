package secretary

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBriefCapsFit(t *testing.T) {
	sum := capNow + capPrayer + capTasks + capCalendar + capHealth + capTraining + capMail + capSync
	if sum+briefOverhead > BriefBudget {
		t.Fatalf("caps %d + overhead %d exceed %d", sum, briefOverhead, BriefBudget)
	}
}

// The final guard drops whole sections from the bottom and names them. now is
// never dropped, and no section is shortened.
func TestBriefGuardDropsWholeSectionsBottomUp(t *testing.T) {
	section := func(n int) json.RawMessage { return mustJSON(map[string]string{"v": strings.Repeat("x", n)}) }
	b := dayBrief{Now: section(10), Prayer: section(100), Tasks: section(100), Calendar: section(100), Health: section(100), Training: section(100), Mail: section(100), Sync: section(100)}
	caps := map[string]int{"prayer": 500, "tasks": 500, "calendar": 500, "health": 500, "training": 500, "mail": 500, "sync": 500}
	for _, tc := range []struct {
		limit int
		cut   string
	}{{10000, ""}, {700, "sync,mail"}, {250, "sync,mail,training,health,calendar,tasks"}, {1, "sync,mail,training,health,calendar,tasks,prayer"}} {
		out, err := finalize(b, caps, tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatal(err)
		}
		if _, ok := m["now"]; !ok {
			t.Fatalf("limit %d: now was dropped", tc.limit)
		}
		if tc.cut == "" {
			if _, has := m["cut"]; has || len(out) != len(mustMarshal(t, b)) {
				t.Fatalf("limit %d: cut %v", tc.limit, m["cut"])
			}
			continue
		}
		var cut []string
		for _, c := range m["cut"].([]any) {
			cut = append(cut, c.(string))
		}
		if strings.Join(cut, ",") != tc.cut {
			t.Fatalf("limit %d: cut %v, want %s", tc.limit, cut, tc.cut)
		}
		// Whole sections only: every section kept is byte-identical to the original.
		for _, name := range sectionOrder {
			if v, ok := m[name]; ok && name != "now" {
				if got, _ := json.Marshal(v); string(got) != string(*b.slot(name)) {
					t.Fatalf("section %s was changed", name)
				}
			}
		}
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A section over its own cap is dropped whole and named, not shortened.
func TestBriefSectionOverItsCapIsDropped(t *testing.T) {
	big := mustJSON(map[string]string{"v": strings.Repeat("x", 400)})
	b := dayBrief{Now: mustJSON(map[string]string{"v": "n"}), Mail: big}
	out, err := finalize(b, map[string]int{"mail": capMail}, BriefBudget)
	if err != nil || !strings.Contains(string(out), `"cut":["mail"]`) || strings.Contains(string(out), "xxxx") {
		t.Fatalf("%s %v", out, err)
	}
}

// GYST-180: the health section stays inside its cap with food.ai_estimated and
// every other value at its longest.
func TestBriefHealthWithAIEstimatedFitsItsCap(t *testing.T) {
	s := func(v string) *string { return &v }
	n := 15
	h := HealthBrief{Stale: true, FoodEatenKcal: s("12,345"), FoodTargetKcal: s("12,345"), FoodBudgetLeftKcal: s("12,345"), FoodOverBudgetByKcal: s("12,345"),
		ProteinEatenG: s("1,234"), ProteinTargetG: s("1,234"), Steps: s("123,456"), StepsSource: "phone", BurnedSoFarKcal: s("12,345"), EnergyBalanceSoFarKcal: s("−12,345"),
		WaterLoggedL: s("12.5"), WaterTargetL: s("12.5"), WeightKg: s("123.4"),
		FoodAIEstimatedItems: &n, FoodAIEstimatedKcal: s("12,345"), FoodAIEstimatedProteinG: s("1,234")}
	raw := briefHealth(h, true, true, func(time.Time) any { return "2026-11-24T23:59" })
	if len(raw) > capHealth {
		t.Fatalf("health section %d bytes, cap %d: %s", len(raw), capHealth, raw)
	}
}
