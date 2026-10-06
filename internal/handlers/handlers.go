package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/authlimit"
	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/healthcoach"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
	"lifeops-tracker/internal/xp"
)

type Handler struct {
	Service     *services.Service
	TemplateDir string
	// AllowPublicFeeds gates the unauthenticated plain .ics routes. Off by
	// default (production-safe); the tokenized /calendar/feed/{token} routes are
	// unaffected. Enable in local/dev via ALLOW_PUBLIC_CALENDAR_FEEDS=true.
	AllowPublicFeeds bool
	HealthSyncToken  string // Server-only Health API secret; never rendered.
	// Auth holds the password hash used to sign in. Never rendered or logged.
	Auth services.AuthConfig
	// AuthDisabled is an emergency escape hatch for a lockout. It serves every
	// route unauthenticated and logs loudly on each request; never leave it on.
	AuthDisabled bool
	// loginLimiter throttles failed sign-ins (GYST-VN-006). It is built in New.
	// A nil limiter makes POST /login fail closed rather than run unthrottled.
	loginLimiter *authlimit.Limiter
	// Coach answers Health questions (Issue #64). Nil means no AI provider is
	// configured: the Coach page shows that state and the rest of Health works.
	Coach      healthcoach.Provider
	coachLimit coachLimiter
}

type TodayData struct {
	ChecklistOnly  bool
	NextReminder   *models.ReminderView
	Greeting       string
	HealthSnapshot TodayHealthSnapshot
	Page           string
	Today          string
	Progress       int
	ShipStats      models.ShipStats
	Mission        models.DailyMissionData
	Pending        []models.TaskOccurrence
	Completed      []models.TaskOccurrence
	Skipped        []models.TaskOccurrence
	OverdueTasks   []models.Task
	RemindersToday []models.ReminderView
	Learning       models.TodayLearning
	Prayer         PrayerTodayData
}

type PrayerTodayData struct {
	Schedule    models.PrayerSchedule
	Next        models.NextPrayer
	NextAt      string
	CurrentName string
	Config      models.PrayerConfig
}

type TaskListData struct {
	Page  string
	Today string
	Tasks []models.Task
	// StatusFilter is the ?status= view filter: pending | completed | skipped |
	// overdue | all. Presentation-only; "overdue" means pending with a past due
	// date.
	StatusFilter string
}

type TaskFormData struct {
	Page   string
	Today  string
	Title  string
	Action string
	Task   models.Task
}

type TaskDetailData struct {
	Page          string
	Task          models.Task
	Checklist     []models.TaskChecklistItem
	ShipStats     models.ShipStats
	ExternalLinks []models.ExternalLink
}

type PlanImportData struct {
	Page        string
	Markdown    string
	Template    string
	Plan        models.AIPlan
	Preview     bool
	Destination string
	TargetID    int64
	Projects    []models.ProjectWithProgress
	Tasks       []models.Task
}

type MonthlyData struct {
	Page      string
	MonthName string
	Progress  int
	Chart     models.ChartData
	Items     []models.TaskOccurrence
}

type ProjectListData struct {
	Page              string
	ShipStats         models.ShipStats
	ShipRoute         models.RocketRoute
	Tree              models.ProjectTree
	Unlocked          bool
	Roadmaps          []models.Roadmap
	SelectedRoadmapID int64
	SelectedRoadmap   models.Roadmap
	RoadmapNames      map[int64]string
	StatusFilter      string
	// All-roadmaps view: projects grouped into a section per roadmap.
	Grouped       bool
	Groups        []RoadmapGroup
	AnyProjects   bool
	OpenAllGroups bool
}

// RoadmapGroup is one roadmap's section on the all-roadmaps Projects view: its
// metadata plus its own (top-level-only, subprojects-nested) project tree.
type RoadmapGroup struct {
	Roadmap models.Roadmap
	Tree    models.ProjectTree
	Count   int // top-level trips (Main + Upcoming)
}

// sortRoadmapGroups orders sections: featured first, then priority DESC, then
// most-recently-updated, then name ASC.
func sortRoadmapGroups(groups []RoadmapGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i].Roadmap, groups[j].Roadmap
		if a.Featured != b.Featured {
			return a.Featured
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.UpdatedAt != b.UpdatedAt {
			return a.UpdatedAt > b.UpdatedAt
		}
		return a.Name < b.Name
	})
}

// projectsRoute builds the rocket-route nodes (trips) from main projects.
func projectsRoute(title string, mains []models.ProjectWithProgress) models.RocketRoute {
	nodes := make([]models.RouteNode, 0, len(mains))
	for _, project := range mains {
		nodes = append(nodes, models.RouteNode{
			Name:  project.Project.Name,
			URL:   fmt.Sprintf("/projects/%d", project.Project.ID),
			State: services.RouteState(project.Project.Status, project.Progress),
		})
	}
	return services.BuildRocketRoute("deck", title, "galaxy", nodes)
}

type ProjectFormData struct {
	Page     string
	Today    string
	Title    string
	Action   string
	Project  models.Project
	Projects []models.ProjectWithProgress
	Roadmaps []models.Roadmap
}

type ProjectDetailData struct {
	Page             string
	Project          models.ProjectWithProgress
	Items            []models.ProjectItem
	Stops            []models.ProjectWithProgress
	Route            models.RocketRoute
	Roadmap          models.Roadmap
	HasRoadmap       bool
	Parent           models.Project
	HasParent        bool
	NextDestinations []models.Project
	ExternalLinks    []models.ExternalLink
	ItemLinks        map[int64][]models.ExternalLink
}

type GoalListData struct {
	Page  string
	Goals []models.GoalWithProgress
	Chart models.ChartData
}

type GoalFormData struct {
	Page   string
	Today  string
	Title  string
	Action string
	Goal   models.Goal
}

type GoalDetailData struct {
	Page          string
	Today         string
	Goal          models.GoalWithProgress
	Entries       []models.GoalProgressEntry
	HistoryLabels []string
	HistoryValues []int
}

type RemindersData struct {
	Page             string
	Today            string
	RemindersToday   []models.ReminderView
	OverdueReminders []models.ReminderView
}

type AnalyticsData struct {
	Page      string
	Dashboard models.DashboardData
}

type HealthData struct {
	Page        string
	HealthGoals []models.GoalWithProgress
	HealthTasks []models.Task
	Mission     models.DailyMissionData
	BodyRhythm  models.ChartData
	// Today carries real observations. Absent metrics render as "No data";
	// nothing here is projected or synthesized.
	Today HealthTodayView
	// Health v3 (#53).
	HealthTab string
	Weight    healthWeightView
	Checklist []healthChecklistRow
	// Today redesign (#70).
	ChecklistDone int
	Nudge         *HealthNudge
}

type ArchiveData struct {
	Page     string
	Tasks    []models.Task
	Projects []models.ProjectWithProgress
	Goals    []models.GoalWithProgress
}

type SettingsData struct {
	Page          string
	CalendarToken string
	BaseURL       string
	Prayer        models.PrayerConfig
}

type RoadmapFormData struct {
	Page    string
	Title   string
	Action  string
	Roadmap models.Roadmap
	IsEdit  bool
}

