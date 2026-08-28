package models

import "time"

type Task struct {
	ID            int64
	UUID          string
	Title         string
	Description   string
	Category      string
	Priority      string
	Status        string
	DueDate       string
	RepeatType    string
	ReminderTime  string
	StarsReward   int
	ProofRequired bool
	ProofType     string
	ProofURL      string
	ProofPath     string
	StarsAwarded  int
	CreatedAt     string
	UpdatedAt     string
	CompletedAt   string
	ArchivedAt    string
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
	ID                 int64
	UUID               string
	ParentProjectID    int64
	RoadmapID          int64
	Name               string
	Description        string
	Category           string
	ProjectKind        string
	Status             string
	StartDate          string
	TargetDate         string
	Notes              string
	UnlockNotes        string
	DependsOnProjectID int64
	AutoUnlock         bool
	CreatedAt          string
	UpdatedAt          string
	ArchivedAt         string
	PosX               *float64
	PosY               *float64
	PosZ               *float64
}

type GalaxyPlanet struct {
	ID       int64          `json:"id"`
	Name     string         `json:"name"`
	Category string         `json:"category"`
	Summary  string         `json:"summary"`
	Target   string         `json:"target,omitempty"`
	Next     string         `json:"next,omitempty"`
	Missions int            `json:"missions"`
	URL      string         `json:"url"`
	State    string         `json:"state"`
	Size     float64        `json:"size"`
	Progress int            `json:"progress"`
	X        float64        `json:"x"`
	Y        float64        `json:"y"`
	Z        float64        `json:"z"`
	Moons    []GalaxyPlanet `json:"moons"`
}

type ProjectItem struct {
	ID            int64
	ProjectID     int64
	Title         string
	Description   string
	Status        string
	Priority      string
	DueDate       string
	SortOrder     int
	StarsReward   int
	ProofRequired bool
	ProofType     string
	ProofURL      string
	ProofPath     string
	StarsAwarded  int
	CreatedAt     string
	UpdatedAt     string
	CompletedAt   string
}

