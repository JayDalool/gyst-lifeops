package clock

import "time"

// InstantLayout is the canonical stored-instant format (GYST-VN-004): UTC,
// always 30 characters, with a fixed 9-digit fraction. The fixed width is
// load-bearing: SQL TEXT comparison and ORDER BY are only chronological if
// every value has the same width (RFC3339Nano trims zeros, so "...:00.5Z"
// would sort before "...:00Z"). Sessions already store this layout.
const InstantLayout = "2006-01-02T15:04:05.000000000Z"

// FormatInstant renders t as a canonical stored instant, always in UTC.
func FormatInstant(t time.Time) string {
	return t.UTC().Format(InstantLayout)
}

// ParseInstant reads RFC 3339 with Z or a numeric offset and 0 to 9
// fractional digits, and returns the instant in UTC. A value with no offset,
// the SQLite "YYYY-MM-DD HH:MM:SS" form, a date only, or "" is an error.
func ParseInstant(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
