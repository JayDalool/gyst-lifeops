package services

import (
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"time"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

// Service embeds the application clock: s.Now() is the instant source and
// s.Today(), s.TodayString(), s.DateOf() and s.Location() give the local
// calendar day in the application timezone (GYST-VN-003).
type Service struct {
	clock.Clock
	Training *TrainingSync
	Repo     *repository.Repository
	// AttentionProducers feed Attention(); New fills in the GYST sources.
	AttentionProducers []AttentionProducer
	// BackupStatusPath is the backup status file (data/backups/last_run.json).
	// Empty means unconfigured: the backup producer reports nothing.
	BackupStatusPath string

	// calendarAfterReads is a test seam run after the calendar's project data
	// reads and before the privacy lookup. Nil in production.
	calendarAfterReads func()
}

type DailyStat struct {
	Date      string
	Label     string
	Total     int
	Completed int
	Percent   int
}

func New(repo *repository.Repository) *Service {
	s := &Service{
		Clock: clock.System(),
		Repo:  repo,
	}
	s.AttentionProducers = s.gystAttentionProducers()
	return s
}

func (s *Service) Dashboard() (models.DashboardData, error) {
	today := s.Today()
	trackerItems, err := s.Repo.ListTrackerItems()
	if err != nil {
		return models.DashboardData{}, err
	}
	trackerStart := today.AddDate(0, 0, -29)
	trackerLogs, err := s.Repo.TrackerLogsBetween(formatDate(trackerStart), formatDate(today))
	if err != nil {
		return models.DashboardData{}, err
	}
	dailyMission := buildDailyMission(today, trackerItems, trackerLogs)
	deenConsistency := trackerPercentChart(trackerItems, trackerLogs, trackerStart, today, map[string]bool{"prayer": true, "athkar": true})
	bodyRhythm := gymRhythmChart(trackerItems, trackerLogs, today.AddDate(0, 0, -13), today)

	todayItems, err := s.TaskOccurrencesOn(today)
	if err != nil {
		return models.DashboardData{}, err
	}

	overdueTasks, err := s.Repo.ListActiveOneTimeOverdueTasks(formatDate(today))
	if err != nil {
		return models.DashboardData{}, err
	}

	projects, err := s.Repo.ListProjects(false)
	if err != nil {
		return models.DashboardData{}, err
	}
	activeProjects := make([]models.ProjectWithProgress, 0, len(projects))
	upcomingProjects := make([]models.ProjectWithProgress, 0)
	projectChart := models.ChartData{}
	for _, project := range projects {
		project.Momentum = trendChart(project.Progress)
		if project.Project.Status == "active" {
			activeProjects = append(activeProjects, project)
		}
		if project.Project.Status == "upcoming" || project.Project.Status == "locked" {
			upcomingProjects = append(upcomingProjects, project)
		}
		if project.TotalItems > 0 {
			projectChart.Labels = append(projectChart.Labels, project.Project.Name)
			projectChart.Values = append(projectChart.Values, project.Progress)
		}
	}
	projectMomentumURL := "/projects"
	if len(activeProjects) > 0 {
		projectMomentumURL = "/projects/" + strconv.FormatInt(activeProjects[0].Project.ID, 10)
	}
	projectMomentum := projectMomentumChart(activeProjects)

	goals, err := s.Repo.ListGoals(false)
	if err != nil {
		return models.DashboardData{}, err
	}
	yearlyGoals := make([]models.GoalWithProgress, 0, len(goals))
	yearlyGoalChart := models.ChartData{}
	for _, goal := range goals {
		goal.Momentum = trendChart(goal.Progress)
		if goal.Goal.Type != "yearly" {
			continue
		}
		yearlyGoals = append(yearlyGoals, goal)
		yearlyGoalChart.Labels = append(yearlyGoalChart.Labels, goal.Goal.Title)
		yearlyGoalChart.Values = append(yearlyGoalChart.Values, goal.Progress)
	}
	goalSignal := goalProgressSparkline(yearlyGoals)

	dailyStats, err := s.DailyCompletionStats(today.AddDate(0, 0, -29), today)
	if err != nil {
		return models.DashboardData{}, err
	}
	dailyChart := models.ChartData{}
	for _, stat := range dailyStats {
		dailyChart.Labels = append(dailyChart.Labels, stat.Label)
		dailyChart.Values = append(dailyChart.Values, stat.Percent)
	}
	lastSevenChart := tailChart(dailyChart, 7)

	monthlyChart, err := s.MonthlyCompletionChart(today.Year(), today.Month())
	if err != nil {
		return models.DashboardData{}, err
	}
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)
	monthItems, err := s.TaskOccurrencesBetween(monthStart, monthEnd)
	if err != nil {
		return models.DashboardData{}, err
	}
	monthStats, err := s.DailyCompletionStats(monthStart, today)
	if err != nil {
		return models.DashboardData{}, err
	}
	monthlyScoreChart := models.ChartData{}
	for _, stat := range monthStats {
		monthlyScoreChart.Labels = append(monthlyScoreChart.Labels, stat.Label)
		monthlyScoreChart.Values = append(monthlyScoreChart.Values, stat.Percent)
	}

	remindersToday, overdueReminders, err := s.ReminderViews(today)
	if err != nil {
		return models.DashboardData{}, err
	}
	nextReminder := models.ReminderView{}
	if len(remindersToday) > 0 {
		nextReminder = remindersToday[0]
	} else if len(overdueReminders) > 0 {
		nextReminder = overdueReminders[0]
	}
	reminderLoad := models.ChartData{
		Labels: []string{"Today", "Overdue"},
		Values: []int{len(remindersToday), len(overdueReminders)},
	}
	player, err := s.Repo.PlayerStats()
	if err != nil {
		return models.DashboardData{}, err
	}
	player.ActiveStreak = streakFromStats(dailyStats)
	achievements, err := s.Repo.RecentAchievements(5)
	if err != nil {
		return models.DashboardData{}, err
	}
	shipStats, err := s.ShipStats()
	if err != nil {
		return models.DashboardData{}, err
	}

	// Scope the bridge hero / Active Trips / Planned Routes to the featured
	// roadmap. Charts above stay global; only the journey view is scoped.
	heroActive := activeProjects
	heroUpcoming := upcomingProjects
	dashboardRoadmap, drErr := s.Repo.GetDashboardRoadmap()
	hasRoadmap := drErr == nil
	if drErr != nil && !errors.Is(drErr, sql.ErrNoRows) {
		return models.DashboardData{}, drErr
	}
	if hasRoadmap {
		// Self-heal: adopt any projects not yet bound to a roadmap (seed rows,
		// projects created via the form) into the featured roadmap so the bridge
		// hero is populated.
		if err := s.Repo.AdoptOrphanProjects(dashboardRoadmap.ID); err != nil {
			return models.DashboardData{}, err
		}
		tree, err := s.Repo.ProjectTreeByRoadmap(dashboardRoadmap.ID)
		if err != nil {
			return models.DashboardData{}, err
		}
		heroActive = make([]models.ProjectWithProgress, 0, len(tree.Main))
		for _, project := range tree.Main {
			if project.Project.Status == "active" {
				project.Momentum = trendChart(project.Progress)
				heroActive = append(heroActive, project)
			}
		}
		heroUpcoming = tree.Upcoming
	}
	roadmaps, err := s.Repo.ListRoadmaps(false)
	if err != nil {
		return models.DashboardData{}, err
	}

	completedToday, skippedToday := 0, 0
	for _, occ := range todayItems {
		switch occ.Status {
		case "completed":
			completedToday++
		case "skipped":
			skippedToday++
		}
	}
	activity := models.ActivitySummary{
		Reminders:      len(remindersToday),
		Overdue:        len(overdueTasks),
		CompletedToday: completedToday,
		Skipped:        skippedToday,
	}

	// Galaxy route: top-level destinations only (parent_project_id IS NULL).
	// Subprojects live inside their parent's solar system, never on this route.
	routeTitle := "Ship Bridge"
	if hasRoadmap {
		routeTitle = dashboardRoadmap.Name
	}
	var shipRoute models.RocketRoute
	if hasRoadmap {
		shipRoute, err = s.RocketRouteForRoadmap(dashboardRoadmap.ID, routeTitle)
		if err != nil {
			return models.DashboardData{}, err
		}
	} else {
		nodes := make([]models.RouteNode, 0, len(heroActive))
		for _, project := range heroActive {
			if project.Project.ParentProjectID == 0 {
				nodes = append(nodes, projectNode(project))
			}
		}
		shipRoute = BuildRocketRoute("galaxy", routeTitle, "galaxy", nodes)
	}

	return models.DashboardData{
		Page:                     "dashboard",
		ShipStats:                shipStats,
		ShipRoute:                shipRoute,
		DashboardRoadmap:         dashboardRoadmap,
		HasRoadmap:               hasRoadmap,
		Roadmaps:                 roadmaps,
		Activity:                 activity,
		Today:                    formatDate(today),
		TodayItems:               todayItems,
		OverdueTasks:             overdueTasks,
		TodayCompletion:          CompletionPercentage(todayItems),
		MonthlyCompletionPercent: CompletionPercentage(monthItems),
		CurrentStreak:            player.ActiveStreak,
		Player:                   player,
		ActiveProjects:           heroActive,
		UpcomingProjects:         heroUpcoming,
		RecentAchievements:       achievements,
		YearlyGoals:              yearlyGoals,
		RemindersToday:           remindersToday,
		OverdueReminders:         overdueReminders,
		NextReminder:             nextReminder,
		DailyMission:             dailyMission,
		DailyCompletion:          dailyChart,
		LastSevenCompletion:      lastSevenChart,
		DeenConsistency:          tailChart(deenConsistency, 7),
		BodyRhythm:               bodyRhythm,
		MonthlyCompletion:        monthlyChart,
		MonthlyScore:             monthlyScoreChart,
		TasksCompletedMini:       tailChart(monthlyChart, 7),
		ProjectProgress:          projectChart,
		ProjectMomentum:          projectMomentum,
		ProjectMomentumURL:       projectMomentumURL,
		YearlyGoalProgress:       yearlyGoalChart,
		GoalProgressSignal:       goalSignal,
		ReminderLoad:             reminderLoad,
	}, nil
}

