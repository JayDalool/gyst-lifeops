package handlers

import (
	"net/url"
	"strings"
)

// isLocalPath reports whether candidate is safe to redirect to or link to as a
// same-origin path. A plain "starts with one slash" check is not enough:
// browsers treat "\" as "/" and drop tabs and newlines from URLs, so
// "/\evil.example" or "/\t/evil.example" become "//evil.example", a
// redirect to another site. Those characters never appear in GYST's own
// paths, so they are rejected outright (raw or percent-encoded).
func isLocalPath(candidate string) bool {
	if candidate == "" || !strings.HasPrefix(candidate, "/") || strings.HasPrefix(candidate, "//") {
		return false
	}
	for _, c := range candidate {
		if c == '\\' || c < 0x20 || c == 0x7f {
			return false
		}
	}
	if strings.Contains(strings.ToLower(candidate), "%5c") {
		return false
	}
	parsed, err := url.Parse(candidate)
	return err == nil && parsed.Scheme == "" && parsed.Host == ""
}
