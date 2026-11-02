package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
)

func TestOverdueSince(t *testing.T) {
	d := func(s string) time.Time {
		v, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	zone, err := clock.Load(clock.DefaultTimezone)
	if err != nil {
		t.Fatal(err)
	}
	local := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02", s, zone)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct{ due, today, want string }{
		{"2026-10-08", "2026-10-09", "since yesterday"},
		{"2026-10-07", "2026-10-09", "since Wednesday"},
		{"2026-10-03", "2026-10-09", "since Saturday"},
		{"2026-10-02", "2026-10-09", "since Oct 2"},
		{"2026-01-05", "2026-10-09", "since Jan 5"},
		{"2025-12-31", "2026-01-01", "since yesterday"},
		{"2025-10-02", "2026-10-09", "since Oct 2, 2025"},
		{"2025-12-20", "2026-01-02", "since Dec 20, 2025"},
	}
	for _, tc := range cases {
		if got := OverdueSince(d(tc.due), d(tc.today)); got != tc.want {
			t.Errorf("%s -> %s = %q, want %q", tc.due, tc.today, got, tc.want)
		}
	}
	// Across the DST changes, where local midnights are 23 or 25 hours apart,
	// and across 2026-11-01, when Manitoba stayed on UTC-05 (24 hours).
	for _, tc := range []struct{ due, today, want string }{
		{"2025-11-01", "2025-11-02", "since yesterday"},
		{"2025-11-02", "2025-11-03", "since yesterday"},
		{"2025-11-01", "2025-11-03", "since Saturday"},
		{"2026-03-07", "2026-03-08", "since yesterday"},
		{"2026-03-08", "2026-03-09", "since yesterday"},
		{"2026-03-07", "2026-03-09", "since Saturday"},
		{"2026-10-31", "2026-11-01", "since yesterday"},
		{"2026-11-01", "2026-11-02", "since yesterday"},
		{"2026-10-31", "2026-11-02", "since Saturday"},
	} {
		if got := OverdueSince(local(tc.due), local(tc.today)); got != tc.want {
			t.Errorf("DST %s -> %s = %q, want %q", tc.due, tc.today, got, tc.want)
		}
	}
}
