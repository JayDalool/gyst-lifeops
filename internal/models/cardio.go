package models

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Cardio notes convention (documented in docs/HEVY_SETUP.md):
//
//  1. Keyword form, exercise-level: "incline 8% · 5.0 km/h" applies to every
//     timed set. A bare percent ("30% effort") is never an incline.
//  2. Per-set tokens "8%/5.0": when the token count equals the number of timed
//     sets, the nth token belongs to the nth timed set.
//  3. Any other mix (disagreeing values, token count not matching) gives no
//     reading for any set.
const (
	maxInclinePct = 40.0
	minSpeedKmh   = 0.5
	maxSpeedKmh   = 25.0
	kmhPerMph     = 1.609344
)

const decimalPattern = `(\d+(?:[.,]\d+)?)`
const unitPattern = `(km\s*/\s*h|kmh|kph|mph)\b`

var (
	tokenRe      = regexp.MustCompile(`(?:^|[^\d.,\-])` + decimalPattern + `\s*%\s*/\s*` + decimalPattern + `(?:\s*` + unitPattern + `)?`)
	inclineKeyRe = regexp.MustCompile(`incline\s*[:=]?\s*` + decimalPattern)
	speedUnitRe  = regexp.MustCompile(`(?:^|[^\d.,\-])` + decimalPattern + `\s*` + unitPattern)
)

// CardioReading is the incline (percent) and speed (km/h) of one timed set.
// A nil field is unknown, never 0.
type CardioReading struct {
	InclinePct, SpeedKmh *float64
}

func parseDecimal(s string) float64 {
	v, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	return v
}

// toKmh converts and rounds to 0.1; unit "" means km/h.
func toKmh(value, unit string) float64 {
	v := parseDecimal(value)
	if strings.HasPrefix(unit, "mph") {
		v *= kmhPerMph
	}
	return math.Round(v*10) / 10
}

func boundedIncline(v float64) *float64 {
	if v < 0 || v > maxInclinePct {
		return nil
	}
	return &v
}

func boundedSpeed(v float64) *float64 {
	if v < minSpeedKmh || v > maxSpeedKmh {
		return nil
	}
	return &v
}

func allEqual(v []float64) bool {
	for _, x := range v {
		if x != v[0] {
			return false
		}
	}
	return true
}

// ParseCardioNotes returns one reading per timed set (timed is the number of
// timed sets in the exercise). Unreadable, ambiguous or out-of-bounds values
// (incline 0-40%, speed 0.5-25 km/h) stay nil.
func ParseCardioNotes(notes string, timed int) []CardioReading {
	out := make([]CardioReading, max(timed, 0))
	if timed <= 0 {
		return out
	}
	s := strings.ToLower(notes)
	var tokInc, tokSpd, keyInc, keySpd []float64
	for _, m := range tokenRe.FindAllStringSubmatch(s, -1) {
		tokInc = append(tokInc, parseDecimal(m[1]))
		tokSpd = append(tokSpd, toKmh(m[2], m[3]))
	}
	for _, m := range inclineKeyRe.FindAllStringSubmatchIndex(s, -1) {
		// "incline 30 degrees" is not a percent.
		if rest := strings.TrimSpace(s[m[1]:]); strings.HasPrefix(rest, "deg") || strings.HasPrefix(rest, "°") {
			continue
		}
		keyInc = append(keyInc, parseDecimal(s[m[2]:m[3]]))
	}
	for _, m := range speedUnitRe.FindAllStringSubmatch(s, -1) {
		keySpd = append(keySpd, toKmh(m[1], m[2]))
	}
	if len(tokInc) == 0 {
		// Keyword form: exercise-level, one agreed value or nothing.
		if len(keyInc) > 0 && !allEqual(keyInc) || len(keySpd) > 0 && !allEqual(keySpd) {
			return out
		}
		var r CardioReading
		if len(keyInc) > 0 {
			r.InclinePct = boundedIncline(keyInc[0])
		}
		if len(keySpd) > 0 {
			r.SpeedKmh = boundedSpeed(keySpd[0])
		}
		for i := range out {
			out[i] = r
		}
		return out
	}
	if len(keyInc) > 0 {
		// Keyword plus tokens: only valid when they all agree.
		if !allEqual(keyInc) || !allEqual(tokInc) || keyInc[0] != tokInc[0] {
			return out
		}
		// Speeds from both forms must agree too.
		if !allEqual(append(keySpd, tokSpd...)) {
			return out
		}
		r := CardioReading{InclinePct: boundedIncline(keyInc[0]), SpeedKmh: boundedSpeed(tokSpd[0])}
		for i := range out {
			out[i] = r
		}
		return out
	}
	if len(tokInc) != timed {
		return out
	}
	for i := range out {
		out[i] = CardioReading{InclinePct: boundedIncline(tokInc[i]), SpeedKmh: boundedSpeed(tokSpd[i])}
	}
	return out
}
