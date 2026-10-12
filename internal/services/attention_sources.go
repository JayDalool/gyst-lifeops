package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// HealthSyncStaleAfter is when the day's Health data stops being described as
// current. A three-times-daily Shortcut means a gap this long is a missed
// run, not a quiet afternoon. The Health screen and Attention share it.
const HealthSyncStaleAfter = 8 * time.Hour

const (
	backupWarnAfter     = 36 * time.Hour
	backupCriticalAfter = 72 * time.Hour
	backupStatusMaxSize = 64 << 10
	appleHealthSource   = "apple_shortcuts"
)

// gystAttentionProducers returns the producers backed by GYST's own data.
func (s *Service) gystAttentionProducers() []AttentionProducer {
	return []AttentionProducer{
		taskAttention{s}, reminderAttention{s}, learningAttention{s},
		healthSyncAttention{s}, backupAttention{s},
	}
}

// dayAfter is local midnight at the end of date, as UTC: the first instant a
// date-based item is overdue.
func (s *Service) dayAfter(date string) (time.Time, bool) {
	day, err := time.ParseInLocation("2006-01-02", date, s.Location())
	if err != nil {
		return time.Time{}, false
	}
	next := day.AddDate(0, 0, 1)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, s.Location()).UTC(), true
}

// dayStart is local midnight at the start of date, as UTC.
func (s *Service) dayStart(date string) (time.Time, bool) {
	day, err := time.ParseInLocation("2006-01-02", date, s.Location())
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, s.Location()).UTC(), true
}

type taskAttention struct{ s *Service }

func (taskAttention) Source() string { return "gyst.tasks" }
func (taskAttention) Label() string  { return "GYST tasks" }

func (p taskAttention) Produce(_ context.Context, _ time.Time) ([]AttentionItem, error) {
	tasks, err := p.s.Repo.ListActiveOneTimeOverdueTasks(p.s.TodayString())
	if err != nil {
		return nil, err
	}
	var items []AttentionItem
	for _, task := range tasks {
		first, ok := p.s.dayAfter(task.DueDate)
		if !ok {
			continue
		}
		key := strconv.FormatInt(task.ID, 10)
		items = append(items, AttentionItem{
			Source: p.Source(), SourceKey: key,
			Kind: AttentionProblem, Severity: AttentionWarning,
			Title: task.Title, Context: "Overdue since " + task.DueDate,
			Target:      "/tasks/" + key,
			FirstSeen:   first,
			Fingerprint: AttentionFingerprint(p.Source(), key, AttentionWarning, task.DueDate),
		})
	}
	return items, nil
}

type reminderAttention struct{ s *Service }

func (reminderAttention) Source() string { return "gyst.reminders" }
func (reminderAttention) Label() string  { return "GYST reminders" }

func (p reminderAttention) Produce(_ context.Context, _ time.Time) ([]AttentionItem, error) {
	today := p.s.TodayString()
	reminders, err := p.s.Repo.ListStandaloneRemindersThrough(today)
	if err != nil {
		return nil, err
	}
	var items []AttentionItem
	for _, r := range reminders {
		if r.ReminderDate == "" || r.ReminderDate >= today {
			continue
		}
		first, ok := p.s.dayAfter(r.ReminderDate)
		if !ok {
			continue
		}
		key := strconv.FormatInt(r.ID, 10)
		items = append(items, AttentionItem{
			Source: p.Source(), SourceKey: key,
			Kind: AttentionProblem, Severity: AttentionWarning,
			Title: r.Title, Context: "Overdue since " + r.ReminderDate,
			Target:      "/reminders",
			FirstSeen:   first,
			Fingerprint: AttentionFingerprint(p.Source(), key, AttentionWarning, r.ReminderDate+" "+r.ReminderTime),
		})
	}
	return items, nil
}

type learningAttention struct{ s *Service }

func (learningAttention) Source() string { return "gyst.learning" }
func (learningAttention) Label() string  { return "GYST learning" }

