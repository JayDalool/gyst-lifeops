package models

type Task struct {
	ID           int64
	UUID         string
	Title        string
	Description  string
	Category     string
	Priority     string
	Status       string
	DueDate      string
	RepeatType   string
	ReminderTime string
	CreatedAt    string
	UpdatedAt    string
	CompletedAt  string
	ArchivedAt   string
}

type TaskCompletion struct {
	ID             int64
	TaskID         int64
	OccurrenceDate string
	Status         string
	CompletedAt    string
	Notes          string
}

type TaskOccurrence struct {
	Task          Task
	Date          string
	Status        string
	CompletedAt   string
	OccurrenceKey string
}

type Project struct {
	ID          int64
	UUID        string
	Name        string
	Description string
	Category    string
	Status      string
	StartDate   string
	TargetDate  string
	Notes       string
	CreatedAt   string
	UpdatedAt   string
	ArchivedAt  string
}

type ProjectItem struct {
	ID          int64
	ProjectID   int64
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     string
	SortOrder   int
	CreatedAt   string
	UpdatedAt   string
	CompletedAt string
}

type ProjectWithProgress struct {
	Project        Project
	TotalItems     int
	CompletedItems int
	Progress       int
	NextDueItem    string
	NextDueDate    string
	Momentum       ChartData
}

type RoadmapProjectMeta struct {
	ID               int64
	ProjectID        int64
	SourceImportID   int64
	ProjectType      string
	ProjectTypeLabel string
	Tags             []string
	TagsText         string
	IsPrivate        bool
	LocalOnly        bool
	CurrentFocus     bool
	PublicSummary    string
	CreatedAt        string
	UpdatedAt        string
}

type RoadmapMilestone struct {
	ID          int64
	ProjectID   int64
	Title       string
	Description string
	Status      string
	DueDate     string
	SortOrder   int
	CreatedAt   string
	UpdatedAt   string
	CompletedAt string
}

type RoadmapResource struct {
	ID             int64
	ProjectID      int64
	MilestoneID    int64
	ProjectItemID  int64
	SourceImportID int64
	Title          string
	URL            string
	ResourceType   string
	Notes          string
	IsPrivate      bool
	CreatedAt      string
	UpdatedAt      string
}

type RoadmapImport struct {
	ID             int64
	UUID           string
	GoalID         int64
	GoalTitle      string
	GoalDeadline   string
	SourceMarkdown string
	ImportedAt     string
	CreatedAt      string
	UpdatedAt      string
}

type RoadmapWeeklyReview struct {
	ID        int64
	WeekStart string
	Finished  string
	Blocked   string
	NextSteps string
	CreatedAt string
	UpdatedAt string
}

type RoadmapDeadline struct {
	Kind        string
	Title       string
	ProjectName string
	DueDate     string
	ProjectID   int64
	ItemID      int64
}

type RoadmapDailyTask struct {
	Title       string
	ProjectName string
	ProjectType string
	DueDate     string
	ProjectID   int64
	ItemID      int64
}

type RoadmapDailyTasks struct {
	AITask        RoadmapDailyTask
	CourseTask    RoadmapDailyTask
	CodingTask    RoadmapDailyTask
	PortfolioTask RoadmapDailyTask
}

type RoadmapProject struct {
	Project          ProjectWithProgress
	Meta             RoadmapProjectMeta
	Items            []ProjectItem
	Milestones       []RoadmapMilestone
	Resources        []RoadmapResource
	Progress         int
	CompletedWork    int
	TotalWork        int
	WeeklyCompletion int
}

type RoadmapData struct {
	Page               string
	Today              string
	WeekStart          string
	WeekEnd            string
	GoalTitle          string
	GoalDeadline       string
	LatestImport       RoadmapImport
	Projects           []RoadmapProject
	OverallProgress    int
	WeeklyCompletion   int
	MissedDeadlines    []RoadmapDeadline
	CurrentFocus       []RoadmapProject
	Daily              RoadmapDailyTasks
	Resources          []RoadmapResource
	WeeklyReviews      []RoadmapWeeklyReview
	ProjectTypeOptions []RoadmapProjectTypeOption
}

