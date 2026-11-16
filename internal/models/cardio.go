package models

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Cardio notes convention (documented in docs/HEVY_SETUP.md): free text that
// carries an incline in percent and a speed in km/h or mph, for example
// "incline 8% · 5.0 km/h", "incline 10 3.1mph" or "8%/5.0".
const (
	maxInclinePct = 40.0
	minSpeedKmh   = 0.5
	maxSpeedKmh   = 25.0
	kmhPerMph     = 1.609344
)

const num = `(\d+(?:\.\d+)?)`

var (
	shortNotesRe = regexp.MustCompile(`(?:^|[^\d.\-])` + num + `\s*%\s*/\s*` + num + `(?:\s*(km\s*/\s*h|kmh|kph|mph)\b)?`)
	inclineKeyRe = regexp.MustCompile(`incline\s*[:=]?\s*` + num)
	inclinePctRe = regexp.MustCompile(`(?:^|[^\d.\-])` + num + `\s*%`)
	speedUnitRe  = regexp.MustCompile(`(?:^|[^\d.\-])` + num + `\s*(km\s*/\s*h|kmh|kph|mph)\b`)
)

func speedKmh(value, unit string) *float64 {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	if strings.HasPrefix(unit, "mph") {
		v *= kmhPerMph
	}
	v = math.Round(v*10) / 10
	if v < minSpeedKmh || v > maxSpeedKmh {
		return nil
	}
	return &v
}

func inclinePct(value string) *float64 {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil || v < 0 || v > maxInclinePct {
		return nil
	}
	return &v
}

// ParseCardioNotes reads incline (percent) and speed (km/h) from exercise
// notes. Either result is nil when it is missing, unreadable or outside
// 0-40% / 0.5-25 km/h. It never returns 0 for "unknown".
func ParseCardioNotes(notes string) (incline, speed *float64) {
	s := strings.ToLower(notes)
	if m := shortNotesRe.FindStringSubmatch(s); m != nil {
		unit := m[3]
		if unit == "" {
			unit = "kmh"
		}
		return inclinePct(m[1]), speedKmh(m[2], unit)
	}
	if m := inclineKeyRe.FindStringSubmatch(s); m != nil {
		incline = inclinePct(m[1])
	} else if m := inclinePctRe.FindStringSubmatch(s); m != nil {
		incline = inclinePct(m[1])
	}
	if m := speedUnitRe.FindStringSubmatch(s); m != nil {
		speed = speedKmh(m[1], m[2])
	}
	return incline, speed
}
