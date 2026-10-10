package handlers

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

// Development fixtures only: no route, database or production renderer includes this page.
func galleryHTML(t *testing.T) []byte {
	t.Helper()
	read := func(name string) template.CSS {
		b, err := os.ReadFile("../../web/static/css/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return template.CSS(b)
	}
	data := struct {
		Tokens, Components template.CSS
		Colors             []struct {
			Name  string
			Value template.CSS
		}
		Icons           []string
		Actions         []UIAction
		Fields          []UIField
		Choices         []UIChoice
		Periods         []UISegment
		Sections        UISegments
		Rows, SheetRows []UIRow
		Tabs            []struct {
			Label, Icon string
			Selected    bool
		}
		Ring, EmptyRing UIMetric
		Metrics         []UIMetric
		States          []UIStateCard
		VN              vnGallery
		SC              stateGallery
	}{Tokens: read("ui-tokens.css"), Components: read("ui-components.css"), VN: vnGalleryData(), SC: stateGalleryData()}
	data.Colors = []struct {
		Name  string
		Value template.CSS
	}{{"Navy", "#0f172a"}, {"Blue", "#2563eb"}, {"Green", "#10b981"}, {"Orange", "#f59e0b"}, {"Purple", "#8b5cf6"}, {"Surface", "#ffffff"}}
	data.Icons = []string{"house", "folder-heart", "heart", "book-open", "bell", "settings"}
	data.Actions = []UIAction{{Label: "Save changes"}, {Label: "Learn more", Variant: "secondary"}, {Label: "Favorite", Icon: "heart", Variant: "icon"}, {Label: "Add example", Icon: "plus", Variant: "fab"}, {Label: "Unavailable", Disabled: true}}
	data.Fields = []UIField{{ID: "search", Name: "search", Label: "Search", Type: "search", Placeholder: "Search anything…"}, {ID: "title", Name: "title", Label: "Goal title", Value: "", Placeholder: "Enter your goal", Required: true, Error: "Give your goal a title."}, {ID: "category", Name: "category", Label: "Category", Type: "select", Options: []UIOption{{Value: "personal", Label: "Personal", Selected: true}, {Value: "learning", Label: "Learning"}}}}
	data.Choices = []UIChoice{{ID: "alerts", Name: "alerts", Label: "Notifications", Kind: "switch", Checked: true}, {ID: "complete", Name: "complete", Label: "Completed example", Checked: true}, {ID: "radio-day", Name: "period", Label: "Daily", Kind: "radio", Value: "daily", Checked: true}, {ID: "radio-week", Name: "period", Label: "Weekly", Kind: "radio", Value: "weekly"}}
	data.Periods = []UISegment{{Label: "Day", Selected: true}, {Label: "Week"}, {Label: "Month"}, {Label: "Year"}}
	data.Sections = UISegments{Label: "Example Health sections", Items: []UISegment{{Label: "Summary", URL: "#metrics-title", Selected: true}, {Label: "Nutrition", URL: "#metrics-title"}, {Label: "Training", URL: "#states-title"}}}
	data.Rows = []UIRow{{Label: "Daily checklist", Detail: "Small steps, every day", Icon: "check", URL: "#controls-title"}, {Label: "Reminders", Meta: "Example", Icon: "bell", URL: "#states-title"}, {Label: "Settings", Icon: "settings", URL: "#foundation-title"}}
	data.SheetRows = []UIRow{{Label: "New item", Icon: "plus"}, {Label: "Reminder", Icon: "bell"}}
	data.Tabs = []struct {
		Label, Icon string
		Selected    bool
	}{{"Today", "house", true}, {"Projects", "folder-heart", false}, {"Health", "heart", false}, {"Courses", "book-open", false}, {"More", "ellipsis", false}}
	data.Ring = UIMetric{Label: "Food consumed · fixture", Value: "1,440", Unit: "kcal", Present: true, Detail: "of 2,000 kcal example target", Progress: UIProgress{Present: true, Percent: 72}, Status: UIStatus{Message: "Example only"}}
	data.EmptyRing = UIMetric{Label: "Food consumed · missing", Status: UIStatus{State: "empty", Message: "No intake synced yet"}}
	data.Metrics = []UIMetric{
		{Label: "Protein · fixture", Value: "96", Unit: "g", Present: true, Detail: "of 120 g example target", Progress: UIProgress{Present: true, Percent: 80, Label: "Example protein: 96 of 120 grams", Detail: "24 g remaining"}},
		{Label: "Watch steps · observed zero", Value: "0", Unit: "steps", Present: true, Status: UIStatus{Message: "An observed zero"}},
		{Label: "Active energy · partial", Value: "240", Unit: "kcal", Present: true, Status: UIStatus{State: "partial", Message: "Resting energy not received"}},
		{Label: "Watch steps · stale", Value: "4,200", Unit: "steps", Present: true, Status: UIStatus{State: "stale", Message: "Example updated 9 hours ago"}},
		{Label: "Weight · missing", Status: UIStatus{State: "empty", Message: "Not synced"}},
	}
	data.States = []UIStateCard{{Title: "No items yet", Icon: "inbox", Status: UIStatus{State: "empty", Message: "Your next step starts here"}, Action: UIAction{Label: "Explore examples", URL: "#controls-title", Variant: "secondary"}}, {Title: "Loading your summary", Icon: "heart", Status: UIStatus{State: "loading", Message: "Waiting for data"}}, {Title: "Could not load summary", Icon: "circle-alert", Status: UIStatus{State: "error", Message: "Try again when you are connected"}, Action: UIAction{Label: "Review states", URL: "#states-title", Variant: "secondary"}}}
	tmpl, err := template.ParseFiles("../../web/templates/ui_icons.html", "../../web/templates/ui_components.html", "../../web/templates/ui_gallery.html")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = tmpl.ExecuteTemplate(&out, "gallery", data); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestUIGalleryRender(t *testing.T) { galleryHTML(t) }

// Explicit export only: GYST_UI_GALLERY_OUT=/tmp/gyst-ui0 go test ./internal/handlers -run TestExportUIGallery -count=1
func TestExportUIGallery(t *testing.T) {
	dir := os.Getenv("GYST_UI_GALLERY_OUT")
	if dir == "" {
		t.Skip("set GYST_UI_GALLERY_OUT to export the development gallery")
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("gallery output must be an absolute development path")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), galleryHTML(t), 0644); err != nil {
		t.Fatal(err)
	}
}

// stateGallery holds the fictional VN-022 state component examples.
type stateGallery struct {
	States  []UIState
	Unknown UIState
	Banners []UIBanner
	Error   UIErrorBanner
	Empty   UIEmpty
	Stale   UIStale
	Loaded  string
}

func stateGalleryData() stateGallery {
	return stateGallery{
		States:  []UIState{{Kind: "ok"}, {Kind: "info"}, {Kind: "warn"}, {Kind: "crit"}, {Kind: "idle", Word: "Paused"}},
		Unknown: UIState{Kind: "mystery"},
		Banners: []UIBanner{
			{State: UIState{Kind: "info"}, Message: "Example sync in progress", Action: UIAction{Label: "View", URL: "#states-title", Variant: "secondary"}},
			{State: UIState{Kind: "warn"}, Message: "Example source is behind", Detail: "Last sync 6:40"},
			{State: UIState{Kind: "crit"}, Message: "Example service is down", Action: UIAction{Label: "Details", URL: "#states-title", Variant: "secondary"}},
			{State: UIState{Kind: "idle", Word: "Paused"}, Message: "Example reminders are paused"},
		},
		Error:  UIErrorBanner{Message: "Could not load the example summary", LastGood: true, Retry: UIAction{Label: "Retry", URL: "#states-title", Variant: "secondary"}},
		Empty:  UIEmpty{Line: "Nothing to study yet", Action: UIAction{Label: "Add course", URL: "#states-title", Variant: "secondary"}},
		Stale:  UIStale{Source: "Example source · last sync 6:40", Age: "6:40"},
		Loaded: "09:15",
	}
}
