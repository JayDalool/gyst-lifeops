package handlers

import (
	"strings"
	"testing"
)

// GYST-VN-019 (#42, DD-1): Settings → Appearance is one three-value control,
// System / Light / Dark, that replaces the "Light mode" checkbox. theme.js
// owns the pressed state; the server renders System pressed as the default.
func TestSettingsAppearanceIsSystemLightDark(t *testing.T) {
	h := authHandler(t)
	page := getSettings(t, h, signIn(t, h))
	for _, want := range []string{
		`<div class="gyst-scope gyst-segments" role="group" aria-labelledby="appearance-label" aria-describedby="appearance-help">`,
		`<b id="appearance-label">Theme</b>`,
		`<button class="gyst-segment" type="button" data-theme-choice="system" aria-pressed="true">System</button>`,
		`<button class="gyst-segment" type="button" data-theme-choice="light" aria-pressed="false">Light</button>`,
		`<button class="gyst-segment" type="button" data-theme-choice="dark" aria-pressed="false">Dark</button>`,
		`/static/js/theme.js?v=appearance-1`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("GET /settings is missing %s", want)
		}
	}
	// Three in Settings plus the same three in the launcher's Do panel (GYST-103).
	if got := strings.Count(page, "data-theme-choice="); got != 6 {
		t.Errorf("appearance choices = %d, want 6", got)
	}
	if dialog := page[strings.Index(page, "<dialog"):]; strings.Count(dialog, "data-theme-choice=") != 3 {
		t.Errorf("launcher should hold exactly three appearance choices")
	}
	for _, gone := range []string{"data-theme-toggle", "Light mode"} {
		if strings.Contains(page, gone) {
			t.Errorf("GET /settings still renders %q", gone)
		}
	}
}
