package clock

import (
	"testing"
	"time"
)

func at(value string) func() time.Time {
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return instant }
}

func TestLoadDefaultsToWinnipeg(t *testing.T) {
	for _, name := range []string{"", "  "} {
		location, err := Load(name)
		if err != nil || location.String() != DefaultTimezone {
			t.Fatalf("Load(%q) = %v, %v; want %s", name, location, err, DefaultTimezone)
		}
	}
	if got := (Clock{}).Location().String(); got != DefaultTimezone {
		t.Fatalf("zero Clock location = %s", got)
	}
	if got := System().Location().String(); got != DefaultTimezone {
		t.Fatalf("System location = %s", got)
	}
}

func TestLoadAcceptsValidZone(t *testing.T) {
	location, err := Load("Asia/Tokyo")
	if err != nil || location.String() != "Asia/Tokyo" {
		t.Fatalf("Load(Asia/Tokyo) = %v, %v", location, err)
	}
}

func TestLoadRejectsInvalidZone(t *testing.T) {
	for _, name := range []string{"Mars/Base", "america/winnipeg_x", "Local", "../etc/passwd"} {
		if _, err := Load(name); err == nil {
			t.Fatalf("Load(%q) succeeded; want an error", name)
		}
	}
}

// The production condition: the pinned instant is UTC, the day is Winnipeg's.
// Manitoba's last clock change was the 2026-03-08 spring forward; from then
// on it is UTC-05 all year (Issue #193). Earlier history keeps its offsets.
func TestTodayUsesApplicationTimezone(t *testing.T) {
	cases := []struct{ instant, day string }{
		// Fall back 2025-11-02, the last one (02:00 CDT -> 01:00 CST at 07:00Z)
		{"2025-11-02T04:59:00Z", "2025-11-01"}, // 23:59 CDT
		{"2025-11-02T05:00:00Z", "2025-11-02"}, // 00:00 CDT
		{"2025-11-02T06:30:00Z", "2025-11-02"}, // first 01:30 (CDT)
		{"2025-11-02T07:30:00Z", "2025-11-02"}, // second 01:30 (CST)
		{"2025-11-03T05:59:00Z", "2025-11-02"}, // 23:59 CST, the 25-hour day
		{"2025-11-03T06:00:00Z", "2025-11-03"}, // 00:00 CST
		// CST (UTC-6)
		{"2026-01-16T05:30:00Z", "2026-01-15"}, // 23:30 local
		{"2026-01-16T05:59:00Z", "2026-01-15"}, // 23:59 local
		{"2026-01-16T06:00:00Z", "2026-01-16"}, // 00:00 local
		{"2026-01-16T06:30:00Z", "2026-01-16"}, // 00:30 local
		{"2026-01-16T00:00:00Z", "2026-01-15"}, // 00:00 UTC = 18:00 local
		// CDT (UTC-5)
		{"2026-07-16T04:30:00Z", "2026-07-15"}, // 23:30 local
		{"2026-07-16T04:59:00Z", "2026-07-15"}, // 23:59 local
		{"2026-07-16T05:00:00Z", "2026-07-16"}, // 00:00 local
		{"2026-07-16T05:30:00Z", "2026-07-16"}, // 00:30 local
		{"2026-07-16T00:00:00Z", "2026-07-15"}, // 00:00 UTC = 19:00 local
		// Spring forward 2026-03-08 (02:00 CST -> 03:00 CDT at 08:00Z), the last change
		{"2026-03-08T05:59:00Z", "2026-03-07"},
		{"2026-03-08T06:00:00Z", "2026-03-08"},
		{"2026-03-08T07:59:00Z", "2026-03-08"},
		{"2026-03-08T08:00:00Z", "2026-03-08"},
		{"2026-03-08T00:00:00Z", "2026-03-07"},
		{"2026-03-09T00:00:00Z", "2026-03-08"},
		{"2026-03-09T04:59:00Z", "2026-03-08"},
		{"2026-03-09T05:00:00Z", "2026-03-09"},
		// 2026-10-31 / 2026-11-01: permanent UTC-05, no fall-back, no repeated hour
		{"2026-10-31T04:59:00Z", "2026-10-30"},
		{"2026-10-31T05:00:00Z", "2026-10-31"},
		{"2026-11-01T04:59:00Z", "2026-10-31"},
		{"2026-11-01T05:00:00Z", "2026-11-01"},
		{"2026-11-01T06:30:00Z", "2026-11-01"}, // 01:30, once
		{"2026-11-01T07:30:00Z", "2026-11-01"}, // 02:30 (old rules: the second 01:30)
		{"2026-11-01T00:00:00Z", "2026-10-31"},
		{"2026-11-02T00:00:00Z", "2026-11-01"}, // 19:00
		{"2026-11-02T04:59:00Z", "2026-11-01"}, // 23:59
		{"2026-11-02T05:00:00Z", "2026-11-02"}, // 00:00 (old rules: 23:00 CST on the 1st)
		{"2026-11-02T05:59:00Z", "2026-11-02"}, // 00:59 (old rules: 23:59 CST on the 1st)
		// Late 2026 and the new year
		{"2026-12-21T04:59:00Z", "2026-12-20"},
		{"2026-12-21T05:00:00Z", "2026-12-21"},
		{"2027-01-01T04:59:00Z", "2026-12-31"},
		{"2027-01-01T05:00:00Z", "2027-01-01"},
		// March 2027: no spring forward on 2027-03-14
		{"2027-03-14T04:59:00Z", "2027-03-13"},
		{"2027-03-14T05:00:00Z", "2027-03-14"},
		{"2027-03-14T05:59:00Z", "2027-03-14"}, // 00:59 (old rules: 23:59 CST on the 13th)
		{"2027-03-15T04:59:00Z", "2027-03-14"},
		{"2027-03-15T05:00:00Z", "2027-03-15"},
		// November 2027: no fall-back on 2027-11-07
		{"2027-11-07T04:59:00Z", "2027-11-06"},
		{"2027-11-07T05:00:00Z", "2027-11-07"},
		{"2027-11-08T04:59:00Z", "2027-11-07"},
		{"2027-11-08T05:00:00Z", "2027-11-08"},
		{"2027-11-08T05:59:00Z", "2027-11-08"}, // 00:59 (old rules: 23:59 CST on the 7th)
	}
	for _, tc := range cases {
		c := Clock{Now: at(tc.instant)}
		if got := c.TodayString(); got != tc.day {
			t.Errorf("TodayString at %s = %s; want %s", tc.instant, got, tc.day)
		}
		if got := c.DateOf(c.Now()); got != tc.day {
			t.Errorf("DateOf at %s = %s; want %s", tc.instant, got, tc.day)
		}
		today := c.Today()
		if today.Hour() != 0 || today.Minute() != 0 || today.Location() != c.Location() {
			t.Errorf("Today at %s = %s; want local midnight", tc.instant, today)
		}
	}
}

