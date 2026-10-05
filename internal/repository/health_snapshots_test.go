package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

// ListDailyHealthSnapshots is inclusive at both ends and ordered by date
// (Issue #55 review).
func TestListDailyHealthSnapshotsRange(t *testing.T) {
	r := newTestRepo(t)
	kcal := func(v float64) *float64 { return &v }
	for _, d := range []string{"2026-09-03", "2026-09-04", "2026-09-07", "2026-09-10", "2026-09-11"} {
		if err := r.UpsertDailyHealthSnapshot(models.DailyHealthSnapshot{Date: d, LastSyncedAt: "2026-09-11T00:00:00Z", HealthMetrics: models.HealthMetrics{DietaryCalories: kcal(1000)}}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.ListDailyHealthSnapshots("2026-09-04", "2026-09-10")
	if err != nil {
		t.Fatal(err)
	}
	dates := []string{}
	for _, s := range got {
		dates = append(dates, s.Date)
	}
	if len(dates) != 3 || dates[0] != "2026-09-04" || dates[1] != "2026-09-07" || dates[2] != "2026-09-10" {
		t.Fatalf("dates %v", dates)
	}
	if none, err := r.ListDailyHealthSnapshots("2026-10-01", "2026-10-07"); err != nil || len(none) != 0 || none == nil {
		t.Fatalf("empty range: %v %v", none, err)
	}
}
