package services

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func windowInput(date string) models.HealthSyncRequest {
	return models.HealthSyncRequest{Date: date, Source: "apple_shortcuts", HealthMetrics: models.HealthMetrics{DietaryCalories: healthFloat(1357)}}
}

func TestHealthSyncWindowBoundaries(t *testing.T) {
	// 2026-10-11 18:00Z is Oct 11 in Winnipeg.
	now := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		date string
		ok   bool
	}{
		{"2026-10-11", true}, {"2026-10-04", true}, {"2026-10-05", true},
		{"2026-10-03", false}, {"2026-10-12", false}, {"2020-01-01", false}, {"2099-01-01", false},
	} {
		t.Run(test.date, func(t *testing.T) {
			s := testService(t)
			s.Now = func() time.Time { return now }
			_, err := s.SyncHealth(windowInput(test.date))
			var window *HealthDateWindowError
			if test.ok {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.As(err, &window) || window.Earliest != "2026-10-04" || window.Latest != "2026-10-11" {
				t.Fatalf("want window error, got %v", err)
			}
			if err.Error() != "date must be between 2026-10-04 and 2026-10-11" {
				t.Fatal(err.Error())
			}
			if _, err := s.GetDailyHealthSnapshot(test.date); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("snapshot written: %v", err)
			}
			state, _ := s.GetHealthSyncState("apple_shortcuts")
			if state.LastSyncedAt != "" {
				t.Fatalf("sync state changed: %+v", state)
			}
		})
	}
}

func TestHealthSyncWindowFollowsLocalMidnight(t *testing.T) {
	// America/Winnipeg is CDT (UTC-5) on these dates.
	for _, test := range []struct {
		now          time.Time
		date         string
		wantAccepted bool
	}{
		{time.Date(2026, 10, 9, 4, 30, 0, 0, time.UTC), "2026-10-09", false}, // 23:30 on Oct 8 local
		{time.Date(2026, 10, 9, 4, 30, 0, 0, time.UTC), "2026-10-01", true},
		{time.Date(2026, 10, 9, 5, 30, 0, 0, time.UTC), "2026-10-09", true}, // 00:30 on Oct 9 local
		{time.Date(2026, 10, 9, 5, 30, 0, 0, time.UTC), "2026-10-01", false},
	} {
		s := testService(t)
		now := test.now
		s.Now = func() time.Time { return now }
		_, err := s.SyncHealth(windowInput(test.date))
		if (err == nil) != test.wantAccepted {
			t.Errorf("now=%s date=%s: err=%v want accepted=%v", now.Format(time.RFC3339), test.date, err, test.wantAccepted)
		}
	}
}

func TestHealthSyncWindowAcrossDSTAndOtherZone(t *testing.T) {
	// US DST ends on 2026-11-01; the window must still be seven civil days.
	s := testService(t)
	s.Now = func() time.Time { return time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC) }
	if _, err := s.SyncHealth(windowInput("2026-10-27")); err != nil {
		t.Fatalf("earliest across DST: %v", err)
	}
	var window *HealthDateWindowError
	if _, err := s.SyncHealth(windowInput("2026-10-26")); !errors.As(err, &window) || window.Earliest != "2026-10-27" {
		t.Fatalf("day before earliest across DST: %v", err)
	}
	// Spring forward in Winnipeg: 2027-03-14.
	s.Now = func() time.Time { return time.Date(2027, 3, 16, 18, 0, 0, 0, time.UTC) }
	if _, err := s.SyncHealth(windowInput("2027-03-09")); err != nil {
		t.Fatalf("earliest across spring DST: %v", err)
	}
	if _, err := s.SyncHealth(windowInput("2027-03-08")); err == nil {
		t.Fatal("accepted day before earliest across spring DST")
	}
	// A different zone moves the window: 2026-10-11T22:00Z is already Oct 12 in Tokyo.
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	s.Loc = loc
	s.Now = func() time.Time { return time.Date(2026, 10, 11, 22, 0, 0, 0, time.UTC) }
	if _, err := s.SyncHealth(windowInput("2026-10-12")); err != nil {
		t.Fatalf("tokyo today: %v", err)
	}
	if _, err := s.SyncHealth(windowInput("2026-10-13")); err == nil {
		t.Fatal("tokyo tomorrow accepted")
	}
}

func TestHealthSyncMalformedDateStaysValidationError(t *testing.T) {
	s := testService(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC) }
	_, err := s.SyncHealth(windowInput("2026-02-30"))
	var invalid *HealthValidationError
	var window *HealthDateWindowError
	if !errors.As(err, &invalid) || errors.As(err, &window) {
		t.Fatalf("malformed date: %v", err)
	}
}

func TestHealthSyncRefusalLogHasNoMetricOrToken(t *testing.T) {
	var buf bytes.Buffer
	origOut, origFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() { log.SetOutput(origOut); log.SetFlags(origFlags) }()
	s := testService(t)
	s.Now = func() time.Time { return time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC) }
	in := windowInput("2026-09-01")
	in.ProteinG = healthFloat(7654.25)
	if _, err := s.SyncHealth(in); err == nil {
		t.Fatal("accepted")
	}
	out := buf.String()
	if strings.Count(out, "SECURITY: health sync refused: date outside window") != 1 || strings.Count(out, "\n") != 1 {
		t.Fatalf("want one line, got %q", out)
	}
	for _, bad := range []string{"7654", "1357", "dietary", "protein", "token", "Bearer"} {
		if strings.Contains(out, bad) {
			t.Fatalf("log contains %q: %s", bad, out)
		}
	}
	if !strings.Contains(out, "2026-09-01") || !strings.Contains(out, "2026-10-04") || !strings.Contains(out, "2026-10-11") {
		t.Fatalf("log lacks date or bounds: %s", out)
	}
}
