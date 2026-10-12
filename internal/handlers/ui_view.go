package handlers

import "strings"

// UI-0 inputs are presentation-only. Callers supply formatted values and state;
// components never query services or compute nutrition/energy targets.
type UIStatus struct {
	State   string // normal, empty, partial, stale, loading, error
	Message string
}

func (s UIStatus) Kind() string {
	switch s.State {
	case "empty", "partial", "stale", "loading", "error":
		return s.State
	}
	return "normal"
}
func (s UIStatus) Label() string {
	switch s.Kind() {
	case "empty":
		return "No data"
	case "partial":
		return "Partial data"
	case "stale":
		return "Stale data"
	case "loading":
		return "Loading"
	case "error":
		return "Unable to load"
	}
	return "Current"
}

type UIAction struct {
	Label, URL, Icon, Variant string
	Disabled                  bool
}
type UIHeader struct {
	Title, Subtitle string
	Action          UIAction
}
type UIRow struct{ Label, Detail, Meta, URL, Icon string }
type UISegments struct {
	Label string
	Items []UISegment
}
type UISegment struct {
	Label, URL string
	Selected   bool
}
type UIOption struct {
	Value, Label string
	Selected     bool
}
type UIField struct {
	ID, Name, Label, Value, Placeholder, Help, Error, Type string
	Required, Disabled                                     bool
	Options                                                []UIOption
}

func (f UIField) InputType() string {
	switch f.Type {
	case "search", "email", "url", "date", "time", "number", "password":
		return f.Type
	}
	return "text"
}

type UIChoice struct {
	ID, Name, Label, Value, Kind string
	Checked, Disabled            bool
}

func (c UIChoice) InputType() string {
	if c.Kind == "radio" {
		return "radio"
	}
	return "checkbox"
}

type UIProgress struct {
	Label, Detail string
	Present       bool
	Percent       int
}

func (p UIProgress) Fill() int {
	if p.Percent < 0 {
		return 0
	}
	if p.Percent > 100 {
		return 100
	}
	return p.Percent
}

type UIMetric struct {
	Label, Value, Unit, Detail string
	Present                    bool
	Status                     UIStatus
	Progress                   UIProgress
}

// Only normal/partial/stale values may render. Loading and failures do not
// accidentally expose fixture values or an obsolete zero from a caller.
func (m UIMetric) ShowValue() bool {
	k := m.Status.Kind()
	return m.Present && (k == "normal" || k == "partial" || k == "stale")
}
func (m UIMetric) ShowProgress() bool { return m.ShowValue() && m.Progress.Present }

type UIStateCard struct {
	Title, Icon string
	Status      UIStatus
	Action      UIAction
}

// UIState is the shared status vocabulary of 11 §I: five families, each with a
// word. Anything unrecognised is idle "Unknown", never ok (13 §17).
type UIState struct{ Kind, Word string }

var uiStateFamilies = map[string]struct{ word, icon string }{
	"ok":   {"Healthy", "check"},
	"info": {"Working", "info"},
	"warn": {"Stale", "triangle-alert"},
	"crit": {"Failed", "circle-alert"},
	"idle": {"Unknown", "circle-dashed"},
}

// uiOKWords are the 11 §I words that only the ok family may show.
var uiOKWords = map[string]bool{"healthy": true, "done": true, "up to date": true, "current": true, "valid": true}

// Family returns ok, info, warn, crit or idle; anything else is idle.
func (s UIState) Family() string {
	k := strings.ToLower(strings.TrimSpace(s.Kind))
	if _, ok := uiStateFamilies[k]; ok {
		return k
	}
	return "idle"
}

// Label is the caller's word, or the family default when it is empty or, for a
// non-ok family, one of the ok words.
func (s UIState) Label() string {
	f := s.Family()
	w := strings.TrimSpace(s.Word)
	if w == "" || (f != "ok" && uiOKWords[strings.ToLower(w)]) {
		return uiStateFamilies[f].word
	}
	return w
}

// Icon is the Lucide icon for tiles and banners.
func (s UIState) Icon() string { return uiStateFamilies[s.Family()].icon }

// UIBanner is an inline strip: state icon, state word, one message, optional action.
type UIBanner struct {
	State           UIState
	Message, Detail string
	Action          UIAction
}

// UIErrorBanner is the crit banner for a failed source (13 §17). LastGood says
// the previous data is still shown.
type UIErrorBanner struct {
	Message  string
	LastGood bool
	Retry    UIAction
}

// UIEmpty is one line plus at most one action.
type UIEmpty struct {
	Line   string
	Action UIAction
}

// UIStale is the amber chip with the age the caller passes in ("6:40").
type UIStale struct{ Source, Age string }

// UIOverlayHead is the title and close button at the top of a sheet or popover (VN-023).
type UIOverlayHead struct{ ID, Title string }

// UIConfirm is an in-page confirmation dialog (VN-023, 13 §8). Destructive styles the
// confirm button as dangerous; Cancel always takes the initial focus.
type UIConfirm struct {
	ID, Title, Consequence, ConfirmLabel string
	Destructive                          bool
}
