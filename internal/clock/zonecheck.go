package clock

import (
	"errors"
	"fmt"
	"time"
)

// ZoneProbe is a known UTC offset of DefaultTimezone at one instant.
type ZoneProbe struct {
	At     time.Time
	Offset time.Duration
	What   string
}

// DefaultZoneProbes are known America/Winnipeg UTC offsets around Manitoba's
// move to permanent UTC-05 (Issue #193; Manitoba's notice at
// gov.mb.ca/timechange and IANA tzdb 2026e). The 2026-03-08 spring forward
// was the last clock change. Earlier history must not move, and a stale
// zone database (before 2026e) still falls back to UTC-06 on 2026-11-01, so
// it fails the later probes. Offsets only: IANA models the change with the
// abbreviation EST and says the details are temporary.
var DefaultZoneProbes = []ZoneProbe{
	{probeAt("2025-07-15T12:00:00Z"), -5 * time.Hour, "summer 2025 (CDT)"},
	{probeAt("2025-11-02T06:59:59Z"), -5 * time.Hour, "last second before the 2025 fall-back"},
	{probeAt("2025-11-02T07:00:00Z"), -6 * time.Hour, "2025-11-02 fall-back to CST"},
	{probeAt("2026-01-15T12:00:00Z"), -6 * time.Hour, "winter 2026 (CST)"},
	{probeAt("2026-03-08T07:59:59Z"), -6 * time.Hour, "last second of CST"},
	{probeAt("2026-03-08T08:00:00Z"), -5 * time.Hour, "2026-03-08 spring forward, the last change"},
	{probeAt("2026-10-31T12:00:00Z"), -5 * time.Hour, "2026-10-31, permanent UTC-05 in law"},
	{probeAt("2026-11-01T07:00:00Z"), -5 * time.Hour, "2026-11-01 02:00, no fall-back"},
	{probeAt("2026-11-15T12:00:00Z"), -5 * time.Hour, "November 2026"},
	{probeAt("2027-01-15T12:00:00Z"), -5 * time.Hour, "winter 2027"},
	{probeAt("2027-03-14T08:00:00Z"), -5 * time.Hour, "2027-03-14, no spring forward"},
	{probeAt("2027-07-15T12:00:00Z"), -5 * time.Hour, "summer 2027"},
	{probeAt("2027-11-07T07:00:00Z"), -5 * time.Hour, "2027-11-07, no fall-back"},
	{probeAt("2040-01-15T12:00:00Z"), -5 * time.Hour, "2040, past the transition table"},
}

func probeAt(value string) time.Time {
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return instant
}

// CheckZone compares location with every DefaultZoneProbes entry and
// returns one error naming each probe it gets wrong.
func CheckZone(location *time.Location) error {
	var errs []error
	for _, probe := range DefaultZoneProbes {
		_, seconds := probe.At.In(location).Zone()
		if got := time.Duration(seconds) * time.Second; got != probe.Offset {
			errs = append(errs, fmt.Errorf("%s (%s): UTC offset %s, want %s",
				probe.At.Format(time.RFC3339), probe.What, formatOffset(got), formatOffset(probe.Offset)))
		}
	}
	return errors.Join(errs...)
}

// CheckDefaultZone runs CheckZone on DefaultTimezone as the application
// loads it.
func CheckDefaultZone() error {
	location, err := Load(DefaultTimezone)
	if err != nil {
		return err
	}
	return CheckZone(location)
}

func formatOffset(d time.Duration) string {
	sign := "+"
	if d < 0 {
		sign, d = "-", -d
	}
	return fmt.Sprintf("%s%02d:%02d", sign, int(d.Hours()), int(d.Minutes())%60)
}
