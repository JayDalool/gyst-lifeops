package repository

import (
	"reflect"
	"testing"

	"lifeops-tracker/internal/models"
)

// GYST-165: created_at has one-second resolution, so projects made in the same
// second tie. SQLite scans such ties in id order (creation order, which imports
// rely on) as a plan accident; the explicit id ASC tiebreak now guarantees it.
// Newest created_at stays first.
func TestListProjectsOrdersNewestFirstThenLowerID(t *testing.T) {
	repo := newTestRepo(t)
	var ids []int64
	for _, name := range []string{"Fictional Atlas", "Orbit Alpha", "Orbit Beta"} {
		p := models.Project{Name: name, ProjectKind: "main", Status: "active"}
		if err := repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	// Pin an identical timestamp for the first three.
	if _, err := repo.db.Exec(`UPDATE projects SET created_at = '2030-01-01 00:00:00'`); err != nil {
		t.Fatal(err)
	}
	// Row 4 is newer by created_at, so it leads; the tied rows follow by id ASC.
	p := models.Project{Name: "Newest", ProjectKind: "main", Status: "active"}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`UPDATE projects SET created_at = '2030-01-02 00:00:00' WHERE id = ?`, p.ID); err != nil {
		t.Fatal(err)
	}
	want := []int64{p.ID, ids[0], ids[1], ids[2]}
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
