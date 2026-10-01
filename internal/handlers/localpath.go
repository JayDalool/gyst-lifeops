package handlers

import (
	"net/http"
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

// sameHostReferer returns the request's Referer as a local path (path, query
// and fragment) only when the Referer is on the same host as the request, or
// is itself a path. Anything else, such as an empty, unparsable, cross-host,
// non-http(s) or non-local Referer, yields fallback, so redirectBack can never
// become an open redirect even if a GET caller is added later.
func sameHostReferer(r *http.Request, fallback string) string {
	ref := r.Header.Get("Referer")
	if ref == "" {
		return fallback
	}
	u, err := url.Parse(ref)
	if err != nil || u.Opaque != "" {
		return fallback
	}
	if u.Host != "" {
		if (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, r.Host) {
			return fallback
		}
	} else if u.Scheme != "" {
		return fallback
	}
	candidate := u.EscapedPath()
	if candidate == "" {
		candidate = "/"
	}
	if u.RawQuery != "" {
		candidate += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		candidate += "#" + u.EscapedFragment()
	}
	if !isLocalPath(candidate) {
		return fallback
	}
	return candidate
}
