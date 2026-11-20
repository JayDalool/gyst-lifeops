package handlers

import (
	"bytes"
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"
)

// spriteIDs returns the symbol names in the makeover icon sprite, in file order, without the "i-" prefix.
func spriteIDs(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile("../../web/static/icons.svg")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range regexp.MustCompile(`<symbol id="i-([a-z0-9-]+)"`).FindAllStringSubmatch(string(raw), -1) {
		ids = append(ids, m[1])
	}
	if len(ids) == 0 {
		t.Fatal("icons.svg has no symbols")
	}
	return ids
}

// GYST-169, 13B DD-15: the helper draws a symbol from the one same-origin sprite. The markup is
// pinned whole because the href is what keeps the icons inside img-src 'self'.
func TestVNIconHelper(t *testing.T) {
	tmpl, err := template.ParseFiles("../../web/templates/ui_icons.html")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = tmpl.ExecuteTemplate(&out, "vn.icon", "home"); err != nil {
		t.Fatal(err)
	}
	want := `<svg class="vn-icon" aria-hidden="true" focusable="false"><use href="/static/icons.svg?v=r7-1#i-home"/></svg>`
	if out.String() != want {
		t.Fatalf("vn.icon rendered\n%s\nwant\n%s", out.String(), want)
	}
}

func TestVNIconRule(t *testing.T) {
	css := readCSS(t, "ui-components.css")
	i := strings.Index(css, "\n.vn-icon {")
	if i < 0 {
		t.Fatal("ui-components.css has no .vn-icon rule")
	}
	rule := css[i:]
	rule = rule[:strings.Index(rule, "}")]
	// DD-15: 1.75 stroke, round caps and joins, no fill, currentColor; 20 px is the row and button size.
	for _, decl := range []string{"width: 20px;", "height: 20px;", "fill: none;", "stroke: currentColor;", "stroke-width: 1.75;", "stroke-linecap: round;", "stroke-linejoin: round;"} {
		if !strings.Contains(rule, decl) {
			t.Errorf(".vn-icon lacks %q", decl)
		}
	}
}

func TestUIGalleryShowsIconSet(t *testing.T) {
	body := string(galleryHTML(t))
	ids := spriteIDs(t)
	for _, id := range ids {
		if !strings.Contains(body, `<use href="/static/icons.svg?v=r7-1#i-`+id+`"/></svg><code>`+id+`</code>`) {
			t.Errorf("gallery does not show icon %s with its name", id)
		}
	}
	if got := strings.Count(body, `<svg class="vn-icon"`); got != len(ids) {
		t.Errorf("gallery draws %d sprite icons, the sprite has %d", got, len(ids))
	}
}

// TestUIAccentsIsLinkedNowhere stood here while GYST-169 shipped ui-accents.css unlinked. GYST-168 links
// it (layout.html, login.html, the precache list); TestMakeoverAssetVersions now asserts that.

// Issue #259, lead note: the sign-in page stays dark through a static data-gyst-theme="dark" and no
// JS, so the dark values must sit on that plain selector, not only on :root or a chosen Look.
func TestUIAccentsDarkOnPlainThemeAttribute(t *testing.T) {
	css := readCSS(t, "ui-accents.css")
	// Each header is at the start of a line, so neither matches a longer selector that ends the same way.
	dark := vnBlock(t, css, "\n[data-gyst-theme=\"dark\"] {")
	light := vnBlock(t, css, "\n:root, [data-gyst-theme=\"light\"] {")
	// 13B DD-7 (Aurora), DD-13 and DD-6 (Iris), dark.
	for name, want := range map[string]string{
		"--vn-canvas": "#09080F", "--vn-surface": "#13111F", "--vn-surface-2": "#1B182D", "--vn-elevated": "#1C192E",
		"--vn-ink": "#F3F2F5", "--vn-ink-2": "#A7A5B6", "--vn-ink-3": "#8B899A", "--vn-separator": "rgb(255 255 255 / 9%)",
		"--vn-ok-text": "#6EE7A0", "--vn-warn-text": "#FCD062", "--vn-crit-text": "#FCA5A5",
		"--vn-accent": "#96A2FE", "--vn-accent-text": "#96A2FE", "--vn-accent-soft": "#1F2343", "--vn-on-accent": "#0B0D12",
	} {
		if dark[name] != want {
			t.Errorf("plain dark %s = %q, want %q", name, dark[name], want)
		}
	}
	// A token set in light and missing in dark would keep its light value on a dark page.
	for name := range light {
		if _, ok := dark[name]; !ok {
			t.Errorf("plain dark rule lacks %s", name)
		}
	}
	if len(light) == 0 {
		t.Fatal("no light tokens found")
	}
}
