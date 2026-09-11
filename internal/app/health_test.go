package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthAPIConfigAndRootMux(t *testing.T) {
	for _, token := range []string{"", "test-only-app-health-token"} {
		t.Run("configured="+map[bool]string{true: "yes", false: "no"}[token != ""], func(t *testing.T) {
			a, err := New(Config{DBPath: filepath.Join(t.TempDir(), "health.db"), MigrationsDir: "../../migrations", TemplateDir: "../../web/templates", StaticDir: "../../web/static", LoadSeed: false, HealthSyncToken: token})
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			req := httptest.NewRequest(http.MethodGet, "/api/health/today", nil)
			req.Header.Set("Authorization", "Bearer test-only-app-health-token")
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, req)
			want := 200
			if token == "" {
				want = 503
			}
			if rec.Code != want || !json.Valid(rec.Body.Bytes()) {
				t.Fatalf("root API routing: %d %s", rec.Code, rec.Body.String())
			}
			for _, path := range []string{"/health", "/health/plan", "/today", "/galaxy", "/projects", "/courses"} {
				rec := httptest.NewRecorder()
				a.Handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
				if rec.Code != 200 {
					t.Errorf("%s: %d", path, rec.Code)
				}
				if strings.Contains(rec.Body.String(), "test-only-app-health-token") {
					t.Fatal("token leaked to HTML")
				}
			}
		})
	}
}
