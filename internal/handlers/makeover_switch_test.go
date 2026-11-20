package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// GYST-168, "the switch": the static half of its required tests. The browser
// half (computed styles, contrast audit, Galaxy zero diff, offline font) is
// run by the tools kept with the ticket's evidence.

const legacyAliasSelector = `:root, :root[data-theme="light"]`
const gystAliasSelector = `:root, [data-gyst-theme="light"], [data-gyst-theme="dark"]`

// The legacy variables in styles.css are declared three times: two older
// blocks near the top and the alias block. The alias block must be the last
// word for :root and for :root[data-theme="light"], whose (0,2,0) specificity
// would otherwise beat a lone :root block in light mode.
func TestLegacyVariablesEndAsAliases(t *testing.T) {
	rules := parseCSS(readStatic(t, "css/styles.css"))
	alias := -1
	for i, r := range rules {
		if r.Selector == legacyAliasSelector && r.Media == "" {
			if alias >= 0 {
				t.Fatal("styles.css has two legacy alias blocks")
			}
			alias = i
		}
	}
	if alias < 0 {
		t.Fatalf("styles.css has no %q alias block", legacyAliasSelector)
	}
	legacy := map[string]bool{}
	for _, scope := range []string{":root", `:root[data-theme="light"]`} {
		last := map[string]int{}
		for i, r := range rules {
			if !r.has(scope) {
				continue
			}
			for _, n := range r.Names {
				if strings.HasPrefix(n, "--") && !strings.HasPrefix(n, "--vn-") && !strings.HasPrefix(n, "--gyst-") && !strings.HasPrefix(n, "--health-") {
					last[n] = i
					legacy[n] = true
				}
			}
		}
		if len(last) == 0 {
			t.Fatalf("no legacy variable found for %s", scope)
		}
		for n, i := range last {
			if i != alias {
				t.Errorf("%s: the last declaration of %s is in %q (%s), not the alias block", scope, n, rules[i].Selector, rules[i].Media)
			}
		}
	}
	for n := range legacy {
		v, ok := rules[alias].Decls[n]
		if !ok {
			t.Errorf("legacy variable %s is missing from the alias block", n)
			continue
		}
		// Shadows and glows are switched off; everything else follows a token.
		if !strings.Contains(v, "var(--vn-") && !strings.HasPrefix(v, "0 0 0 ") {
			t.Errorf("legacy variable %s = %q does not follow a --vn- token", n, v)
		}
	}
	for _, n := range []string{"--bg", "--panel", "--text", "--muted", "--quiet", "--border", "--accent", "--cyan", "--green", "--red"} {
		if !legacy[n] {
			t.Errorf("expected legacy variable %s was not seen; the test is reading the wrong rules", n)
		}
	}
}

// Each theme-dependent --gyst-* name is an alias, declared once, on all three
// theme scopes, so a nested theme scope resolves it against its own tokens.
func TestGystAliasesOnAllThemeScopes(t *testing.T) {
	rules := parseCSS(readStatic(t, "css/"+tokensCSS))
	count := map[string]int{}
	var alias *cssRule
	for i, r := range rules {
		for _, n := range r.Names {
			if strings.HasPrefix(n, "--gyst-") {
				count[n]++
				// --gyst-border's dark compatibility rules are checked on their own below.
				if n != "--gyst-border" && strings.Contains(r.Decls[n], "var(--vn-") && r.Selector != gystAliasSelector {
					t.Errorf("%s is an alias but sits on %q, not on all three theme scopes", n, r.Selector)
				}
			}
		}
		if r.Selector == gystAliasSelector {
			alias = &rules[i]
		}
	}
	if alias == nil {
		t.Fatalf("ui-tokens.css has no %q block", gystAliasSelector)
	}
	// One exception, a legacy-alias compatibility value and not a 13B token: in dark, --gyst-border is
	// stronger than the separator so that the control outlines reading it are not fainter than on
	// master. It is restated on the two dark selectors right after the alias block.
	const darkBorder = "color-mix(in srgb, var(--vn-ink) 20%, var(--vn-surface))"
	darkRules := 0
	for _, r := range rules {
		if v, ok := r.Decls["--gyst-border"]; ok && r.Selector != gystAliasSelector {
			darkRules++
			dark := (r.Selector == selDark && r.Media == "") || (r.Selector == selSystemDark && r.Media == "@media (prefers-color-scheme: dark)")
			if !dark || v != darkBorder || len(r.Decls) != 1 {
				t.Errorf("unexpected --gyst-border rule %q (%s): %v", r.Selector, r.Media, r.Decls)
			}
		}
	}
	if darkRules != 2 {
		t.Errorf("--gyst-border has %d dark compatibility rules, want 2 (the toggle and the OS setting)", darkRules)
	}
	for n, c := range count {
		want := 1
		if n == "--gyst-border" {
			want = 3
		}
		if c != want {
			t.Errorf("%s is declared %d times, want %d", n, c, want)
		}
	}
	for n, want := range map[string]string{
		"--gyst-bg": "var(--vn-canvas)", "--gyst-surface": "var(--vn-surface)", "--gyst-text": "var(--vn-ink)",
		"--gyst-text-secondary": "var(--vn-ink-2)", "--gyst-control-text": "var(--vn-ink-2)", "--gyst-border": "var(--vn-separator)",
		"--gyst-control-border": "var(--vn-ink-3)", "--gyst-blue": "var(--vn-accent-text)", "--gyst-purple": "var(--vn-accent-text)",
		"--gyst-blue-soft": "var(--vn-accent-soft)", "--gyst-control-bg": "var(--vn-surface-2)",
		"--gyst-green": "var(--vn-ok)", "--gyst-orange": "var(--vn-warn)", "--gyst-red": "var(--vn-crit)",
		"--gyst-positive-text": "var(--vn-ok-text)", "--gyst-warning-text": "var(--vn-warn-text)", "--gyst-danger-text": "var(--vn-crit-text)",
		"--gyst-font": "var(--vn-font)", "--gyst-radius-sm": "var(--vn-r-md)", "--gyst-radius-md": "var(--vn-r-lg)", "--gyst-radius-xl": "var(--vn-r-xl)",
		"--gyst-shadow-card": "0 0 0 transparent", "--gyst-shadow-raised": "var(--vn-e-float)", "--gyst-shadow-floating": "var(--vn-e-float)",
		"--gyst-duration": "var(--vn-base)",
	} {
		if got := alias.Decls[n]; got != want {
			t.Errorf("%s = %q, want %q", n, got, want)
		}
	}
}

