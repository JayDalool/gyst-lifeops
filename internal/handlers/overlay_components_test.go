package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Fictional fixtures only. VN-023 overlay components (11 §N, 13 §6-§9).

// overlayGallery holds the fictional overlay examples for the development gallery.
type overlayGallery struct {
	SheetHead, PopoverHead UIOverlayHead
	Menu                   []struct {
		Label       string
		Destructive bool
	}
	Confirm UIConfirm
	Inspect []struct{ Key, Label, Detail string }
}

func overlayGalleryData() overlayGallery {
	g := overlayGallery{
		SheetHead:   UIOverlayHead{ID: "sheet-demo", Title: "A little progress today"},
		PopoverHead: UIOverlayHead{ID: "popover-demo", Title: "Example options"},
		Confirm:     UIConfirm{ID: "confirm-demo", Title: "Delete this example?", Consequence: "This is a preview. Nothing will be deleted.", ConfirmLabel: "Delete example", Destructive: true},
		Inspect: []struct{ Key, Label, Detail string }{
			{"a", "Daily checklist", "Example: 5 small steps, every day"},
			{"b", "Reading goal", "Example: 20 pages a week"},
			{"c", "Review notes", "Example: due on Friday"},
		},
	}
	for _, label := range []string{"Rename", "Duplicate", "Move to another list", "Archive"} {
		g.Menu = append(g.Menu, struct {
			Label       string
			Destructive bool
		}{label, false})
	}
	g.Menu = append(g.Menu, struct {
		Label       string
		Destructive bool
	}{"Delete", true})
	return g
}

func TestOverlayHeadHasNamedCloseButton(t *testing.T) {
	out := componentHTML(t, "gyst.overlay-head", UIOverlayHead{ID: "demo", Title: "Example title"})
	for _, want := range []string{`<h2 id="demo-title"`, `class="gyst-overlay-close"`, `type="button"`, `data-gyst-close`, `aria-label="Close Example title"`} {
		if !strings.Contains(out, want) {
			t.Errorf("overlay head lacks %q: %s", want, out)
		}
	}
}

func TestConfirmDialogWiring(t *testing.T) {
	out := componentHTML(t, "gyst.confirm", UIConfirm{ID: "stop", Title: "Stop this example?", Consequence: "The example run ends.", ConfirmLabel: "Stop", Destructive: true})
	for _, want := range []string{`<dialog id="stop"`, `role="alertdialog"`, `aria-labelledby="stop-title"`, `aria-describedby="stop-desc"`, `<h2 id="stop-title"`, `<p id="stop-desc"`, `data-gyst-confirm-ok`, `gyst-button--danger`} {
		if !strings.Contains(out, want) {
			t.Errorf("confirm lacks %q: %s", want, out)
		}
	}
	cancel := regexp.MustCompile(`<button[^>]*>Cancel</button>`).FindString(out)
	if !strings.Contains(cancel, "autofocus") || !strings.Contains(cancel, `type="button"`) || !strings.Contains(cancel, "data-gyst-close") {
		t.Errorf("Cancel must be a typed, autofocused close button: %q", cancel)
	}
	if strings.Contains(out, "onsubmit") || strings.Contains(out, "onclick") {
		t.Errorf("confirm has inline handlers: %s", out)
	}
	plain := componentHTML(t, "gyst.confirm", UIConfirm{ID: "ok", Title: "Continue?", Consequence: "Nothing is lost.", ConfirmLabel: "Continue"})
	if strings.Contains(plain, "gyst-button--danger") {
		t.Errorf("non-destructive confirm styled as danger: %s", plain)
	}
}

func TestGalleryRendersOverlays(t *testing.T) {
	html := string(galleryHTML(t))
	for _, want := range []string{
		`id="sheet-demo"`, `id="popover-demo"`, `id="confirm-demo"`, `data-gyst-inspector`,
		`data-gyst-open="sheet-demo"`, `data-gyst-open="popover-demo"`, `data-gyst-confirm="confirm-demo"`,
		`data-gyst-inspect-panel="b"`, `GYSTOverlay`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("gallery lacks %q", want)
		}
	}
	if n := strings.Count(html, `class="gyst-menu-item`); n != 5 {
		t.Errorf("popover has %d items, want 5 (max 6)", n)
	}
	if strings.Contains(html, "data-open-dialog") {
		t.Error("old inline dialog opener still present")
	}
}

func TestOverlayCSSRules(t *testing.T) {
	raw, err := os.ReadFile("../../web/static/css/ui-components.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	if strings.Contains(css, "--vn-xp") {
		t.Error("overlay CSS must not use the identity violet")
	}
	// .gyst-inspector is hidden outside the 1280px media query.
	i := strings.Index(css, ".gyst-inspector { display: none; }")
	if i < 0 {
		t.Fatal(".gyst-inspector must default to display:none")
	}
	m := strings.Index(css, "@media (min-width: 1280px)")
	if m < i || !strings.Contains(css[m:], ".gyst-inspector { display: block;") {
		t.Error(".gyst-inspector must only be shown inside @media (min-width: 1280px)")
	}
	for _, sel := range []string{".gyst-overlay-close {", ".gyst-menu-item {"} {
		start := strings.Index(css, sel)
		if start < 0 {
			t.Fatalf("missing %s", sel)
		}
		rule := css[start : start+strings.Index(css[start:], "}")]
		if !strings.Contains(rule, "44px") {
			t.Errorf("%s must be at least 44px: %s", sel, rule)
		}
	}
	for _, want := range []string{"max-height: 88dvh", "html.gyst-scroll-lock", "prefers-reduced-motion", "380px", "440px"} {
		if !strings.Contains(css, want) {
			t.Errorf("css lacks %q", want)
		}
	}
}