func (s *Service) DailyMission() (models.DailyMissionData, error) {
	today := s.Today()
	trackerItems, err := s.Repo.ListTrackerItems()
	if err != nil {
		return models.DailyMissionData{}, err
	}
	logs, err := s.Repo.TrackerLogsBetween(formatDate(today.AddDate(0, 0, -6)), formatDate(today))
	if err != nil {
		return models.DailyMissionData{}, err
	}
	return buildDailyMission(today, trackerItems, logs), nil
}

func (s *Service) BodyRhythmChart() (models.ChartData, error) {
	today := s.Today()
	trackerItems, err := s.Repo.ListTrackerItems()
	if err != nil {
		return models.ChartData{}, err
	}
	start := today.AddDate(0, 0, -13)
	logs, err := s.Repo.TrackerLogsBetween(formatDate(start), formatDate(today))
	if err != nil {
		return models.ChartData{}, err
	}
	return gymRhythmChart(trackerItems, logs, start, today), nil
}

func TrendChart(current int) models.ChartData {
	return trendChart(current)
}

func (s *Service) TaskOccurrencesOn(date time.Time) ([]models.TaskOccurrence, error) {
	occurrences, err := s.TaskOccurrencesBetween(date, date)
	if err != nil {
		return nil, err
	}
	return occurrences, nil
}