// The sign-in page is dark through a static attribute and no script, so the
// generated file must carry the default Look's dark values on a plain
// [data-gyst-theme="dark"] selector, with no :root. The accent is read from
// the Aurora block rather than repeated here: Aurora's starting accent is one
// seed in scripts/build-accents.cjs.
func TestAccentsDarkValuesOnPlainSelector(t *testing.T) {
	_, _, dark := themeBlocks(t, accentsCSS)
	var aurora *cssRule
	rules := parseCSS(readStatic(t, "css/"+accentsCSS))
	for i, r := range rules {
		if r.Media == "" && r.has(`:root[data-look="aurora"][data-gyst-theme="dark"]`) {
			aurora = &rules[i]
		}
	}
	if aurora == nil {
		t.Fatal("ui-accents.css has no Aurora dark block")
	}
	if len(dark.Decls) != len(aurora.Decls) {
		t.Errorf("the plain dark block has %d tokens, Aurora dark has %d", len(dark.Decls), len(aurora.Decls))
	}
	for n, v := range aurora.Decls {
		if dark.Decls[n] != v {
			t.Errorf("plain [data-gyst-theme=\"dark\"] %s = %q, Aurora dark is %q", n, dark.Decls[n], v)
		}
	}
	if dark.Decls["--vn-canvas"] != "#09080F" {
		t.Errorf("dark canvas = %q, want #09080F", dark.Decls["--vn-canvas"])
	}
	for _, n := range []string{"--vn-accent", "--vn-accent-text", "--vn-accent-soft", "--vn-on-accent"} {
		if dark.Decls[n] == "" {
			t.Errorf("the plain dark block lacks %s", n)
		}
	}
}

