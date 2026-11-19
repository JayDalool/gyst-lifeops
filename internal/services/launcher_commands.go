package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Launcher Do mode (GYST-103). One fixed, code-defined registry: every command
// is a navigation link, an existing CSRF-protected POST (shown behind a confirm
// step by the client), or the browser-only appearance choice (rendered in the
// layout, never listed here). Nothing is added by the client and there are no
// destructive, infrastructure or agent commands. Labels never carry health
// values (no totals, weights or targets).

const (
	LauncherKindLink = "link"
	LauncherKindPost = "post"
)

// LauncherCommand is one Do-mode entry. Action and Fields are set only for
// Kind "post"; Fields holds the hidden form fields (occurrence_date only).
type LauncherCommand struct {
	ID             string            `json:"id"`
	Label          string            `json:"label"`
	Meta           string            `json:"meta"`
	Kind           string            `json:"kind"`
	Href           string            `json:"href,omitempty"`
	Action         string            `json:"action,omitempty"`
	Fields         map[string]string `json:"fields,omitempty"`
	Confirm        string            `json:"confirm,omitempty"`
	DisabledReason string            `json:"disabledReason,omitempty"`
	Terms          string            `json:"-"`
}

// LauncherCommands builds the fixed list. hevyConfigured gates "Sync Hevy now"
// (absent, not disabled, when Hevy is not set up on this host).
func (s *Service) LauncherCommands(hevyConfigured bool) ([]LauncherCommand, error) {
	cmds := []LauncherCommand{
		{ID: "new-task", Label: "New task", Meta: "Open", Kind: LauncherKindLink, Href: "/tasks/new", Terms: "add create mission"},
		{ID: "new-project", Label: "New project", Meta: "Open", Kind: LauncherKindLink, Href: "/projects/new", Terms: "add create"},
		{ID: "import-plan", Label: "Import a plan", Meta: "Open", Kind: LauncherKindLink, Href: "/plan-import", Terms: "ai chatgpt markdown tasks checklist"},
		{ID: "log-weight", Label: "Log weight", Meta: "Open", Kind: LauncherKindLink, Href: "/health/progress", Terms: "health progress body"},
		{ID: "prayer-times", Label: "Prayer times", Meta: "Open", Kind: LauncherKindLink, Href: "/settings#prayer", Terms: "settings salah"},
		{ID: "reminders", Label: "Reminders", Meta: "Open", Kind: LauncherKindLink, Href: "/reminders", Terms: "deadlines"},
	}

	learn := LauncherCommand{ID: "continue-learning", Label: "Continue learning", Meta: "Open", Kind: LauncherKindLink, Href: "/courses", Terms: "study course block"}
	tl, err := s.TodayLearning()
	if err != nil {
		return nil, err
	}
	if len(tl.Blocks) > 0 {
		b := tl.Blocks[0]
		learn.Href = fmt.Sprintf("/courses/%d#block-%d", b.CourseID, b.ID)
	}
	cmds = append(cmds, learn)

	// Water follows the Today/Health Tappable rule: a plan with a bottle size
	// and target, and the target not yet reached.
	water := LauncherCommand{ID: "water", Label: "Add a bottle of water", Meta: "Health", Kind: LauncherKindPost, Action: "/health/water", Terms: "drink hydration log",
		Confirm: "Log one bottle of water for today?"}
	today := s.TodayString()
	cycle, err := s.GetHealthCycleForDate(today)
	switch {
	case errors.Is(err, sql.ErrNoRows) || (err == nil && (cycle.WaterBottleML <= 0 || cycle.WaterTargetML <= 0)):
		water.DisabledReason = "Set a bottle size in Health"
	case err != nil:
		return nil, err
	default:
		total, _, werr := s.Repo.HealthWaterDay(today)
		if werr != nil {
			return nil, werr
		}
		if total >= cycle.WaterTargetML {
			water.DisabledReason = "Today's water target is reached"
		}
	}
	cmds = append(cmds, water)

	// Complete uses the same first pending occurrence as Today's Focus card.
	complete := LauncherCommand{ID: "complete-focus", Label: "Complete: focus task", Meta: "Task", Kind: LauncherKindPost, Terms: "done finish mission now focus"}
	items, err := s.TaskOccurrencesOn(s.Today())
	if err != nil {
		return nil, err
	}
	pending, _, _ := SplitOccurrences(items)
	if len(pending) == 0 {
		complete.DisabledReason = "No pending tasks today"
	} else {
		occ := pending[0]
		title := strings.TrimSpace(occ.Task.Title)
		if r := []rune(title); len(r) > 60 {
			title = strings.TrimSpace(string(r[:60])) + "…"
		}
		complete.Label = "Complete: " + title
		complete.Action = fmt.Sprintf("/tasks/%d/complete", occ.Task.ID)
		complete.Fields = map[string]string{"occurrence_date": occ.Date}
		complete.Confirm = "Mark “" + title + "” done for today?"
	}
	cmds = append(cmds, complete)

	if hevyConfigured {
		cmds = append(cmds, LauncherCommand{ID: "sync-hevy", Label: "Sync Hevy now", Meta: "Training", Kind: LauncherKindPost, Action: "/health/training/sync", Terms: "workout training refresh",
			Confirm: "Fetch your latest workouts from Hevy now?"})
	}
	return cmds, nil
}

// FilterLauncherCommands keeps the commands whose label or hidden terms contain
// every word of q (case-insensitive). An empty q keeps all of them.
func FilterLauncherCommands(cmds []LauncherCommand, q string) []LauncherCommand {
	words := strings.Fields(strings.ToLower(NormalizeSearchQuery(q)))
	out := make([]LauncherCommand, 0, len(cmds))
	for _, c := range cmds {
		hay := strings.ToLower(c.Label + " " + c.Terms)
		ok := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, c)
		}
	}
	return out
}
