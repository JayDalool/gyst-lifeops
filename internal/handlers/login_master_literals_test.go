package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

// GYST-188 c4: what /login and /logout answer, apart from the page markup, is what master answered
// before the sign-in page was rebuilt. The literals below were captured by running this same test on
// master (2ed90ca) with the dump printed, then pasted; only pre-existing helpers are used so the file
// compiles there unchanged. A change to a status, header or cookie of the sign-in handler fails here.

// responseShape is the status, every header except the per-request id, and the Set-Cookie values with
// the (random) token blanked, one per line.
func responseShape(rec *httptest.ResponseRecorder) string {
	var lines []string
	lines = append(lines, "status "+itoa(rec.Code))
	res := rec.Result()
	for name, vals := range res.Header {
		if name == "X-Request-Id" {
			continue
		}
		for _, v := range vals {
			if name == "Set-Cookie" {
				parts := strings.SplitN(v, ";", 2)
				cookieName, _, _ := strings.Cut(parts[0], "=")
				rest := ""
				if len(parts) == 2 {
					rest = ";" + parts[1]
				}
				// Expires moves with the clock; its presence is what is pinned.
				if i := strings.Index(rest, "; Expires="); i >= 0 {
					j := strings.Index(rest[i+2:], ";")
					if j < 0 {
						rest = rest[:i] + "; Expires=<time>"
					} else {
						rest = rest[:i] + "; Expires=<time>" + rest[i+2+j:]
					}
				}
				v = cookieName + "=<token>" + rest
			}
			lines = append(lines, name+": "+v)
		}
	}
	sort.Strings(lines[1:])
	return strings.Join(lines, "\n")
}

func TestLoginResponseShapesMatchMaster(t *testing.T) {
	const html = "Cache-Control: no-store\nContent-Type: text/html; charset=utf-8"
	formReq := func(h *Handler, method, path string, values url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("X-Forwarded-For", "192.0.2.7")
		r.Header.Set("CF-Connecting-IP", "203.0.113.50")
		r.Host = "gyst.zaylolabs.com"
		if cookie != nil {
			r.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, r)
		return rec
	}
	h, clk := throttledHandler(t)

	get := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/login", nil))
	wrong := formReq(h, http.MethodPost, "/login", url.Values{"password": {wrongPW}}, nil)
	big := formReq(h, http.MethodPost, "/login", url.Values{"password": {strings.Repeat("a", 9000)}}, nil)
	failFive(t, h, clk, "203.0.113.60")
	throttledWrong := pub(h, wrongPW, "203.0.113.60")
	throttledRight := pub(h, testPassword, "203.0.113.60")
	ok := formReq(h, http.MethodPost, "/login", url.Values{"password": {testPassword}, "next": {"/today"}}, nil)
	var session *http.Cookie
	for _, c := range ok.Result().Cookies() {
		if c.Name == sessionCookieName {
			session = c
		}
	}
	clk.Advance(time.Second)
	logout := formReq(h, http.MethodPost, "/logout", url.Values{}, session)
	h.Auth.PasswordHash = ""
	unconfigured := formReq(h, http.MethodPost, "/login", url.Values{"password": {wrongPW}}, nil)

	cookieAttrs := "; Path=/; Expires=<time>; HttpOnly; SameSite=Lax"
	cases := []struct {
		name string
		rec  *httptest.ResponseRecorder
		want string
	}{
		{"GET /login", get, "status 200\n" + html},
		{"POST wrong", wrong, "status 401\n" + html},
		{"POST unreadable", big, "status 400\n" + html},
		{"POST throttled, wrong password", throttledWrong, "status 429\n" + html},
		{"POST throttled, right password", throttledRight, "status 429\n" + html},
		{"POST success", ok, "status 303\nLocation: /today\nSet-Cookie: " + sessionCookieName + "=<token>" + cookieAttrs},
		{"POST /logout", logout, "status 303\nLocation: /login\nSet-Cookie: " + sessionCookieName + "=<token>; Path=/; Max-Age=0; HttpOnly; SameSite=Lax"},
		{"POST unconfigured", unconfigured, "status 503\n" + html},
	}
	for _, c := range cases {
		if got := responseShape(c.rec); got != c.want {
			t.Errorf("%s:\ngot:\n%s\nwant:\n%s", c.name, got, c.want)
		}
	}
}
