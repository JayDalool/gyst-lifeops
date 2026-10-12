package handlers

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func componentHTML(t *testing.T, name string, data any) string {
	t.Helper()
	tmpl, err := template.ParseFiles("../../web/templates/ui_icons.html", "../../web/templates/ui_components.html")
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err = tmpl.ExecuteTemplate(&b, name, data); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
func TestUIMetricPresenceAndStates(t *testing.T) {
	for _, state := range []string{"normal", "partial", "stale", "empty", "loading", "error"} {
		for _, present := range []bool{false, true} {
			m := UIMetric{Label: "Steps", Value: "98765", Present: present, Status: UIStatus{State: state}, Progress: UIProgress{Present: true, Percent: 50}}
			for _, name := range []string{"gyst.metric", "gyst.ring"} {
				body := componentHTML(t, name, m)
				want := present && (state == "normal" || state == "partial" || state == "stale")
				if strings.Contains(body, "98765") != want {
					t.Fatalf("%s state=%s present=%v leaked or lost value: %s", name, state, present, body)
				}
				if name == "gyst.ring" && strings.Contains(body, `class="gyst-ring-fill"`) != want {
					t.Fatalf("ring fill present without meaningful data: %s", body)
				}
			}
		}
	}
	zero := componentHTML(t, "gyst.metric", UIMetric{Label: "Steps", Value: "0", Present: true})
	if !strings.Contains(zero, `class="gyst-number">0`) {
		t.Fatal("observed zero disappeared")
	}
}
func TestUIProgressBoundsAndMissing(t *testing.T) {
	for _, tt := range []struct{ input, want int }{{-10, 0}, {0, 0}, {73, 73}, {145, 100}} {
		p := UIProgress{Present: true, Percent: tt.input, Label: "Actual intake", Detail: "145 of 100"}
		if p.Fill() != tt.want {
			t.Fatalf("fill(%d)=%d", tt.input, p.Fill())
		}
		body := componentHTML(t, "gyst.progress", p)
		if !strings.Contains(body, "145 of 100") {
			t.Fatal("actual over-target detail lost")
		}
	}
	body := componentHTML(t, "gyst.progress", UIProgress{Percent: 50})
	if strings.Contains(body, "<progress") {
		t.Fatal("missing progress rendered as numeric")
	}
}
func TestUIEscapingAndLabels(t *testing.T) {
	bad := `<script>alert("x")</script>`
	body := componentHTML(t, "gyst.row", UIRow{Label: bad, URL: "javascript:alert(1)", Icon: "heart"})
	if strings.Contains(body, "<script>") || strings.Contains(body, `href="javascript:`) {
		t.Fatal("unsafe content rendered")
	}
	body = componentHTML(t, "gyst.field", UIField{ID: "goal", Name: "title", Label: "Goal", Value: bad, Error: "Required"})
	for _, part := range []string{`for="goal"`, `id="goal"`, `aria-invalid="true"`, `goal-error`, `&lt;script&gt;`} {
		if !strings.Contains(body, part) {
			t.Fatalf("field missing %s", part)
		}
	}
	body = componentHTML(t, "gyst.action", UIAction{Label: "Add task", Variant: "icon", Icon: "plus"})
	if !strings.Contains(body, `aria-label="Add task"`) || !strings.Contains(body, `type="button"`) {
		t.Fatal("icon button missing accessible name or safe type")
	}
}

func TestUIFoundationDoesNotExposeGallery(t *testing.T) {
	h := testHandler(t)
	for _, path := range []string{"/ui", "/ui-gallery"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("development gallery route %s exposed: %d", path, rec.Code)
		}
	}
	body := renderBody(t, "today.html", TodayData{Page: "today", Today: "2026-09-11"})
	if strings.Contains(body, "fictional fixtures") {
		t.Fatal("UI-0 opted the production page into demo markup")
	}
	for _, asset := range []string{"ui-tokens.css?v=vn-2", "ui-components.css?v=goals-1"} {
		if !strings.Contains(body, asset) {
			t.Fatalf("foundation asset %s not loaded", asset)
		}
	}
}
