package services

import (
	"sort"
	"time"

	"lifeops-tracker/internal/models"
)

// nowTimedHorizon is the one place that defines how far ahead the "next timed
// item" looks: to the end of the current local day, the same horizon as the
// Today page's Next up. Jay chose this over a rolling 12 hours (2026-10-11).
// An item dated tomorrow never qualifies, even a few minutes after midnight.
const nowTimedHorizon = "end of the current local day"

// NowInput is everything SelectNow needs. The caller gathers it; SelectNow
// reads no database and no clock.
type NowInput struct {
	Now      time.Time               // already in the application timezone
	Pending  []models.TaskOccurrence // today's pending occurrences only
	Learning models.TodayLearning
	// Reminders is today's due-today slice from ReminderViews. It already holds
	// pending task reminders and standalone reminders, so task times are read
	// from here and never counted twice from Pending.
	Reminders []models.ReminderView
	Prayer    models.NextPrayer
}

// NowAction is the single thing to do now.
type NowAction struct {
	Kind   string // "task", "session" or "continue"
	Task   *models.TaskOccurrence
	Block  *models.LearningBlock
	Course *models.Course
}

// NowTimed is the next timed item later today.
type NowTimed struct {
	Kind       string // "reminder" or "session"
	Title      string
	At         time.Time
	SourceType string
	SourceID   int64
}

// NowSet is the result of SelectNow.
type NowSet struct {
	Action *NowAction
	Prayer models.NextPrayer
	Timed  *NowTimed
}

const nowTimeLayout = "2006-01-02 15:04"

// SelectNow chooses the Now action, the next prayer and the next timed item.
// It is pure and deterministic.
//
// Next action (first match wins):
//
//  1. Pending task occurrences for today. Completed, skipped and overdue
//     one-time tasks are never candidates (overdue belongs to Attention).
//  2. The first pending study block, in the order given (due date, timed
//     first, start time, id).
//  3. A "continue" action for Learning.SuggestCourse when SuggestStudy is set.
//  4. No action (nil). There is no "next routine" fallback.
//
// Task order (each key only breaks ties of the one before):
//
//	key 1  priority      high, then medium, then anything else (priorityRank)
//	key 2  ReminderTime  earlier first; a task with a time before one without
//	key 3  task ID       lower first (tasks have no manual order column)
//
// Prayer is passed through unchanged: strictly after Now, and after Isha it is
// tomorrow's Fajr with Tomorrow = true.
//
// Next timed item: the earliest of today's reminders (task or standalone) and
// timed study blocks due today, strictly after Now and within nowTimedHorizon.
// A reminder at exactly Now is skipped. The item chosen as the action is
// excluded. Ties:
//
//	key 1  time          earlier first
//	key 2  kind          reminder before session
//	key 3  title         byte order, A to Z
//	key 4  source ID     lower first
//
// Blank or unparsable times are skipped.
func SelectNow(in NowInput) NowSet {
	out := NowSet{Prayer: in.Prayer}
	var chosenTaskID, chosenBlockID int64

	if task := firstNowTask(in.Pending); task != nil {
		out.Action = &NowAction{Kind: "task", Task: task}
		chosenTaskID = task.Task.ID
	} else if block := firstNowBlock(in.Learning.Blocks); block != nil {
		out.Action = &NowAction{Kind: "session", Block: block}
		chosenBlockID = block.ID
	} else if in.Learning.SuggestStudy {
		course := in.Learning.SuggestCourse
		out.Action = &NowAction{Kind: "continue", Course: &course}
	}

	out.Timed = nextNowTimed(in, chosenTaskID, chosenBlockID)
	return out
}

func firstNowTask(pending []models.TaskOccurrence) *models.TaskOccurrence {
	candidates := make([]models.TaskOccurrence, 0, len(pending))
	for _, item := range pending {
		if item.Status == "completed" || item.Status == "skipped" {
			continue
		}
		candidates = append(candidates, item)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if ra, rb := priorityRank(a.Task.Priority), priorityRank(b.Task.Priority); ra != rb {
			return ra < rb
		}
		ta, tb := a.Task.ReminderTime, b.Task.ReminderTime
		if (ta == "") != (tb == "") {
			return ta != ""
		}
		if ta != tb {
			return ta < tb
		}
		return a.Task.ID < b.Task.ID
	})
	first := candidates[0]
	return &first
}

func firstNowBlock(blocks []models.LearningBlock) *models.LearningBlock {
	for _, block := range blocks {
		if block.Status != "" && block.Status != "pending" {
			continue
		}
		first := block
		return &first
	}
	return nil
}

func nextNowTimed(in NowInput, chosenTaskID, chosenBlockID int64) *NowTimed {
	today := in.Now.Format("2006-01-02")
	var items []NowTimed
	add := func(kind, title, date, clock, sourceType string, id int64) {
		// Horizon: nowTimedHorizon, enforced by requiring the current local date.
		if date != today || clock == "" {
			return
		}
		at, err := time.ParseInLocation(nowTimeLayout, date+" "+clock, in.Now.Location())
		if err != nil || !at.After(in.Now) {
			return
		}
		items = append(items, NowTimed{Kind: kind, Title: title, At: at, SourceType: sourceType, SourceID: id})
	}
	for _, reminder := range in.Reminders {
		if chosenTaskID != 0 && reminder.SourceType == "task" && reminder.SourceID == chosenTaskID {
			continue
		}
		add("reminder", reminder.Title, reminder.ReminderDate, reminder.ReminderTime, reminder.SourceType, reminder.SourceID)
	}
	for _, block := range in.Learning.Blocks {
		if chosenBlockID != 0 && block.ID == chosenBlockID {
			continue
		}
		if block.Status != "" && block.Status != "pending" {
			continue
		}
		add("session", block.Title, block.DueDate, block.StartTime, "learning_block", block.ID)
	}
	if len(items) == 0 {
		return nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if !a.At.Equal(b.At) {
			return a.At.Before(b.At)
		}
		if a.Kind != b.Kind {
			return a.Kind == "reminder"
		}
		if a.Title != b.Title {
			return a.Title < b.Title
		}
		return a.SourceID < b.SourceID
	})
	first := items[0]
	return &first
}

// TodayNow gathers the inputs from existing queries and selects the Now set.
// It is not named Now because Service embeds clock.Clock, whose Now is a field.
func (s *Service) TodayNow() (NowSet, error) {
	now := s.Now().In(s.Location())
	items, err := s.TaskOccurrencesOn(s.Today())
	if err != nil {
		return NowSet{}, err
	}
	pending, _, _ := SplitOccurrences(items)
	learning, err := s.TodayLearning()
	if err != nil {
		return NowSet{}, err
	}
	reminders, _, err := s.ReminderViews(s.Today())
	if err != nil {
		return NowSet{}, err
	}
	prayer, err := s.NextPrayer(now)
	if err != nil {
		return NowSet{}, err
	}
	return SelectNow(NowInput{Now: now, Pending: pending, Learning: learning, Reminders: reminders, Prayer: prayer}), nil
}
