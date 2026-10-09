package handlers

import (
	"lifeops-tracker/internal/models"
	"math"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Fictional fixtures only. VN-022 state components (11 §I/§N, 13 §17).

func TestStateFamiliesRenderWordAndMarker(t *testing.T) {
	for _, kind := range []string{"ok", "info", "warn", "crit", "idle"} {
		s := UIState{Kind: kind}
		chip := componentHTML(t, "gyst.chip", s)
		line := componentHTML(t, "gyst.state-line", s)
		tile := componentHTML(t, "gyst.sev-tile", s)
		banner := componentHTML(t, "gyst.banner", UIBanner{State: s, Message: "Example message"})
		for name, body := range map[string]string{"chip": chip, "line": line, "tile": tile, "banner": banner} {
			if !strings.Contains(body, `data-state="`+kind+`"`) {
				t.Errorf("%s %s lacks data-state: %s", kind, name, body)
			}
		}
		for name, body := range map[string]string{"chip": chip, "line": line, "banner": banner} {
			if !strings.Contains(body, s.Label()) {
				t.Errorf("%s %s lacks its word %q: %s", kind, name, s.Label(), body)
			}
		}
		if !strings.Contains(chip, `class="gyst-dot" aria-hidden="true"`) || !strings.Contains(line, `class="gyst-dot" aria-hidden="true"`) {
			t.Errorf("%s chip/line lack the dot", kind)
		}
		for name, body := range map[string]string{"tile": tile, "banner": banner} {
			if !strings.Contains(body, `<svg class="gyst-icon"`) || !strings.Contains(body, `aria-hidden="true"`) {
				t.Errorf("%s %s lacks an aria-hidden icon: %s", kind, name, body)
			}
		}
	}
}

func TestStateUnknownAndOKWordsNeverLeak(t *testing.T) {
	for _, kind := range []string{"", "bogus", "healthy"} {
		s := UIState{Kind: kind}
		if s.Family() != "idle" {
			t.Errorf("kind %q family = %s, want idle", kind, s.Family())
		}
		body := componentHTML(t, "gyst.chip", s)
		if !strings.Contains(body, "Unknown") || strings.Contains(body, "Healthy") {
			t.Errorf("kind %q must render idle Unknown: %s", kind, body)
		}
	}
	if (UIState{Kind: " OK "}).Family() != "ok" {
		t.Error("kind is trimmed and case-insensitive")
	}
	// currentColor in the SVG is not a word; only visible text is checked.
	svg := regexp.MustCompile(`(?s)<svg.*?</svg>`)
	for _, kind := range []string{"info", "warn", "crit", "idle", "nonsense"} {
		for _, word := range []string{"Healthy", "done", "Up to date", "Current", "VALID"} {
			s := UIState{Kind: kind, Word: word}
			for _, name := range []string{"gyst.chip", "gyst.state-line"} {
				body := svg.ReplaceAllString(componentHTML(t, name, s), "")
				if strings.Contains(strings.ToLower(body), strings.ToLower(word)) {
					t.Errorf("%s %s shows ok word %q: %s", kind, name, word, body)
				}
			}
			b := svg.ReplaceAllString(componentHTML(t, "gyst.banner", UIBanner{State: s, Message: "Example"}), "")
			if strings.Contains(strings.ToLower(b), strings.ToLower(word)) {
				t.Errorf("%s banner shows ok word %q: %s", kind, word, b)
			}
		}
	}
	if got := (UIState{Kind: "crit", Word: "Healthy"}).Label(); got != "Failed" {
		t.Errorf("crit with Healthy renders %q, want Failed", got)
	}
	if got := (UIState{Kind: "ok", Word: "Done"}).Label(); got != "Done" {
		t.Errorf("ok keeps its own word, got %q", got)
	}
	icons := map[string]string{"ok": "check", "info": "info", "warn": "triangle-alert", "crit": "circle-alert", "idle": "circle-dashed"}
	for kind, icon := range icons {
		if got := (UIState{Kind: kind}).Icon(); got != icon {
			t.Errorf("%s icon = %s, want %s", kind, got, icon)
		}
	}
}

func TestErrorBannerNeverHealthy(t *testing.T) {
	body := componentHTML(t, "gyst.error-banner", UIErrorBanner{Message: "Could not load the example", LastGood: true, Retry: UIAction{Label: "Retry", URL: "#x"}})
	for _, want := range []string{`data-state="crit"`, `role="alert"`, "Failed", "Last good data kept", "Retry"} {
		if !strings.Contains(body, want) {
			t.Errorf("error banner missing %q: %s", want, body)
		}
	}
	if strings.Contains(strings.ToLower(body), "ealthy") {
		t.Errorf("error banner mentions healthy: %s", body)
	}
	if strings.Contains(componentHTML(t, "gyst.error-banner", UIErrorBanner{Message: "x"}), "Last good data kept") {
		t.Error("last-good note shown without LastGood")
	}
}

func TestEmptyIsOneLineAndStaleShowsAge(t *testing.T) {
	body := componentHTML(t, "gyst.empty", UIEmpty{Line: "Nothing to study yet", Action: UIAction{Label: "Add course", URL: "#x"}})
	if strings.Count(body, "<p>") != 1 || strings.Count(body, "<a ") != 1 || !strings.Contains(body, "Nothing to study yet") {
		t.Errorf("empty must be one line plus one action: %s", body)
	}
	if bare := componentHTML(t, "gyst.empty", UIEmpty{Line: "Nothing yet"}); strings.Contains(bare, "<a ") || strings.Contains(bare, "<button") {
		t.Errorf("empty without action rendered one: %s", bare)
	}
	st := componentHTML(t, "gyst.stale", UIStale{Source: "Example source", Age: "6:40"})
	if !strings.Contains(st, "Stale · 6:40") || !strings.Contains(st, `data-state="warn"`) || !strings.Contains(st, "Example source") {
		t.Errorf("stale chip: %s", st)
	}
}

func TestOfflineBannerMarkup(t *testing.T) {
	body := componentHTML(t, "gyst.offline-banner", "09:15")
	for _, want := range []string{"data-gyst-offline", " hidden", `role="status"`, `data-state="idle"`, "Offline", "showing saved data", "Loaded 09:15", "<svg"} {
		if !strings.Contains(body, want) {
			t.Errorf("offline banner missing %q: %s", want, body)
		}
	}
	if strings.Contains(componentHTML(t, "gyst.offline-banner", ""), "Loaded") {
		t.Error("Loaded shown without a time")
	}
}

func TestOfflineBannerInLayoutExceptGalaxy(t *testing.T) {
	h := testHandler(t)
	pinClock(t, h, "2026-03-04T15:15:00Z")
	want := h.Service.Now().In(h.Service.Location()).Format("15:04")
	pages := map[string]any{
		"today.html":         TodayData{Page: "today", Today: "2026-03-04"},
		"projects_list.html": ProjectListData{Page: "projects", Tree: models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}}, RoadmapNames: map[int64]string{}},
		"health.html":        map[string]any{"Page": "health"},
	}
	for page, data := range pages {
		rec := httptest.NewRecorder()
		h.render(rec, page, data)
		body := rec.Body.String()
		if !strings.Contains(body, "data-gyst-offline") {
			t.Errorf("%s layout lacks the offline banner (status %d)", page, rec.Code)
		}
		if !strings.Contains(body, "Loaded "+want) {
			t.Errorf("%s offline banner lacks app time %s", page, want)
		}
	}
	rec := httptest.NewRecorder()
	h.render(rec, "galaxy.html", struct{ Page string }{Page: "galaxy"})
	if rec.Code != 200 {
		t.Fatalf("galaxy status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "data-gyst-offline") {
		t.Error("offline banner must not appear on /galaxy")
	}
}

