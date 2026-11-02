// Package clock is the one source of "now" and of the application's local
// calendar day (GYST-VN-003).
//
// Rules:
//   - Instants (sessions, sync times, completed_at, ...) are absolute and are
//     stored in UTC. Take them from Clock.Now().
//   - Calendar days (task occurrences, tracker log dates, prayer day, Health
//     day, quick capture, ...) are YYYY-MM-DD dates in the application
//     timezone. Derive them from Clock.Today(), Clock.TodayString() or
//     Clock.DateOf(). Never take a day straight from a raw time.Now().
//
// The application timezone comes from GYST_TIMEZONE and defaults to
// America/Winnipeg. America/Winnipeg's rules are pinned in the binary
// (tzdata/, see its README): Manitoba's permanent UTC-05 from 2026-11-01 is
// newer than Go's embedded zone database and than older container images, so
// the default zone depends on neither. Any other zone goes through
// time.LoadLocation (system zoneinfo first, then Go's embedded copy).
package clock

import (
	_ "embed"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // fallback zone database for zones other than the default
)

// DefaultTimezone is the application timezone when GYST_TIMEZONE is unset.
const DefaultTimezone = "America/Winnipeg"

//go:embed tzdata/America_Winnipeg
var winnipegTZif []byte

//go:embed tzdata/VERSION
var pinnedVersion string

// PinnedTZDataVersion is the IANA tzdata release the default zone is pinned to.
func PinnedTZDataVersion() string {
	return strings.TrimSpace(pinnedVersion)
}

var defaultLocation = mustLoadPinned()

func mustLoadPinned() *time.Location {
	location, err := time.LoadLocationFromTZData(DefaultTimezone, winnipegTZif)
	if err != nil {
		panic(fmt.Sprintf("clock: load pinned %s: %v", DefaultTimezone, err))
	}
	return location
}

// Clock supplies instants and the application's local calendar day.
//
// Now is a field so tests can pin it. Loc is the application timezone; nil
// means DefaultTimezone.
type Clock struct {
	Now func() time.Time
	Loc *time.Location
}

// System returns the production clock in the default application timezone.
func System() Clock {
	return Clock{Now: time.Now, Loc: defaultLocation}
}

// Load resolves the configured application timezone. An empty name or
// DefaultTimezone gives the pinned DefaultTimezone; any other name must be a
// valid IANA zone. "Local" is
// rejected because it would make the result depend on the process zone.
func Load(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == DefaultTimezone {
		return defaultLocation, nil
	}
	if name == "Local" {
		return nil, fmt.Errorf("invalid GYST_TIMEZONE %q: use an IANA zone name such as %s", name, DefaultTimezone)
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid GYST_TIMEZONE %q: %w", name, err)
	}
	return location, nil
}

// Location is the application timezone.
func (c Clock) Location() *time.Location {
	if c.Loc == nil {
		return defaultLocation
	}
	return c.Loc
}

// Today is 00:00 of the current local calendar day, in the application
// timezone. The value is only ever used for civil-date arithmetic (AddDate,
// Weekday, Year/Month/Day) and leaves the process as a YYYY-MM-DD string.
// America/Winnipeg's clock changes (the last on 2026-03-08) were at 02:00, so
// local midnight always exists.
func (c Clock) Today() time.Time {
	local := c.Now().In(c.Location())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, c.Location())
}

// TodayString is Today as YYYY-MM-DD.
func (c Clock) TodayString() string {
	return c.Today().Format(DateLayout)
}

// InLocal shows an instant in the application timezone, for display only.
func (c Clock) InLocal(t time.Time) time.Time {
	return t.In(c.Location())
}

// DateOf is the local calendar day of an instant, as YYYY-MM-DD.
func (c Clock) DateOf(t time.Time) string {
	return t.In(c.Location()).Format(DateLayout)
}

// DateLayout is the canonical stored calendar-day format.
const DateLayout = "2006-01-02"

// ParseLocalDate accepts only a canonical, valid YYYY-MM-DD date (for example
// it rejects 2026-2-3, 2026-02-30 and 0000-01-01) and returns it unchanged.
func ParseLocalDate(value string) (string, error) {
	parsed, err := time.Parse(DateLayout, value)
	if err != nil || parsed.Format(DateLayout) != value || parsed.Year() < 1 {
		return "", fmt.Errorf("date must be YYYY-MM-DD")
	}
	return value, nil
}
