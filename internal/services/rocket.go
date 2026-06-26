package services

import (
	"math"

	"lifeops-tracker/internal/models"
)

// ShipStats computes the Captain Stats / Ship Status bar entirely from existing
// data. PlayerStats already implements the total_fuel/level/completed formula.
func (s *Service) ShipStats() (models.ShipStats, error) {
	player, err := s.Repo.PlayerStats()
	if err != nil {
		return models.ShipStats{}, err
	}

	today := s.Today()
	dailyStats, err := s.DailyCompletionStats(today.AddDate(0, 0, -29), today)
	if err != nil {
		return models.ShipStats{}, err
	}
	streak := streakFromStats(dailyStats)

	projects, err := s.Repo.ListProjects(false)
	if err != nil {
		return models.ShipStats{}, err
	}
	activeTrips := 0
	for _, project := range projects {
		if project.Project.Status == "active" {
			activeTrips++
		}
	}

	medalCount, err := s.Repo.CountAchievements()
	if err != nil {
		return models.ShipStats{}, err
	}
	recentMedal := ""
	if recent, err := s.Repo.RecentAchievements(1); err == nil && len(recent) > 0 {
		recentMedal = recent[0].Title
	}

	return models.ShipStats{
		TotalFuel:         player.TotalStars,
		Level:             player.CurrentLevel,
		LevelProgress:     player.TotalStars % 100,
		CompletedMissions: player.TaskDoneCount,
		ActiveTrips:       activeTrips,
		Streak:            streak,
		MedalCount:        medalCount,
		RecentMedal:       recentMedal,
	}, nil
}

// RouteState maps a project status + progress to a rocket-route node state.
func RouteState(status string, progress int) string {
	switch status {
	case "completed", "done", "archived":
		return "done"
	case "locked", "upcoming", "paused":
		return "locked"
	case "blocked":
		return "blocked"
	default:
		if progress >= 100 {
			return "done"
		}
		return "active"
	}
}

// BuildRocketRoute caps and positions nodes along the curved route. LeftPct
// spreads nodes horizontally; TopPct follows a gentle wave so they sit near the
// decorative path. Positions are percentages consumed by inline styles.
func BuildRocketRoute(title string, nodes []models.RouteNode) models.RocketRoute {
	const maxNodes = 6
	if len(nodes) > maxNodes {
		nodes = nodes[:maxNodes]
	}
	n := len(nodes)
	for i := range nodes {
		nodes[i].Index = i
		if n <= 1 {
			nodes[i].LeftPct = 50
		} else {
			nodes[i].LeftPct = 10 + (i*80)/(n-1)
		}
		top := 50 - int(math.Round(24*math.Sin(float64(i)*0.95)))
		if top < 22 {
			top = 22
		}
		if top > 74 {
			top = 74
		}
		nodes[i].TopPct = top
	}
	return models.RocketRoute{Title: title, Nodes: nodes}
}
