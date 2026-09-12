package handlers

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
