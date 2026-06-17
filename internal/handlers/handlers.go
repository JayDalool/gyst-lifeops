package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

type Handler struct {
	Service     *services.Service
	TemplateDir string
}

type TodayData struct {
	Page           string
	Today          string
	Progress       int
	Mission        models.DailyMissionData
	Pending        []models.TaskOccurrence
	Completed      []models.TaskOccurrence
	Skipped        []models.TaskOccurrence
	OverdueTasks   []models.Task
	RemindersToday []models.ReminderView
}

type TaskListData struct {
	Page  string
	Today string
	Tasks []models.Task
}

type TaskFormData struct {
	Page   string
	Today  string
	Title  string
	Action string
	Task   models.Task
}

type TaskDetailData struct {
	Page string
	Task models.Task
}

type MonthlyData struct {
	Page      string
	MonthName string
	Progress  int
	Chart     models.ChartData
	Items     []models.TaskOccurrence
}

type ProjectListData struct {
	Page     string
	Projects []models.ProjectWithProgress
}

type ProjectFormData struct {
	Page    string
	Today   string
	Title   string
	Action  string
	Project models.Project
}

type ProjectDetailData struct {
	Page    string
	Project models.ProjectWithProgress
	Items   []models.ProjectItem
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
}

type ArchiveData struct {
	Page     string
	Tasks    []models.Task
	Projects []models.ProjectWithProgress
	Goals    []models.GoalWithProgress
}

type SettingsData struct {
	Page string
}