func TestTodayArithmeticAcrossDST(t *testing.T) {
	for _, tc := range []struct{ instant, next string }{
		{"2025-11-02T05:30:00Z", "2025-11-03"}, // Sun 00:30 CDT (the 25-hour fall-back day) -> Mon
		{"2026-03-08T05:30:00Z", "2026-03-08"}, // Sat 23:30 CST -> Sun (spring forward)
		{"2026-10-31T04:30:00Z", "2026-10-31"}, // Fri 23:30 CDT -> Sat
		{"2026-11-01T05:30:00Z", "2026-11-02"}, // Sun 00:30 UTC-05 (no fall-back) -> Mon
		{"2027-03-14T05:30:00Z", "2027-03-15"}, // Sun 00:30 UTC-05 (no spring forward) -> Mon
	} {
		c := Clock{Now: at(tc.instant)}
		next := c.Today().AddDate(0, 0, 1)
		if next.Format(DateLayout) != tc.next || next.Hour() != 0 {
			t.Errorf("Today+1 at %s = %s; want %s 00:00", tc.instant, next, tc.next)
		}
	}
}

func TestCustomLocation(t *testing.T) {
	tokyo, err := Load("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	c := Clock{Now: at("2026-01-15T16:00:00Z"), Loc: tokyo} // 01:00 JST on the 16th
	if got := c.TodayString(); got != "2026-01-16" {
		t.Fatalf("Tokyo TodayString = %s", got)
	}
}

func TestParseLocalDate(t *testing.T) {
	for _, valid := range []string{"2026-01-15", "2024-02-29", "0001-01-01"} {
		if got, err := ParseLocalDate(valid); err != nil || got != valid {
			t.Errorf("ParseLocalDate(%q) = %q, %v", valid, got, err)
		}
	}
	for _, invalid := range []string{"", "x", "2026-2-3", "2026-02-30", "2026-13-01", "0000-01-01", "2026-01-15T00:00:00Z", " 2026-01-15", "2026/01/15", "2025-02-29"} {
		if _, err := ParseLocalDate(invalid); err == nil {
			t.Errorf("ParseLocalDate(%q) succeeded; want an error", invalid)
		}
	}
}
