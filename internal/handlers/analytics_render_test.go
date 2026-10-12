package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestRenderAnalyticsShell(t *testing.T) {
	series := models.ChartData{Labels: []string{"a", "b", "c"}, Values: []int{10, 20, 30}}
	data := AnalyticsData{Page: "analytics", Dashboard: models.DashboardData{
		TodayCompletion: 40, CurrentStreak: 5, MonthlyCompletionPercent: 60,
		DailyCompletion: series, ProjectProgress: series, YearlyGoalProgress: series,
	}}
	h := &Handler{TemplateDir: "../../web/templates"}
	rec := httptest.NewRecorder()
	h.render(rec, "analytics.html", data)
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "executing \"") {
		t.Fatalf("analytics render failed: %d %s", rec.Code, body)
	}
	for _, c := range []struct {
		s string
		n int
	}{
		{"data-chart=", 3}, {`data-max="100"`, 3}, {`class="chart"`, 3},
		{`data-labels="a|b|c"`, 3}, {`data-values="10|20|30"`, 3}, {"<h1", 1},
		{`data-empty="Complete tasks to build the trend."`, 1},
	} {
		if got := strings.Count(body, c.s); got != c.n {
			t.Errorf("%q count = %d, want %d", c.s, got, c.n)
		}
	}
	for _, s := range []string{"Today", "Streak", "This month", "Overdue", "Last 30 days", "Project progress", "Yearly goals", "gyst-scope gyst-more"} {
		if !strings.Contains(body, s) {
			t.Errorf("missing %q", s)
		}
	}
	if strings.Contains(body, `class="stats"`) {
		t.Error("legacy stats strip still rendered")
	}
	renderOK(t, "analytics.html", AnalyticsData{Page: "analytics"})
}
