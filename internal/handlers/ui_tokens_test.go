package handlers

import (
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"
)

// vnToken is one vNext design token (GYST-VN-018). Dark is empty for tokens
// that do not change with the theme; they are declared once, in the light block.
type vnToken struct{ Group, Name, Light, Dark string }

// vnTokens is the 11_DESIGN_SYSTEM_VNEXT.md §C–§I set plus the 13 §1
// breakpoints. It drives both the CSS assertions and the gallery swatches.
var vnTokens = []vnToken{
	{"Type", "--vn-t-display", "34px", ""}, {"Type", "--vn-t-display-lh", "40px", ""},
	{"Type", "--vn-t-title", "28px", ""}, {"Type", "--vn-t-title-lh", "32px", ""},
	{"Type", "--vn-t-section", "20px", ""}, {"Type", "--vn-t-section-lh", "25px", ""},
	{"Type", "--vn-t-body", "16px", ""}, {"Type", "--vn-t-body-lh", "23px", ""},
	{"Type", "--vn-t-sub", "14px", ""}, {"Type", "--vn-t-sub-lh", "19px", ""},
	{"Type", "--vn-t-foot", "13px", ""}, {"Type", "--vn-t-foot-lh", "18px", ""},
	{"Type", "--vn-t-caption", "12px", ""}, {"Type", "--vn-t-caption-lh", "16px", ""},
	{"Spacing", "--vn-s1", "4px", ""}, {"Spacing", "--vn-s2", "8px", ""}, {"Spacing", "--vn-s3", "12px", ""},
	{"Spacing", "--vn-s4", "16px", ""}, {"Spacing", "--vn-s6", "24px", ""}, {"Spacing", "--vn-s8", "32px", ""},
	{"Spacing", "--vn-s12", "48px", ""},
	{"Radii", "--vn-r-sm", "8px", ""}, {"Radii", "--vn-r-md", "12px", ""}, {"Radii", "--vn-r-lg", "16px", ""},
	{"Radii", "--vn-r-xl", "24px", ""}, {"Radii", "--vn-r-pill", "999px", ""},
	{"Motion", "--vn-fast", "150ms", ""}, {"Motion", "--vn-base", "240ms", ""},
	{"Motion", "--vn-ease", "cubic-bezier(.2,.8,.2,1)", ""},
	{"Breakpoints", "--vn-bp-tablet", "600px", ""}, {"Breakpoints", "--vn-bp-laptop", "1024px", ""},
	{"Breakpoints", "--vn-bp-desktop", "1280px", ""}, {"Breakpoints", "--vn-bp-wide", "1600px", ""},
	{"Surfaces", "--vn-canvas", "#f4f6fa", "#0b111c"}, {"Surfaces", "--vn-surface", "#ffffff", "#141c2b"},
	{"Surfaces", "--vn-surface-2", "#eef1f6", "#1b2537"}, {"Surfaces", "--vn-surface-3", "#e4e8ef", "#243047"},
	{"Surfaces", "--vn-elevated", "#ffffff", "#1a2334"},
	{"Galaxy space (always dark)", "--vn-space", "#070b18", ""}, {"Galaxy space (always dark)", "--vn-space-2", "#0f1630", ""},
	{"Elevation", "--vn-e1", "0 1px 0 rgb(14 23 38 / 4%)", "0 0 0 transparent"},
	{"Elevation", "--vn-e2", "0 6px 20px rgb(14 23 38 / 12%), 0 1px 3px rgb(14 23 38 / 8%)", "0 8px 24px rgb(0 0 0 / 45%)"},
	{"Elevation", "--vn-e3", "0 18px 50px rgb(14 23 38 / 22%)", "0 20px 60px rgb(0 0 0 / 60%)"},
	{"Text and lines", "--vn-ink", "#0e1726", "#eef2f8"}, {"Text and lines", "--vn-ink-2", "#566174", "#a9b3c4"},
	{"Text and lines", "--vn-ink-3", "#8790a0", "#7b8699"}, {"Text and lines", "--vn-separator", "#e2e6ed", "#243049"},
	{"Action", "--vn-accent", "#2563eb", "#6ea8ff"}, {"Action", "--vn-accent-soft", "#e8f0ff", "#15284a"},
	{"Identity only (DD-4)", "--vn-xp", "#6d5dfc", "#a99bff"}, {"Identity only (DD-4)", "--vn-xp-soft", "#eeecff", "#231f45"},
	{"ok", "--vn-ok", "#16a34a", "#4ade80"}, {"ok", "--vn-ok-text", "#15803d", "#6ee7a0"}, {"ok", "--vn-ok-soft", "#e7f6ec", "#10301f"},
	{"info", "--vn-info", "var(--vn-accent)", "var(--vn-accent)"}, {"info", "--vn-info-text", "#1d4ed8", "#93bdff"}, {"info", "--vn-info-soft", "var(--vn-accent-soft)", "var(--vn-accent-soft)"},
	{"warn", "--vn-warn", "#d97706", "#fbbf24"}, {"warn", "--vn-warn-text", "#b45309", "#fcd062"}, {"warn", "--vn-warn-soft", "#fdf1e1", "#33260c"},
	{"crit", "--vn-crit", "#dc2626", "#f87171"}, {"crit", "--vn-crit-text", "#b91c1c", "#fca5a5"}, {"crit", "--vn-crit-soft", "#fdeaea", "#3a1618"},
	{"idle", "--vn-idle", "#8790a0", "#7b8699"}, {"idle", "--vn-idle-text", "#566174", "#a9b3c4"}, {"idle", "--vn-idle-soft", "var(--vn-surface-2)", "var(--vn-surface-2)"},
}

