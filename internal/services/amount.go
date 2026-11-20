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

// validThousands reports whether s is [1-9]\d{0,2}(,\d{3})+ with an optional
// .digits tail. Anything else (a comma after the dot, "2.000,00", a leading
// zero as in "0,125") is not a thousands number and falls back to the diary
// rule, so it is refused instead of silently reinterpreted.
func validThousands(s string) bool {
	whole, frac, hasDot := strings.Cut(s, ".")
	if !strings.Contains(whole, ",") {
		return false
	}
	if hasDot {
		if frac == "" {
			return false
		}
		for _, c := range frac {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	for i, g := range strings.Split(whole, ",") {
		if i == 0 && (len(g) < 1 || len(g) > 3 || g[0] == '0') || i > 0 && len(g) != 3 {
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
