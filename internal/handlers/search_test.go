package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

type searchBody struct {
	Groups []services.SearchGroup `json:"groups"`
}

func doSearch(t *testing.T, h *Handler, host, query string) (searchBody, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+query, nil)
	if host != "" {
		req.Host = host
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body searchBody
	if rec.Code == 200 {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("bad JSON %q: %v", rec.Body.String(), err)
		}
	}
	return body, rec
}

func searchHrefs(b searchBody) []string {
	var out []string
	for _, g := range b.Groups {
		for _, it := range g.Items {
			out = append(out, it.Href)
		}
	}
	return out
}

func TestSearchAPIRequiresSession(t *testing.T) {
	h := authHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=today", nil)
	rec := httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "unauthorized") || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("signed out: %d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
	cookie := signIn(t, h)
	req = httptest.NewRequest(http.MethodGet, "/api/search?q=today", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.Authenticated().ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("signed in: %d %v", rec.Code, rec.Header())
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req = httptest.NewRequest(method, "/api/search?q=today", nil)
		req.AddCookie(cookie)
		req.Header.Set("Origin", "http://example.com")
		req.Host = "example.com"
		rec = httptest.NewRecorder()
		h.Authenticated().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET" || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d %v", method, rec.Code, rec.Header())
		}
	}
}

func TestSearchAPIQueryCap(t *testing.T) {
	h := testHandler(t)
	if err := h.Service.Repo.CreateProject(&models.Project{Name: strings.Repeat("a", 100), ProjectKind: "main", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	// 101 characters are cut to 100, so the 100-character name still matches.
	body, rec := doSearch(t, h, "", strings.Repeat("a", 101))
	if rec.Code != 200 || len(body.Groups) == 0 || body.Groups[0].Items[0].Href != "/projects/1" {
		t.Fatalf("capped query: %d %s", rec.Code, rec.Body.String())
	}
	// Whitespace is trimmed; an empty query returns suggestions only.
	body, _ = doSearch(t, h, "", "%20%20")
	if len(body.Groups) != 1 || body.Groups[0].Label != "Suggestions" {
		t.Fatalf("empty query: %+v", body)
	}
	var labels []string
	for _, it := range body.Groups[0].Items {
		labels = append(labels, it.Label)
	}
	if strings.Join(labels, ",") != "Today,Projects,Galaxy,Health" {
		t.Fatalf("suggestions = %v", labels)
	}
}

func TestSearchAPIGalaxyResult(t *testing.T) {
	h := testHandler(t)
	body, _ := doSearch(t, h, "", "gal")
	if len(body.Groups) == 0 || body.Groups[0].Label != "Top hit" || body.Groups[0].Items[0].Label != "Galaxy" || body.Groups[0].Items[0].Href != "/galaxy" {
		t.Fatalf("galaxy result: %+v", body)
	}
	if body, _ = doSearch(t, h, "", "open%20galaxy"); len(body.Groups) != 0 {
		t.Fatalf("old Open Galaxy label must be gone: %+v", body)
	}
}

func TestSearchAPITailnetAppsHiddenOnPublicHost(t *testing.T) {
	h := testHandler(t)
	h.appRegistry = appsFixture(fxWork, fxTailnt)
	for _, host := range []string{"gyst.zaylolabs.com", "", "box.example.com:8081"} {
		body, rec := doSearch(t, h, host, "tool")
		hrefs := strings.Join(searchHrefs(body), " ")
		if !strings.Contains(hrefs, "https://alpha.example.com/") || strings.Contains(rec.Body.String(), "Delta") || strings.Contains(rec.Body.String(), "9443") || strings.Contains(rec.Body.String(), "tailnet_host") {
			t.Fatalf("host %q leaks the tailnet app: %s", host, rec.Body.String())
		}
	}
	body, _ := doSearch(t, h, "Example.ts.net:3443", "delta")
	if len(body.Groups) == 0 || body.Groups[0].Items[0].Href != "https://example.ts.net:9443/dash" || !body.Groups[0].Items[0].External {
		t.Fatalf("tailnet host: %+v", body)
	}
}

func TestSearchAPIHealthNamesOnly(t *testing.T) {
	h := testHandler(t)
	steps, kcal := int64(98765), 4321.5
	if err := h.Service.Repo.UpsertDailyHealthSnapshot(models.DailyHealthSnapshot{Date: "2026-09-12", HealthMetrics: models.HealthMetrics{StepsWatch: &steps, DietaryCalories: &kcal}}); err != nil {
		t.Fatal(err)
	}
	body, rec := doSearch(t, h, "", "health")
	if len(body.Groups) == 0 {
		t.Fatal("no Health results")
	}
	for _, leak := range []string{"98765", "4321", "steps"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Fatalf("response leaks %q: %s", leak, rec.Body.String())
		}
	}
	for _, g := range body.Groups {
		for _, it := range g.Items {
			if strings.HasPrefix(it.Href, "/health") && it.Meta != "Health" {
				t.Fatalf("health result carries more than a name: %+v", it)
			}
		}
	}
	if strings.Contains(rec.Body.String(), "/health/coach") {
		t.Fatal("coach page listed without a provider")
	}
}

func TestSearchAPIGroupsAndGoPrivacy(t *testing.T) {
	h := testHandler(t)
	repo := h.Service.Repo
	for _, p := range []models.Project{{Name: "Orbit Alpha", ProjectKind: "main", Status: "active"}, {Name: "Orbit Gone", ProjectKind: "main", Status: "active"}} {
		p := p
		if err := repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		if p.Name == "Orbit Gone" {
			if err := repo.ArchiveProject(p.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := repo.CreateTask(&models.Task{Title: "Orbit errand", Status: "pending", Priority: "medium", Category: "personal", RepeatType: "none", ProofType: "none"}); err != nil {
		t.Fatal(err)
	}
	body, rec := doSearch(t, h, "", "ORBIT")
	if strings.Contains(rec.Body.String(), "Orbit Gone") || strings.Contains(rec.Body.String(), "errand") {
		t.Fatalf("archived project or task returned: %s", rec.Body.String())
	}
	if len(body.Groups) != 1 || body.Groups[0].Label != "Top hit" || body.Groups[0].Items[0].Label != "Orbit Alpha" {
		t.Fatalf("groups: %+v", body)
	}
}

func TestLauncherMarkup(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/today", nil))
	html := rec.Body.String()
	for _, want := range []string{"data-command-open", "data-quick-task", `data-launcher-panel="capture" hidden`, `data-launcher-panel="go"`, `role="combobox"`, `role="listbox"`, "data-today=", "data-tomorrow=", "/static/js/launcher.js?v="} {
		if !strings.Contains(html, want) {
			t.Fatalf("launcher markup missing %q", want)
		}
	}
	if strings.Contains(html, "Open Galaxy") || strings.Contains(html, "data-command-item") {
		t.Fatal("the fixed palette links must be gone")
	}
	if strings.Contains(strings.ToLower(html), "data-launcher-mode=\"do\"") {
		t.Fatal("no Do tab until GYST-103")
	}
}
