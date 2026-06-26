package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// renderOK exercises the real render() path (layout + page + funcmap) and
// fails on any non-200 or embedded template execution error. go test does not
// otherwise cover template rendering, so this guards the routes against future
// field-reference or pipeline breakage.
func renderOK(t *testing.T, page string, data any) {
	t.Helper()
	h := &Handler{TemplateDir: "../../web/templates"}
	rec := httptest.NewRecorder()
	h.render(rec, page, data)
	if rec.Code != 200 {
		t.Fatalf("%s rendered status %d: %s", page, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "executing \"") {
		t.Fatalf("%s contains template error: %s", page, rec.Body.String())
	}
}

func TestRenderDashboard(t *testing.T) {
	data := models.DashboardData{
		Page:            "dashboard",
		Today:           "2026-06-25",
		TodayCompletion: 40,
		ShipStats:       models.ShipStats{TotalFuel: 240, Level: 3, LevelProgress: 40, CompletedMissions: 18, ActiveTrips: 2, Streak: 5, MedalCount: 4, RecentMedal: "First Launch"},
		ShipRoute: services.BuildRocketRoute("Ship Bridge", []models.RouteNode{
			{Name: "Real Estate AI", URL: "/projects/1", State: "active"},
			{Name: "Healthcare AI", URL: "/projects/2", State: "locked"},
			{Name: "Portfolio", URL: "/projects/3", State: "done"},
		}),
		Player: models.PlayerStats{TotalStars: 240, CurrentLevel: 3, ActiveStreak: 5, ProofCount: 2, TaskDoneCount: 18},
		TodayItems: []models.TaskOccurrence{
			{Task: models.Task{ID: 1, Title: "Build lead scoring API", StarsReward: 3, ProofRequired: true, ProofType: "url", Priority: "high"}, Date: "2026-06-25", Status: "pending"},
			{Task: models.Task{ID: 2, Title: "Read paper", StarsReward: 1, Priority: "low"}, Date: "2026-06-25", Status: "completed"},
		},
		ActiveProjects: []models.ProjectWithProgress{
			{Project: models.Project{ID: 1, Name: "Real Estate AI", Status: "active", ProjectKind: "main", Notes: "scoring"}, Progress: 55, NextDueItem: "Ship API"},
		},
		UpcomingProjects: []models.ProjectWithProgress{
			{Project: models.Project{ID: 2, Name: "Healthcare AI", Status: "locked", AutoUnlock: true, UnlockNotes: "after RE"}},
		},
		RecentAchievements: []models.Achievement{
			{Title: "First Launch", StarsBonus: 10, UnlockedAt: "2026-06-20 10:00:00", Description: "Deployed first server"},
		},
	}
	renderOK(t, "dashboard.html", data)

	// empty-slice path: no active destination, no missions, no medals
	renderOK(t, "dashboard.html", models.DashboardData{Page: "dashboard", Today: "2026-06-25"})
}

func TestRenderRoadmap(t *testing.T) {
	main := models.RoadmapProject{
		Project:  models.ProjectWithProgress{Project: models.Project{ID: 1, Name: "Real Estate AI System", Status: "active", ProjectKind: "main", ParentProjectID: 0}},
		Meta:     models.RoadmapProjectMeta{ProjectType: "real_estate_ai", ProjectTypeLabel: "Real Estate AI", CurrentFocus: true},
		Progress: 50,
		Items: []models.ProjectItem{
			{ID: 10, Title: "Build API", Status: "completed", Priority: "high"},
			{ID: 11, Title: "Add case study", Status: "pending", Priority: "medium"},
		},
		Milestones: []models.RoadmapMilestone{{ID: 5, Title: "MVP", Status: "pending", DueDate: "2026-07-10"}},
	}
	sub := models.RoadmapProject{
		Project:  models.ProjectWithProgress{Project: models.Project{ID: 3, Name: "Deploy Station", Status: "locked", ProjectKind: "subproject", ParentProjectID: 1}},
		Meta:     models.RoadmapProjectMeta{ProjectType: "learning", ProjectTypeLabel: "Learning"},
		Progress: 0,
	}
	data := models.RoadmapData{
		Page:               "roadmap",
		ShipStats:          models.ShipStats{TotalFuel: 300, Level: 4, LevelProgress: 0, ActiveTrips: 2, MedalCount: 3, RecentMedal: "Server Deployed"},
		ShipRoute:          services.BuildRocketRoute("Galaxy Map", []models.RouteNode{{Name: "Real Estate AI System", URL: "/projects/1", State: "active"}}),
		Today:              "2026-06-25",
		WeekStart:          "2026-06-22",
		WeekEnd:            "2026-06-28",
		GoalTitle:          "Become an AI Engineer",
		GoalDeadline:       "2026-11-01",
		Projects:           []models.RoadmapProject{main, sub},
		OverallProgress:    42,
		WeeklyCompletion:   30,
		CurrentFocus:       []models.RoadmapProject{main},
		MissedDeadlines:    []models.RoadmapDeadline{{Kind: "task", Title: "Late", ProjectName: "RE", DueDate: "2026-06-01"}},
		WeeklyReviews:      []models.RoadmapWeeklyReview{{WeekStart: "2026-06-15", Finished: "x", Blocked: "y", NextSteps: "z"}},
		ProjectTypeOptions: models.RoadmapProjectTypeOptions(),
	}
	renderOK(t, "roadmap.html", data)

	// empty roadmap path
	renderOK(t, "roadmap.html", models.RoadmapData{Page: "roadmap", GoalTitle: "Goal", ProjectTypeOptions: models.RoadmapProjectTypeOptions()})
}

func TestRenderProjectsList(t *testing.T) {
	tree := models.ProjectTree{
		Main: []models.ProjectWithProgress{
			{Project: models.Project{ID: 1, Name: "Real Estate AI", Status: "active", ProjectKind: "main"}, Progress: 55, TotalItems: 4, CompletedItems: 2},
		},
		Children: map[int64][]models.ProjectWithProgress{
			1: {{Project: models.Project{ID: 2, Name: "Phase 1", Status: "active", ProjectKind: "subproject"}, Progress: 30}},
		},
		Upcoming: []models.ProjectWithProgress{
			{Project: models.Project{ID: 9, Name: "Healthcare AI", Status: "locked", AutoUnlock: true, UnlockNotes: "later"}},
		},
	}
	renderOK(t, "projects_list.html", ProjectListData{
		Page:      "projects",
		Tree:      tree,
		Unlocked:  true,
		ShipStats: models.ShipStats{TotalFuel: 120, Level: 2, LevelProgress: 20, ActiveTrips: 1, MedalCount: 1},
		ShipRoute: services.BuildRocketRoute("Navigation Deck", []models.RouteNode{
			{Name: "Real Estate AI", URL: "/projects/1", State: "active"},
		}),
	})
	renderOK(t, "projects_list.html", ProjectListData{Page: "projects", Tree: models.ProjectTree{Children: map[int64][]models.ProjectWithProgress{}}})
}

func TestRenderProjectDetail(t *testing.T) {
	data := ProjectDetailData{
		Page: "projects",
		Project: models.ProjectWithProgress{
			Project:  models.Project{ID: 1, Name: "Real Estate AI", Category: "business", Status: "active", Notes: "Build the scorer", TargetDate: "2026-08-15", UnlockNotes: "after intro"},
			Progress: 55, TotalItems: 4, CompletedItems: 2, NextDueItem: "Ship API",
		},
		Items: []models.ProjectItem{
			{ID: 10, Title: "Build API", Status: "completed", Priority: "high", DueDate: "2026-07-10", Description: "endpoint"},
			{ID: 11, Title: "Case study", Status: "pending", Priority: "medium"},
		},
		Stops: []models.ProjectWithProgress{
			{Project: models.Project{ID: 2, Name: "Phase 1", Status: "active", ProjectKind: "subproject"}, Progress: 30},
			{Project: models.Project{ID: 3, Name: "Phase 2", Status: "locked", ProjectKind: "subproject", UnlockNotes: "after phase 1"}},
		},
	}
	renderOK(t, "project_detail.html", data)

	// minimal trip with no stops/items
	renderOK(t, "project_detail.html", ProjectDetailData{
		Page:    "projects",
		Project: models.ProjectWithProgress{Project: models.Project{ID: 5, Name: "Bare Trip", Status: "active"}},
	})
}

func TestRenderTaskDetail(t *testing.T) {
	renderOK(t, "task_detail.html", TaskDetailData{
		Page: "tasks",
		Task: models.Task{ID: 1, Title: "Deploy server", Category: "coding", Status: "pending", Priority: "high", DueDate: "2026-06-26", StarsReward: 5, StarsAwarded: 0, ProofRequired: true, ProofType: "link", Description: "ship it"},
	})
	renderOK(t, "task_detail.html", TaskDetailData{Page: "tasks", Task: models.Task{ID: 2, Title: "Bare Mission", Status: "completed"}})
}

func TestRenderToday(t *testing.T) {
	data := TodayData{
		Page:     "today",
		Today:    "2026-06-25",
		Progress: 40,
		Mission:  models.DailyMissionData{Date: "2026-06-25", PrayerCompletion: 60, AthkarCompletion: 50, DeenCompletion: 55},
		Pending: []models.TaskOccurrence{
			{Task: models.Task{ID: 1, Title: "Workout", Category: "health", RepeatType: "daily", Priority: "medium"}, Date: "2026-06-25", Status: "pending"},
		},
		Completed:      []models.TaskOccurrence{{Task: models.Task{ID: 2, Title: "Read"}, Date: "2026-06-25", Status: "completed", CompletedAt: "2026-06-25 08:00:00"}},
		Skipped:        []models.TaskOccurrence{{Task: models.Task{ID: 3, Title: "Skipped one", Category: "personal"}, Date: "2026-06-25", Status: "skipped"}},
		OverdueTasks:   []models.Task{{ID: 4, Title: "Late task", DueDate: "2026-06-20", Priority: "high"}},
		RemindersToday: []models.ReminderView{{Title: "Standup", ReminderTime: "09:00"}},
	}
	renderOK(t, "today.html", data)
	renderOK(t, "today.html", TodayData{Page: "today", Today: "2026-06-25"})
}
