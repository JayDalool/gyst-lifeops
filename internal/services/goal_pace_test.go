package services

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func TestGoalPace(t *testing.T) {
	base := models.Goal{Status: "active", TargetValue: 100, StartDate: "2026-01-01", EndDate: "2026-01-11"} // 10 days
	cases := []struct {
		name    string
		mod     func(g *models.Goal)
		today   string
		current float64
		want    string
		ok      bool
	}{
		{"on track exactly at expected", nil, "2026-01-06", 50, "On track", true},
		{"on track ahead", nil, "2026-01-06", 80, "On track", true},
		{"behind", nil, "2026-01-06", 49, "Behind", true},
		{"start day expects zero", nil, "2026-01-01", 0, "On track", true},
		{"after end expects full target", nil, "2026-02-01", 99, "Behind", true},
		{"before start has no pace", nil, "2025-12-31", 0, "", false},
		{"missing start", func(g *models.Goal) { g.StartDate = "" }, "2026-01-06", 0, "", false},
		{"missing end", func(g *models.Goal) { g.EndDate = "" }, "2026-01-06", 0, "", false},
		{"target zero", func(g *models.Goal) { g.TargetValue = 0 }, "2026-01-06", 0, "", false},
		{"completed goal", func(g *models.Goal) { g.Status = "completed" }, "2026-01-06", 100, "", false},
		{"end before start", func(g *models.Goal) { g.EndDate = "2025-12-01" }, "2026-01-06", 0, "", false},
		{"bad today", nil, "", 0, "", false},
		{"app-day boundary: last day before start", nil, "2025-12-31", 100, "", false},
		{"app-day boundary: start day", nil, "2026-01-01", 0, "On track", true},
	}
	for _, c := range cases {
		g := base
		if c.mod != nil {
			c.mod(&g)
		}
		g.CurrentValue = c.current
		got, ok := GoalPace(g, c.today)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", c.name, got, ok, c.want, c.ok)
		}
	}
}