func (p learningAttention) Produce(_ context.Context, _ time.Time) ([]AttentionItem, error) {
	today := p.s.TodayString()
	blocks, err := p.s.Repo.LearningBlocksThrough(today)
	if err != nil {
		return nil, err
	}
	var items []AttentionItem
	for _, b := range blocks {
		severity, reason := AttentionWarning, "Overdue since "+b.DueDate
		first, ok := p.s.dayAfter(b.DueDate)
		if b.DueDate == today {
			severity, reason = AttentionInfo, "Due today"
			first, ok = p.s.dayStart(b.DueDate)
		}
		if !ok {
			continue
		}
		anchor := "block-"
		if b.IsCatchup {
			anchor = "catchup-"
		}
		key := strconv.FormatInt(b.ID, 10)
		items = append(items, AttentionItem{
			Source: p.Source(), SourceKey: key,
			Kind: AttentionProblem, Severity: severity,
			Title: b.Title, Context: reason,
			Target:      "/courses/" + strconv.FormatInt(b.CourseID, 10) + "#" + anchor + key,
			FirstSeen:   first,
			Fingerprint: AttentionFingerprint(p.Source(), key, severity, b.DueDate),
		})
	}
	return items, nil
}

type healthSyncAttention struct{ s *Service }

func (healthSyncAttention) Source() string { return "gyst.health_sync" }
func (healthSyncAttention) Label() string  { return "Apple Health sync" }

// Produce uses only the sync time, never a health value.
func (p healthSyncAttention) Produce(_ context.Context, now time.Time) ([]AttentionItem, error) {
	state, err := p.s.Repo.GetHealthSyncState(appleHealthSource)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	synced, err := time.Parse(time.RFC3339Nano, state.LastSyncedAt)
	if err != nil {
		return nil, fmt.Errorf("health sync time: %w", err)
	}
	if now.Sub(synced) < HealthSyncStaleAfter {
		return nil, nil
	}
	return []AttentionItem{{
		Source: p.Source(), SourceKey: appleHealthSource,
		Kind: AttentionProblem, Severity: AttentionWarning,
		Title: "Apple Health not synced", Context: "Last sync " + synced.UTC().Format(time.RFC3339),
		Target:      "/health",
		FirstSeen:   synced.Add(HealthSyncStaleAfter).UTC(),
		Fingerprint: AttentionFingerprint(p.Source(), appleHealthSource, AttentionWarning, state.LastSyncedAt),
	}}, nil
}

type backupAttention struct{ s *Service }

func (backupAttention) Source() string { return "gyst.backup" }
func (backupAttention) Label() string  { return "GYST backup" }

type backupStatus struct {
	FinishedAt string `json:"finished_at"`
	Result     string `json:"result"`
	Verified   bool   `json:"verified"`
}

// Produce reads the status file the scheduled backup writes. A missing,
// unreadable or invalid file is an error (reported as unavailable), never
// "healthy". The file's other fields, such as its location, are not read.
func (p backupAttention) Produce(_ context.Context, now time.Time) ([]AttentionItem, error) {
	path := p.s.BackupStatusPath
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, backupStatusMaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > backupStatusMaxSize {
		return nil, errors.New("backup status file too large")
	}
	var st backupStatus
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	finished, err := time.Parse(time.RFC3339, st.FinishedAt)
	if err != nil {
		return nil, fmt.Errorf("backup finished_at: %w", err)
	}
	age := now.Sub(finished)
	if age < 0 {
		return nil, errors.New("backup finished_at is in the future")
	}

	var severity AttentionSeverity
	var reason string
	first := finished.UTC()
	switch {
	case age >= backupCriticalAfter:
		severity, reason = AttentionCritical, "Last backup finished more than 72 hours ago"
		first = finished.Add(backupWarnAfter).UTC()
	case age >= backupWarnAfter:
		severity, reason = AttentionWarning, "Last backup finished more than 36 hours ago"
		first = finished.Add(backupWarnAfter).UTC()
	case st.Result != "ok" || !st.Verified:
		severity, reason = AttentionWarning, "Last backup failed or was not verified"
	default:
		return nil, nil
	}
	state := fmt.Sprintf("%s %s %t", st.FinishedAt, st.Result, st.Verified)
	return []AttentionItem{{
		Source: p.Source(), SourceKey: "backup",
		Kind: AttentionProblem, Severity: severity,
		Title: "GYST backup", Context: reason,
		FirstSeen:   first,
		Fingerprint: AttentionFingerprint(p.Source(), "backup", severity, state),
	}}, nil
}