// ExternalLink is a read-only reference to work managed outside LifeOps. It is
// deliberately separate from proof: attaching or editing one never completes
// work and never awards XP.
type ExternalLink struct {
	ID                   int64
	UUID                 string
	EntityType           string
	EntityID             int64
	Provider             string
	ExternalKey          string
	URL                  string
	Title                string
	Summary              string
	StatusText           string
	SyncSource           string
	LastSyncedAt         string
	SyncVersion          int
	LastModifiedDeviceID string
	CreatedAt            string
	UpdatedAt            string
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

type ProjectTree struct {
	Main     []ProjectWithProgress
	Children map[int64][]ProjectWithProgress
	Upcoming []ProjectWithProgress
}

// Course is a self-paced learning container (Udemy, bootcamp, YouTube series,
// API/SQL/AI course, etc.). ProgressPercent is entered manually.
type Course struct {
	ID              int64
	UUID            string
	Name            string
	Provider        string
	Instructor      string
	URL             string
	Status          string
	ProgressPercent int
	TargetDate      string
	Notes           string
	CreatedAt       string
	UpdatedAt       string
	ArchivedAt      string
}

// LearningBlock is any unit of a course: a lesson/module, study/review block,
// assignment/exam deadline, or optional live session. DueDate/StartTime/EndTime
// are all optional so self-paced lessons need no schedule.
type LearningBlock struct {
	ID          int64
	UUID        string
	CourseID    int64
	Title       string
	BlockType   string
	Status      string
	StartTime   string
	EndTime     string
	DueDate     string
	URL         string
	RepeatType  string
	Notes       string
	SortOrder   int
	IsCatchup   bool
	CreatedAt   string
	UpdatedAt   string
	CompletedAt string
	ArchivedAt  string
	// CourseName is populated for cross-course views (Today, unified calendar).
	CourseName string
}

// CourseWithProgress decorates a course with block counts and the next block /
// next deadline for the Learning Deck.
type CourseWithProgress struct {
	Course          Course
	BlockCount      int
	CompletedBlocks int
	NextBlock       LearningBlock
	HasNextBlock    bool
	NextDeadline    LearningBlock
	HasNextDeadline bool
}

// TodayLearning is the compact "Learning Missions" payload for the Today page.
type TodayLearning struct {
	Blocks        []LearningBlock // scheduled/due today + overdue catch-ups
	SuggestStudy  bool            // active course exists but nothing is due today
	SuggestCourse Course          // the course to suggest a study block for
}

// DeadlineRow is a dated row (project mission item, roadmap milestone, or
// standalone reminder) used only to build calendar events. It carries the row
// uuid for a stable ICS UID since the view models don't expose it. Time is set
// only for reminders (which always have a clock time); ProjectID/ProjectName are
// set only for items/milestones.
type DeadlineRow struct {
	UUID        string
	Title       string
	DueDate     string
	Time        string
	ProjectID   int64
	ProjectName string
}

type Achievement struct {
	ID          int64
	Title       string
	Description string
	StarsBonus  int
	UnlockedAt  string
	TriggerType string
	CreatedAt   string
}

type PlayerStats struct {
	TotalStars    int
	CurrentLevel  int
	ActiveStreak  int
	ProofCount    int
	TaskDoneCount int
}

// PrayerConfig controls the local solar calculation. Adjustments are signed
// minutes applied after astronomical times are calculated.
type PrayerConfig struct {
	Latitude    float64
	Longitude   float64
	Timezone    string
	Method      string
	AsrMethod   string
	Adjustments map[string]int
	AlertSound  string
	AthanFile   string
}

type PrayerTime struct {
	Name string
	Time string
	At   time.Time
}

type PrayerSchedule struct {
	Date     string
	Location string
	Times    []PrayerTime
}

type NextPrayer struct {
	Prayer    PrayerTime
	Remaining time.Duration
	Tomorrow  bool
}

type PushSubscription struct {
	ID        int64
	Endpoint  string
	P256DH    string
	Auth      string
	UserAgent string
}

// Roadmap is a top-level journey that groups projects (trips). It is distinct
// from the markdown-import RoadmapData feature.
type Roadmap struct {
	ID          int64
	UUID        string
	Name        string
	Description string
	Priority    int
	Status      string
	Privacy     string
	Featured    bool
	Icon        string
	Color       string
	CreatedAt   string
	UpdatedAt   string
	ArchivedAt  string
}

// RoadmapWithProgress augments a Roadmap with rollup counts for the sidebar.
type RoadmapWithProgress struct {
	Roadmap      Roadmap
	ProjectCount int
	ActiveCount  int
	Progress     int
}

// ShipStats is the Captain Stats / Ship Status bar shown on the bridge and the
// trip/galaxy pages. All values derive from existing data (no schema change).
type ShipStats struct {
	TotalFuel         int
	Level             int
	LevelProgress     int // 0-99, fuel into the current level
	CompletedMissions int
	ActiveTrips       int
	Streak            int
	MedalCount        int
	RecentMedal       string
}

// RouteNode is one planet/station along the animated rocket route.
type RouteNode struct {
	Name    string
	URL     string
	State   string // done | active | locked | pending
	Index   int
	LeftPct int
	TopPct  int
	// PathPct is the node's position along the SVG route path (0-100). The
	// browser places nodes precisely with getPointAtLength; LeftPct/TopPct stay
	// as the no-JS fallback.
	PathPct float64
	// Current marks the ship's resting destination (first active node).
	Current bool
}

// RocketRoute feeds the reusable rocket_route.html partial. Mode selects the
// visual treatment: "galaxy" (roadmap overview), "solar" (one project's system),
// or "compact" (small strip on detail pages). ID is unique per page so multiple
// routes can each own their <path id>.
type RocketRoute struct {
	ID    string
	Title string
	Mode  string
	Nodes []RouteNode
}

// SolarSystem is one project rendered as its own system: the project at the
// centre, its child projects (planets/stations) orbiting, and its work items
// (missions). Missions are project_items, the only project-scoped work in the
// data model (standalone Tasks do not carry a project link).
type SolarSystem struct {
	Center     ProjectWithProgress
	Roadmap    Roadmap
	HasRoadmap bool
	Children   []ProjectWithProgress
	Missions   []ProjectItem
	Route      RocketRoute
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
	ShipStats          ShipStats
	ShipRoute          RocketRoute
	Roadmaps           []Roadmap
	SelectedRoadmap    Roadmap
	HasSelectedRoadmap bool
	ScopedTree         ProjectTree
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
	Name               string
	Description        string
	ProjectType        string
	ProjectKind        string
	ParentProjectID    int64
	ParentName         string
	Status             string
	Deadline           string
	Notes              string
	UnlockNotes        string
	DependsOn          string
	DependsOnProjectID int64
	AutoUnlock         bool
	Tags               []string
	IsPrivate          bool
	LocalOnly          bool
	CurrentFocus       bool
	PublicSummary      string
	Milestones         []RoadmapMilestonePlan
	Tasks              []RoadmapTaskPlan
	Resources          []RoadmapResourcePlan
}

type RoadmapMilestonePlan struct {
	Title       string
	Description string
	Status      string
	DueDate     string
	SortOrder   int
}

type RoadmapTaskPlan struct {
	Title         string
	Description   string
	Status        string
	Priority      string
	DueDate       string
	SortOrder     int
	StarsReward   int
	ProofRequired bool
	ProofType     string
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

// ActivitySummary is the compact dashboard widget that replaces the old big
// reminders/overdue/skipped blocks.
type ActivitySummary struct {
	Reminders      int
	Overdue        int
	CompletedToday int
	Skipped        int
}

type DashboardData struct {
	Page                     string
	ShipStats                ShipStats
	ShipRoute                RocketRoute
	DashboardRoadmap         Roadmap
	HasRoadmap               bool
	Roadmaps                 []Roadmap
	Activity                 ActivitySummary
	Today                    string
	TodayItems               []TaskOccurrence
	OverdueTasks             []Task
	TodayCompletion          int
	MonthlyCompletionPercent int
	CurrentStreak            int
	Player                   PlayerStats
	ActiveProjects           []ProjectWithProgress
	UpcomingProjects         []ProjectWithProgress
	RecentAchievements       []Achievement
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
