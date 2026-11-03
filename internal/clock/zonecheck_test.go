package clock

import (
	"archive/zip"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The regression probe for Issue #193: the application's own zone loading
// gives Winnipeg's real offsets, history included, on any machine.
func TestDefaultZoneProbes(t *testing.T) {
	if err := CheckDefaultZone(); err != nil {
		t.Fatalf("America/Winnipeg as the application loads it:\n%v", err)
	}
	if err := CheckZone(System().Location()); err != nil {
		t.Fatalf("System clock location:\n%v", err)
	}
	if err := CheckZone(Clock{}.Location()); err != nil {
		t.Fatalf("zero Clock location:\n%v", err)
	}
}

// The probe has teeth: all-year CST (the stale future) and all-year UTC-05
// (history rewritten) both fail it.
func TestCheckZoneRejectsWrongRules(t *testing.T) {
	stale := CheckZone(time.FixedZone("CST", -6*60*60))
	if stale == nil || !strings.Contains(stale.Error(), "2026-11-01T07:00:00Z") {
		t.Fatalf("all-year UTC-06 passed or missed the 2026-11-01 probe: %v", stale)
	}
	rewritten := CheckZone(time.FixedZone("EST", -5*60*60))
	if rewritten == nil || !strings.Contains(rewritten.Error(), "2025-11-02T07:00:00Z") || !strings.Contains(rewritten.Error(), "2026-01-15T12:00:00Z") {
		t.Fatalf("all-year UTC-05 passed or missed the history probes: %v", rewritten)
	}
}

// The data GYST used before #193 where the host had no zone files: the copy
// in the Go toolchain (2025b for Go 1.25). It still falls back on 2026-11-01,
// and the probe catches it.
func TestCheckZoneRejectsGoToolchainCopy(t *testing.T) {
	path := filepath.Join(runtime.GOROOT(), "lib", "time", "zoneinfo.zip")
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Skipf("no Go toolchain zone database at %s: %v", path, err)
	}
	defer archive.Close()
	var data []byte
	for _, file := range archive.File {
		if file.Name == DefaultTimezone {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if data == nil {
		t.Fatalf("%s has no %s", path, DefaultTimezone)
	}
	goCopy, err := time.LoadLocationFromTZData(DefaultTimezone, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, offset := probeAt("2027-01-15T12:00:00Z").In(goCopy).Zone(); offset == -5*60*60 {
		t.Skipf("the toolchain copy at %s already has permanent UTC-05", path)
	}
	if err := CheckZone(goCopy); err == nil || !strings.Contains(err.Error(), "2026-11-01T07:00:00Z") {
		t.Fatalf("Go toolchain copy passed the probe or missed 2026-11-01: %v", err)
	}
}

// Local wall-clock times map to the right instants around the change: no
// repeated 01:00 hour on 2026-11-01, no skipped 02:00 hour on 2027-03-14,
// and earlier history unchanged.
func TestWinnipegWallClockAroundTheChange(t *testing.T) {
	location := System().Location()
	for _, tc := range []struct {
		local time.Time
		utc   string
	}{
		{time.Date(2025, 11, 2, 3, 0, 0, 0, location), "2025-11-02T09:00:00Z"},  // CST after the last fall-back
		{time.Date(2026, 1, 15, 12, 0, 0, 0, location), "2026-01-15T18:00:00Z"}, // CST
		{time.Date(2026, 3, 8, 3, 30, 0, 0, location), "2026-03-08T08:30:00Z"},  // CDT after the last spring forward
		{time.Date(2026, 10, 31, 12, 0, 0, 0, location), "2026-10-31T17:00:00Z"},
		{time.Date(2026, 11, 1, 1, 30, 0, 0, location), "2026-11-01T06:30:00Z"}, // happens once
		{time.Date(2026, 11, 1, 2, 30, 0, 0, location), "2026-11-01T07:30:00Z"},
		{time.Date(2026, 12, 25, 9, 0, 0, 0, location), "2026-12-25T14:00:00Z"},
		{time.Date(2027, 3, 14, 2, 30, 0, 0, location), "2027-03-14T07:30:00Z"}, // exists: no gap
		{time.Date(2027, 11, 7, 1, 30, 0, 0, location), "2027-11-07T06:30:00Z"}, // happens once
	} {
		if got := tc.local.UTC().Format(time.RFC3339); got != tc.utc {
			t.Errorf("%s local = %s; want %s", tc.local.Format("2006-01-02 15:04"), got, tc.utc)
		}
	}
}