// vnBlock returns the declarations of the rule whose selector starts at the
// last occurrence of header (the vNext blocks follow the legacy --gyst ones).
func vnBlock(t *testing.T, css, header string) map[string]string {
	t.Helper()
	i := strings.LastIndex(css, header)
	if i < 0 {
		t.Fatalf("token block %q missing", header)
	}
	body := css[i+len(header):]
	body = body[:strings.Index(body, "}")]
	body = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(body, "")
	out := map[string]string{}
	for _, decl := range strings.Split(body, ";") {
		name, value, ok := strings.Cut(decl, ":")
		if name = strings.TrimSpace(name); ok && strings.HasPrefix(name, "--") {
			out[name] = strings.Join(strings.Fields(value), " ")
		}
	}
	return out
}

func TestVNextTokensDefinedLightAndDark(t *testing.T) {
	raw, err := os.ReadFile("../../web/static/css/ui-tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	light := vnBlock(t, css, `:root, [data-gyst-theme="light"] {`)
	system := vnBlock(t, css, "@media (prefers-color-scheme: dark) {\n  :root:not([data-gyst-theme=\"light\"]) {")
	explicit := vnBlock(t, css, `[data-gyst-theme="dark"] {`)
	if len(explicit) == 0 || strings.Count(css, `[data-gyst-theme="dark"] {`) != 2 {
		t.Fatal("expected the legacy --gyst dark block followed by one vNext dark block")
	}
	for _, tok := range vnTokens {
		if got := light[tok.Name]; got != tok.Light {
			t.Errorf("light %s = %q, want %q", tok.Name, got, tok.Light)
		}
		for mode, block := range map[string]map[string]string{"system dark": system, "explicit dark": explicit} {
			got, ok := block[tok.Name]
			if tok.Dark == "" && ok {
				t.Errorf("%s: theme-independent %s must not be redeclared", mode, tok.Name)
			}
			if tok.Dark != "" && got != tok.Dark {
				t.Errorf("%s %s = %q, want %q", mode, tok.Name, got, tok.Dark)
			}
		}
	}
	if len(system) != len(explicit) {
		t.Errorf("system dark (%d) and explicit dark (%d) blocks differ", len(system), len(explicit))
	}
	// The vNext blocks must never set values for existing tokens.
	for _, block := range []map[string]string{light, system, explicit} {
		for name := range block {
			if !strings.HasPrefix(name, "--vn-") {
				t.Errorf("vNext block declares non-namespaced %s", name)
			}
		}
	}
	if regexp.MustCompile(`[^-]color-scheme\s*:`).MatchString(css) {
		t.Error("ui-tokens.css must not set color-scheme (it restyles native controls on existing screens)")
	}
}

// DD-4: violet is identity only. No state token may resolve to the XP colours.
func TestVNextStateTokensNeverViolet(t *testing.T) {
	violet := map[string]bool{}
	for _, tok := range vnTokens {
		if strings.HasPrefix(tok.Name, "--vn-xp") {
			violet[tok.Light], violet[tok.Dark] = true, true
		}
	}
	for _, tok := range vnTokens {
		switch tok.Group {
		case "ok", "info", "warn", "crit", "idle":
			if violet[tok.Light] || violet[tok.Dark] || strings.Contains(tok.Light+tok.Dark, "--vn-xp") {
				t.Errorf("state token %s uses the violet identity accent", tok.Name)
			}
		}
	}
}

type vnSwatch struct {
	Name, Light, Dark, Kind string
	Var                     template.CSS
}

type vnSwatchGroup struct {
	Label, Word string
	Tokens      []vnSwatch
	// StateStyle follows 11 §I: ok is quiet (--ink-2 text and a green dot),
	// info and idle use their text colour, only warn and crit are tinted chips.
	StateStyle template.CSS
}

type vnGallery struct {
	Themes                 []string
	Colors, States, Scales []vnSwatchGroup
}

// One word per state family from 11 §I; state is never shown by colour alone.
var vnStateStyles = map[string]template.CSS{
	"ok":   "color:var(--vn-ink-2)",
	"info": "color:var(--vn-info-text)",
	"warn": "color:var(--vn-warn-text);background:var(--vn-warn-soft)",
	"crit": "color:var(--vn-crit-text);background:var(--vn-crit-soft)",
	"idle": "color:var(--vn-idle-text)",
}

var vnStateWords = map[string]string{"ok": "Healthy", "info": "Working", "warn": "Stale", "crit": "Failed", "idle": "Paused"}

func vnGalleryData() vnGallery {
	g := vnGallery{Themes: []string{"light", "dark"}}
	add := func(groups *[]vnSwatchGroup, tok vnToken) {
		dark := tok.Dark
		if dark == "" {
			dark = tok.Light
		}
		kind := map[string]string{"Type": "text", "Spacing": "space", "Radii": "radius", "Breakpoints": "bp", "Elevation": "shadow"}[tok.Group]
		if strings.HasSuffix(tok.Name, "-lh") {
			kind = "line"
		}
		s := vnSwatch{Name: tok.Name, Light: tok.Light, Dark: dark, Kind: kind, Var: template.CSS("var(" + tok.Name + ")")}
		if n := len(*groups); n > 0 && (*groups)[n-1].Label == tok.Group {
			(*groups)[n-1].Tokens = append((*groups)[n-1].Tokens, s)
			return
		}
		*groups = append(*groups, vnSwatchGroup{Label: tok.Group, Word: vnStateWords[tok.Group], StateStyle: vnStateStyles[tok.Group], Tokens: []vnSwatch{s}})
	}
	for _, tok := range vnTokens {
		switch tok.Group {
		case "ok", "info", "warn", "crit", "idle":
			add(&g.States, tok)
		case "Type", "Spacing", "Radii", "Motion", "Breakpoints":
			add(&g.Scales, tok)
		default:
			add(&g.Colors, tok)
		}
	}
	return g
}

func TestUIGalleryShowsEveryVNextToken(t *testing.T) {
	body := string(galleryHTML(t))
	for _, tok := range vnTokens {
		if !strings.Contains(body, tok.Name) {
			t.Errorf("gallery does not show %s", tok.Name)
		}
	}
	for _, part := range []string{`data-gyst-theme="light"`, `data-gyst-theme="dark"`, "Identity only (DD-4)", "never a state"} {
		if !strings.Contains(body, part) {
			t.Errorf("gallery token section missing %q", part)
		}
	}
}
