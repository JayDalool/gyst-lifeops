package services

import (
	"strconv"
	"strings"
)

// ParseTypedNumber is the one rule for a typed number (GYST-154). A single
// comma with no dot and one or two digits after it is a decimal comma ("1,5");
// any other comma is refused rather than guessed. With thousands set, a comma
// that groups digits in threes ("2,000", "1,234,567.5") is also read as a
// thousands separator: a reference target of "2,000" kcal was saved as 2.
// The diary passes thousands=false and keeps refusing "1,000".
func ParseTypedNumber(raw string, thousands bool) (float64, bool) {
	s := strings.TrimSpace(raw)
	if thousands && validThousands(s) {
		s = strings.ReplaceAll(s, ",", "")
	}
	if strings.Contains(s, ",") {
		before, after, _ := strings.Cut(s, ",")
		if strings.Contains(s, ".") || strings.Count(s, ",") != 1 || before == "" || len(after) == 0 || len(after) > 2 {
			return 0, false
		}
		s = before + "." + after
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// validThousands reports whether every comma in s sits between a leading group
// of 1-3 digits and groups of exactly 3, before any decimal dot.
func validThousands(s string) bool {
	if !strings.Contains(s, ",") {
		return false
	}
	whole, _, _ := strings.Cut(s, ".")
	groups := strings.Split(whole, ",")
	for i, g := range groups {
		if i == 0 && (len(g) < 1 || len(g) > 3) || i > 0 && len(g) != 3 {
			return false
		}
		for _, c := range g {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