func New(service *services.Service, templateDir string) *Handler {
	return &Handler{Service: service, TemplateDir: templateDir}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		h.dashboard(w, r)
		return
	}

	segments := strings.Split(path, "/")
	switch segments[0] {
	case "today":
		h.today(w, r)
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
	case "health":
		h.health(w, r)
	case "archive":
		h.archive(w, r)
	case "settings":
		h.settings(w, r)
	case "trackers":
		h.trackers(w, r, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	data, err := h.Service.Dashboard()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "dashboard.html", data)
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
	h.render(w, "today.html", TodayData{
		Page:           "today",
		Today:          h.Service.TodayString(),
		Progress:       services.CompletionPercentage(items),
		Mission:        mission,
		Pending:        pending,
		Completed:      completed,
		Skipped:        skipped,
		OverdueTasks:   overdueTasks,
		RemindersToday: remindersToday,
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
		h.render(w, "tasks_list.html", TaskListData{Page: "tasks", Today: h.Service.TodayString(), Tasks: tasks})
		return
	}

	if len(segments) == 2 && segments[1] == "new" {
		h.newTask(w, r)
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
		h.render(w, "task_detail.html", TaskDetailData{Page: "tasks", Task: task})
		return
	}

	switch segments[2] {
	case "edit":
		h.editTask(w, r, id)
	case "complete":
		h.setTaskStatus(w, r, id, "completed")
	case "skip":
		h.setTaskStatus(w, r, id, "skipped")
	case "archive":
		h.archiveTask(w, r, id)
	default:
		http.NotFound(w, r)
	}
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
				Category:   "personal",
				Priority:   "medium",
				Status:     "pending",
				DueDate:    h.Service.TodayString(),
				RepeatType: "none",
			},
		})
	case http.MethodPost:
		task, err := taskFromRequest(r)
		if err != nil {
			serverError(w, err)
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
	date := r.FormValue("occurrence_date")
	if date == "" {
		date = h.Service.TodayString()
	}
	if err := h.Service.Repo.SetTaskOccurrenceStatus(id, date, status); err != nil {
		serverError(w, err)
		return
	}
	redirectBack(w, r, "/today")
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
		projects, err := h.Service.Repo.ListProjects(false)
		if err != nil {
			serverError(w, err)
			return
		}
		for index := range projects {
			projects[index].Momentum = services.TrendChart(projects[index].Progress)
		}
		h.render(w, "projects_list.html", ProjectListData{Page: "projects", Projects: projects})
		return
	}

	if len(segments) == 2 && segments[1] == "new" {
		h.newProject(w, r)
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
	case "items":
		h.projectItems(w, r, id, segments)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) newProject(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.render(w, "project_form.html", ProjectFormData{
			Page:   "projects",
			Today:  h.Service.TodayString(),
			Title:  "New Project",
			Action: "/projects/new",
			Project: models.Project{
				Category:  "other",
				Status:    "active",
				StartDate: h.Service.TodayString(),
			},
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
	h.render(w, "project_detail.html", ProjectDetailData{
		Page:    "projects",
		Project: project,
		Items:   items,
	})
}

func (h *Handler) editProject(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		project, err := h.Service.Repo.ProjectByID(id)
		if err != nil {
			notFoundOrError(w, err)
			return
		}
		h.render(w, "project_form.html", ProjectFormData{
			Page:    "projects",
			Today:   h.Service.TodayString(),
			Title:   "Edit Project",
			Action:  fmt.Sprintf("/projects/%d/edit", id),
			Project: project,
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

	switch segments[4] {
	case "complete":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "completed"); err != nil {
			serverError(w, err)
			return
		}
	case "reopen":
		if err := h.Service.Repo.SetProjectItemStatus(projectID, itemID, "pending"); err != nil {
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
	http.Redirect(w, r, fmt.Sprintf("/projects/%d", projectID), http.StatusSeeOther)
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
		if err := h.Service.Repo.CreateGoal(&goal); err != nil {
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
	entryDate := cleanForm(r.FormValue("entry_date"), h.Service.TodayString())
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
	goals, err := h.Service.Repo.ListGoals(false)
	if err != nil {
		serverError(w, err)
		return
	}
	tasks, err := h.Service.Repo.ListTasks(false)
	if err != nil {
		serverError(w, err)
		return
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
		serverError(w, err)
		return
	}
	bodyRhythm, err := h.Service.BodyRhythmChart()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, "health.html", HealthData{Page: "health", HealthGoals: healthGoals, HealthTasks: healthTasks, Mission: mission, BodyRhythm: bodyRhythm})
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

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	h.render(w, "settings.html", SettingsData{Page: "settings"})
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
	logDate := cleanForm(r.FormValue("log_date"), h.Service.TodayString())
	next, err := h.Service.Repo.ToggleTrackerLogByKey(segments[1], logDate)
	if err != nil {
		serverError(w, err)
		return
	}
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, `{"status":%q}`, next)
		return
	}
	redirectBack(w, r, "/")
}

func wantsJSON(r *http.Request) bool {
	if r.Header.Get("X-Requested-With") == "fetch" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func (h *Handler) render(w http.ResponseWriter, page string, data any) {
	files := []string{
		filepath.Join(h.TemplateDir, "layout.html"),
		filepath.Join(h.TemplateDir, page),
	}
	tmpl, err := template.New("layout.html").Funcs(template.FuncMap{
		"join":         strings.Join,
		"joinInts":     joinInts,
		"niceDate":     niceDate,
		"niceDateTime": niceDateTime,
		"niceNumber":   niceNumber,
		"greeting":     greeting,
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
	return models.Task{
		Title:        strings.TrimSpace(r.FormValue("title")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Category:     cleanForm(r.FormValue("category"), "personal"),
		Priority:     cleanForm(r.FormValue("priority"), "medium"),
		Status:       cleanForm(r.FormValue("status"), "pending"),
		DueDate:      strings.TrimSpace(r.FormValue("due_date")),
		RepeatType:   cleanForm(r.FormValue("repeat_type"), "none"),
		ReminderTime: strings.TrimSpace(r.FormValue("reminder_time")),
	}, nil
}

func projectFromRequest(r *http.Request) (models.Project, error) {
	if err := r.ParseForm(); err != nil {
		return models.Project{}, err
	}
	return models.Project{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Category:    cleanForm(r.FormValue("category"), "other"),
		Status:      cleanForm(r.FormValue("status"), "active"),
		StartDate:   strings.TrimSpace(r.FormValue("start_date")),
		TargetDate:  strings.TrimSpace(r.FormValue("target_date")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}, nil
}

func projectItemFromRequest(r *http.Request) (models.ProjectItem, error) {
	if err := r.ParseForm(); err != nil {
		return models.ProjectItem{}, err
	}
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	return models.ProjectItem{
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Priority:    cleanForm(r.FormValue("priority"), "medium"),
		DueDate:     strings.TrimSpace(r.FormValue("due_date")),
		SortOrder:   sortOrder,
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

func cleanForm(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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

func greeting() string {
	hour := time.Now().Hour()
	switch {
	case hour < 12:
		return "Good Morning, Jay"
	case hour < 17:
		return "Good Afternoon, Jay"
	default:
		return "Good Evening, Jay"
	}
}

func redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	back := r.Header.Get("Referer")
	if back == "" {
		back = fallback
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
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
