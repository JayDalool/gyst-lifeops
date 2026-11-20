package repository

import (
	"reflect"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-165: created_at has one-second resolution, so projects made in the same
// second tied and SQLite returned them in either order. Ties now break by id
// DESC, which keeps the list newest-first.
func TestListProjectsSameCreatedAtOrdersByIDDesc(t *testing.T) {
	repo := newTestRepo(t)
	var ids []int64
	for _, name := range []string{"Fictional Atlas", "Orbit Alpha", "Orbit Beta"} {
		p := models.Project{Name: name, ProjectKind: "main", Status: "active"}
		if err := repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	// Pin an identical timestamp, and make the oldest row sort last by rowid
	// so a missing tiebreak cannot pass by accident of scan order.
	if _, err := repo.db.Exec(`UPDATE projects SET created_at = '2030-01-01 00:00:00'`); err != nil {
		t.Fatal(err)
	}
	want := []int64{ids[2], ids[1], ids[0]}
	for _, list := range []func() ([]models.ProjectWithProgress, error){
		func() ([]models.ProjectWithProgress, error) { return repo.ListProjects(false) },
	} {
		got, err := list()
		if err != nil {
			t.Fatal(err)
		}
		var order []int64
		for _, p := range got {
			order = append(order, p.Project.ID)
		}
		if !reflect.DeepEqual(order, want) {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