func (s *Service) TaskOccurrencesBetween(start, end time.Time) ([]models.TaskOccurrence, error) {
	tasks, err := s.Repo.ListTasks(false)
	if err != nil {
		return nil, err
	}

	completions, err := s.Repo.TaskCompletionsBetween(formatDate(start), formatDate(end))
	if err != nil {
		return nil, err
	}
	completionByKey := make(map[string]models.TaskCompletion, len(completions))
	for _, completion := range completions {
		completionByKey[completionKey(completion.TaskID, completion.OccurrenceDate)] = completion
	}

	var occurrences []models.TaskOccurrence
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateValue := formatDate(date)
		for _, task := range tasks {
			if !taskDueOn(task, date) {
				continue
			}

			status := "pending"
			completedAt := ""
			if task.RepeatType == "none" && task.Status != "" {
				status = task.Status
				completedAt = task.CompletedAt
			}
			if completion, ok := completionByKey[completionKey(task.ID, dateValue)]; ok {
				status = completion.Status
				completedAt = completion.CompletedAt
			}

			occurrences = append(occurrences, models.TaskOccurrence{
				Task:          task,
				Date:          dateValue,
				Status:        status,
				CompletedAt:   completedAt,
				OccurrenceKey: completionKey(task.ID, dateValue),
			})
		}
	}

	sort.SliceStable(occurrences, func(i, j int) bool {
		if occurrences[i].Date != occurrences[j].Date {
			return occurrences[i].Date < occurrences[j].Date
		}
		left := priorityRank(occurrences[i].Task.Priority)
		right := priorityRank(occurrences[j].Task.Priority)
		if left != right {
			return left < right
		}
		return occurrences[i].Task.Title < occurrences[j].Task.Title
	})

	return occurrences, nil
}

