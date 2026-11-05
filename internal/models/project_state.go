package models

import "strings"

// ProjectState is the one meaning of a stored project status. Every surface
// that reads a status (Galaxy, the rocket route, the roadmap and project
// pages, the Projects filters and the project tree) goes through it so they
// cannot disagree (GYST-VN-037, 17A §N.5). Stored values are never changed.
type ProjectState string

const (
	ProjectStateActive   ProjectState = "active"
	ProjectStatePlanned  ProjectState = "planned" // stored "upcoming"
	ProjectStateBlocked  ProjectState = "blocked" // stored "locked"
	ProjectStatePaused   ProjectState = "paused"
	ProjectStateDone     ProjectState = "done" // stored "done" or "completed"
	ProjectStateArchived ProjectState = "archived"
)

// ProjectStateOf maps a stored project status to its state. Input is trimmed
// and lower-cased; any unknown value is active, the same default as the
// repository's status cleaner.
func ProjectStateOf(status string) ProjectState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "upcoming":
		return ProjectStatePlanned
	case "locked":
		return ProjectStateBlocked
	case "paused":
		return ProjectStatePaused
	case "done", "completed":
		return ProjectStateDone
	case "archived":
		return ProjectStateArchived
	default:
		return ProjectStateActive
	}
}

// Finished reports whether the project has left the working set: stored done,
// completed or archived. Galaxy excludes exactly these. Progress is never used.
func (s ProjectState) Finished() bool {
	return s == ProjectStateDone || s == ProjectStateArchived
}

// RouteState maps a stored status and progress to the node state used by the
// rocket route, Galaxy data and the status dots: active, locked or done.
func RouteState(status string, progress int) string {
	state := ProjectStateOf(status)
	switch {
	case state.Finished():
		return "done"
	case state == ProjectStatePlanned, state == ProjectStateBlocked, state == ProjectStatePaused:
		return "locked"
	case progress >= 100:
		return "done"
	}
	return "active"
}