func New(service *services.Service, templateDir string) *Handler {
	return &Handler{
		Service:      service,
		TemplateDir:  templateDir,
		loginLimiter: authlimit.New(clock.System().Now, authlimit.DefaultPolicy),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		h.today(w, r)
		return
	}

	segments := strings.Split(path, "/")
	switch segments[0] {
	case "dashboard":
		http.Redirect(w, r, "/today", http.StatusPermanentRedirect)
	case "checklist":
		if len(segments) != 1 {
			http.NotFound(w, r)
			return
		}
		h.today(w, r)
	case "today":
		h.today(w, r)
	case "galaxy":
		if len(segments) == 2 && segments[1] == "data" {
			h.galaxyData(w, r)
		} else {
			h.galaxy(w, r)
		}
	case "more":
		h.more(w, r)
	case "monthly":
		h.monthly(w, r)
	case "tasks":
		h.tasks(w, r, segments)
	case "projects":
		h.projects(w, r, segments)
	case "goals":
		h.goals(w, r, segments)
	case "reminders":
		h.reminders(w, r)
	case "analytics":
		h.analytics(w, r)
	case "login":
		h.login(w, r)
	case "logout":
		h.logout(w, r)
	case "health":
		if len(segments) == 1 {
			h.health(w, r)
		} else if len(segments) == 2 && segments[1] == "progress" {
			h.healthProgress(w, r)
		} else if len(segments) == 2 && segments[1] == "weight" {
			h.healthWeight(w, r)
		} else if len(segments) == 4 && segments[1] == "weight" && segments[3] == "undo" {
			h.healthWeightUndo(w, r, segments[2])
		} else if len(segments) == 3 && segments[1] == "training" && segments[2] == "sync" {
			h.trainingSync(w, r)
		} else if len(segments) == 2 && segments[1] == "training" {
			h.training(w, r)
		} else if len(segments) == 2 && segments[1] == "nutrition" {
			h.nutrition(w, r)
		} else if len(segments) == 2 && segments[1] == "plan" {
			h.healthPlan(w, r)
		} else if len(segments) == 2 && segments[1] == "more" {
			h.healthMore(w, r)
		} else if len(segments) == 2 && segments[1] == "checklist" {
			h.healthChecklist(w, r)
		} else if len(segments) == 4 && segments[1] == "checklist" {
			h.healthChecklistItemAction(w, r, segments[2], segments[3])
		} else if len(segments) == 2 && segments[1] == "coach" {
			h.healthCoach(w, r)
		} else if len(segments) == 3 && segments[1] == "coach" && segments[2] == "clear" {
			h.healthCoachClear(w, r)
		} else if len(segments) == 2 && segments[1] == "profile" {
			h.healthProfile(w, r)
		} else if len(segments) == 2 && segments[1] == "foods" {
			h.healthFoods(w, r)
		} else if len(segments) == 3 && segments[1] == "foods" {
			h.healthFood(w, r, segments[2])
		} else {
			http.NotFound(w, r)
		}
	case "roadmap":
		h.roadmap(w, r, segments)
	case "roadmaps":
		h.roadmaps(w, r, segments)
	case "courses":
		h.courses(w, r, segments)
	case "calendar.ics":
		h.allCalendarICS(w, r)
	case "calendar":
		h.calendarFeed(w, r, segments)
	case "archive":
		h.archive(w, r)
	case "settings":
		h.settings(w, r, segments)
	case "trackers":
		h.trackers(w, r, segments)
	case "external-links":
		h.externalLinks(w, r, segments)
	case "plan-import":
		h.planImport(w, r)
	case "api":
		if len(segments) >= 2 && segments[1] == "health" {
			h.healthAPI(w, r, segments)
		} else {
			h.pushAPI(w, r, segments)
		}
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) pushAPI(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 3 || segments[1] != "push" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch segments[2] {
	case "vapid-key":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		key := h.Service.PushPublicKey()
		if key == "" {
			http.Error(w, `{"error":"push is not configured"}`, http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"publicKey": key})
	case "subscribe":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		var input struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256DH string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil {
			http.Error(w, `{"error":"invalid subscription"}`, http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(input.Endpoint, "https://") || len(input.Endpoint) > 4096 {
			http.Error(w, `{"error":"invalid endpoint"}`, http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if err := h.Service.Repo.DeletePushSubscription(input.Endpoint); err != nil {
				serverError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if input.Keys.P256DH == "" || input.Keys.Auth == "" {
			http.Error(w, `{"error":"subscription keys required"}`, http.StatusBadRequest)
			return
		}
		if err := h.Service.Repo.UpsertPushSubscription(models.PushSubscription{Endpoint: input.Endpoint, P256DH: input.Keys.P256DH, Auth: input.Keys.Auth, UserAgent: r.UserAgent()}); err != nil {
			serverError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) galaxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	h.render(w, "galaxy.html", struct{ Page string }{Page: "galaxy"})
}

func (h *Handler) galaxyData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	planets, err := h.Service.GalaxyData()
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(planets); err != nil {
		serverError(w, err)
	}
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, err := h.Service.TaskOccurrencesOn(h.Service.Today())
	if err != nil {
		serverError(w, err)
		return
	}
	overdueTasks, err := h.Service.Repo.ListActiveOneTimeOverdueTasks(h.Service.TodayString())
	if err != nil {
		serverError(w, err)
		return
	}
	remindersToday, _, err := h.Service.ReminderViews(h.Service.Today())
	if err != nil {
		serverError(w, err)
		return
	}
	mission, err := h.Service.DailyMission()
	if err != nil {
		serverError(w, err)
		return
	}
	pending, completed, skipped := services.SplitOccurrences(items)
	stats, err := h.Service.ShipStats()
	if err != nil {
		serverError(w, err)
		return
	}
	learning, err := h.Service.TodayLearning()
	if err != nil {
		serverError(w, err)
		return
	}
	// One instant for the prayer, greeting and next-reminder decisions below,
	// read in the application timezone.
	now := h.Service.Now().In(h.Service.Location())
	schedule, err := h.Service.PrayerTimes(now)
	if err != nil {
		serverError(w, err)
		return
	}
	prayerConfig, err := h.Service.PrayerConfig()
	if err != nil {
		serverError(w, err)
		return
	}
	nextPrayer, err := h.Service.NextPrayer(now)
	if err != nil {
		serverError(w, err)
		return
	}
	currentName := ""
	for i, prayer := range schedule.Times {
		if !prayer.At.After(now) {
			currentName = prayer.Name
		}
		if i == 0 && prayer.At.After(now) {
			currentName = ""
		}
	}
	checklistOnly := strings.Trim(r.URL.Path, "/") == "checklist"
	var healthSnapshot TodayHealthSnapshot
	if !checklistOnly {
		health, healthErr := h.Service.HealthToday()
		healthSnapshot = buildTodayHealthSnapshot(buildHealthTodayView(health, now), healthErr)
	}
	greeting := greetingFor(now.Hour())
	page := "today.html"
	if checklistOnly {
		page = "checklist.html"
	}
	h.render(w, page, TodayData{
		ChecklistOnly:  checklistOnly,
		Greeting:       greeting,
		NextReminder:   nextTodayReminder(remindersToday, now),
		HealthSnapshot: healthSnapshot,
		Page:           "today",
		Today:          h.Service.TodayString(),
		Progress:       services.CompletionPercentage(items),
		ShipStats:      stats,
		Mission:        mission,
		Pending:        pending,
		Completed:      completed,
		Skipped:        skipped,
		OverdueTasks:   overdueTasks,
		RemindersToday: remindersToday,
		Learning:       learning,
		Prayer: PrayerTodayData{
			Schedule: schedule, Next: nextPrayer,
			NextAt: nextPrayer.Prayer.At.Format(time.RFC3339), CurrentName: currentName, Config: prayerConfig,
		},
	})
}

func (h *Handler) monthly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	today := h.Service.Today()
	start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	end := start.AddDate(0, 1, -1)
	items, err := h.Service.TaskOccurrencesBetween(start, end)
	if err != nil {
		serverError(w, err)
		return
	}
	chart, err := h.Service.MonthlyCompletionChart(today.Year(), today.Month())
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "monthly.html", MonthlyData{
		Page:      "monthly",
		MonthName: today.Format("January 2006"),
		Progress:  services.CompletionPercentage(items),
		Chart:     chart,
		Items:     items,
	})
}

func (h *Handler) tasks(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		tasks, err := h.Service.Repo.ListTasks(false)
		if err != nil {
			serverError(w, err)
			return
		}
		today := h.Service.TodayString()
		filter := r.URL.Query().Get("status")
		switch filter {
		case "pending", "completed", "skipped", "overdue":
			filtered := make([]models.Task, 0, len(tasks))
			for _, task := range tasks {
				switch filter {
				case "overdue":
					if task.Status == "pending" && task.DueDate != "" && task.DueDate < today {
						filtered = append(filtered, task)
					}
				default:
					if task.Status == filter {
						filtered = append(filtered, task)
					}
				}
			}
			tasks = filtered
		default:
			filter = "all"
		}
		h.render(w, "tasks_list.html", TaskListData{Page: "tasks", Today: today, Tasks: tasks, StatusFilter: filter})
		return
	}

	if len(segments) == 2 && segments[1] == "new" {
		h.newTask(w, r)
		return
	}
	if len(segments) == 2 && segments[1] == "quick" {
		h.quickTask(w, r)
		return
	}

	id, ok := parseID(segmentAt(segments, 1))
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(segments) == 2 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		task, err := h.Service.Repo.TaskByID(id)
		if err != nil {
			notFoundOrError(w, err)
			return
		}
		stats, err := h.Service.ShipStats()
		if err != nil {
			serverError(w, err)
			return
		}
		links, err := h.Service.Repo.ExternalLinks("task", id)
		if err != nil {
			serverError(w, err)
			return
		}
		checklist, err := h.Service.Repo.TaskChecklist(id)
		if err != nil {
			serverError(w, err)
			return
		}
		h.render(w, "task_detail.html", TaskDetailData{Page: "tasks", Task: task, Checklist: checklist, ShipStats: stats, ExternalLinks: links})
		return
	}

	switch segments[2] {
	case "edit":
		h.editTask(w, r, id)
	case "complete":
		h.setTaskStatus(w, r, id, "completed")
	case "skip":
		h.setTaskStatus(w, r, id, "skipped")
	case "undo":
		h.undoTaskCompletion(w, r, id)
	case "archive":
		h.archiveTask(w, r, id)
	case "checklist":
		h.taskChecklist(w, r, id, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) taskChecklist(w http.ResponseWriter, r *http.Request, taskID int64, segments []string) {
	if r.Method != http.MethodPost || len(segments) != 4 {
		methodNotAllowed(w)
		return
	}
	itemID, ok := parseID(segments[3])
	if !ok {
		http.NotFound(w, r)
		return
	}
	status := "pending"
	if r.FormValue("completed") != "" {
		status = "completed"
	}
	if err := h.Service.Repo.SetTaskChecklistStatus(taskID, itemID, status); err != nil {
		notFoundOrError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/tasks/%d", taskID), http.StatusSeeOther)
}

func (h *Handler) planImport(w http.ResponseWriter, r *http.Request) {
	projects, err := h.Service.Repo.ListProjects(false)
	if err != nil {
		serverError(w, err)
		return
	}
	tasks, err := h.Service.Repo.ListTasks(false)
	if err != nil {
		serverError(w, err)
		return
	}
	data := PlanImportData{Page: "plan-import", Template: services.AIPlanPromptTemplate, Projects: projects, Tasks: tasks, Destination: "new"}
	if r.Method == http.MethodGet {
		h.render(w, "plan_import.html", data)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	data.Markdown = strings.TrimSpace(r.FormValue("markdown"))
	data.Destination = r.FormValue("destination")
	data.TargetID, _ = parseID(r.FormValue("target_id"))
	if data.Markdown == "" {
		http.Error(w, "paste a Markdown plan first", http.StatusBadRequest)
		return
	}
	data.Plan = services.ParseAIPlanMarkdown(data.Markdown, h.Service.Today())
	if r.FormValue("action") != "import" {
		data.Preview = true
		h.render(w, "plan_import.html", data)
		return
	}
	redirect, err := h.Service.ImportAIPlan(data.Plan, data.Destination, data.TargetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (h *Handler) quickTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	task, err := taskFromRequest(r)
	if err != nil {
		serverError(w, err)
		return
	}
	if strings.TrimSpace(task.Title) == "" {
		http.Error(w, "task title is required", http.StatusBadRequest)
		return
	}
	if !validDueDate(task.DueDate) {
		invalidDate(w, "due_date")
		return
	}
	if task.DueDate == "" {
		task.DueDate = h.Service.TodayString()
	}
	if err := h.Service.Repo.CreateTask(&task); err != nil {
		serverError(w, err)
		return
	}
	redirectAfter(w, r, "/today")
}

func (h *Handler) newTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.render(w, "task_form.html", TaskFormData{
			Page:   "tasks",
			Today:  h.Service.TodayString(),
			Title:  "New Task",
			Action: "/tasks/new",
			Task: models.Task{
				Category:    "personal",
				Priority:    "medium",
				Status:      "pending",
				DueDate:     h.Service.TodayString(),
				RepeatType:  "none",
				StarsReward: xp.TaskReward("medium"),
				ProofType:   "none",
			},
		})
	case http.MethodPost:
		task, err := taskFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if !validDueDate(task.DueDate) {
			invalidDate(w, "due_date")
			return
		}
		if err := h.Service.Repo.CreateTask(&task); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/tasks/%d", task.ID), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) editTask(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		task, err := h.Service.Repo.TaskByID(id)
		if err != nil {
			notFoundOrError(w, err)
			return
		}
		h.render(w, "task_form.html", TaskFormData{
			Page:   "tasks",
			Today:  h.Service.TodayString(),
			Title:  "Edit Task",
			Action: fmt.Sprintf("/tasks/%d/edit", id),
			Task:   task,
		})
	case http.MethodPost:
		task, err := taskFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if !validDueDate(task.DueDate) {
			invalidDate(w, "due_date")
			return
		}
		task.ID = id
		if err := h.Service.Repo.UpdateTask(&task); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/tasks/%d", id), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) setTaskStatus(w http.ResponseWriter, r *http.Request, id int64, status string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	date, err := h.formLocalDate(r.FormValue("occurrence_date"))
	if err != nil {
		invalidDate(w, "occurrence_date")
		return
	}
	var before models.PlayerStats
	var haveBefore bool
	if status == "completed" {
		before, haveBefore = h.statsSnapshot()
	}
	if err := h.Service.Repo.SetTaskOccurrenceStatus(id, date, status); err != nil {
		if refuseCompletion(w, r, err, fmt.Sprintf("/today#task-%d", id)) {
			return
		}
		serverError(w, err)
		return
	}
	if status == "completed" {
		task, err := h.Service.Repo.TaskByID(id)
		if err == nil {
			gained, level := h.rewardDelta(before, haveBefore)
			redirectWithCelebration(w, r, fmt.Sprintf("/today#task-%d", id), celebration{
				Kind: "task", Title: task.Title, XP: gained, Level: level,
				Undo: fmt.Sprintf("/tasks/%d/undo", id), Date: date,
			})
			return
		}
	}
	redirectAfter(w, r, fmt.Sprintf("/today#task-%d", id))
}

// undoTaskCompletion reverses a completion made from the toast. POST only; the
// CSRF guard in Authenticated() applies. A second call changes nothing.
func (h *Handler) undoTaskCompletion(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	date, err := h.formLocalDate(r.FormValue("occurrence_date"))
	if err != nil {
		invalidDate(w, "occurrence_date")
		return
	}
	if _, err := h.Service.Repo.UndoTaskCompletion(id, date); err != nil {
		notFoundOrError(w, err)
		return
	}
	redirectAfter(w, r, fmt.Sprintf("/today#task-%d", id))
}

func (h *Handler) archiveTask(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := h.Service.Repo.ArchiveTask(id); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *Handler) projects(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		roadmaps, err := h.Service.Repo.ListRoadmaps(false)
		if err != nil {
			serverError(w, err)
			return
		}
		roadmapNames := make(map[int64]string, len(roadmaps))
		for _, roadmap := range roadmaps {
			roadmapNames[roadmap.ID] = roadmap.Name
		}
		stats, err := h.Service.ShipStats()
		if err != nil {
			serverError(w, err)
			return
		}
		statusFilter := projectFilter(r.URL.Query().Get("status"))
		unlocked := r.URL.Query().Get("unlocked") == "1"

		// A specific roadmap is selected → single filtered deck (radio tabs).
		if id, ok := parseID(r.URL.Query().Get("roadmap")); ok {
			selectedRoadmap, _ := h.Service.Repo.GetRoadmap(id)
			tree, err := h.Service.Repo.ProjectTreeByRoadmap(id)
			if err != nil {
				serverError(w, err)
				return
			}
			h.render(w, "projects_list.html", ProjectListData{
				Page:              "projects",
				ShipStats:         stats,
				ShipRoute:         projectsRoute(selectedRoadmap.Name, tree.Main),
				Tree:              tree,
				Unlocked:          unlocked,
				Roadmaps:          roadmaps,
				SelectedRoadmapID: id,
				SelectedRoadmap:   selectedRoadmap,
				RoadmapNames:      roadmapNames,
				StatusFilter:      statusFilter,
			})
			return
		}

		// All roadmaps. The overview route shows every top-level destination.
		allTree, err := h.Service.Repo.ProjectTree()
		if err != nil {
			serverError(w, err)
			return
		}

		// With no roadmaps at all (e.g. all archived), fall back to one ungrouped
		// deck so orphan trips still appear.
		if len(roadmaps) == 0 {
			h.render(w, "projects_list.html", ProjectListData{
				Page:         "projects",
				ShipStats:    stats,
				ShipRoute:    projectsRoute("Navigation Deck", allTree.Main),
				Tree:         allTree,
				Unlocked:     unlocked,
				Roadmaps:     roadmaps,
				RoadmapNames: roadmapNames,
				StatusFilter: statusFilter,
			})
			return
		}

		// Self-heal: adopt orphan trips into the dashboard roadmap so every trip
		// lands in a group (mirrors the dashboard's adoption).
		if dashboard, drErr := h.Service.Repo.GetDashboardRoadmap(); drErr == nil {
			if err := h.Service.Repo.AdoptOrphanProjects(dashboard.ID); err != nil {
				serverError(w, err)
				return
			}
		}

		groups := make([]RoadmapGroup, 0, len(roadmaps))
		anyProjects := false
		for _, roadmap := range roadmaps {
			tree, err := h.Service.Repo.ProjectTreeByRoadmap(roadmap.ID)
			if err != nil {
				serverError(w, err)
				return
			}
			count := len(tree.Main) + len(tree.Upcoming)
			if count > 0 {
				anyProjects = true
			}
			groups = append(groups, RoadmapGroup{Roadmap: roadmap, Tree: tree, Count: count})
		}
		sortRoadmapGroups(groups)

		h.render(w, "projects_list.html", ProjectListData{
			Page:          "projects",
			ShipStats:     stats,
			ShipRoute:     projectsRoute("Navigation Deck", allTree.Main),
			Unlocked:      unlocked,
			Roadmaps:      roadmaps,
			RoadmapNames:  roadmapNames,
			StatusFilter:  statusFilter,
			Grouped:       true,
			Groups:        groups,
			AnyProjects:   anyProjects,
			OpenAllGroups: len(groups) <= 2,
		})
		return
	}

	if len(segments) == 2 && segments[1] == "new" {
		h.newProject(w, r)
		return
	}

	// /projects/calendar.ics must be matched before the segment is parsed as an
	// ID (otherwise "calendar.ics" fails parseID and 404s).
	if len(segments) == 2 && segments[1] == "calendar.ics" {
		h.projectsCalendarICS(w, r)
		return
	}

	id, ok := parseID(segmentAt(segments, 1))
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(segments) == 2 {
		h.projectDetail(w, r, id)
		return
	}

	switch segments[2] {
	case "edit":
		h.editProject(w, r, id)
	case "archive":
		h.archiveProject(w, r, id)
	case "status":
		h.projectStatus(w, r, id)
	case "items":
		h.projectItems(w, r, id, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) newProject(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := h.Service.Repo.ListProjects(false)
		if err != nil {
			serverError(w, err)
			return
		}
		roadmaps, err := h.Service.Repo.ListRoadmaps(false)
		if err != nil {
			serverError(w, err)
			return
		}
		newProject := models.Project{
			Category:    "other",
			Color:       "cyan",
			ProjectKind: "main",
			Status:      "active",
			StartDate:   h.Service.TodayString(),
			AutoUnlock:  true,
		}
		if id, ok := parseID(r.URL.Query().Get("roadmap")); ok {
			newProject.RoadmapID = id
		}
		h.render(w, "project_form.html", ProjectFormData{
			Page:     "projects",
			Today:    h.Service.TodayString(),
			Title:    "New Project",
			Action:   "/projects/new",
			Project:  newProject,
			Projects: projects,
			Roadmaps: roadmaps,
		})
	case http.MethodPost:
		project, err := projectFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := h.Service.Repo.CreateProject(&project); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/projects/%d", project.ID), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) projectDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	project, err := h.Service.Repo.ProjectWithProgress(id)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	project.Momentum = services.TrendChart(project.Progress)
	items, err := h.Service.Repo.ProjectItems(id)
	if err != nil {
		serverError(w, err)
		return
	}
	externalLinks, err := h.Service.Repo.ExternalLinks("project", id)
	if err != nil {
		serverError(w, err)
		return
	}
	itemLinks := make(map[int64][]models.ExternalLink, len(items))
	for _, item := range items {
		links, linkErr := h.Service.Repo.ExternalLinks("project_item", item.ID)
		if linkErr != nil {
			serverError(w, linkErr)
			return
		}
		itemLinks[item.ID] = links
	}
	// Stops = subprojects of this trip (any status), derived from the project
	// hierarchy. No schema change: we filter the existing project list.
	allProjects, err := h.Service.Repo.ListProjects(false)
	if err != nil {
		serverError(w, err)
		return
	}
	var stops []models.ProjectWithProgress
	var parent models.Project
	hasParent := false
	var nextDestinations []models.Project
	for _, candidate := range allProjects {
		if candidate.Project.ParentProjectID == id {
			stops = append(stops, candidate)
		}
		if project.Project.ParentProjectID != 0 && candidate.Project.ID == project.Project.ParentProjectID {
			parent = candidate.Project
			hasParent = true
		}
		if candidate.Project.DependsOnProjectID == id {
			nextDestinations = append(nextDestinations, candidate.Project)
		}
	}
	var roadmap models.Roadmap
	hasRoadmap := false
	if project.Project.RoadmapID != 0 {
		if found, rmErr := h.Service.Repo.GetRoadmap(project.Project.RoadmapID); rmErr == nil {
			roadmap = found
			hasRoadmap = true
		}
	}
	route, err := h.Service.RocketRouteForProject(id)
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "project_detail.html", ProjectDetailData{
		Page:             "projects",
		Project:          project,
		Items:            items,
		Stops:            stops,
		Route:            route,
		Roadmap:          roadmap,
		HasRoadmap:       hasRoadmap,
		Parent:           parent,
		HasParent:        hasParent,
		NextDestinations: nextDestinations,
		ExternalLinks:    externalLinks,
		ItemLinks:        itemLinks,
	})
}

func (h *Handler) externalLinks(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	redirectTarget := safeReturnTo(r)
	if redirectTarget == "" {
		redirectTarget = "/"
	}
	if len(segments) == 1 {
		entityID, err := strconv.ParseInt(r.FormValue("entity_id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid link target", http.StatusBadRequest)
			return
		}
		link, err := h.Service.BuildExternalLink(r.FormValue("entity_type"), entityID, r.FormValue("url"), r.FormValue("summary"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.Service.Repo.CreateExternalLink(&link); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
		return
	}
	id, ok := parseID(segmentAt(segments, 1))
	if !ok || len(segments) != 3 {
		http.NotFound(w, r)
		return
	}
	switch segments[2] {
	case "edit":
		if _, err := h.Service.Repo.ExternalLinkByID(id); err != nil {
			notFoundOrError(w, err)
			return
		}
		if err := h.Service.Repo.UpdateExternalLinkSummary(id, strings.TrimSpace(r.FormValue("summary"))); err != nil {
			serverError(w, err)
			return
		}
	case "delete":
		if err := h.Service.Repo.DeleteExternalLink(id); err != nil {
			notFoundOrError(w, err)
			return
		}
	case "sync":
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := h.Service.SyncExternalLink(ctx, id); err != nil {
			http.Error(w, "sync failed: "+err.Error(), http.StatusBadGateway)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
}

func (h *Handler) editProject(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		project, err := h.Service.Repo.ProjectByID(id)
		if err != nil {
			notFoundOrError(w, err)
			return
		}
		projects, err := h.Service.Repo.ListProjects(false)
		if err != nil {
			serverError(w, err)
			return
		}
		roadmaps, err := h.Service.Repo.ListRoadmaps(false)
		if err != nil {
			serverError(w, err)
			return
		}
		h.render(w, "project_form.html", ProjectFormData{
			Page:     "projects",
			Today:    h.Service.TodayString(),
			Title:    "Edit Project",
			Action:   fmt.Sprintf("/projects/%d/edit", id),
			Project:  project,
			Projects: projects,
			Roadmaps: roadmaps,
		})
	case http.MethodPost:
		project, err := projectFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		project.ID = id
		if err := h.Service.Repo.UpdateProject(&project); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/projects/%d", id), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) archiveProject(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := h.Service.Repo.ArchiveProject(id); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (h *Handler) projectStatus(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	status := r.FormValue("status")
	before, _ := h.Service.Repo.ProjectByID(id)
	// Detect whether marking this trip done will unlock a planned route, so the
	// Navigation Deck can play the "New Destination Unlocked" cue once.
	willUnlock := 0
	if cleanForm(status, "") == "done" {
		willUnlock, _ = h.Service.Repo.CountUnlockableDependents(id)
	}
	statsBefore, haveStats := h.statsSnapshot()
	if err := h.Service.Repo.SetProjectStatus(id, status); err != nil {
		serverError(w, err)
		return
	}
	if cleanForm(status, "") == "done" {
		gained, level := h.rewardDelta(statsBefore, haveStats)
		redirectWithCelebration(w, r, "/projects", celebration{Kind: "project", Title: before.Name, XP: gained, Level: level})
		return
	}
	if willUnlock > 0 {
		http.Redirect(w, r, "/projects?unlocked=1", http.StatusSeeOther)
		return
	}
	redirectAfter(w, r, "/projects")
}

func (h *Handler) projectItems(w http.ResponseWriter, r *http.Request, projectID int64, segments []string) {
	if len(segments) == 3 {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		item, err := projectItemFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		item.ProjectID = projectID
		if err := h.Service.Repo.CreateProjectItem(&item); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/projects/%d", projectID), http.StatusSeeOther)
		return
	}

	if len(segments) != 5 {
		http.NotFound(w, r)
		return
	}
	itemID, ok := parseID(segments[3])
	if !ok || r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	projectBefore, _ := h.Service.Repo.ProjectByID(projectID)
	itemTitle := "Mission complete"
	if items, err := h.Service.Repo.ProjectItems(projectID); err == nil {
		for _, item := range items {
			if item.ID == itemID {
				itemTitle = item.Title
				break
			}
		}
	}
	action := segments[4]
	var statsBefore models.PlayerStats
	var haveStats bool
	if action == "complete" {
		statsBefore, haveStats = h.statsSnapshot()
	}
	switch action {
	case "undo":
		if _, err := h.Service.Repo.UndoProjectItemCompletion(projectID, itemID); err != nil {
			serverError(w, err)
			return
		}
	case "complete":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "completed"); err != nil {
			if refuseCompletion(w, r, err, fmt.Sprintf("/projects/%d", projectID)) {
				return
			}
			serverError(w, err)
			return
		}
	case "reopen":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "pending"); err != nil {
			serverError(w, err)
			return
		}
	case "proof":
		if err := r.ParseForm(); err != nil {
			serverError(w, err)
			return
		}
		if err := h.Service.Repo.UpdateProjectItemProof(
			projectID,
			itemID,
			cleanForm(r.FormValue("proof_type"), "none"),
			strings.TrimSpace(r.FormValue("proof_url")),
			strings.TrimSpace(r.FormValue("proof_path")),
		); err != nil {
			serverError(w, err)
			return
		}
	case "delete":
		if err := h.Service.Repo.DeleteProjectItem(projectID, itemID); err != nil {
			serverError(w, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	if action == "complete" {
		gained, level := h.rewardDelta(statsBefore, haveStats)
		projectAfter, err := h.Service.Repo.ProjectByID(projectID)
		if err == nil && projectBefore.Status != "done" && projectBefore.Status != "completed" && (projectAfter.Status == "done" || projectAfter.Status == "completed") {
			// Finishing a project unlocks dependents and adds a bonus that cannot be
			// reversed safely, so this toast carries no Undo.
			redirectWithCelebration(w, r, fmt.Sprintf("/projects/%d", projectID), celebration{Kind: "project", Title: projectAfter.Name, XP: gained, Level: level})
			return
		}
		redirectWithCelebration(w, r, fmt.Sprintf("/projects/%d#mission-%d", projectID, itemID), celebration{
			Kind: "task", Title: itemTitle, XP: gained, Level: level,
			Undo: fmt.Sprintf("/projects/%d/items/%d/undo", projectID, itemID),
		})
		return
	}
	// Stay on the project detail at the acted-on mission instead of jumping to
	// the top of a generic list.
	redirectAfter(w, r, fmt.Sprintf("/projects/%d#mission-%d", projectID, itemID))
}

func (h *Handler) goals(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		goals, err := h.Service.Repo.ListGoals(false)
		if err != nil {
			serverError(w, err)
			return
		}
		for index := range goals {
			goals[index].Momentum = services.TrendChart(goals[index].Progress)
		}
		chart := models.ChartData{}
		for _, goal := range goals {
			chart.Labels = append(chart.Labels, goal.Goal.Title)
			chart.Values = append(chart.Values, goal.Progress)
		}
		h.render(w, "goals_list.html", GoalListData{Page: "goals", Goals: goals, Chart: chart})
		return
	}

	if len(segments) == 2 && segments[1] == "new" {
		h.newGoal(w, r)
		return
	}

	id, ok := parseID(segmentAt(segments, 1))
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(segments) == 2 {
		h.goalDetail(w, r, id)
		return
	}

	switch segments[2] {
	case "progress":
		h.addGoalProgress(w, r, id)
	case "archive":
		h.archiveGoal(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) newGoal(w http.ResponseWriter, r *http.Request) {
	today := h.Service.Today()
	switch r.Method {
	case http.MethodGet:
		h.render(w, "goal_form.html", GoalFormData{
			Page:   "goals",
			Today:  h.Service.TodayString(),
			Title:  "New Goal",
			Action: "/goals/new",
			Goal: models.Goal{
				Type:      "yearly",
				Category:  "health",
				Unit:      "sessions",
				StartDate: h.Service.TodayString(),
				EndDate:   time.Date(today.Year(), time.December, 31, 0, 0, 0, 0, today.Location()).Format("2006-01-02"),
				Status:    "active",
			},
		})
	case http.MethodPost:
		goal, err := goalFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := h.Service.Repo.CreateGoal(&goal, h.Service.TodayString()); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/goals/%d", goal.ID), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) goalDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	goal, err := h.Service.Repo.GoalWithProgress(id)
	if err != nil {
		notFoundOrError(w, err)
		return
	}
	entries, err := h.Service.Repo.GoalProgressEntries(id)
	if err != nil {
		serverError(w, err)
		return
	}
	labels := make([]string, 0, len(entries))
	values := make([]int, 0, len(entries))
	for _, entry := range entries {
		labels = append(labels, entry.EntryDate)
		values = append(values, int(math.Round(entry.Value)))
	}
	h.render(w, "goal_detail.html", GoalDetailData{
		Page:          "goals",
		Today:         h.Service.TodayString(),
		Goal:          goal,
		Entries:       entries,
		HistoryLabels: labels,
		HistoryValues: values,
	})
}

func (h *Handler) addGoalProgress(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	value, _ := strconv.ParseFloat(r.FormValue("value"), 64)
	entryDate, err := h.formLocalDate(r.FormValue("entry_date"))
	if err != nil {
		invalidDate(w, "entry_date")
		return
	}
	notes := strings.TrimSpace(r.FormValue("notes"))
	if err := h.Service.Repo.AddGoalProgress(id, entryDate, value, notes); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/goals/%d", id), http.StatusSeeOther)
}

func (h *Handler) archiveGoal(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := h.Service.Repo.ArchiveGoal(id); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/goals", http.StatusSeeOther)
}

func (h *Handler) reminders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	remindersToday, overdueReminders, err := h.Service.ReminderViews(h.Service.Today())
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "reminders.html", RemindersData{
		Page:             "reminders",
		Today:            h.Service.TodayString(),
		RemindersToday:   remindersToday,
		OverdueReminders: overdueReminders,
	})
}

func (h *Handler) analytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	data, err := h.Service.Dashboard()
	if err != nil {
		serverError(w, err)
		return
	}
	data.Page = "analytics"
	h.render(w, "analytics.html", AnalyticsData{Page: "analytics", Dashboard: data})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	data, ok := h.healthBase(w)
	if !ok {
		return
	}
	data.HealthTab = "today"
	h.render(w, "health.html", data)
}

// healthBase loads everything the Health landing page and More show.
func (h *Handler) healthBase(w http.ResponseWriter) (HealthData, bool) {
	fail := func(err error) (HealthData, bool) { serverError(w, err); return HealthData{}, false }
	goals, err := h.Service.Repo.ListGoals(false)
	if err != nil {
		return fail(err)
	}
	tasks, err := h.Service.Repo.ListTasks(false)
	if err != nil {
		return fail(err)
	}
	healthGoals := make([]models.GoalWithProgress, 0)
	for _, goal := range goals {
		goal.Momentum = services.TrendChart(goal.Progress)
		if goal.Goal.Category == "health" {
			healthGoals = append(healthGoals, goal)
		}
	}
	healthTasks := make([]models.Task, 0)
	for _, task := range tasks {
		if task.Category == "health" {
			healthTasks = append(healthTasks, task)
		}
	}
	mission, err := h.Service.DailyMission()
	if err != nil {
		return fail(err)
	}
	bodyRhythm, err := h.Service.BodyRhythmChart()
	if err != nil {
		return fail(err)
	}
	// Read through the service rather than this app's own HTTP API: an internal
	// round-trip would buy nothing and add a second error and auth path.
	today, err := h.Service.HealthToday()
	if err != nil {
		return fail(err)
	}
	weights, err := h.Service.ListHealthWeightEntries()
	if err != nil {
		return fail(err)
	}
	view := buildHealthTodayView(today, h.Service.Now())
	items, err := h.Service.Repo.ListHealthChecklistItems(today.Date)
	if err != nil {
		return fail(err)
	}
	cycleStart := ""
	if today.HealthCycle != nil {
		cycleStart = today.HealthCycle.StartDate
	}
	// The nudge is optional: if the Hevy cache can't be read, Today, More and
	// Coach still render, just without a nudge (we can't tell whether a
	// workout happened, so suggesting more movement could be wrong).
	training, trainingErr := h.Service.Repo.TrainingCache("hevy")
	nudge := nudgeInput{
		Hour:        h.Service.Now().In(h.Service.Location()).Hour(),
		HasSnapshot: view.HasSnapshot,
		Stale:       view.LastSyncedStale,
		Workout:     todayWorkoutTitle(training.Entries, today.Date, h.Service.Location()),
	}
	if s := today.Snapshot; s != nil {
		nudge.Steps, nudge.Exercise = s.StepsWatch, s.ExerciseMinutes
		if nudge.Steps == nil {
			nudge.Steps = s.StepsPhone
		}
	}
	if c := today.HealthCycle; c != nil {
		nudge.StepTarget = c.StepTarget
	}
	rows := buildChecklistRows(items, view)
	return HealthData{
		Page:          "health",
		HealthGoals:   healthGoals,
		HealthTasks:   healthTasks,
		Mission:       mission,
		BodyRhythm:    bodyRhythm,
		Today:         view,
		Weight:        buildHealthWeight(weights, cycleStart, h.Service.Now(), h.Service.Location()),
		Checklist:     rows,
		ChecklistDone: checklistDone(rows),
		Nudge:         nudgeUnlessErr(buildHealthNudge(nudge), trainingErr),
	}, true
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	allTasks, err := h.Service.Repo.ListTasks(true)
	if err != nil {
		serverError(w, err)
		return
	}
	allProjects, err := h.Service.Repo.ListProjects(true)
	if err != nil {
		serverError(w, err)
		return
	}
	allGoals, err := h.Service.Repo.ListGoals(true)
	if err != nil {
		serverError(w, err)
		return
	}
	tasks := make([]models.Task, 0)
	for _, task := range allTasks {
		if task.ArchivedAt != "" {
			tasks = append(tasks, task)
		}
	}
	projects := make([]models.ProjectWithProgress, 0)
	for _, project := range allProjects {
		if project.Project.ArchivedAt != "" || project.Project.Status == "archived" {
			projects = append(projects, project)
		}
	}
	goals := make([]models.GoalWithProgress, 0)
	for _, goal := range allGoals {
		if goal.Goal.ArchivedAt != "" || goal.Goal.Status == "archived" {
			goals = append(goals, goal)
		}
	}
	h.render(w, "archive.html", ArchiveData{Page: "archive", Tasks: tasks, Projects: projects, Goals: goals})
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 2 && segments[1] == "prayer" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "the prayer settings form could not be read", http.StatusBadRequest)
			return
		}
		if err := h.Service.SavePrayerConfig(r.Form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/settings#prayer", http.StatusSeeOther)
		return
	}
	// POST /settings/calendar-token — generate/rotate or revoke the feed token.
	if len(segments) == 2 && segments[1] == "calendar-token" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		if r.FormValue("action") == "revoke" {
			if err := h.Service.RevokeCalendarToken(); err != nil {
				serverError(w, err)
				return
			}
		} else if _, err := h.Service.GenerateCalendarToken(); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	if len(segments) != 1 {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	token, err := h.Service.CalendarToken()
	if err != nil {
		serverError(w, err)
		return
	}
	prayer, err := h.Service.PrayerConfig()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "settings.html", SettingsData{Page: "settings", CalendarToken: token, BaseURL: requestBaseURL(r), Prayer: prayer})
}

// roadmaps handles the multiple-roadmaps entity: create, edit, set-as-dashboard.
func (h *Handler) roadmaps(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 2 && segments[1] == "new" {
		h.newRoadmap(w, r)
		return
	}

	id, ok := parseID(segmentAt(segments, 1))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(segments) == 3 {
		switch segments[2] {
		case "edit":
			h.editRoadmap(w, r, id)
			return
		case "feature":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			if err := h.Service.Repo.SetFeaturedRoadmap(id); err != nil {
				serverError(w, err)
				return
			}
			redirectBack(w, r, "/")
			return
		case "archive":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			if err := h.Service.Repo.ArchiveRoadmap(id); err != nil {
				serverError(w, err)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	http.NotFound(w, r)
}

func roadmapFromRequest(r *http.Request) (models.Roadmap, error) {
	if err := r.ParseForm(); err != nil {
		return models.Roadmap{}, err
	}
	return models.Roadmap{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Priority:    parseIntForm(r.FormValue("priority"), 0),
		Status:      cleanForm(r.FormValue("status"), "active"),
		Privacy:     cleanForm(r.FormValue("privacy"), "private"),
		Icon:        cleanForm(r.FormValue("icon"), "rocket"),
		Color:       cleanForm(r.FormValue("color"), "cyan"),
		Featured:    r.FormValue("featured") == "on",
	}, nil
}

func (h *Handler) newRoadmap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.render(w, "roadmap_form.html", RoadmapFormData{
			Page:    "roadmap",
			Title:   "New Roadmap",
			Action:  "/roadmaps/new",
			Roadmap: models.Roadmap{Status: "active", Privacy: "private", Icon: "rocket", Color: "cyan", Priority: 5},
		})
	case http.MethodPost:
		roadmap, err := roadmapFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		if roadmap.Name == "" {
			http.Redirect(w, r, "/roadmaps/new", http.StatusSeeOther)
			return
		}
		if err := h.Service.Repo.CreateRoadmap(&roadmap); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/roadmap?roadmap="+strconv.FormatInt(roadmap.ID, 10), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) editRoadmap(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		roadmap, err := h.Service.Repo.GetRoadmap(id)
		if err != nil {
			notFoundOrError(w, err)
			return
		}
		h.render(w, "roadmap_form.html", RoadmapFormData{
			Page:    "roadmap",
			Title:   "Edit Roadmap",
			Action:  fmt.Sprintf("/roadmaps/%d/edit", id),
			Roadmap: roadmap,
			IsEdit:  true,
		})
	case http.MethodPost:
		roadmap, err := roadmapFromRequest(r)
		if err != nil {
			serverError(w, err)
			return
		}
		roadmap.ID = id
		if err := h.Service.Repo.UpdateRoadmap(&roadmap); err != nil {
			serverError(w, err)
			return
		}
		if roadmap.Featured {
			if err := h.Service.Repo.SetFeaturedRoadmap(id); err != nil {
				serverError(w, err)
				return
			}
		}
		http.Redirect(w, r, "/roadmap?roadmap="+strconv.FormatInt(id, 10), http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (h *Handler) roadmap(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		selectedRoadmapID, _ := parseID(r.URL.Query().Get("roadmap"))
		data, err := h.Service.Roadmap(selectedRoadmapID)
		if err != nil {
			serverError(w, err)
			return
		}
		h.render(w, "roadmap.html", data)
		return
	}

	switch segments[1] {
	case "import":
		h.importRoadmap(w, r)
	case "export":
		h.exportRoadmap(w, r, segments)
	case "weekly-review":
		h.createRoadmapWeeklyReview(w, r)
	case "projects":
		h.roadmapProjects(w, r, segments)
	case "milestones":
		h.roadmapMilestones(w, r, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) importRoadmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	source := strings.TrimSpace(r.FormValue("markdown"))
	if source == "" {
		http.Redirect(w, r, "/roadmap", http.StatusSeeOther)
		return
	}
	if _, err := h.Service.ImportRoadmapMarkdown(source); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/roadmap", http.StatusSeeOther)
}

func (h *Handler) exportRoadmap(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if len(segments) != 3 || (segments[2] != "public" && segments[2] != "private") {
		http.NotFound(w, r)
		return
	}
	public := segments[2] == "public"
	body, err := h.Service.ExportRoadmap(public)
	if err != nil {
		serverError(w, err)
		return
	}
	filename := "ai-engineer-roadmap-private.md"
	if public {
		filename = "ai-engineer-portfolio-summary.md"
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if _, err := w.Write([]byte(body)); err != nil {
		serverError(w, err)
	}
}

func (h *Handler) createRoadmapWeeklyReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	review := models.RoadmapWeeklyReview{
		WeekStart: strings.TrimSpace(r.FormValue("week_start")),
		Finished:  strings.TrimSpace(r.FormValue("finished")),
		Blocked:   strings.TrimSpace(r.FormValue("blocked")),
		NextSteps: strings.TrimSpace(r.FormValue("next_steps")),
	}
	if err := h.Service.CreateRoadmapWeeklyReview(review); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/roadmap", http.StatusSeeOther)
}

func (h *Handler) roadmapProjects(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if len(segments) < 4 {
		http.NotFound(w, r)
		return
	}
	projectID, ok := parseID(segments[2])
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(segments) == 4 && segments[3] == "meta" {
		meta, err := roadmapMetaFromRequest(r, projectID)
		if err != nil {
			serverError(w, err)
			return
		}
		if err := h.Service.Repo.SetRoadmapProjectMeta(meta); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/roadmap", http.StatusSeeOther)
		return
	}

	if len(segments) != 6 || segments[3] != "items" {
		http.NotFound(w, r)
		return
	}
	itemID, ok := parseID(segments[4])
	if !ok {
		http.NotFound(w, r)
		return
	}
	projectBefore, _ := h.Service.Repo.ProjectByID(projectID)
	itemTitle := "Mission complete"
	if items, err := h.Service.Repo.ProjectItems(projectID); err == nil {
		for _, item := range items {
			if item.ID == itemID {
				itemTitle = item.Title
				break
			}
		}
	}
	action := segments[5]
	var statsBefore models.PlayerStats
	var haveStats bool
	if action == "complete" {
		statsBefore, haveStats = h.statsSnapshot()
	}
	switch action {
	case "complete":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "completed"); err != nil {
			if refuseCompletion(w, r, err, fmt.Sprintf("/projects/%d", projectID)) {
				return
			}
			serverError(w, err)
			return
		}
	case "reopen":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "pending"); err != nil {
			serverError(w, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	if action == "complete" {
		gained, level := h.rewardDelta(statsBefore, haveStats)
		projectAfter, err := h.Service.Repo.ProjectByID(projectID)
		if err == nil && projectBefore.Status != "done" && projectBefore.Status != "completed" && (projectAfter.Status == "done" || projectAfter.Status == "completed") {
			redirectWithCelebration(w, r, "/roadmap", celebration{Kind: "project", Title: projectAfter.Name, XP: gained, Level: level})
			return
		}
		// The Undo route lives under /projects; the toast's return_to brings the
		// user back to /roadmap.
		redirectWithCelebration(w, r, "/roadmap", celebration{
			Kind: "task", Title: itemTitle, XP: gained, Level: level,
			Undo: fmt.Sprintf("/projects/%d/items/%d/undo", projectID, itemID),
		})
		return
	}
	redirectAfter(w, r, "/roadmap")
}

func (h *Handler) roadmapMilestones(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if len(segments) != 4 {
		http.NotFound(w, r)
		return
	}
	milestoneID, ok := parseID(segments[2])
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch segments[3] {
	case "complete":
		if err := h.Service.Repo.SetRoadmapMilestoneStatus(milestoneID, "completed"); err != nil {
			serverError(w, err)
			return
		}
	case "reopen":
		if err := h.Service.Repo.SetRoadmapMilestoneStatus(milestoneID, "pending"); err != nil {
			serverError(w, err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	redirectAfter(w, r, "/roadmap")
}

func (h *Handler) trackers(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if len(segments) != 3 || segments[2] != "toggle" {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		serverError(w, err)
		return
	}
	logDate, err := h.formLocalDate(r.FormValue("log_date"))
	if err != nil {
		invalidDate(w, "log_date")
		return
	}
	next, err := h.Service.Repo.ToggleTrackerLogByKey(segments[1], logDate)
	if err != nil {
		serverError(w, err)
		return
	}

	if wantsJSON(r) {
		// The toggle write already succeeded, so status is authoritative and the
		// response must be 200 no matter what follows. The recomputed mission
		// scores are best-effort extras: if DailyMission() errors we still return
		// the new status so the client never reverts a persisted change.
		resp := trackerToggleResponse{Status: next}
		if mission, err := h.Service.DailyMission(); err == nil {
			resp.Prayer = &mission.PrayerCompletion
			resp.Athkar = &mission.AthkarCompletion
			resp.Deen = &mission.DeenCompletion
			resp.Gym = &mission.GymWeekCompletion
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			serverError(w, err)
		}
		return
	}
	redirectBack(w, r, "/")
}

type trackerToggleResponse struct {
	Status string `json:"status"`
	Prayer *int   `json:"prayer,omitempty"`
	Athkar *int   `json:"athkar,omitempty"`
	Deen   *int   `json:"deen,omitempty"`
	Gym    *int   `json:"gym,omitempty"`
}

func wantsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "fetch" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// appClock is the application clock; without a service (render tests) it is
// the default system clock.
func (h *Handler) appClock() clock.Clock {
	if h.Service == nil {
		return clock.System()
	}
	return h.Service.Clock
}

func (h *Handler) render(w http.ResponseWriter, page string, data any) {
	playerXP := models.PlayerStats{CurrentLevel: 1}
	if h.Service != nil && h.Service.Repo != nil {
		var err error
		playerXP, err = h.Service.Repo.PlayerStats()
		if err != nil {
			serverError(w, err)
			return
		}
	}
	files := []string{
		filepath.Join(h.TemplateDir, "layout.html"),
		filepath.Join(h.TemplateDir, "rocket_route.html"),
		filepath.Join(h.TemplateDir, "ui_icons.html"),
		filepath.Join(h.TemplateDir, "ui_components.html"),
		filepath.Join(h.TemplateDir, "ui_shell.html"),
		filepath.Join(h.TemplateDir, "today_components.html"),
		filepath.Join(h.TemplateDir, "project_components.html"),
		filepath.Join(h.TemplateDir, "health_components.html"),
		filepath.Join(h.TemplateDir, page),
	}
	tmpl, err := template.New("layout.html").Funcs(template.FuncMap{
		"primarySection": primarySection,
		"projectCard":    projectCard,
		"xpStats":        func() models.PlayerStats { return playerXP },
		"join":           strings.Join,
		"joinInts":       joinInts,
		"niceDate":       niceDate,
		"niceDateTime":   niceDateTime,
		"niceNumber":     niceNumber,
		"healthTab":      healthTab,
		"coachEnabled":   func() bool { return h.Coach != nil },
		// deref reads an optional number; templates guard with {{if}} first.
		"deref": func(v *float64) float64 {
			if v == nil {
				return 0
			}
			return *v
		},
		"greeting": func() string {
			return greeting(h.appClock().Now().In(h.appClock().Location()).Hour())
		},
		// Quick capture's Today/Tomorrow chips use the server's local day. With
		// no service (render tests) they are empty and app.js falls back to the
		// browser date.
		"appToday": func() string {
			if h.Service == nil {
				return ""
			}
			return h.Service.TodayString()
		},
		"appTomorrow": func() string {
			if h.Service == nil {
				return ""
			}
			return h.Service.Today().AddDate(0, 0, 1).Format(clock.DateLayout)
		},
		"nodeState":   nodeState,
		"levelPct":    levelPct,
		"toNextLevel": toNextLevel,
		"blockLabel":  services.BlockTypeLabel,
		"routeDone": func(nodes []models.RouteNode) int {
			done := 0
			for _, node := range nodes {
				if node.State == "done" {
					done++
				}
			}
			return done
		},
		"routeNext": func(nodes []models.RouteNode) string {
			for _, node := range nodes {
				if node.State != "done" && !node.Current {
					return node.Name
				}
			}
			return ""
		},
		"progressWidth": func(value int) template.CSS {
			if value < 0 {
				value = 0
			}
			if value > 100 {
				value = 100
			}
			return template.CSS(strconv.Itoa(value) + "%")
		},
	}).ParseFiles(files...)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		serverError(w, err)
	}
}

func taskFromRequest(r *http.Request) (models.Task, error) {
	if err := r.ParseForm(); err != nil {
		return models.Task{}, err
	}
	priority := cleanForm(r.FormValue("priority"), "medium")
	return models.Task{
		Title:         strings.TrimSpace(r.FormValue("title")),
		Description:   strings.TrimSpace(r.FormValue("description")),
		Category:      cleanForm(r.FormValue("category"), "personal"),
		Priority:      priority,
		Status:        cleanForm(r.FormValue("status"), "pending"),
		DueDate:       strings.TrimSpace(r.FormValue("due_date")),
		RepeatType:    cleanForm(r.FormValue("repeat_type"), "none"),
		ReminderTime:  strings.TrimSpace(r.FormValue("reminder_time")),
		StarsReward:   parseIntForm(r.FormValue("stars_reward"), xp.TaskReward(priority)),
		ProofRequired: r.FormValue("proof_required") == "on",
		ProofType:     cleanForm(r.FormValue("proof_type"), "none"),
		ProofURL:      strings.TrimSpace(r.FormValue("proof_url")),
		ProofPath:     strings.TrimSpace(r.FormValue("proof_path")),
	}, nil
}

func projectFromRequest(r *http.Request) (models.Project, error) {
	if err := r.ParseForm(); err != nil {
		return models.Project{}, err
	}
	return models.Project{
		ParentProjectID:    parseInt64Form(r.FormValue("parent_project_id")),
		Name:               strings.TrimSpace(r.FormValue("name")),
		Description:        strings.TrimSpace(r.FormValue("description")),
		Category:           cleanForm(r.FormValue("category"), "other"),
		Color:              cleanForm(r.FormValue("color"), "cyan"),
		ProjectKind:        cleanForm(r.FormValue("project_kind"), "main"),
		Status:             cleanForm(r.FormValue("status"), "active"),
		StartDate:          strings.TrimSpace(r.FormValue("start_date")),
		TargetDate:         strings.TrimSpace(r.FormValue("target_date")),
		Notes:              strings.TrimSpace(r.FormValue("notes")),
		UnlockNotes:        strings.TrimSpace(r.FormValue("unlock_notes")),
		DependsOnProjectID: parseInt64Form(r.FormValue("depends_on_project_id")),
		AutoUnlock:         r.FormValue("auto_unlock") == "on",
		RoadmapID:          parseInt64Form(r.FormValue("roadmap_id")),
	}, nil
}

func projectItemFromRequest(r *http.Request) (models.ProjectItem, error) {
	if err := r.ParseForm(); err != nil {
		return models.ProjectItem{}, err
	}
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	priority := cleanForm(r.FormValue("priority"), "medium")
	return models.ProjectItem{
		Title:         strings.TrimSpace(r.FormValue("title")),
		Description:   strings.TrimSpace(r.FormValue("description")),
		Priority:      priority,
		DueDate:       strings.TrimSpace(r.FormValue("due_date")),
		SortOrder:     sortOrder,
		StarsReward:   parseIntForm(r.FormValue("stars_reward"), xp.MissionReward(priority)),
		ProofRequired: r.FormValue("proof_required") == "on",
		ProofType:     cleanForm(r.FormValue("proof_type"), "none"),
		ProofURL:      strings.TrimSpace(r.FormValue("proof_url")),
		ProofPath:     strings.TrimSpace(r.FormValue("proof_path")),
	}, nil
}

func roadmapMetaFromRequest(r *http.Request, projectID int64) (models.RoadmapProjectMeta, error) {
	if err := r.ParseForm(); err != nil {
		return models.RoadmapProjectMeta{}, err
	}
	localOnly := r.FormValue("local_only") == "on"
	return models.RoadmapProjectMeta{
		ProjectID:     projectID,
		ProjectType:   cleanForm(r.FormValue("project_type"), "learning"),
		TagsText:      strings.TrimSpace(r.FormValue("tags")),
		IsPrivate:     r.FormValue("is_private") == "on" || localOnly,
		LocalOnly:     localOnly,
		CurrentFocus:  r.FormValue("current_focus") == "on",
		PublicSummary: strings.TrimSpace(r.FormValue("public_summary")),
	}, nil
}

func goalFromRequest(r *http.Request) (models.Goal, error) {
	if err := r.ParseForm(); err != nil {
		return models.Goal{}, err
	}
	target, _ := strconv.ParseFloat(r.FormValue("target_value"), 64)
	current, _ := strconv.ParseFloat(r.FormValue("current_value"), 64)
	return models.Goal{
		Title:        strings.TrimSpace(r.FormValue("title")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Type:         cleanForm(r.FormValue("type"), "yearly"),
		Category:     cleanForm(r.FormValue("category"), "personal"),
		TargetValue:  target,
		CurrentValue: current,
		Unit:         cleanForm(r.FormValue("unit"), "tasks"),
		StartDate:    strings.TrimSpace(r.FormValue("start_date")),
		EndDate:      strings.TrimSpace(r.FormValue("end_date")),
		Status:       cleanForm(r.FormValue("status"), "active"),
	}, nil
}

func parseID(value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}

func segmentAt(segments []string, index int) string {
	if index < 0 || index >= len(segments) {
		return ""
	}
	return segments[index]
}

// formLocalDate validates a browser-supplied calendar day (GYST-VN-003). An
// empty value becomes the application's local Today; anything else must be a
// canonical, valid YYYY-MM-DD date.
func (h *Handler) formLocalDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return h.Service.TodayString(), nil
	}
	return clock.ParseLocalDate(value)
}

// invalidDate answers a malformed date with 400; nothing has been written.
func invalidDate(w http.ResponseWriter, field string) {
	http.Error(w, field+" must be a valid YYYY-MM-DD date", http.StatusBadRequest)
}

// validDueDate reports whether a task due date is empty or a canonical date.
func validDueDate(value string) bool {
	if value == "" {
		return true
	}
	_, err := clock.ParseLocalDate(value)
	return err == nil
}

func cleanForm(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func parseIntForm(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64Form(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func joinInts(values []int, sep string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, sep)
}

func niceDate(value string) string {
	if value == "" {
		return "No date"
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("Jan 2, 2006")
}

func niceDateTime(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", value)
	if err != nil {
		return value
	}
	return parsed.Format("Jan 2, 3:04 PM")
}

func niceNumber(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.0001 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}

// nodeState maps a project/mission status + progress into a campaign-map state
// token used purely for visual styling: done, active, blocked, locked, pending.
func nodeState(status string, progress int) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "done", "archived":
		return "done"
	case "blocked":
		return "blocked"
	case "locked", "upcoming":
		return "locked"
	case "paused":
		return "blocked"
	case "active", "pending", "in_progress":
		if progress >= 100 {
			return "done"
		}
		return "active"
	default:
		if progress >= 100 {
			return "done"
		}
		return "pending"
	}
}

// levelPct returns how far into the current level the player is (0-99),
// where every 100 stars is one level. Used for the captain XP bar width.
func levelPct(stars int) int {
	if stars < 0 {
		return 0
	}
	return stars % 100
}

// toNextLevel returns stars remaining until the next level threshold.
func toNextLevel(stars int) int {
	if stars < 0 {
		return 100
	}
	return 100 - (stars % 100)
}

// greetingFor is the Today page greeting for a local hour.
func greetingFor(hour int) string {
	switch {
	case hour >= 18:
		return "Good evening"
	case hour >= 12:
		return "Good afternoon"
	default:
		return "Good morning"
	}
}

// greeting is the dashboard template's greeting for a local hour.
func greeting(hour int) string {
	switch {
	case hour < 12:
		return "Good Morning, Jay"
	case hour < 17:
		return "Good Afternoon, Jay"
	default:
		return "Good Evening, Jay"
	}
}

// redirectBack sends the user back to the page they came from, using only the
// path, query and fragment of a same-host Referer (see sameHostReferer). A
// missing, cross-host or non-local Referer goes to fallback instead.
func redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	http.Redirect(w, r, sameHostReferer(r, fallback), http.StatusSeeOther)
}

// safeReturnTo returns the form's return_to value only when it is a same-origin
// path (starts with a single "/"), preventing open-redirects via protocol-
// relative ("//host") or absolute URLs. Returns "" when absent or unsafe.
func safeReturnTo(r *http.Request) string {
	rt := r.FormValue("return_to")
	if !isLocalPath(rt) {
		return ""
	}
	return rt
}

// redirectAfter sends the user back to the exact page+section they acted from.
// It prefers an explicit return_to (carries a #fragment so the browser lands on
// the item), then a same-host Referer, then the fallback — so the no-JS POST never
// dead-ends at the top of a generic list.
func redirectAfter(w http.ResponseWriter, r *http.Request, fallback string) {
	if rt := safeReturnTo(r); rt != "" {
		http.Redirect(w, r, rt, http.StatusSeeOther)
		return
	}
	redirectBack(w, r, fallback)
}

// celebration is what the completion toast is built from after a redirect.
// Kind is "task" or "project"; Undo is the same-origin POST path that reverses
// this completion (empty when no safe reversal exists); XP is the real stars
// gained and Level is the new level when this completion crossed one.
type celebration struct {
	Kind, Title, Undo, Date string
	XP, Level               int
}

const celebrationTitleRunes = 80

func redirectWithCelebration(w http.ResponseWriter, r *http.Request, fallback string, c celebration) {
	destination := safeReturnTo(r)
	if destination == "" {
		destination = fallback
	}
	parsed, err := url.Parse(destination)
	if err != nil {
		parsed, _ = url.Parse(fallback)
	}
	title := []rune(c.Title)
	if len(title) > celebrationTitleRunes {
		title = title[:celebrationTitleRunes]
	}
	query := parsed.Query()
	for _, key := range []string{"celebrate", "celebrate_title", "celebrate_xp", "celebrate_level", "celebrate_undo", "celebrate_date"} {
		query.Del(key)
	}
	query.Set("celebrate", c.Kind)
	query.Set("celebrate_title", string(title))
	query.Set("celebrate_xp", strconv.Itoa(c.XP))
	if c.Level > 0 {
		query.Set("celebrate_level", strconv.Itoa(c.Level))
	}
	if c.Undo != "" {
		query.Set("celebrate_undo", c.Undo)
	}
	if c.Date != "" {
		query.Set("celebrate_date", c.Date)
	}
	parsed.RawQuery = query.Encode()
	http.Redirect(w, r, parsed.String(), http.StatusSeeOther)
}

// statsSnapshot reads the player's totals before a write so the toast can show
// the stars the write really added. ok is false when the read fails; the write
// is still honoured and the toast simply shows no XP.
func (h *Handler) statsSnapshot() (models.PlayerStats, bool) {
	stats, err := h.Service.Repo.PlayerStats()
	return stats, err == nil
}

// rewardDelta returns the stars gained (never negative) and, when the level
// went up, the new level.
func (h *Handler) rewardDelta(before models.PlayerStats, ok bool) (xpGained, level int) {
	if !ok {
		return 0, 0
	}
	after, ok := h.statsSnapshot()
	if !ok {
		return 0, 0
	}
	xpGained = max(0, after.TotalStars-before.TotalStars)
	if after.CurrentLevel > before.CurrentLevel {
		level = after.CurrentLevel
	}
	return xpGained, level
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func serverError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func notFoundOrError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	serverError(w, err)
}