func (s *Service) DailyCompletionStats(start, end time.Time) ([]DailyStat, error) {
	occurrences, err := s.TaskOccurrencesBetween(start, end)
	if err != nil {
		return nil, err
	}

	byDate := map[string][]models.TaskOccurrence{}
	for _, occurrence := range occurrences {
		byDate[occurrence.Date] = append(byDate[occurrence.Date], occurrence)
	}

	var stats []DailyStat
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateValue := formatDate(date)
		items := byDate[dateValue]
		completed := 0
		for _, item := range items {
			if item.Status == "completed" {
				completed++
			}
		}
		stats = append(stats, DailyStat{
			Date:      dateValue,
			Label:     date.Format("Jan 2"),
			Total:     len(items),
			Completed: completed,
			Percent:   CompletionPercentage(items),
		})
	}

	return stats, nil
}

func (s *Service) MonthlyCompletionChart(year int, month time.Month) (models.ChartData, error) {
	start := time.Date(year, month, 1, 0, 0, 0, 0, s.Today().Location())
	end := start.AddDate(0, 1, -1)
	occurrences, err := s.TaskOccurrencesBetween(start, end)
	if err != nil {
		return models.ChartData{}, err
	}

	completedByDate := map[string]int{}
	for _, occurrence := range occurrences {
		if occurrence.Status == "completed" {
			completedByDate[occurrence.Date]++
		}
	}

	chart := models.ChartData{}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateValue := formatDate(date)
		chart.Labels = append(chart.Labels, date.Format("2"))
		chart.Values = append(chart.Values, completedByDate[dateValue])
	}
	return chart, nil
}

func (s *Service) ReminderViews(today time.Time) ([]models.ReminderView, []models.ReminderView, error) {
	todayValue := formatDate(today)
	todayItems, err := s.TaskOccurrencesOn(today)
	if err != nil {
		return nil, nil, err
	}

	var dueToday []models.ReminderView
	for _, item := range todayItems {
		if item.Task.ReminderTime == "" || item.Status != "pending" {
			continue
		}
		dueToday = append(dueToday, models.ReminderView{
			SourceType:   "task",
			SourceID:     item.Task.ID,
			Title:        item.Task.Title,
			ReminderDate: todayValue,
			ReminderTime: item.Task.ReminderTime,
		})
	}

	overdueTasks, err := s.Repo.ListActiveOneTimeOverdueTasks(todayValue)
	if err != nil {
		return nil, nil, err
	}
	var overdue []models.ReminderView
	for _, task := range overdueTasks {
		if task.ReminderTime == "" {
			continue
		}
		overdue = append(overdue, models.ReminderView{
			SourceType:   "task",
			SourceID:     task.ID,
			Title:        task.Title,
			ReminderDate: task.DueDate,
			ReminderTime: task.ReminderTime,
			Overdue:      true,
		})
	}

	standalone, err := s.Repo.ListStandaloneRemindersThrough(todayValue)
	if err != nil {
		return nil, nil, err
	}
	for _, reminder := range standalone {
		view := models.ReminderView{
			SourceType:   reminder.SourceType,
			SourceID:     reminder.SourceID,
			Title:        reminder.Title,
			ReminderDate: reminder.ReminderDate,
			ReminderTime: reminder.ReminderTime,
		}
		if reminder.ReminderDate == "" || reminder.ReminderDate == todayValue {
			dueToday = append(dueToday, view)
			continue
		}
		view.Overdue = true
		overdue = append(overdue, view)
	}

	sort.SliceStable(dueToday, func(i, j int) bool {
		return dueToday[i].ReminderTime < dueToday[j].ReminderTime
	})
	sort.SliceStable(overdue, func(i, j int) bool {
		if overdue[i].ReminderDate != overdue[j].ReminderDate {
			return overdue[i].ReminderDate < overdue[j].ReminderDate
		}
		return overdue[i].ReminderTime < overdue[j].ReminderTime
	})

	return dueToday, overdue, nil
}

