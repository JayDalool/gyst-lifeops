package handlers

import (
	"fmt"
	"lifeops-tracker/internal/models"
	"net/url"
	"strconv"
)

func projectFilter(value string) string {
	switch value {
	case "active", "completed", "upcoming":
		return value
	}
	return "all"
}
func projectMatches(project models.ProjectWithProgress, filter string) bool {
	switch projectFilter(filter) {
	case "completed":
		return models.ProjectStateOf(project.Project.Status) == models.ProjectStateDone
	case "upcoming":
		state := models.ProjectStateOf(project.Project.Status)
		return state == models.ProjectStatePlanned || state == models.ProjectStateBlocked
	case "active":
		state := models.ProjectStateOf(project.Project.Status)
		return state == models.ProjectStateActive || state == models.ProjectStatePaused
	}
	return true
}
func (d ProjectListData) FilterURL(status string, roadmapID int64) string {
	values := url.Values{"status": {projectFilter(status)}}
	if roadmapID > 0 {
		values.Set("roadmap", strconv.FormatInt(roadmapID, 10))
	}
	return "/projects?" + values.Encode()
}
func (d ProjectListData) Filters() UISegments {
	out := UISegments{Label: "Project status"}
	for _, item := range []struct{ value, label string }{{"all", "All"}, {"active", "Active"}, {"completed", "Completed"}, {"upcoming", "Upcoming"}} {
		out.Items = append(out.Items, UISegment{Label: item.label, URL: d.FilterURL(item.value, d.SelectedRoadmapID), Selected: projectFilter(d.StatusFilter) == item.value})
	}
	return out
}

// Filter copies the tree and retains a parent when a child matches. The parent
// is context, not a newly activated/completed project. Repository state is untouched.
func (d ProjectListData) VisibleTree(tree models.ProjectTree) models.ProjectTree {
	out := models.ProjectTree{Children: make(map[int64][]models.ProjectWithProgress)}
	filter := func(items []models.ProjectWithProgress) []models.ProjectWithProgress {
		var result []models.ProjectWithProgress
		for _, item := range items {
			var children []models.ProjectWithProgress
			for _, child := range tree.Children[item.Project.ID] {
				if projectMatches(child, d.StatusFilter) {
					children = append(children, child)
				}
			}
			if projectMatches(item, d.StatusFilter) || len(children) > 0 {
				result = append(result, item)
				out.Children[item.Project.ID] = children
			}
		}
		return result
	}
	out.Main = filter(tree.Main)
	out.Upcoming = filter(tree.Upcoming)
	return out
}

type UIProjectCard struct {
	models.ProjectWithProgress
	Icon         string
	ProgressView UIProgress
}

func projectCard(project models.ProjectWithProgress) UIProjectCard {
	icon := "folder-heart"
	if project.Project.Category == "health" {
		icon = "heart"
	}
	return UIProjectCard{ProjectWithProgress: project, Icon: icon, ProgressView: UIProgress{Label: project.Project.Name + " completion", Present: true, Percent: project.Progress, Detail: fmt.Sprintf("%d of %d missions complete", project.CompletedItems, project.TotalItems)}}
}
