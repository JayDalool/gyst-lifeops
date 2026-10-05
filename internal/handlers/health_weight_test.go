package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func TestParseManualWeight(t *testing.T) {
	location, _ := time.LoadLocation("America/Winnipeg")
	now := time.Date(2026, 10, 4, 20, 30, 0, 0, location)
	for _, tc := range []struct {
		weight, measured, wantErr string
		wantKg                    float64
		wantAt                    string
	}{
		{weight: "134.5", wantKg: 134.5, wantAt: "2026-10-04T20:30:00-05:00"},
		{weight: " 134,5 ", wantKg: 134.5},
		{weight: "134.456", wantKg: 134.46},
		{weight: "134.5", measured: "2026-10-04T07:15", wantKg: 134.5, wantAt: "2026-10-04T07:15:00-05:00"},
		{weight: "134.5", measured: "2026-10-04T20:34", wantKg: 134.5},
		{weight: "", wantErr: "Enter your weight"},
		{weight: "abc", wantErr: "must be a number"},
		{weight: "NaN", wantErr: "must be a number"},
		{weight: "Inf", wantErr: "must be a number"},
		{weight: "20", wantErr: "between 20 and 400"},
		{weight: "13.45", wantErr: "between 20 and 400"},
		{weight: "400", wantErr: "between 20 and 400"},
		{weight: "1345", wantErr: "between 20 and 400"},
		{weight: "134.5", measured: "2026-10-04T21:00", wantErr: "can't be in the future"},
		{weight: "134.5", measured: "yesterday", wantErr: "valid date and time"},
	} {
		entry, err := parseManualWeight(tc.weight, tc.measured, now, location)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("%q %q: error %v, want %q", tc.weight, tc.measured, err, tc.wantErr)
			}
			continue
		}
		if err != nil || entry.WeightKg != tc.wantKg || entry.Source != "manual" {
			t.Errorf("%q %q: got %+v, %v", tc.weight, tc.measured, entry, err)
		}
		if tc.wantAt != "" && entry.MeasuredAt != tc.wantAt {
			t.Errorf("%q %q: measured %s, want %s", tc.weight, tc.measured, entry.MeasuredAt, tc.wantAt)
		}
	}
}

func postWeight(h *Handler, path string, form url.Values, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func getBody(t *testing.T, h *Handler, path string) string {
	t.Helper()
	path, _, _ = strings.Cut(path, "#") // browsers never send the fragment
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: %d", path, w.Code)
	}
	return w.Body.String()
}

func TestLogWeightAndUndo(t *testing.T) {
	h := testHandler(t)
	w := postWeight(h, "/health/weight", url.Values{"weight_kg": {"134,5"}}, "http://example.com")
	if w.Code != http.StatusSeeOther || !strings.HasPrefix(w.Header().Get("Location"), "/health/progress?logged=") {
		t.Fatalf("log: %d %q %s", w.Code, w.Header().Get("Location"), w.Body.String())
	}
	entries, _ := h.Service.ListHealthWeightEntries()
	if len(entries) != 1 || entries[0].WeightKg != 134.5 || entries[0].Source != "manual" {
		t.Fatalf("stored %+v", entries)
	}
	body := getBody(t, h, w.Header().Get("Location"))
	if !strings.Contains(body, "Logged <strong>134.5 kg</strong>") || !strings.Contains(body, `class="weight-dot"`) {
		t.Fatal("confirmation or reading missing after logging")
	}
	undo := "/health/weight/" + url.PathEscape(strings.TrimPrefix(strings.Split(w.Header().Get("Location"), "#")[0], "/health/progress?logged=")) + "/undo"
	if !strings.Contains(body, `action="`+undo+`"`) {
		t.Fatalf("undo form missing for %s", undo)
	}
	if w := postWeight(h, undo, nil, ""); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/health/progress?undone=1#log" {
		t.Fatalf("undo: %d %q", w.Code, w.Header().Get("Location"))
	}
	if entries, _ := h.Service.ListHealthWeightEntries(); len(entries) != 0 {
		t.Fatalf("undo left %+v", entries)
	}
	if w := postWeight(h, undo, nil, ""); w.Code != http.StatusNotFound {
		t.Fatalf("second undo: %d", w.Code)
	}
	if body := getBody(t, h, "/health/progress?undone=1"); !strings.Contains(body, "Reading removed.") || !strings.Contains(body, "No weight readings") {
		t.Fatal("undone page should confirm and show no readings")
	}
}

func TestLogWeightRejectsBadInput(t *testing.T) {
	h := testHandler(t)
	future := h.Service.Now().In(h.Service.Location()).Add(time.Hour).Format(localDateTime)
	for _, form := range []url.Values{{"weight_kg": {"abc"}}, {"weight_kg": {""}}, {"weight_kg": {"1345"}}, {"weight_kg": {"134.5"}, "measured_at": {future}}} {
		w := postWeight(h, "/health/weight", form, "")
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `role="alert"`) || !strings.Contains(w.Body.String(), `aria-invalid="true"`) {
			t.Fatalf("%v: %d", form, w.Code)
		}
		if v := form.Get("weight_kg"); v != "" && !strings.Contains(w.Body.String(), `value="`+v+`"`) {
			t.Fatalf("%v: typed value not kept", form)
		}
	}
	if w := postWeight(h, "/health/weight", url.Values{"weight_kg": {"134.5"}}, "https://evil.example"); w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	for _, path := range []string{"/health/weight", "/health/weight/1/undo"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET %s: %d", path, w.Code)
		}
		if authExemptPath(path) {
			t.Fatalf("%s must require authentication", path)
		}
	}
	if entries, _ := h.Service.ListHealthWeightEntries(); len(entries) != 0 {
		t.Fatalf("rejected input stored %+v", entries)
	}
}

func TestUndoOnlyRecentManualEntries(t *testing.T) {
	h := testHandler(t)
	synced := models.HealthWeightEntry{MeasuredAt: time.Now().Format(time.RFC3339), WeightKg: 134, Source: "apple_shortcuts"}
	manual := models.HealthWeightEntry{MeasuredAt: time.Now().Format(time.RFC3339), WeightKg: 135, Source: "manual"}
	for _, e := range []*models.HealthWeightEntry{&synced, &manual} {
		if err := h.Service.CreateHealthWeightEntry(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/health/weight/" + idString(synced.ID) + "/undo", "/health/weight/999/undo", "/health/weight/x/undo", "/health/weight/0/undo"} {
		if w := postWeight(h, path, nil, ""); w.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if body := getBody(t, h, "/health/progress?logged="+idString(synced.ID)); strings.Contains(body, "Undo") {
		t.Fatal("synced reading offered undo")
	}
	h.Service.Clock.Now = func() time.Time { return time.Now().Add(11 * time.Minute) }
	if w := postWeight(h, "/health/weight/"+idString(manual.ID)+"/undo", nil, ""); w.Code != http.StatusNotFound {
		t.Fatalf("expired undo: %d", w.Code)
	}
	if entries, _ := h.Service.ListHealthWeightEntries(); len(entries) != 2 {
		t.Fatalf("entries deleted: %+v", entries)
	}
}

func idString(n int64) string { return strconv.FormatInt(n, 10) }