func CompletionPercentage(items []models.TaskOccurrence) int {
	if len(items) == 0 {
		return 0
	}
	completed := 0
	for _, item := range items {
		if item.Status == "completed" {
			completed++
		}
	}
	return (completed * 100) / len(items)
}

func SplitOccurrences(items []models.TaskOccurrence) (pending, completed, skipped []models.TaskOccurrence) {
	for _, item := range items {
		switch item.Status {
		case "completed":
			completed = append(completed, item)
		case "skipped":
			skipped = append(skipped, item)
		default:
			pending = append(pending, item)
		}
	}
	return pending, completed, skipped
}

func buildDailyMission(today time.Time, items []models.TrackerItem, logs []models.TrackerLog) models.DailyMissionData {
	todayValue := formatDate(today)
	statusByItemDate := trackerStatusMap(logs)
	mission := models.DailyMissionData{Date: todayValue}

	var gymItem models.TrackerItem
	for _, item := range items {
		status := trackerStatus(item, todayValue, statusByItemDate)
		view := models.TrackerStatusView{
			Item:   item,
			Date:   todayValue,
			Status: status,
		}
		switch item.Type {
		case "prayer":
			mission.PrayerItems = append(mission.PrayerItems, view)
		case "athkar":
			mission.AthkarItems = append(mission.AthkarItems, view)
		case "gym":
			gymItem = item
		}
	}

	mission.PrayerCompletion = trackerCompletion(mission.PrayerItems)
	mission.AthkarCompletion = trackerCompletion(mission.AthkarItems)
	deenItems := append([]models.TrackerStatusView{}, mission.PrayerItems...)
	deenItems = append(deenItems, mission.AthkarItems...)
	mission.DeenCompletion = trackerCompletion(deenItems)

	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := today.AddDate(0, 0, -(weekday - 1))
	doneGymDays := 0
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		dateValue := formatDate(date)
		status := "rest"
		if gymItem.ID != 0 {
			status = trackerStatus(gymItem, dateValue, statusByItemDate)
		}
		if status == "done" {
			doneGymDays++
		}
		mission.GymWeek = append(mission.GymWeek, models.GymDayView{
			Label:  date.Format("Mon"),
			Date:   dateValue,
			Status: status,
		})
	}
	mission.GymWeekCompletion = (doneGymDays * 100) / 7

	return mission
}

func trackerPercentChart(items []models.TrackerItem, logs []models.TrackerLog, start, end time.Time, types map[string]bool) models.ChartData {
	statusByItemDate := trackerStatusMap(logs)
	chart := models.ChartData{}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateValue := formatDate(date)
		total := 0
		done := 0
		for _, item := range items {
			if !types[item.Type] {
				continue
			}
			total++
			if trackerStatus(item, dateValue, statusByItemDate) == "done" {
				done++
			}
		}
		chart.Labels = append(chart.Labels, date.Format("Jan 2"))
		if total == 0 {
			chart.Values = append(chart.Values, 0)
		} else {
			chart.Values = append(chart.Values, (done*100)/total)
		}
	}
	return chart
}

