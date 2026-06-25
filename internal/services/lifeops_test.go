package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func trackerItemsFixture() []models.TrackerItem {
	return []models.TrackerItem{
		{ID: 1, Type: "prayer"},
		{ID: 2, Type: "prayer"},
		{ID: 3, Type: "prayer"},
		{ID: 4, Type: "prayer"},
		{ID: 5, Type: "prayer"},
		{ID: 6, Type: "athkar"},
		{ID: 7, Type: "athkar"},
		{ID: 8, Type: "gym"},
	}
}

func doneLog(itemID int64, date string) models.TrackerLog {
	return models.TrackerLog{TrackerItemID: itemID, LogDate: date, Status: "done"}
}

// 4 of 5 prayers done, 1 of 2 athkar done.
func TestBuildDailyMissionCompletion(t *testing.T) {
	today := time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)
	date := formatDate(today)
	logs := []models.TrackerLog{
		doneLog(1, date),
		doneLog(2, date),
		doneLog(3, date),
		doneLog(4, date),
		doneLog(6, date),
	}

	m := buildDailyMission(today, trackerItemsFixture(), logs)

	if m.PrayerCompletion != 80 {
		t.Errorf("PrayerCompletion = %d, want 80", m.PrayerCompletion)
	}
	if m.AthkarCompletion != 50 {
		t.Errorf("AthkarCompletion = %d, want 50", m.AthkarCompletion)
	}
	// Daily Mission / Deen score = 5 of 7 deen items done = floor(500/7) = 71.
	if m.DeenCompletion != 71 {
		t.Errorf("DeenCompletion = %d, want 71", m.DeenCompletion)
	}
}

func TestBuildDailyMissionEmpty(t *testing.T) {
	today := time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)
	m := buildDailyMission(today, trackerItemsFixture(), nil)

	if m.PrayerCompletion != 0 || m.AthkarCompletion != 0 || m.DeenCompletion != 0 {
		t.Errorf("expected all-zero completion with no logs, got prayer=%d athkar=%d deen=%d",
			m.PrayerCompletion, m.AthkarCompletion, m.DeenCompletion)
	}
	if len(m.PrayerItems) != 5 || len(m.AthkarItems) != 2 {
		t.Errorf("expected 5 prayer / 2 athkar views, got %d / %d", len(m.PrayerItems), len(m.AthkarItems))
	}
}

func TestBuildDailyMissionGymWeek(t *testing.T) {
	today := time.Date(2026, time.June, 17, 12, 0, 0, 0, time.UTC)
	date := formatDate(today)
	// One gym day done this week => floor(1 * 100 / 7) = 14.
	logs := []models.TrackerLog{doneLog(8, date)}

	m := buildDailyMission(today, trackerItemsFixture(), logs)

	if len(m.GymWeek) != 7 {
		t.Fatalf("GymWeek length = %d, want 7", len(m.GymWeek))
	}
	if m.GymWeekCompletion != 14 {
		t.Errorf("GymWeekCompletion = %d, want 14", m.GymWeekCompletion)
	}
}
