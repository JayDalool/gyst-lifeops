package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func coachWaterJSON(t *testing.T, v healthWaterView) string {
	t.Helper()
	b, err := json.Marshal(coachToday(HealthData{Water: v})["water"])
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCoachWaterSummary(t *testing.T) {
	const note = `"note":"Logged by bottle taps on Today; taps_today counts taps, not litres. null means nothing logged, not zero. GYST stops adding bottles at the target, so the real amount may be higher. Nothing is estimated from food or Apple Health."`
	plan := func(target, bottle int64) *models.HealthCycle {
		return &models.HealthCycle{WaterTargetML: target, WaterBottleML: bottle}
	}
	cases := []struct {
		name  string
		cycle *models.HealthCycle
		total int64
		taps  int
		want  string
	}{
		{"no plan", nil, 0, 0, `{"bottle_l":null,"logged_l":null,` + note + `,"taps_today":0,"target_l":null,"target_reached":null}`},
		{"target 0 with bottle", plan(0, 900), 0, 0, `{"bottle_l":"0.9","logged_l":null,` + note + `,"taps_today":0,"target_l":null,"target_reached":null}`},
		{"target, no taps", plan(2700, 900), 0, 0, `{"bottle_l":"0.9","logged_l":null,` + note + `,"taps_today":0,"target_l":"2.7","target_reached":false}`},
		{"one of three bottles", plan(2700, 900), 900, 1, `{"bottle_l":"0.9","logged_l":"0.9",` + note + `,"taps_today":1,"target_l":"2.7","target_reached":false}`},
		{"at target", plan(2700, 900), 2700, 3, `{"bottle_l":"0.9","logged_l":"2.7",` + note + `,"taps_today":3,"target_l":"2.7","target_reached":true}`},
		{"over target", plan(1800, 900), 2700, 3, `{"bottle_l":"0.9","logged_l":"2.7",` + note + `,"taps_today":3,"target_l":"1.8","target_reached":true}`},
		{"no bottle size", plan(2700, 0), 0, 0, `{"bottle_l":null,"logged_l":null,` + note + `,"taps_today":0,"target_l":"2.7","target_reached":false}`},
	}
	for _, c := range cases {
		if got := coachWaterJSON(t, buildHealthWater(c.cycle, c.total, c.taps, -1)); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

func TestCoachSeesWaterWithoutChecklistRow(t *testing.T) {
	h, now := waterHandler(t, "2.7", "0.9")
	fc := &fakeCoach{call: []string{"get_health_today", "get_daily_checklist"}, reply: "OK"}
	h.Coach = fc
	if w := ask(h, "How is my water?", "http://example.com"); w.Code != http.StatusSeeOther {
		t.Fatalf("ask: %d", w.Code)
	}
	today := fc.toolData["get_health_today"]
	if !strings.Contains(today, `"logged_l":null`) || !strings.Contains(today, `"taps_today":0`) || strings.Contains(today, `"logged_l":"0.0"`) {
		t.Fatalf("zero taps must be null: %s", today)
	}
	if !strings.Contains(fc.seen.System, `"water":`) {
		t.Fatal("system prompt lacks water")
	}
	if w := cfgPost(h, "/health/water", nil, ""); w.Code != http.StatusSeeOther {
		t.Fatalf("tap: %d", w.Code)
	}
	*now = now.Add(time.Minute)
	if w := ask(h, "And now?", "http://example.com"); w.Code != http.StatusSeeOther {
		t.Fatalf("ask: %d", w.Code)
	}
	today = fc.toolData["get_health_today"]
	for _, want := range []string{`"logged_l":"0.9"`, `"taps_today":1`, `"target_l":"2.7"`, `"target_reached":false`} {
		if !strings.Contains(today, want) {
			t.Fatalf("missing %s in %s", want, today)
		}
	}
	if strings.Contains(fc.toolData["get_daily_checklist"], "ater") {
		t.Fatalf("checklist changed: %s", fc.toolData["get_daily_checklist"])
	}
}