func gymRhythmChart(items []models.TrackerItem, logs []models.TrackerLog, start, end time.Time) models.ChartData {
	statusByItemDate := trackerStatusMap(logs)
	var gymItem models.TrackerItem
	for _, item := range items {
		if item.Type == "gym" {
			gymItem = item
			break
		}
	}

	chart := models.ChartData{}
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateValue := formatDate(date)
		status := "rest"
		if gymItem.ID != 0 {
			status = trackerStatus(gymItem, dateValue, statusByItemDate)
		}
		value := 35
		switch status {
		case "done":
			value = 100
		case "not_done":
			value = 0
		}
		chart.Labels = append(chart.Labels, date.Format("Mon"))
		chart.Values = append(chart.Values, value)
	}
	return chart
}

func projectMomentumChart(projects []models.ProjectWithProgress) models.ChartData {
	if len(projects) == 0 {
		return models.ChartData{}
	}
	return trendChart(projects[0].Progress)
}

func goalProgressSparkline(goals []models.GoalWithProgress) models.ChartData {
	if len(goals) == 0 {
		return models.ChartData{}
	}
	return trendChart(goals[0].Progress)
}

func trendChart(current int) models.ChartData {
	if current < 0 {
		current = 0
	}
	if current > 100 {
		current = 100
	}
	offsets := []int{24, 18, 13, 9, 5, 2, 0}
	chart := models.ChartData{
		Labels: []string{"D-6", "D-5", "D-4", "D-3", "D-2", "D-1", "Now"},
	}
	for _, offset := range offsets {
		value := current - offset
		if value < 0 {
			value = 0
		}
		chart.Values = append(chart.Values, value)
	}
	return chart
}

func trackerStatusMap(logs []models.TrackerLog) map[int64]map[string]string {
	statusByItemDate := make(map[int64]map[string]string)
	for _, log := range logs {
		if _, ok := statusByItemDate[log.TrackerItemID]; !ok {
			statusByItemDate[log.TrackerItemID] = make(map[string]string)
		}
		statusByItemDate[log.TrackerItemID][log.LogDate] = log.Status
	}
	return statusByItemDate
}

func trackerStatus(item models.TrackerItem, date string, statusByItemDate map[int64]map[string]string) string {
	if byDate, ok := statusByItemDate[item.ID]; ok {
		if status, ok := byDate[date]; ok {
			return status
		}
	}
	if item.Type == "gym" {
		return "rest"
	}
	return "not_done"
}

func trackerCompletion(items []models.TrackerStatusView) int {
	if len(items) == 0 {
		return 0
	}
	done := 0
	for _, item := range items {
		if item.Status == "done" {
			done++
		}
	}
	return (done * 100) / len(items)
}

// OverdueSince labels how long a task has been overdue: "since yesterday",
// the weekday for 2-6 days, "since Oct 2" for a week or more, and the year too
// when it is not this year. Days are counted from the calendar dates, not from
// durations, so DST days cannot shift the count.
func OverdueSince(due, today time.Time) string {
	d := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	days := int(t.Sub(d) / (24 * time.Hour))
	switch {
	case days <= 1:
		return "since yesterday"
	case days < 7:
		return "since " + d.Weekday().String()
	case d.Year() == t.Year():
		return "since " + d.Format("Jan 2")
	default:
		return "since " + d.Format("Jan 2, 2006")
	}
}

// OverdueTasks lists the overdue one-time tasks, oldest first, with labels.
func (s *Service) OverdueTasks() ([]models.OverdueTask, error) {
	today := s.Today()
	tasks, err := s.Repo.ListActiveOneTimeOverdueTasks(formatDate(today))
	if err != nil {
		return nil, err
	}
	out := make([]models.OverdueTask, 0, len(tasks))
	for _, task := range tasks {
		item := models.OverdueTask{Task: task, Since: "since earlier"}
		if due, ok := parseDate(task.DueDate, s.Location()); ok {
			item.Since = OverdueSince(due, today)
		}
		out = append(out, item)
	}
	return out, nil
}

