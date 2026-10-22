package services

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

// serviceWithDB is testService plus the raw handle, so a test can seed a
// parent cycle that the repository now refuses to write.
func serviceWithDB(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return New(repository.New(database)), database
}

func galaxyProject(t *testing.T, svc *Service, name string, parent int64) models.Project {
	t.Helper()
	p := models.Project{Name: name, ProjectKind: "main", Status: "active", ParentProjectID: parent, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&p); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return p
}

func planetKeys(t *testing.T, p models.GalaxyPlanet) []string {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Without a cycle the Galaxy data is exactly what the parent links say.
func TestGalaxyDataAcyclicUnchanged(t *testing.T) {
	svc := testService(t)
	a := galaxyProject(t, svc, "Fictional Atlas", 0)
	b := galaxyProject(t, svc, "Orbit Alpha", a.ID)
	c := galaxyProject(t, svc, "Orbit Beta", 0)
	d := galaxyProject(t, svc, "Orbit Gamma", c.ID)

	planets, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	if len(planets) != 2 || planets[0].ID != a.ID || planets[1].ID != c.ID {
		t.Fatalf("planets = %+v", planets)
	}
	if len(planets[0].Moons) != 1 || planets[0].Moons[0].ID != b.ID || len(planets[1].Moons) != 1 || planets[1].Moons[0].ID != d.ID {
		t.Fatalf("moons wrong: %+v", planets)
	}
	for _, planet := range planets {
		stored, err := svc.Repo.ProjectByID(planet.ID)
		if err != nil {
			t.Fatal(err)
		}
		x, y, z := galaxyPosition(stored)
		if planet.X != x || planet.Y != y || planet.Z != z {
			t.Fatalf("planet %d moved", planet.ID)
		}
	}
	want := []string{"category", "color", "id", "missions", "moons", "name", "progress", "size", "state", "summary", "url", "x", "y", "z"}
	if got := planetKeys(t, planets[0]); !reflect.DeepEqual(got, want) {
		t.Fatalf("planet JSON keys changed: %v", got)
	}
}

// A parent cycle has no root; its lowest ID becomes the planet and the other a
// moon, so neither world disappears.
func TestGalaxyDataShowsParentCycle(t *testing.T) {
	svc, database := serviceWithDB(t)
	a := galaxyProject(t, svc, "Fictional Atlas", 0)
	b := galaxyProject(t, svc, "Orbit Alpha", a.ID)
	other := galaxyProject(t, svc, "Orbit Beta", 0)
	if _, err := database.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	planets, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	if len(planets) != 2 || planets[0].ID != a.ID || planets[1].ID != other.ID {
		t.Fatalf("planets = %+v", planets)
	}
	if len(planets[0].Moons) != 1 || planets[0].Moons[0].ID != b.ID {
		t.Fatalf("cycle moons = %+v", planets[0].Moons)
	}
}

// A phase named like its main used to be linked to itself and vanish.
func TestImportMarkdownPhaseNamedLikeMainKeepsMainAsParent(t *testing.T) {
	svc := testService(t)
	if _, err := svc.ImportRoadmapMarkdown("# Goal: Fictional\n\n## Fictional Atlas\n\n### Fictional Atlas\n"); err != nil {
		t.Fatalf("import: %v", err)
	}
	projects, err := svc.Repo.ListProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	var main, phase models.Project
	for _, p := range projects {
		if p.Project.Name != "Fictional Atlas" {
			continue
		}
		if p.Project.ProjectKind == "main" {
			main = p.Project
		} else {
			phase = p.Project
		}
	}
	if main.ID == 0 || phase.ID == 0 || phase.ParentProjectID != main.ID {
		t.Fatalf("main %+v phase %+v", main, phase)
	}
}
