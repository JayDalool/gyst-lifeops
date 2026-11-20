package secretary

import (
	"errors"
	"strings"
	"time"

	"lifeops-tracker/internal/clock"
)

// DateWindowDays is how far ahead a date tool may look. Today is day 0.
const DateWindowDays = 14

// Fixed errors: they never repeat the input (it came from the model, which
// may have copied it from untrusted text).
var (
	errBadDate    = errors.New("unrecognised date: use YYYY-MM-DD, today, tomorrow, a weekday, or this/next plus a weekday")
	errDateWindow = errors.New("date is outside the range from today to 14 days ahead")
	weekdayNames  = map[string]time.Weekday{"sunday": 0, "sun": 0, "monday": 1, "mon": 1, "tuesday": 2, "tue": 2, "tues": 2, "wednesday": 3, "wed": 3, "thursday": 4, "thu": 4, "thur": 4, "thurs": 4, "friday": 5, "fri": 5, "saturday": 6, "sat": 6}
)

// Day is a resolved local calendar day. Every tool output names both fields so
// the model never has to work out a weekday (S7).
type Day struct {
	Date    string // YYYY-MM-DD
	Weekday string // Monday ... Sunday
	Offset  int    // days from today
	t       time.Time
}

// ResolveDate turns the model's date text into a day inside the window
// (today to today+14) on the application clock. Empty means today.
//
//	YYYY-MM-DD, today, tomorrow
//	friday, this friday: the first Friday from today on (today counts)
//	next friday: the coming Friday, strictly after today (a Friday gives +7)
func ResolveDate(input string, c clock.Clock) (Day, error) {
	today := c.Today()
	text := strings.ToLower(strings.Join(strings.Fields(input), " "))
	var day time.Time
	switch {
	case text == "" || text == "today":
		day = today
	case text == "tomorrow":
		day = today.AddDate(0, 0, 1)
	default:
		if t, err := time.ParseInLocation(clock.DateLayout, text, c.Location()); err == nil && len(text) == len(clock.DateLayout) {
			day = t
			break
		}
		next := false
		name := text
		if rest, ok := strings.CutPrefix(text, "next "); ok {
			next, name = true, rest
		} else if rest, ok := strings.CutPrefix(text, "this "); ok {
			name = rest
		}
		wd, ok := weekdayNames[name]
		if !ok {
			return Day{}, errBadDate
		}
		ahead := (int(wd) - int(today.Weekday()) + 7) % 7
		if next && ahead == 0 {
			ahead = 7
		}
		day = today.AddDate(0, 0, ahead)
	}
	offset := daysBetween(today, day)
	if offset < 0 || offset > DateWindowDays {
		return Day{}, errDateWindow
	}
	return Day{Date: day.Format(clock.DateLayout), Weekday: day.Weekday().String(), Offset: offset, t: day}, nil
}

// daysBetween counts civil days, so a clock change inside the span cannot
// shift the answer by an hour's worth of rounding.
func daysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// Time is the day's local midnight.
func (d Day) Time() time.Time { return d.t }
