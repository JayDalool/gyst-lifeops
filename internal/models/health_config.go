package models

// HealthChecklistItem is one user-configured daily item (Issue #59).
// Kind "habit" is ticked by hand; "protein" is measured from the day's data.
type HealthChecklistItem struct {
	ID       int64
	Name     string
	Kind     string
	Position int64
	Active   bool
	// DoneToday is set for habits ticked on the requested local day.
	DoneToday bool
}

// HealthFood is a food or product Jay usually has. Values are per serving;
// nil means unknown, never zero.
type HealthFood struct {
	ID       int64
	Name     string
	Serving  string
	Kcal     *float64
	ProteinG *float64
	CarbsG   *float64
	FatG     *float64
	FiberG   *float64
	Notes    string
	Active   bool
}