func TestStateCSSRules(t *testing.T) {
	raw, err := os.ReadFile("../../web/static/css/ui-components.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	i := strings.Index(css, "/* VN-022 State components")
	j := strings.Index(css, "/* Central responsive rules")
	if i < 0 || j < i {
		t.Fatal("VN-022 CSS block not found")
	}
	block := css[i:j]
	if strings.Contains(block, "--vn-xp") {
		t.Error("state components must never use the violet identity accent (DD-4)")
	}
	for _, bad := range []string{"--gyst-", "rgb("} {
		if strings.Contains(block, bad) {
			t.Errorf("state block uses %q instead of --vn-* tokens", bad)
		}
	}
	if regexp.MustCompile(`#[0-9a-fA-F]{3,6}\b`).MatchString(block) {
		t.Error("state block hard-codes a colour")
	}
	if !regexp.MustCompile(`\.gyst-banner \.gyst-button, \.gyst-empty \.gyst-button \{[^}]*min-height: 44px; min-width: 44px`).MatchString(block) {
		t.Error("banner/empty actions need min-height and min-width 44px")
	}
	if !strings.Contains(block, ":focus-visible") {
		t.Error("banner/empty actions need a visible focus rule")
	}
}

var vnVarRef = regexp.MustCompile(`^var\((--vn-[a-z0-9-]+)\)$`)

func vnResolve(t *testing.T, name string, dark bool) string {
	t.Helper()
	for _, tok := range vnTokens {
		if tok.Name != name {
			continue
		}
		v := tok.Light
		if dark && tok.Dark != "" {
			v = tok.Dark
		}
		if m := vnVarRef.FindStringSubmatch(v); m != nil {
			return vnResolve(t, m[1], dark)
		}
		return v
	}
	t.Fatalf("token %s missing", name)
	return ""
}

func vnLuminance(t *testing.T, hex string) float64 {
	t.Helper()
	if !regexp.MustCompile(`^#[0-9a-f]{6}$`).MatchString(hex) {
		t.Fatalf("not a hex colour: %q", hex)
	}
	var sum float64
	for i, w := range []float64{.2126, .7152, .0722} {
		n, _ := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		c := float64(n) / 255
		if c <= .03928 {
			c /= 12.92
		} else {
			c = math.Pow((c+.055)/1.055, 2.4)
		}
		sum += w * c
	}
	return sum
}

// Each family's -text on its -soft is at least 4.5:1, and ink-2 on surface for
// the quiet ok chip, in light and dark. ok-text on ok-soft is the ok tile's icon
// (non-text, 3:1), so it is not part of the text check.
func TestStateContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		pairs := [][2]string{{"--vn-info-text", "--vn-info-soft"}, {"--vn-warn-text", "--vn-warn-soft"}, {"--vn-crit-text", "--vn-crit-soft"}, {"--vn-idle-text", "--vn-idle-soft"}, {"--vn-ink-2", "--vn-surface"}, {"--vn-ink-2", "--vn-surface-2"}}
		for _, p := range pairs {
			x, y := vnLuminance(t, vnResolve(t, p[0], dark)), vnLuminance(t, vnResolve(t, p[1], dark))
			if x < y {
				x, y = y, x
			}
			if r := (x + .05) / (y + .05); r < 4.5 {
				t.Errorf("dark=%v %s on %s = %.2f, need 4.5", dark, p[0], p[1], r)
			}
		}
	}
}

func TestGalleryShowsStateComponents(t *testing.T) {
	body := string(galleryHTML(t))
	for _, want := range []string{"09 · State components", "gyst-chip", "gyst-sev-tile", "gyst-banner", "gyst-empty", "gyst-stale", "data-gyst-offline", "Stale · 6:40", "Last good data kept", "Unknown"} {
		if !strings.Contains(body, want) {
			t.Errorf("gallery missing %q", want)
		}
	}
	if n := strings.Count(body, `class="gyst-sev-tile"`); n != 10 {
		t.Errorf("state components must appear in both theme panels, found %d tiles", n)
	}
}