// MoveTaskToToday moves an overdue task to the app-local today. The date comes
// only from the service clock.
func (s *Service) MoveTaskToToday(id int64) (bool, error) {
	return s.Repo.MoveOverdueTaskToDate(id, s.TodayString())
}

func formatDate(date time.Time) string {
	return date.Format("2006-01-02")
}

func tailChart(chart models.ChartData, count int) models.ChartData {
	if count <= 0 || len(chart.Values) <= count {
		return chart
	}
	start := len(chart.Values) - count
	return models.ChartData{
		Labels: append([]string(nil), chart.Labels[start:]...),
		Values: append([]int(nil), chart.Values[start:]...),
	}
}

func completionKey(taskID int64, date string) string {
	return strconv.FormatInt(taskID, 10) + "|" + date
}

// isRepeating reports whether a task repeats on a schedule.
func isRepeating(task models.Task) bool {
	switch task.RepeatType {
	case "daily", "weekly", "monthly", "yearly":
		return true
	}
	return false
}

// TaskActionDate is the date a Complete or Skip from the task list or detail
// page books for a task. A one-time task uses its due date, or today when it has
// none. A repeating task uses the current occurrence on the clock's local
// calendar: today if it is due, else the latest due day before today, else the
// series start when the series starts after today.
func (s *Service) TaskActionDate(task models.Task) string {
	today := s.Today()
	if !isRepeating(task) {
		if task.DueDate != "" {
			return task.DueDate
		}
		return s.TodayString()
	}
	task.ArchivedAt = ""
	start, ok := parseDate(task.DueDate, s.Location())
	if !ok {
		return s.TodayString()
	}
	if start.After(today) {
		return start.Format(clock.DateLayout)
	}
	// The series start is always an occurrence, so the walk ends at or before it.
	for day := today; !day.Before(start); day = day.AddDate(0, 0, -1) {
		if taskDueOn(task, day) {
			return day.Format(clock.DateLayout)
		}
	}
	return start.Format(clock.DateLayout)
}

// IsTaskOccurrence reports whether date (a canonical YYYY-MM-DD) is a day the
// task occurs on. One-time tasks and repeating tasks without a usable start
// date have no series to check, so any date is accepted for them.
func (s *Service) IsTaskOccurrence(task models.Task, date string) bool {
	if !isRepeating(task) {
		return true
	}
	if _, ok := parseDate(task.DueDate, s.Location()); !ok {
		return true
	}
	day, ok := parseDate(date, s.Location())
	if !ok {
		return false
	}
	task.ArchivedAt = ""
	return taskDueOn(task, day)
}

func taskDueOn(task models.Task, date time.Time) bool {
	if task.ArchivedAt != "" || task.DueDate == "" {
		return false
	}

	start, ok := parseDate(task.DueDate, date.Location())
	if !ok || date.Before(start) {
		return false
	}

	switch task.RepeatType {
	case "daily":
		return true
	case "weekly":
		return date.Weekday() == start.Weekday()
	case "monthly":
		return sameMonthDay(start, date)
	case "yearly":
		return sameYearDay(start, date)
	default:
		return sameDay(start, date)
	}
}

func parseDate(value string, location *time.Location) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func sameDay(left, right time.Time) bool {
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func sameMonthDay(start, date time.Time) bool {
	startDay := start.Day()
	lastDay := daysInMonth(date.Year(), date.Month())
	if startDay > lastDay {
		return date.Day() == lastDay
	}
	return date.Day() == startDay
}

func sameYearDay(start, date time.Time) bool {
	if start.Month() == time.February && start.Day() == 29 && !isLeapYear(date.Year()) {
		return date.Month() == time.February && date.Day() == 28
	}
	return start.Month() == date.Month() && start.Day() == date.Day()
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func priorityRank(priority string) int {
	switch priority {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

func streakFromStats(stats []DailyStat) int {
	streak := 0
	for i := len(stats) - 1; i >= 0; i-- {
		stat := stats[i]
		if stat.Total == 0 {
			continue
		}
		if stat.Percent < 100 {
			break
		}
		streak++
	}
	return streak
}
