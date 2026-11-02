package clock

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"
)

// Manitoba's permanent UTC-05 (IANA 2026e): history keeps its seasonal
// offsets, and nothing changes after 2026-11-01. Offsets, not abbreviations,
// because IANA labels the new time EST for compatibility.
func TestWinnipegPermanentUTCMinus5(t *testing.T) {
	const utcMinus6, utcMinus5 = -6 * 3600, -5 * 3600
	cases := []struct {
		instant string
		offset  int
	}{
		{"2025-11-02T06:59:59Z", utcMinus5}, // last fall back still happened
		{"2025-11-02T07:00:00Z", utcMinus6},
		{"2026-01-15T18:00:00Z", utcMinus6},
		{"2026-03-08T07:59:59Z", utcMinus6},
		{"2026-03-08T08:00:00Z", utcMinus5},
		{"2026-07-15T18:00:00Z", utcMinus5},
		{"2026-10-31T18:00:00Z", utcMinus5},
		{"2026-11-01T06:59:59Z", utcMinus5},
		{"2026-11-01T07:00:00Z", utcMinus5}, // former fall-back instant
		{"2026-11-01T08:00:00Z", utcMinus5},
		{"2026-11-02T05:30:00Z", utcMinus5},
		{"2026-12-21T18:00:00Z", utcMinus5},
		{"2027-01-15T18:00:00Z", utcMinus5},
		{"2027-03-14T07:59:59Z", utcMinus5}, // no spring forward
		{"2027-03-14T08:00:00Z", utcMinus5},
		{"2027-03-14T18:00:00Z", utcMinus5},
		{"2027-07-15T18:00:00Z", utcMinus5},
		{"2027-11-07T06:59:59Z", utcMinus5}, // no fall back
		{"2027-11-07T07:00:00Z", utcMinus5},
		{"2027-11-07T18:00:00Z", utcMinus5},
		{"2030-01-15T18:00:00Z", utcMinus5},
	}
	loc := (Clock{}).Location()
	for _, tc := range cases {
		instant, err := time.Parse(time.RFC3339, tc.instant)
		if err != nil {
			t.Fatal(err)
		}
		if _, offset := instant.In(loc).Zone(); offset != tc.offset {
			t.Errorf("offset at %s = %d; want %d", tc.instant, offset, tc.offset)
		}
	}
	// Local wall time round-trips with a single meaning: 01:30 on
	// 2026-11-01 happens once, and 02:00 on 2027-03-14 exists.
	for _, wall := range []struct {
		y    int
		m    time.Month
		d, h int
		utc  string
	}{
		{2026, time.November, 1, 1, "2026-11-01T06:00:00Z"},
		{2026, time.November, 2, 0, "2026-11-02T05:00:00Z"},
		{2027, time.March, 14, 2, "2027-03-14T07:00:00Z"},
		{2027, time.November, 7, 1, "2027-11-07T06:00:00Z"},
	} {
		got := time.Date(wall.y, wall.m, wall.d, wall.h, 0, 0, 0, loc).UTC().Format(time.RFC3339)
		if got != wall.utc {
			t.Errorf("%d-%02d-%02d %02d:00 local = %s; want %s", wall.y, wall.m, wall.d, wall.h, got, wall.utc)
		}
	}
}

// The default zone is the pinned one, whatever ZONEINFO or the system
// zone database say, so a stale image or Go release cannot change it.
func TestDefaultZoneIsPinned(t *testing.T) {
	t.Setenv("ZONEINFO", "/nonexistent")
	for _, name := range []string{"", DefaultTimezone, " " + DefaultTimezone + " "} {
		location, err := Load(name)
		if err != nil || location != defaultLocation {
			t.Fatalf("Load(%q) = %v, %v; want the pinned zone", name, location, err)
		}
	}
	if System().Location() != defaultLocation || (Clock{}).Location() != defaultLocation {
		t.Fatal("System and zero Clock must use the pinned zone")
	}
	if v := PinnedTZDataVersion(); len(v) < 5 || v < "2026e" {
		t.Fatalf("pinned tzdata version = %q; want 2026e or later", v)
	}
	t.Logf("pinned America/Winnipeg tzdata %s", PinnedTZDataVersion())
}

// When the machine running the tests has tzdata at least as new as the pin,
// its America/Winnipeg must agree with the pin; if a newer release changed
// the zone, refresh tzdata/ (see its README).
func TestPinnedWinnipegMatchesSystemTzdata(t *testing.T) {
	version := systemTzdataVersion(t)
	data, err := os.ReadFile("/usr/share/zoneinfo/America/Winnipeg")
	if version == "" || err != nil {
		t.Skip("no system tzdata to compare with")
	}
	system, err := time.LoadLocationFromTZData(DefaultTimezone, data)
	if err != nil {
		t.Fatal(err)
	}
	pinned := PinnedTZDataVersion()
	t.Logf("system tzdata %s, pinned %s", version, pinned)
	start := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2040, 1, 1, 0, 0, 0, 0, time.UTC)
	for at := start; at.Before(end); at = at.Add(time.Hour) {
		_, want := at.In(system).Zone()
		if _, got := at.In(defaultLocation).Zone(); got != want {
			if tzdataOlder(version, pinned) {
				t.Skipf("system tzdata %s is older than the pin %s and differs at %s", version, pinned, at.Format(time.RFC3339))
			}
			t.Fatalf("system tzdata %s gives offset %d at %s, the pinned %s gives %d: refresh internal/clock/tzdata",
				version, want, at.Format(time.RFC3339), pinned, got)
		}
	}
}

func systemTzdataVersion(t *testing.T) string {
	t.Helper()
	f, err := os.Open("/usr/share/zoneinfo/tzdata.zi")
	if err != nil {
		return ""
	}
	defer f.Close()
	line, _ := bufio.NewReader(f).ReadString('\n')
	return strings.TrimSpace(strings.TrimPrefix(line, "# version"))
}

// tzdataOlder compares IANA release names such as 2026e and 2026f.
func tzdataOlder(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}
