package services

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

// calendarTokenKey is the app_settings key holding the private feed secret.
const calendarTokenKey = "calendar_token"

// CalendarEvent is one exportable dated item, source-agnostic so courses,
// projects, and tasks all flow through the same ICS writer.
type CalendarEvent struct {
	UID         string
	Summary     string
	Description string
	URL         string
	Date        string // YYYY-MM-DD (required)
	StartTime   string // HH:MM (optional -> timed event); empty = all-day
	EndTime     string // HH:MM (optional)
}

// ICS line terminator. RFC 5545 requires CRLF; this repo defaults to LF, so it
// is set explicitly and never relies on the source file's newlines.
const icsCRLF = "\r\n"

// icsEscape escapes the characters RFC 5545 reserves in TEXT values.
func icsEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func icsCompactDate(d string) string { return strings.ReplaceAll(d, "-", "") }

// nextDay returns the day after a YYYY-MM-DD date (all-day DTEND is exclusive).
func nextDay(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

// addHour returns HH:MM one hour later, clamped at 23:59, for a default DTEND.
func addHour(hm string) string {
	t, err := time.Parse("15:04", hm)
	if err != nil {
		return hm
	}
	end := t.Add(time.Hour)
	if end.Day() != t.Day() {
		return "23:59"
	}
	return end.Format("15:04")
}

// BuildICS renders a valid VCALENDAR from events. Timed events use floating
// local wall-clock (no TZID needed): for a single-user app the viewer's
// calendar renders them in the local zone, which is the application timezone
// named in X-WR-TIMEZONE. All-day items use VALUE=DATE. now is the DTSTAMP
// instant.
func BuildICS(events []CalendarEvent, baseURL string, now time.Time, timezone string) string {
	stamp := now.UTC().Format("20060102T150405Z")
	var b strings.Builder
	write := func(line string) { b.WriteString(line); b.WriteString(icsCRLF) }

	write("BEGIN:VCALENDAR")
	write("VERSION:2.0")
	write("PRODID:-//GYST LifeOps//Calendar Export//EN")
	write("CALSCALE:GREGORIAN")
	write("METHOD:PUBLISH")
	write("X-WR-CALNAME:GYST LifeOps")
	write("X-WR-TIMEZONE:" + timezone)

	for _, e := range events {
		if e.Date == "" {
			continue
		}
		write("BEGIN:VEVENT")
		write("UID:" + e.UID)
		write("DTSTAMP:" + stamp)
		if e.StartTime != "" {
			start := icsCompactDate(e.Date) + "T" + strings.ReplaceAll(e.StartTime, ":", "") + "00"
			end := e.EndTime
			if end == "" {
				end = addHour(e.StartTime)
			}
			endStamp := icsCompactDate(e.Date) + "T" + strings.ReplaceAll(end, ":", "") + "00"
			write("DTSTART:" + start)
			write("DTEND:" + endStamp)
		} else {
			write("DTSTART;VALUE=DATE:" + icsCompactDate(e.Date))
			write("DTEND;VALUE=DATE:" + icsCompactDate(nextDay(e.Date)))
		}
		write("SUMMARY:" + icsEscape(e.Summary))
		desc := e.Description
		if baseURL != "" && e.URL != "" {
			link := e.URL
			if strings.HasPrefix(link, "/") {
				link = strings.TrimRight(baseURL, "/") + link
			}
			if desc != "" {
				desc += "\n"
			}
			desc += link
		}
		if desc != "" {
			write("DESCRIPTION:" + icsEscape(desc))
		}
		if e.URL != "" && !strings.HasPrefix(e.URL, "/") {
			write("URL:" + icsEscape(e.URL))
		}
		write("END:VEVENT")
	}

	write("END:VCALENDAR")
	return b.String()
}

// --- event gathering -------------------------------------------------------

func courseBlockEvents(blocks []models.LearningBlock) []CalendarEvent {
	events := make([]CalendarEvent, 0, len(blocks))
	for _, b := range blocks {
		if b.DueDate == "" {
			continue
		}
		summary := b.Title
		if b.CourseName != "" {
			summary = b.CourseName + ": " + b.Title
		}
		label := BlockTypeLabel(b.BlockType)
		desc := label
		if b.IsCatchup {
			desc = "Catch-up · " + label
		}
		if b.Notes != "" {
			desc += "\n" + b.Notes
		}
		url := b.URL
		if url == "" {
			url = "/courses"
		}
		events = append(events, CalendarEvent{
			UID:         "block-" + b.UUID + "@gyst",
			Summary:     summary,
			Description: desc,
			URL:         url,
			Date:        b.DueDate,
			StartTime:   b.StartTime,
			EndTime:     b.EndTime,
		})
	}
	return events
}

// CoursesICS exports course learning blocks. courseID == 0 covers all courses.
func (s *Service) CoursesICS(courseID int64, baseURL string) (string, error) {
	blocks, err := s.Repo.CalendarBlocks(courseID)
	if err != nil {
		return "", err
	}
	return BuildICS(courseBlockEvents(blocks), baseURL, s.Now(), s.Location().String()), nil
}

// projectTaskEvents gathers project target dates and one-time task due dates.
// Reminders are not emitted separately: a reminder is a task and would
// double-emit under the same item, so the task due-date covers it.
func (s *Service) projectTaskEvents() ([]CalendarEvent, error) {
	var events []CalendarEvent

	projects, err := s.Repo.ListProjects(false)
	if err != nil {
		return nil, err
	}
	tasks, err := s.Repo.ListTasks(false)
	if err != nil {
		return nil, err
	}
	items, err := s.Repo.ProjectItemDeadlines()
	if err != nil {
		return nil, err
	}
	milestones, err := s.Repo.MilestoneDeadlines()
	if err != nil {
		return nil, err
	}

	// Privacy is read AFTER the project, item and milestone reads it is applied
	// to, so a project made (or imported) private in between is still hidden.
	// A lookup failure fails the whole feed: no feed beats a leaky feed.
	if s.calendarAfterReads != nil {
		s.calendarAfterReads()
	}
	privacy, err := s.Repo.EffectiveProjectPrivacy()
	if err != nil {
		return nil, err
	}
	hidden := func(projectID int64) bool { return privacy[projectID].Hidden() }

	for _, p := range projects {
		pr := p.Project
		if pr.TargetDate == "" || pr.Status == "archived" || hidden(pr.ID) {
			continue
		}
		events = append(events, CalendarEvent{
			UID:         "project-" + pr.UUID + "@gyst",
			Summary:     "Target: " + pr.Name,
			Description: "Project target date",
			URL:         "/projects/" + itoa(pr.ID),
			Date:        pr.TargetDate,
		})
	}

	for _, t := range tasks {
		if t.DueDate == "" || t.RepeatType != "none" || t.Status != "pending" {
			continue
		}
		events = append(events, CalendarEvent{
			UID:         "task-" + t.UUID + "@gyst",
			Summary:     t.Title,
			Description: "Task due",
			URL:         "/tasks/" + itoa(t.ID),
			Date:        t.DueDate,
		})
	}

	// Project mission items with a due date.
	for _, it := range items {
		if hidden(it.ProjectID) {
			continue
		}
		events = append(events, CalendarEvent{
			UID:         "item-" + it.UUID + "@gyst",
			Summary:     it.Title,
			Description: "Mission · " + it.ProjectName,
			URL:         "/projects/" + itoa(it.ProjectID),
			Date:        it.DueDate,
		})
	}

	// Roadmap milestones with a due date.
	for _, m := range milestones {
		if hidden(m.ProjectID) {
			continue
		}
		events = append(events, CalendarEvent{
			UID:         "milestone-" + m.UUID + "@gyst",
			Summary:     m.Title,
			Description: "Milestone · " + m.ProjectName,
			URL:         "/projects/" + itoa(m.ProjectID),
			Date:        m.DueDate,
		})
	}

	// Standalone reminders (task-derived reminders are skipped — they duplicate
	// the source task already emitted above).
	reminders, err := s.Repo.ReminderDeadlines()
	if err != nil {
		return nil, err
	}
	for _, rm := range reminders {
		events = append(events, CalendarEvent{
			UID:         "reminder-" + rm.UUID + "@gyst",
			Summary:     rm.Title,
			Description: "Reminder",
			URL:         "/reminders",
			Date:        rm.DueDate,
			StartTime:   rm.Time,
		})
	}
	return events, nil
}

// --- private feed token ----------------------------------------------------

// CalendarToken returns the configured feed token, or "" when none is set.
func (s *Service) CalendarToken() (string, error) {
	return s.Repo.GetSetting(calendarTokenKey)
}

// GenerateCalendarToken creates (or rotates) a long random feed token and
// persists it. Returns the new token.
func (s *Service) GenerateCalendarToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := s.Repo.SetSetting(calendarTokenKey, token); err != nil {
		return "", err
	}
	return token, nil
}

// RevokeCalendarToken removes the feed token, disabling all tokenized feed URLs.
func (s *Service) RevokeCalendarToken() error {
	return s.Repo.DeleteSetting(calendarTokenKey)
}

// ProjectsICS exports project target dates and task deadlines.
func (s *Service) ProjectsICS(baseURL string) (string, error) {
	events, err := s.projectTaskEvents()
	if err != nil {
		return "", err
	}
	return BuildICS(events, baseURL, s.Now(), s.Location().String()), nil
}

// AllICS is the unified feed: courses + projects + tasks.
func (s *Service) AllICS(baseURL string) (string, error) {
	blocks, err := s.Repo.CalendarBlocks(0)
	if err != nil {
		return "", err
	}
	events := courseBlockEvents(blocks)
	pt, err := s.projectTaskEvents()
	if err != nil {
		return "", err
	}
	events = append(events, pt...)
	return BuildICS(events, baseURL, s.Now(), s.Location().String()), nil
}
