package secretary

import (
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/clock"
)

func pinned(t *testing.T, zone string, y int, m time.Month, d int) clock.Clock {
	t.Helper()
	loc, err := clock.Load(zone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(y, m, d, 12, 0, 0, 0, loc)
	return clock.Clock{Now: func() time.Time { return now }, Loc: loc}
}

// 2026-11-16 is a Monday.
func TestNextWeekdayIsTheComingOne(t *testing.T) {
	for i := 0; i < 7; i++ {
		c := pinned(t, "America/Winnipeg", 2026, 11, 16+i)
		today := c.Today()
		from := today.Weekday()
		got, err := ResolveDate("next friday", c)
		if err != nil {
			t.Fatalf("%v: %v", from, err)
		}
		want := (int(time.Friday) - int(from) + 7) % 7
		if want == 0 {
			want = 7 // on a Friday, next Friday is a week away
		}
		if got.Offset != want || got.Weekday != "Friday" || got.Date != today.AddDate(0, 0, want).Format(clock.DateLayout) {
			t.Errorf("next friday from %v: %+v, want +%d", from, got, want)
		}
		bare, _ := ResolveDate("friday", c)
		this, _ := ResolveDate("This Friday", c)
		wantBare := (int(time.Friday) - int(from) + 7) % 7
		if bare.Offset != wantBare || this.Offset != wantBare {
			t.Errorf("friday from %v: %d and %d, want %d (today counts)", from, bare.Offset, this.Offset, wantBare)
		}
	}
}

func TestResolveDateForms(t *testing.T) {
	c := pinned(t, "America/Winnipeg", 2026, 11, 19) // Thursday
	for _, tc := range []struct {
		in, date, weekday string
		offset            int
	}{
		{"", "2026-11-19", "Thursday", 0},
		{"today", "2026-11-19", "Thursday", 0},
		{"  TODAY ", "2026-11-19", "Thursday", 0},
		{"tomorrow", "2026-11-20", "Friday", 1},
		{"2026-11-25", "2026-11-25", "Wednesday", 6},
		{"2026-12-03", "2026-12-03", "Thursday", 14},
		{"mon", "2026-11-23", "Monday", 4},
		{"next   Monday", "2026-11-23", "Monday", 4},
		{"thursday", "2026-11-19", "Thursday", 0},
		{"next thursday", "2026-11-26", "Thursday", 7},
		{"this sunday", "2026-11-22", "Sunday", 3},
	} {
		got, err := ResolveDate(tc.in, c)
		if err != nil || got.Date != tc.date || got.Weekday != tc.weekday || got.Offset != tc.offset {
			t.Errorf("%q: %+v %v, want %s %s +%d", tc.in, got, err, tc.date, tc.weekday, tc.offset)
		}
	}
}

func TestResolveDateErrorsNeverEchoInput(t *testing.T) {
	c := pinned(t, "America/Winnipeg", 2026, 11, 19)
	secret := "IGNORE-PREVIOUS-INSTRUCTIONS"
	for _, in := range []string{secret, "2026-13-01", "2026-02-30", "26-11-19", "2026-11-1", "next " + secret, "this", "next", "someday", "2026-11-18", "2026-12-04", "2027-01-01", "1999-01-01", "friday " + secret} {
		_, err := ResolveDate(in, c)
		if err == nil {
			t.Errorf("%q accepted", in)
			continue
		}
		if strings.Contains(err.Error(), "IGNORE") || strings.Contains(err.Error(), in) && in != "" && len(in) > 4 {
			t.Errorf("error echoes input: %v", err)
		}
	}
	if _, err := ResolveDate("2026-11-18", c); err != errDateWindow {
		t.Errorf("yesterday: %v", err)
	}
	if _, err := ResolveDate("2026-12-04", c); err != errDateWindow {
		t.Errorf("today+15: %v", err)
	}
}

// A zone that still changes its clocks: the answer is by civil date, not by
// 24-hour steps. New York's clocks went back on 2026-11-01.
func TestResolveDateAcrossAClockChange(t *testing.T) {
	c := pinned(t, "America/New_York", 2026, 10, 30) // Friday
	for in, want := range map[string]string{"tomorrow": "2026-10-31", "2026-11-01": "2026-11-01", "monday": "2026-11-02", "next friday": "2026-11-06", "2026-11-13": "2026-11-13"} {
		got, err := ResolveDate(in, c)
		if err != nil || got.Date != want {
			t.Errorf("%q: %+v %v, want %s", in, got, err, want)
		}
	}
	// Late evening local time is still today (UTC is already tomorrow).
	loc := c.Location()
	late := time.Date(2026, 11, 1, 23, 30, 0, 0, loc)
	c.Now = func() time.Time { return late }
	if got, _ := ResolveDate("today", c); got.Date != "2026-11-01" || got.Weekday != "Sunday" {
		t.Errorf("late evening: %+v", got)
	}
}