type RoadmapProjectTypeOption struct {
	Value string
	Label string
}

type RoadmapImportPlan struct {
	Goal           RoadmapGoalPlan
	Projects       []RoadmapProjectPlan
	Resources      []RoadmapResourcePlan
	SourceMarkdown string
	ImportedDate   string
}

type RoadmapGoalPlan struct {
	Title       string
	Description string
	Deadline    string
	Status      string
	Tags        []string
}

type RoadmapProjectPlan struct {
	Name          string
	Description   string
	ProjectType   string
	Status        string
	Deadline      string
	Tags          []string
	IsPrivate     bool
	LocalOnly     bool
	CurrentFocus  bool
	PublicSummary string
	Milestones    []RoadmapMilestonePlan
	Tasks         []RoadmapTaskPlan
	Resources     []RoadmapResourcePlan
}

type RoadmapMilestonePlan struct {
	Title       string
	Description string
	Status      string
	DueDate     string
	SortOrder   int
}

type RoadmapTaskPlan struct {
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     string
	SortOrder   int
}

type RoadmapResourcePlan struct {
	Title        string
	URL          string
	ResourceType string
	Notes        string
	IsPrivate    bool
}

type Goal struct {
	ID           int64
	UUID         string
	Title        string
	Description  string
	Type         string
	Category     string
	TargetValue  float64
	CurrentValue float64
	Unit         string
	StartDate    string
	EndDate      string
	Status       string
	CreatedAt    string
	UpdatedAt    string
	ArchivedAt   string
}

type GoalProgressEntry struct {
	ID        int64
	GoalID    int64
	EntryDate string
	Value     float64
	Notes     string
	CreatedAt string
}

type GoalWithProgress struct {
	Goal     Goal
	Progress int
	Momentum ChartData
}

type Reminder struct {
	ID           int64
	SourceType   string
	SourceID     int64
	Title        string
	ReminderDate string
	ReminderTime string
	Status       string
}

type ReminderView struct {
	SourceType   string
	SourceID     int64
	Title        string
	ReminderDate string
	ReminderTime string
	Overdue      bool
}

type TrackerItem struct {
	ID        int64
	UUID      string
	Key       string
	Title     string
	Type      string
	Category  string
	SortOrder int
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

type TrackerLog struct {
	ID            int64
	TrackerItemID int64
	LogDate       string
	Status        string
	ValueNumeric  float64
	Note          string
	CreatedAt     string
	UpdatedAt     string
}

type TrackerStatusView struct {
	Item   TrackerItem
	Date   string
	Status string
}

type GymDayView struct {
	Label  string
	Date   string
	Status string
}

type DailyMissionData struct {
	Date              string
	PrayerItems       []TrackerStatusView
	AthkarItems       []TrackerStatusView
	GymWeek           []GymDayView
	PrayerCompletion  int
	AthkarCompletion  int
	DeenCompletion    int
	GymWeekCompletion int
}

type ChartData struct {
	Labels []string
	Values []int
}

type DashboardData struct {
	Page                     string
	Today                    string
	TodayItems               []TaskOccurrence
	OverdueTasks             []Task
	TodayCompletion          int
	MonthlyCompletionPercent int
	CurrentStreak            int
	ActiveProjects           []ProjectWithProgress
	YearlyGoals              []GoalWithProgress
	RemindersToday           []ReminderView
	OverdueReminders         []ReminderView
	NextReminder             ReminderView
	DailyMission             DailyMissionData
	DailyCompletion          ChartData
	LastSevenCompletion      ChartData
	DeenConsistency          ChartData
	BodyRhythm               ChartData
	MonthlyCompletion        ChartData
	MonthlyScore             ChartData
	TasksCompletedMini       ChartData
	ProjectProgress          ChartData
	ProjectMomentum          ChartData
	ProjectMomentumURL       string
	YearlyGoalProgress       ChartData
	GoalProgressSignal       ChartData
	ReminderLoad             ChartData
}
