package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func TestBuildProjectTreeBucketsByProjectState(t *testing.T) {
	var projects []models.ProjectWithProgress
	for i, status := range []string{"active", "paused", "done", "completed", "archived", "upcoming", "locked"} {
		projects = append(projects, models.ProjectWithProgress{Project: models.Project{ID: int64(i + 1), Name: status, Status: status}})
	}
	tree := buildProjectTree(projects)
	names := func(list []models.ProjectWithProgress) string {
		out := ""
		for _, p := range list {
			out += p.Project.Name + ","
		}
		return out
	}
	if got := names(tree.Main); got != "active,paused,done,completed,archived," {
		t.Errorf("Main = %s", got)
	}
	if got := names(tree.Upcoming); got != "upcoming,locked," {
		t.Errorf("Upcoming = %s", got)
	}
}
