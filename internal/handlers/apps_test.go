package handlers

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"lifeops-tracker/internal/services"
)

func appsFixture(apps ...services.App) func() ([]services.App, error) {
	return func() ([]services.App, error) { return apps, nil }
}

var (
	fxWork   = services.App{ID: "w", Name: "Alpha Tool", Monogram: "AT", Group: "work", Environment: "public", URL: "https://alpha.example.com/"}
	fxOp     = services.App{ID: "o", Name: "Beta Tool", Monogram: "BT", Group: "operate", Environment: "public", URL: "https://beta.example.com/"}
	fxLearn  = services.App{ID: "l", Name: "Gamma Tool", Monogram: "GT", Group: "learning", Environment: "public", URL: "https://gamma.example.com/"}
	fxTailnt = services.App{ID: "t", Name: "Delta Tool", Monogram: "DT", Group: "operate", Environment: "tailnet", URL: "https://{tailnet_host}:9443/dash"}
)

func getApps(t *testing.T, h *Handler, host string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/apps", nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET /apps on %q: %d", host, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("apps page must be no-store")
	}
	return rec.Body.String()
}

func TestAppsPublicHostOmitsTailnetURLs(t *testing.T) {
	h := testHandler(t)
	h.appRegistry = appsFixture(fxWork, fxTailnt)
	for _, host := range []string{"gyst.zaylolabs.com", "", "box.example.com:8081"} {
		body := getApps(t, h, host)
		if !strings.Contains(body, "Tailnet only") || !strings.Contains(body, "Delta Tool") {
			t.Fatalf("host %q: tailnet row not shown as locked", host)
		}
		for _, leak := range []string{"9443", "/dash", "{tailnet_host}", "tailnet_host", ".ts.net"} {
			if strings.Contains(body, leak) {
				t.Fatalf("host %q leaks %q", host, leak)
			}
		}
	}
}

func TestAppsTailnetHostLinksTailnetApps(t *testing.T) {
	h := testHandler(t)
	h.appRegistry = appsFixture(fxTailnt)
	body := getApps(t, h, "Example.ts.net:3443")
	if !strings.Contains(body, `href="https://example.ts.net:9443/dash"`) {
		t.Fatalf("tailnet link missing: %s", body)
	}
	if strings.Contains(body, "{tailnet_host}") || strings.Contains(body, "Tailnet only") {
		t.Fatal("placeholder or locked row on tailnet host")
	}
}

func TestAppsExternalLinksOpenSafely(t *testing.T) {
	h := testHandler(t)
	h.appRegistry = appsFixture(fxWork, fxOp, fxTailnt)
	body := getApps(t, h, "example.ts.net")
	links := regexp.MustCompile(`<a class="gyst-row"[^>]*>`).FindAllString(body, -1)
	if len(links) != 3 {
		t.Fatalf("want 3 app links, got %d", len(links))
	}
	for _, l := range links {
		if !strings.Contains(l, `target="_blank"`) || !strings.Contains(l, `rel="noopener noreferrer"`) {
			t.Fatalf("unsafe link: %s", l)
		}
	}
	if strings.Count(body, "(opens in a new tab)") != 3 {
		t.Fatal("hidden new-tab text missing")
	}
	if strings.Count(body, "<h1") != 1 {
		t.Fatal("want exactly one h1")
	}
}

func TestAppsGroupingOrderAndEmptyGroups(t *testing.T) {
	h := testHandler(t)
	h.appRegistry = appsFixture(fxLearn, fxOp, fxWork)
	body := getApps(t, h, "gyst.zaylolabs.com")
	w, o, l := strings.Index(body, ">Work<"), strings.Index(body, ">Operate<"), strings.Index(body, ">Learning<")
	if w < 0 || o < w || l < o {
		t.Fatalf("group order wrong: %d %d %d", w, o, l)
	}
	h.appRegistry = appsFixture(fxOp)
	body = getApps(t, h, "gyst.zaylolabs.com")
	if strings.Contains(body, ">Work<") || strings.Contains(body, ">Learning<") || !strings.Contains(body, ">Operate<") {
		t.Fatal("empty groups must be hidden")
	}
	h.appRegistry = appsFixture()
	body = getApps(t, h, "gyst.zaylolabs.com")
	if strings.Contains(body, "gyst-section-title") || strings.Count(body, "<h1") != 1 {
		t.Fatal("empty registry should render only the header")
	}
}

func TestAppsEmbeddedRegistryRenders(t *testing.T) {
	h := testHandler(t)
	body := getApps(t, h, "gyst.zaylolabs.com")
	for _, want := range []string{"Cloudflare", "Tailscale", "Expenses", "SamVision"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestAppsMethodsAndPaths(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/apps", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET" {
		t.Fatalf("POST: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/apps/x", nil))
	if rec.Code != 404 {
		t.Fatalf("/apps/x: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/apps", nil))
	if rec.Code != 303 || !strings.HasPrefix(rec.Header().Get("Location"), "/login") {
		t.Fatalf("signed-out: %d", rec.Code)
	}
}
