package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
)

func TestRenderAnalyticsShell(t *testing.T) {
	// Each chart gets its own fictional series so a swapped chart, or a changed
	// chart type, fails the test (GYST-142 review finding).
	data := AnalyticsData{Page: "analytics", Dashboard: models.DashboardData{
		TodayCompletion: 40, CurrentStreak: 5, MonthlyCompletionPercent: 60,
		OverdueTasks:       []models.Task{{Title: "Fictional task one"}, {Title: "Fictional task two"}},
		DailyCompletion:    models.ChartData{Labels: []string{"d1", "d2"}, Values: []int{11, 12}},
		ProjectProgress:    models.ChartData{Labels: []string{"p1", "p2"}, Values: []int{21, 22}},
		YearlyGoalProgress: models.ChartData{Labels: []string{"g1", "g2"}, Values: []int{31, 32}},
	}}
	h := &Handler{TemplateDir: "../../web/templates"}
	rec := httptest.NewRecorder()
	h.render(rec, "analytics.html", data)
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "executing \"") {
		t.Fatalf("analytics render failed: %d %s", rec.Code, body)
	}
	// Each chart fragment starts at its own section title, so a canvas moved under
	// another title fails. Whitespace between tags is not part of the contract.
	flat := strings.Join(strings.Fields(body), " ")
	for _, c := range []struct {
		name, s string
	}{
		{"daily line chart", `<h2 class="gyst-section-title">Last 30 days</h2> <div class="gyst-card"> <canvas class="chart" data-chart="line" data-max="100" data-labels="d1|d2" data-values="11|12" data-empty="Complete tasks to build the trend."></canvas>`},
		{"project bar chart", `<h2 class="gyst-section-title">Project progress</h2> <div class="gyst-card"> <canvas class="chart" data-chart="bar" data-max="100" data-labels="p1|p2" data-values="21|22"></canvas>`},
		{"yearly goals bar chart", `<h2 class="gyst-section-title">Yearly goals</h2> <div class="gyst-card"> <canvas class="chart" data-chart="bar" data-max="100" data-labels="g1|g2" data-values="31|32"></canvas>`},
		{"today row", `<span class="gyst-row-copy"><span>Today</span></span><span class="gyst-row-meta num">40%</span>`},
		{"streak row", `<span class="gyst-row-copy"><span>Streak</span></span><span class="gyst-row-meta num">5 days</span>`},
		{"month row", `<span class="gyst-row-copy"><span>This month</span></span><span class="gyst-row-meta num">60%</span>`},
		{"overdue row", `<span class="gyst-row-copy"><span>Overdue</span></span><span class="gyst-row-meta num">2</span>`},
	} {
		if got := strings.Count(flat, c.s); got != 1 {
			t.Errorf("%s: fragment count = %d, want 1", c.name, got)
		}
	}
	if got := strings.Count(body, "<h1"); got != 1 {
		t.Errorf("<h1 count = %d, want 1", got)
	}
	if got := strings.Count(body, "data-chart="); got != 3 {
		t.Errorf("chart canvases = %d, want 3", got)
	}
	for _, s := range []string{"Last 30 days", "Project progress", "Yearly goals", "gyst-scope gyst-more"} {
		if !strings.Contains(body, s) {
			t.Errorf("missing %q", s)
		}
	}
	if strings.Contains(body, `class="stats"`) {
		t.Error("legacy stats strip still rendered")
	}
	renderOK(t, "analytics.html", AnalyticsData{Page: "analytics"})
}
