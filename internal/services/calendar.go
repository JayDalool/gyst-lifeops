package services

import (
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

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
// local wall-clock (no TZID / no LoadLocation needed): for a single-user app the
// viewer's calendar renders them in the local zone, which is the intended
// America/Winnipeg. All-day items use VALUE=DATE.
func BuildICS(events []CalendarEvent, baseURL string) string {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	var b strings.Builder
	write := func(line string) { b.WriteString(line); b.WriteString(icsCRLF) }

	write("BEGIN:VCALENDAR")
	write("VERSION:2.0")
	write("PRODID:-//GYST LifeOps//Calendar Export//EN")
	write("CALSCALE:GREGORIAN")
	write("METHOD:PUBLISH")
	write("X-WR-CALNAME:GYST LifeOps")
	write("X-WR-TIMEZONE:America/Winnipeg")

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
	return BuildICS(courseBlockEvents(blocks), baseURL), nil
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
	for _, p := range projects {
		pr := p.Project
		if pr.TargetDate == "" || pr.Status == "archived" {
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

	tasks, err := s.Repo.ListTasks(false)
	if err != nil {
		return nil, err
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
	return events, nil
}

// ProjectsICS exports project target dates and task deadlines.
func (s *Service) ProjectsICS(baseURL string) (string, error) {
	events, err := s.projectTaskEvents()
	if err != nil {
		return "", err
	}
	return BuildICS(events, baseURL), nil
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
	return BuildICS(events, baseURL), nil
}
