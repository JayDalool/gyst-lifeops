package services

import (
	"math"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/xp"
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
		LevelProgress:     player.TotalStars % xp.XPPerLevel,
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

// BuildRocketRoute caps and positions nodes along the curved route. PathPct is
// the node's fraction along the SVG path (the browser places nodes precisely
// with getPointAtLength); LeftPct/TopPct follow a gentle wave as the no-JS
// fallback. Current marks the ship's resting destination (first active node, or
// the first node if none is active). mode is "galaxy" | "solar" | "compact".
func BuildRocketRoute(id, title, mode string, nodes []models.RouteNode) models.RocketRoute {
	const maxNodes = 6
	if len(nodes) > maxNodes {
		nodes = nodes[:maxNodes]
	}
	n := len(nodes)
	currentSet := false
	for i := range nodes {
		nodes[i].Index = i
		if n <= 1 {
			nodes[i].LeftPct = 50
			nodes[i].PathPct = 50
		} else {
			nodes[i].LeftPct = 10 + (i*80)/(n-1)
			// Spread evenly from 6%..94% so the ship has runway at both ends.
			nodes[i].PathPct = 6 + float64(i)*88/float64(n-1)
		}
		top := 50 - int(math.Round(24*math.Sin(float64(i)*0.95)))
		if top < 22 {
			top = 22
		}
		if top > 74 {
			top = 74
		}
		nodes[i].TopPct = top
		if !currentSet && nodes[i].State == "active" {
			nodes[i].Current = true
			currentSet = true
		}
	}
	if !currentSet && n > 0 {
		nodes[0].Current = true
	}
	return models.RocketRoute{ID: id, Title: title, Mode: mode, Nodes: nodes}
}