func hexLuminance(t *testing.T, hex string) float64 {
	t.Helper()
	if !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(hex) {
		t.Fatalf("%q is not a #RRGGBB colour", hex)
	}
	ch := func(i int) float64 {
		n, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		v := float64(n) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(1) + 0.7152*ch(3) + 0.0722*ch(5)
}

func contrastRatio(t *testing.T, a, b string) float64 {
	t.Helper()
	la, lb := hexLuminance(t, a), hexLuminance(t, b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// No template renders .medal-disc or .medal-chip, so a browser audit cannot
// see them. Their text is --vn-on-accent on a gradient of --cyan, --green and
// --blue; after aliasing, every stop must give that text 4.5:1 in both themes.
func TestMedalTextContrastOnEveryGradientStop(t *testing.T) {
	styles := readStatic(t, "css/styles.css")
	for _, sel := range []string{".medal-disc", ".medal-chip"} {
		found := false
		for _, r := range parseCSS(styles) {
			if r.Selector == sel && r.Decls["background"] != "" {
				found = true
				if r.Decls["color"] != "var(--vn-on-accent)" {
					t.Errorf("%s color = %q, want var(--vn-on-accent)", sel, r.Decls["color"])
				}
				for _, ref := range regexp.MustCompile(`var\((--[a-z-]+)\)`).FindAllStringSubmatch(r.Decls["background"], -1) {
					if ref[1] != "--cyan" && ref[1] != "--green" && ref[1] != "--blue" {
						t.Errorf("%s gradient uses %s, which this test does not cover", sel, ref[1])
					}
				}
			}
		}
		if !found {
			t.Fatalf("%s rule with a background not found", sel)
		}
	}
	var alias cssRule
	for _, r := range parseCSS(styles) {
		if r.Selector == legacyAliasSelector {
			alias = r
		}
	}
	light, _, dark := themeBlocks(t, accentsCSS)
	for theme, block := range map[string]cssRule{"light": light, "dark": dark} {
		text := block.Decls["--vn-on-accent"]
		for _, stop := range []string{"--cyan", "--green", "--blue"} {
			ref := regexp.MustCompile(`^var\((--vn-[a-z0-9-]+)\)$`).FindStringSubmatch(alias.Decls[stop])
			if ref == nil {
				t.Fatalf("%s = %q is not a single token", stop, alias.Decls[stop])
			}
			fill := block.Decls[ref[1]]
			if ratio := contrastRatio(t, text, fill); ratio < 4.5 {
				t.Errorf("%s: --vn-on-accent %s on %s (%s, %s) is %.2f:1, needs 4.5:1", theme, text, stop, ref[1], fill, ratio)
			}
		}
	}
}

// The colour-only swaps of GYST-168: each rule that used the accent fill as
// text, or a literal on an aliased fill, now reads the token made for it.
func TestMakeoverColourSwaps(t *testing.T) {
	type swap struct{ file, selector, prop, want string }
	swaps := []swap{
		{"ui-health-v3.css", `.hv3-tabbar > a[aria-current="page"]`, "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3 a", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-disclose > summary", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-days .is-today", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-tile", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-kdays span.today", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-av", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-tile-h a:hover", "color", "var(--vn-accent-text)"},
		{"ui-health-v3.css", ".hv3-logw", "color", "var(--vn-accent-text)"},
		{"styles.css", `.completion-toast button[type="submit"]`, "color", "var(--vn-accent-text)"},
		{"styles.css", ":root .btn", "color", "var(--vn-on-accent)"},
		{"styles.css", ".medal-disc", "color", "var(--vn-on-accent)"},
		{"styles.css", ".medal-chip", "color", "var(--vn-on-accent)"},
		{"styles.css", ".login-submit", "color", "var(--vn-on-accent)"},
		{"ui-health-v3.css", ".hv3-me", "color", "var(--vn-on-accent)"},
		// A logged workout is a completed state: success fill, never the accent (13A DD-4, 13B DD-6).
		{"ui-health-v3.css", ".hv3-day.done", "background", "var(--vn-ok-text)"},
		{"ui-health-v3.css", ".hv3-day.done", "color", "var(--vn-surface)"},
		// "Current" is a state too, so it stays on the info token (E4).
		{"styles.css", ":root .status.current", "color", "var(--vn-info-text)"},
		{"ui-health-v3.css", ".hv3-btn.secondary", "color", "var(--vn-accent-text)"},
		// Primary controls take the accent everywhere but on the pinned Galaxy page.
		{"ui-components.css", ":where(body:not(.galaxy-page)) .gyst-button", "background", "var(--vn-accent)"},
		{"ui-components.css", ":where(body:not(.galaxy-page)) .gyst-button", "color", "var(--vn-on-accent)"},
		{"ui-components.css", ":where(body:not(.galaxy-page)) .gyst-button:hover", "background", "color-mix(in srgb, var(--vn-accent) 86%, var(--vn-ink))"},
		// One primary fill: the legacy button takes the accent too, off the pinned Galaxy page.
		{"styles.css", ":root :where(body:not(.galaxy-page)) .btn", "background", "var(--vn-accent)"},
		// E4: current and active state graphics read the info colour, never the accent.
		{"styles.css", ":root .phase-strip i.current", "background", "var(--vn-info)"},
		{"styles.css", ":root .map .node.current i", "background", "var(--vn-info)"},
		{"styles.css", ":root .map .node.current i", "border-color", "var(--vn-info)"},
		{"styles.css", ":root .map .node.current::after", "border", "1.5px solid var(--vn-info)"},
		// The legacy button's hover is the same opaque fill; the old 0.9 opacity is reset off Galaxy.
		{"styles.css", ":root :where(body:not(.galaxy-page)) .btn:where(:hover)", "background", "color-mix(in srgb, var(--vn-accent) 86%, var(--vn-ink))"},
		{"styles.css", ":root :where(body:not(.galaxy-page)) .btn:hover", "opacity", "1"},
		{"styles.css", ":root .btn:hover", "opacity", "0.9"}, // Galaxy still reads this one
		// Secondary legacy buttons had only that fade as hover feedback; each now has an opaque tint.
		{"ui-health.css", ":root .health-ui .btn:hover", "background", "var(--gyst-blue-soft)"},
		{"ui-courses.css", ":root .courses-ui .btn:hover", "background", "var(--gyst-blue-soft)"},
		{"ui-courses.css", ":root .courses-ui .btn.danger:hover", "background", "var(--vn-crit-soft)"},
		{"styles.css", ":root :where(body:not(.galaxy-page)) .btn.danger:hover", "background", "var(--vn-crit-soft)"},
		// The current prayer on Today is a state too: border here, outline in ui-today.css (its second
		// colour-only edit, by the lead's ruling), so the tile is one colour.
		{"styles.css", ":root .tracker-dot.prayer-current", "border-color", "var(--vn-info)"},
		{"ui-today.css", ".today-page .tracker-dot.prayer-current", "outline", "2px solid var(--vn-info)"},
		{"ui-components.css", `:where(body:not(.galaxy-page)) .gyst-segment[aria-pressed="true"]`, "background", "var(--vn-accent)"},
		{"ui-components.css", `:where(body:not(.galaxy-page)) .gyst-segment[aria-current="page"]`, "color", "var(--vn-on-accent)"},
		{"ui-components.css", ":where(body:not(.galaxy-page)) .gyst-switch:checked", "background", "var(--vn-accent)"},
		{"ui-components.css", ":where(body:not(.galaxy-page)) .gyst-choice input", "accent-color", "var(--vn-accent)"},
		{"ui-shell.css", ":where(body:not(.galaxy-page)) .gyst-skip", "background", "var(--vn-accent)"},
		{"ui-projects.css", `.projects-ui input[type="checkbox"]`, "accent-color", "var(--vn-accent)"},
		// The one edit in ui-today.css, colour only (lead ruling): the checkbox beside the accent "Add task".
		{"ui-today.css", `.today-page input[type="checkbox"]`, "accent-color", "var(--vn-accent)"},
		// Galaxy reads these base rules, so they keep the literal it had.
		{"ui-components.css", ".gyst-button:hover", "background", "#1d4ed8"},
		{"ui-components.css", ".gyst-switch:checked", "background", "#2563eb"},
		{"styles.css", ".login-input", "border", "1px solid var(--vn-ink-3)"},
		// Demanded by the contrast audit: a field edge at 3:1 and a readable placeholder.
		{"ui-health-v3.css", ".hv3-field input", "border", "1px solid var(--vn-ink-3)"},
		{"styles.css", "body:not(.galaxy-page) :is(input, textarea)::placeholder", "color", "var(--vn-ink-3)"},
		{"styles.css", "body", "font-family", "var(--vn-font)"},
		{"styles.css", "body:not(.galaxy-page)", "-webkit-font-smoothing", "antialiased"},
	}
	parsed := map[string][]cssRule{}
	for _, s := range swaps {
		if parsed[s.file] == nil {
			parsed[s.file] = parseCSS(readStatic(t, "css/"+s.file))
		}
		got, seen := "", false
		for _, r := range parsed[s.file] {
			// The whole selector is tried first: a list split on commas would cut :is(a, b) in two.
			if v, ok := r.Decls[s.prop]; ok && (r.Selector == s.selector || r.has(s.selector)) {
				got, seen = v, true // the last one wins, as in the cascade
			}
		}
		if !seen {
			t.Errorf("%s: no %s declaration on %s", s.file, s.prop, s.selector)
		} else if got != s.want {
			t.Errorf("%s: %s { %s: %s }, want %s", s.file, s.selector, s.prop, got, s.want)
		}
	}
}

// The primary button's hover fill is the accent with 14% ink mixed in. It must differ from the resting
// fill in both themes (in dark, accent-text equals the accent, so that token showed no change) and keep
// the label at 4.5:1.
func TestPrimaryButtonHoverIsDistinctAndReadable(t *testing.T) {
	channel := func(hex string, i int) float64 {
		n, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		return float64(n)
	}
	light, _, dark := themeBlocks(t, accentsCSS)
	for theme, block := range map[string]cssRule{"light": light, "dark": dark} {
		accent, ink, label := block.Decls["--vn-accent"], block.Decls["--vn-ink"], block.Decls["--vn-on-accent"]
		hover := "#"
		for _, i := range []int{1, 3, 5} {
			hover += strings.ToUpper(hex.EncodeToString([]byte{byte(math.Round(0.86*channel(accent, i) + 0.14*channel(ink, i)))}))
		}
		if hover == strings.ToUpper(accent) {
			t.Errorf("%s: hover fill %s equals the resting fill", theme, hover)
		}
		rest, over := contrastRatio(t, label, accent), contrastRatio(t, label, hover)
		if over < 4.5 || over < rest {
			t.Errorf("%s: label %s on hover %s is %.2f:1 (resting %.2f:1); it needs 4.5:1 and must not drop", theme, label, hover, over, rest)
		}
	}
}

// accentNames are the custom properties that resolve to the accent family after GYST-168: the accent
// tokens themselves and every older name aliased onto them.
var accentNames = map[string]bool{
	"--vn-accent": true, "--vn-accent-text": true, "--vn-accent-soft": true, "--vn-on-accent": true,
	"--vn-xp": true, "--vn-xp-soft": true,
	"--accent": true, "--accent-text": true, "--accent-soft": true, "--cyan": true, "--blue": true, "--purple": true,
	"--gyst-blue": true, "--gyst-blue-soft": true, "--gyst-purple": true,
}

// The accent never means a state (13A DD-4, 13B DD-6, AGENTS.md; E4 keeps info blue). Every selector
// here marks a state: current, active, done, info. Its colour is followed through the alias blocks
// and must never reach an accent name. A new state selector belongs in this list.
func TestStateSelectorsNeverResolveToTheAccent(t *testing.T) {
	// Where an older name is defined: the legacy alias block, the --gyst-* aliases, the static tokens.
	defs := map[string]string{}
	for _, r := range parseCSS(readStatic(t, "css/styles.css")) {
		if r.Selector == legacyAliasSelector {
			for n, v := range r.Decls {
				defs[n] = v
			}
		}
	}
	for _, r := range parseCSS(readStatic(t, "css/"+tokensCSS)) {
		if r.Selector == gystAliasSelector || r.Selector == selLight {
			for n, v := range r.Decls {
				defs[n] = v
			}
		}
	}
	// The alias table itself must still say what this test assumes.
	for n := range accentNames {
		if strings.HasPrefix(n, "--vn-accent") || n == "--vn-on-accent" {
			continue
		}
		if v, ok := defs[n]; !ok || !strings.Contains(v, "var(--vn-accent") {
			t.Errorf("%s = %q no longer points at the accent; update accentNames", n, v)
		}
	}
	ref := regexp.MustCompile(`var\((--[a-z0-9-]+)`)
	var reaches func(value string, depth int) string
	reaches = func(value string, depth int) string {
		for _, m := range ref.FindAllStringSubmatch(value, -1) {
			if accentNames[m[1]] {
				return m[1]
			}
			if next, ok := defs[m[1]]; ok && depth < 8 {
				if hit := reaches(next, depth+1); hit != "" {
					return m[1] + " -> " + hit
				}
			}
		}
		return ""
	}
	type state struct{ file, selector string }
	states := []state{
		{"ui-components.css", ".gyst-badge--blue"},
		{"ui-components.css", `.gyst-chip[data-state="info"]`},
		{"ui-components.css", `[data-state="info"] > .gyst-dot`},
		{"ui-components.css", `.gyst-sev-tile[data-state="info"]`},
		{"ui-components.css", `.gyst-banner[data-state="info"]`},
		{"styles.css", ":root .status.current"},
		{"styles.css", ":root .status.done"},
		{"styles.css", ".status-pending"},
		{"styles.css", ":root .phase-strip i.current"},
		{"styles.css", ":root .phase-strip i.done"},
		{"styles.css", ":root .map .node.current i"},
		{"styles.css", ":root .map .node.current::after"},
		{"styles.css", ":root .map .node.done i"},
		{"styles.css", ":root .map .route-done"},
		{"styles.css", ":root .tracker-dot.prayer-current"},
		{"ui-today.css", ".today-page .tracker-dot.prayer-current"},
		{"ui-health-v3.css", ".hv3-day.done"},
		{"ui-health-v3.css", ".hv3-check.done"},
		{"ui-training.css", ".training-ui .training-day.done"},
	}
	parsed := map[string][]cssRule{}
	for _, s := range states {
		if parsed[s.file] == nil {
			parsed[s.file] = parseCSS(readStatic(t, "css/"+s.file))
		}
		found := false
		for _, r := range parsed[s.file] {
			if r.Selector != s.selector && !r.has(s.selector) {
				continue
			}
			found = true
			for _, prop := range []string{"color", "background", "border", "border-color", "outline", "stroke", "fill", "box-shadow"} {
				if hit := reaches(r.Decls[prop], 0); hit != "" {
					t.Errorf("%s: state selector %s { %s: %s } resolves to the accent through %s", s.file, s.selector, prop, r.Decls[prop], hit)
				}
			}
		}
		if !found {
			t.Errorf("%s: state selector %s not found; fix the list", s.file, s.selector)
		}
	}
}

// 13B DD-6 pairs the accent fill with --vn-on-accent. Every rule that fills something with the accent is
// either a graphic with no label (listed here) or a control, and a control's label must be
// --vn-on-accent: not a literal, not inherited, not a link colour. A new accent-filled selector fails
// until it is added to one side, and so does any rule that recolours the label of a filled control.
func TestAccentFillsCarryOnAccent(t *testing.T) {
	graphics := map[string]bool{
		// Bars, dots and knobs: nothing is written on them.
		":root .progress span": true, ".goal-row-bar .progress > span": true, ".hv3-bar i": true, ".hv3-weekbars span": true,
		".hv3-kbars span.today": true, ":root .switch input:checked + i::after": true, ".gyst-switch:checked": true,
		".training-ui .training-bar i": true,
		// Galaxy's own state dot; the page is pinned (E1).
		":root .galaxy-state.state-active, :root .galaxy-state.state-current": true,
		// Older rules for classes that no template renders any more (dots on the old map).
		".mission-spark": true, ".node-active .mission-spark": true, ".stop-dot": true, ".stop-row.node-active .stop-dot": true,
		".tree-dot": true, ".node-active > .tree-dot, .tree-trip-head.node-active .tree-dot": true, ".mission-node .m-dot": true,
	}
	accentFill := regexp.MustCompile(`^(var\(--(vn-accent|accent|cyan|blue|purple|gyst-blue|vn-accent-text)(, [^)]*)?\)|color-mix\(in srgb, var\(--vn-accent\) \d+%, var\(--vn-ink\)\))$`)
	scope := ":where(body:not(.galaxy-page)) "
	controls := 0
	for _, file := range []string{"styles.css", "ui-components.css", "ui-shell.css", "ui-health-v3.css", "ui-health.css", "ui-training.css", "ui-courses.css", "ui-projects.css", "ui-today.css"} {
		rules := parseCSS(readStatic(t, "css/"+file))
		// The label colour of a selector: its own rule, or the same selector without the off-Galaxy scope.
		colourOf := func(sel string) (string, bool) {
			got, ok := "", false
			for _, r := range rules {
				if r.Selector == sel {
					if v, has := r.Decls["color"]; has {
						got, ok = v, true
					}
				}
			}
			return got, ok
		}
		for _, r := range rules {
			fill := r.Decls["background"]
			if fill == "" {
				fill = r.Decls["background-color"]
			}
			if !accentFill.MatchString(fill) {
				continue
			}
			plain := strings.ReplaceAll(r.Selector, scope, "")
			if graphics[plain] {
				continue
			}
			controls++
			// A hover or pressed rule changes the fill only; the label comes from the resting rule.
			base := regexp.MustCompile(`:where\(:hover\)|:hover|:active`).ReplaceAllString(plain, "")
			colour, ok := colourOf(r.Selector)
			for _, candidate := range []string{scope + base, ":root " + scope + strings.TrimPrefix(base, ":root "), base} {
				if !ok {
					colour, ok = colourOf(candidate)
				}
			}
			if !ok || colour != "var(--vn-on-accent)" {
				t.Errorf("%s: %s is filled with %s but its label colour is %q; it must be var(--vn-on-accent), or the selector belongs in the graphics list", file, r.Selector, fill, colour)
			}
		}
		// No rule may recolour the label of a filled control with anything else.
		for _, r := range rules {
			c, ok := r.Decls["color"]
			if !ok || c == "var(--vn-on-accent)" {
				continue
			}
			s := strings.ReplaceAll(r.Selector, scope, "")
			filled := (strings.Contains(s, ".hv3-btn") && !strings.Contains(s, ".hv3-btn.secondary") && !strings.Contains(s, ".hv3-btn.ghost")) ||
				s == ".gyst-button" || s == ":root .btn" || s == ".login-submit" || s == ".hv3-me" || s == ".gyst-skip"
			// The base rules that Galaxy still reads keep the literal they had (its accent is pinned).
			galaxyBase := (s == ".gyst-button" || s == ".gyst-skip") && !strings.Contains(r.Selector, scope)
			if filled && !galaxyBase {
				t.Errorf("%s: %s sets color %s on an accent-filled control; it must be var(--vn-on-accent)", file, r.Selector, c)
			}
		}
	}
	if controls < 8 {
		t.Errorf("only %d accent-filled control rules found; the test is not reading the stylesheets it should", controls)
	}
}

const geistURL = "/static/fonts/Geist-Variable.woff2?v=1.3.0"

func sha256File(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile("../../web/static/" + rel)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func shellAssets(t *testing.T) (cache string, assets []string) {
	t.Helper()
	worker := readStatic(t, "service-worker.js")
	m := regexp.MustCompile(`const CACHE = "([^"]+)";`).FindStringSubmatch(worker)
	if m == nil {
		t.Fatal("service-worker.js has no CACHE name")
	}
	line := strings.Split(strings.Split(worker, "const SHELL = ")[1], ";\n")[0]
	if err := json.Unmarshal([]byte(line), &assets); err != nil {
		t.Fatal(err)
	}
	return m[1], assets
}

func readTemplate(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../../web/templates/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The font is the unmodified GeistVF.woff2 of vercel/geist-font release 1.3.0
// with that release's licence beside it, and one URL names it everywhere: a
// different string in any one place fetches it twice or misses the offline copy.
func TestGeistFontSelfHosted(t *testing.T) {
	if got := sha256File(t, "fonts/Geist-Variable.woff2"); got != "28258d0621216948416a859d32487ab6ad1c9effa0d08795698e70be3c917630" {
		t.Errorf("Geist-Variable.woff2 sha256 = %s; it is not the 1.3.0 release file", got)
	}
	if got := sha256File(t, "licenses/geist.txt"); got != "f38cbd54a72dd027dc854a9bbac9eb5bb8b9b1d6e4b43f037435336ab62a63fc" {
		t.Errorf("geist.txt sha256 = %s; it is not the 1.3.0 LICENSE.TXT", got)
	}
	tokens := readStatic(t, "css/"+tokensCSS)
	var face *cssRule
	rules := parseCSS(tokens)
	for i, r := range rules {
		if r.Selector == "@font-face" {
			if face != nil {
				t.Fatal("more than one @font-face")
			}
			face = &rules[i]
		}
	}
	if face == nil {
		t.Fatal("ui-tokens.css has no @font-face")
	}
	for prop, want := range map[string]string{
		"font-family": `"Geist"`, "font-weight": "100 900", "font-style": "normal", "font-display": "swap",
		"src": `url("` + geistURL + `") format("woff2")`,
	} {
		if face.Decls[prop] != want {
			t.Errorf("@font-face %s = %q, want %q", prop, face.Decls[prop], want)
		}
	}
	preload := `<link rel="preload" href="` + geistURL + `" as="font" type="font/woff2" crossorigin>`
	layout, login := readTemplate(t, "layout.html"), readTemplate(t, "login.html")
	// Galaxy keeps its own font, so its page must not preload (or fetch) Geist.
	if !strings.Contains(layout, `{{if ne .Page "galaxy"}}`+preload+`{{end}}`) {
		t.Error("layout.html must preload the font on every page except Galaxy")
	}
	if strings.Count(layout, "Geist-Variable") != 1 {
		t.Error("layout.html names the font more than once")
	}
	if !strings.Contains(login, preload) {
		t.Error("login.html must preload the font")
	}
	_, assets := shellAssets(t)
	found := false
	for _, a := range assets {
		found = found || a == geistURL
	}
	if !found {
		t.Error("the font URL is not in the service worker SHELL, so it would not be available offline")
	}
	// No remote asset anywhere: the CSP allows only this origin.
	for name, body := range map[string]string{"ui-tokens.css": tokens, "ui-pins.css": readStatic(t, "css/ui-pins.css"), "layout.html": layout, "login.html": login} {
		if regexp.MustCompile(`(?i)(?:url\(|href=|src=)["']?(?:https?:)?//`).MatchString(body) {
			t.Errorf("%s refers to an off-origin asset", name)
		}
	}
}

// The sign-in page stays dark on any OS setting through a static attribute:
// no data-theme keeps the legacy side dark, data-gyst-theme="dark" selects the
// dark tokens. It has no script and its form is untouched.
func TestLoginPageStaysDarkWithoutScript(t *testing.T) {
	login := readTemplate(t, "login.html")
	if strings.Contains(strings.ToLower(login), "<script") {
		t.Error("login.html must not contain a script")
	}
	if !strings.Contains(login, `<html lang="en" data-gyst-theme="dark">`) {
		t.Error(`login.html must set data-gyst-theme="dark" statically and no data-theme`)
	}
	order := []string{
		// Exception E3: the browser bar colour is left as it was; it moves with the shell chunk (R2).
		`<meta name="theme-color" content="#080d16">`,
		`<link rel="stylesheet" href="/static/css/styles.css?v=`,
		`<link rel="stylesheet" href="/static/css/ui-tokens.css?v=`,
		`<link rel="stylesheet" href="/static/css/ui-accents.css?v=`,
		`<form class="login-card" method="post" action="/login" autocomplete="on">`,
		`name="password"`, `type="password"`, `autocomplete="current-password"`,
		`{{if .Next}}<input type="hidden" name="next" value="{{.Next}}">{{end}}`,
		`<button class="login-submit" type="submit">Sign in</button>`,
	}
	at := 0
	for _, part := range order {
		i := strings.Index(login[at:], part)
		if i < 0 {
			t.Fatalf("login.html lacks %s (or it is out of order)", part)
		}
		at += i
	}
}

// Every changed asset has one ?v= string, used identically wherever it is
// named; the cache name moves so installed apps drop the old shell.
func TestMakeoverAssetVersions(t *testing.T) {
	layout, login := readTemplate(t, "layout.html"), readTemplate(t, "login.html")
	cache, assets := shellAssets(t)
	if cache == "lifeops-shell-v81-icons" || !strings.HasPrefix(cache, "lifeops-shell-") {
		t.Errorf("CACHE = %q: it must differ from the last release and keep the lifeops-shell- prefix the activate handler prunes by", cache)
	}
	inShell := map[string]bool{}
	for _, a := range assets {
		inShell[a] = true
	}
	for _, a := range []string{
		"/static/css/styles.css?v=aurora-1", "/static/css/ui-tokens.css?v=aurora-1", "/static/css/ui-accents.css?v=aurora-1",
		"/static/css/ui-pins.css?v=aurora-1", "/static/css/ui-health-v3.css?v=hv3-11",
		"/static/css/ui-galaxy.css?v=galaxywin-1", geistURL,
	} {
		if !strings.Contains(layout, `"`+a+`"`) {
			t.Errorf("layout.html does not load %s", a)
		}
		if !inShell[a] {
			t.Errorf("SHELL lacks %s", a)
		}
	}
	// One version per file: the path may not appear in the layout or the shell with any other ?v=.
	seen := map[string]string{}
	for _, m := range regexp.MustCompile(`/static/[A-Za-z0-9_./-]+\?v=[A-Za-z0-9_.-]+`).FindAllString(layout+login+strings.Join(assets, " "), -1) {
		path, _, _ := strings.Cut(m, "?")
		if prev, ok := seen[path]; ok && prev != m {
			t.Errorf("%s is named with two versions: %s and %s", path, prev, m)
		}
		seen[path] = m
	}
	// Token files load in this order; the pins must come after the tokens they restate.
	at := 0
	for _, css := range []string{"styles.css", "ui-tokens.css", "ui-accents.css", "ui-pins.css", "ui-components.css"} {
		i := strings.Index(layout[at:], "/static/css/"+css+"?v=")
		if i < 0 {
			t.Fatalf("layout.html: %s is missing or out of order", css)
		}
		at += i
	}
}

// ui-pins.css holds the protected Galaxy page and Health v3's type sizes at
// their pre-makeover values. The browser proof is the zero computed-style
// diff; this guards the selectors and the values that proof depends on.
func TestPinsForGalaxyAndHealth(t *testing.T) {
	rules := parseCSS(readStatic(t, "css/ui-pins.css"))
	merged := func(sel string) map[string]string {
		out := map[string]string{}
		for _, r := range rules {
			if r.has(sel) {
				for n, v := range r.Decls {
					out[n] = v
				}
			}
		}
		return out
	}
	hv3 := merged(".hv3")
	for n, want := range map[string]string{"--vn-t-display": "34px", "--vn-t-display-lh": "40px", "--vn-t-title": "28px", "--vn-t-title-lh": "32px", "--vn-t-body-lh": "23px"} {
		if hv3[n] != want {
			t.Errorf(".hv3 %s = %q, want %q", n, hv3[n], want)
		}
	}
	want := map[string]map[string]string{
		"light": {"--vn-on-accent": "#08262e", "--vn-accent-text": "#2563eb", "--vn-accent": "#2563eb", "--gyst-bg": "#f8fafc", "--bg": "#f4f6f8", "--text": "#101923", "--vn-canvas": "#f4f6fa"},
		"dark":  {"--vn-on-accent": "#08262e", "--vn-accent-text": "#6ea8ff", "--vn-accent": "#6ea8ff", "--gyst-bg": "#0f172a", "--bg": "#0f1216", "--text": "#e9eef4", "--vn-canvas": "#0b111c"},
	}
	for theme, values := range want {
		pin := merged(`:root[data-gyst-theme="` + theme + `"] body.galaxy-page`)
		for n, v := range values {
			if pin[n] != v {
				t.Errorf("Galaxy %s pin %s = %q, want %q", theme, n, pin[n], v)
			}
		}
		for n, v := range map[string]string{
			"font-family":    `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,
			"--vn-t-display": "34px", "--vn-r-lg": "16px", "--vn-fast": "150ms", "--vn-base": "240ms", "--gyst-radius-xl": "24px",
		} {
			if pin[n] != v {
				t.Errorf("Galaxy %s pin %s = %q, want %q", theme, n, pin[n], v)
			}
		}
		// A pin that follows a token would move with the makeover; every value is a literal.
		for n, v := range pin {
			if strings.Contains(v, "var(") {
				t.Errorf("Galaxy %s pin %s = %q must be a literal", theme, n, v)
			}
		}
	}
	// Every legacy variable and every aliased --gyst-* name must be pinned in both themes.
	var names []string
	for _, r := range parseCSS(readStatic(t, "css/styles.css")) {
		if r.Selector == legacyAliasSelector {
			for _, n := range r.Names {
				if !strings.HasPrefix(r.Decls[n], "0 0 0 rgba") { // the glows were already zero
					names = append(names, n)
				}
			}
		}
	}
	for _, r := range parseCSS(readStatic(t, "css/"+tokensCSS)) {
		if r.Selector == gystAliasSelector {
			for _, n := range r.Names {
				if n != "--gyst-radius-sm" { // 12px before and after
					names = append(names, n)
				}
			}
		}
	}
	for _, theme := range []string{"light", "dark"} {
		pin := merged(`:root[data-gyst-theme="` + theme + `"] body.galaxy-page`)
		for _, n := range names {
			if pin[n] == "" {
				t.Errorf("Galaxy %s pin lacks %s, which the makeover changes", theme, n)
			}
		}
	}
}
