package services

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

func testService(t *testing.T) *Service {
	t.Helper()
	svc, _ := testServiceDB(t)
	return svc
}

func testServiceDB(t *testing.T) (*Service, *sql.DB) {
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

func TestRocketRouteForRoadmapExcludesSubprojects(t *testing.T) {
	svc := testService(t)
	rm := models.Roadmap{Name: "Galaxy", Priority: 5, Status: "active", Privacy: "private"}
	if err := svc.Repo.CreateRoadmap(&rm); err != nil {
		t.Fatalf("create roadmap: %v", err)
	}
	top := models.Project{Name: "Solar System A", ProjectKind: "main", Status: "active", RoadmapID: rm.ID, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&top); err != nil {
		t.Fatalf("create top: %v", err)
	}
	sub := models.Project{Name: "Planet A1", ProjectKind: "subproject", Status: "active", RoadmapID: rm.ID, ParentProjectID: top.ID, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	route, err := svc.RocketRouteForRoadmap(rm.ID, "Galaxy")
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if len(route.Nodes) != 1 || route.Nodes[0].Name != "Solar System A" {
		t.Fatalf("galaxy route must exclude subprojects; got %+v", route.Nodes)
	}
	if route.Mode != "galaxy" {
		t.Fatalf("expected galaxy mode, got %q", route.Mode)
	}
}

func TestGalaxyDataStablePositionsAndMoonsWithoutWrites(t *testing.T) {
	svc := testService(t)
	top := models.Project{Name: "Stable Planet", ProjectKind: "main", Status: "active", AutoUnlock: true}
	if err := svc.Repo.CreateProject(&top); err != nil {
		t.Fatal(err)
	}
	moon := models.Project{Name: "Moon", ProjectKind: "phase", Status: "locked", ParentProjectID: top.ID, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&moon); err != nil {
		t.Fatal(err)
	}

	first, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(first[0].Moons) != 1 {
		t.Fatalf("galaxy hierarchy = %+v", first)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("galaxy JSON changed between reads")
	}
	// Reads compute the deterministic hash position and never store it.
	projects, err := svc.Repo.ListProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]models.Project{}
	for _, project := range projects {
		byID[project.Project.ID] = project.Project
		if project.Project.PosX != nil || project.Project.PosY != nil || project.Project.PosZ != nil {
			t.Fatalf("position stored by a read for %q", project.Project.Name)
		}
	}
	x, y, z := galaxyPosition(byID[top.ID])
	if first[0].X != x || first[0].Y != y || first[0].Z != z {
		t.Fatal("planet position is not the UUID hash")
	}
	mx, my, mz := galaxyPosition(byID[moon.ID])
	if first[0].Moons[0].X != mx || first[0].Moons[0].Y != my || first[0].Moons[0].Z != mz {
		t.Fatal("moon position is not the UUID hash")
	}
}

func TestGalaxyDataReturnsStoredPositionsUnchanged(t *testing.T) {
	svc, database := testServiceDB(t)
	p := models.Project{Name: "Pinned Planet", ProjectKind: "main", Status: "active", AutoUnlock: true}
	if err := svc.Repo.CreateProject(&p); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`UPDATE projects SET pos_x = 1.5, pos_y = -2.25, pos_z = 3 WHERE id = ?`, p.ID); err != nil {
		t.Fatal(err)
	}
	data, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 || data[0].X != 1.5 || data[0].Y != -2.25 || data[0].Z != 3 {
		t.Fatalf("stored position not returned unchanged: %+v", data)
	}
}

func TestGalaxyDataHidesCompletedPlanetsAndMoons(t *testing.T) {
	svc := testService(t)
	active := models.Project{Name: "Active Planet", ProjectKind: "main", Status: "active", AutoUnlock: true}
	if err := svc.Repo.CreateProject(&active); err != nil {
		t.Fatal(err)
	}
	doneMoon := models.Project{Name: "Completed Moon", ProjectKind: "phase", Status: "done", ParentProjectID: active.ID, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&doneMoon); err != nil {
		t.Fatal(err)
	}
	done := models.Project{Name: "Completed Planet", ProjectKind: "main", Status: "done", AutoUnlock: true}
	if err := svc.Repo.CreateProject(&done); err != nil {
		t.Fatal(err)
	}

	planets, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	if len(planets) != 1 || planets[0].ID != active.ID || len(planets[0].Moons) != 0 {
		t.Fatalf("completed worlds should leave the active galaxy: %+v", planets)
	}
}

func TestProjectSolarSystemIncludesChildren(t *testing.T) {
	svc := testService(t)
	top := models.Project{Name: "Solar System A", ProjectKind: "main", Status: "active", AutoUnlock: true}
	if err := svc.Repo.CreateProject(&top); err != nil {
		t.Fatalf("create top: %v", err)
	}
	sub := models.Project{Name: "Planet A1", ProjectKind: "subproject", Status: "active", ParentProjectID: top.ID, AutoUnlock: true}
	if err := svc.Repo.CreateProject(&sub); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	sys, err := svc.ProjectSolarSystem(top.ID)
	if err != nil {
		t.Fatalf("solar system: %v", err)
	}
	if sys.Center.Project.ID != top.ID {
		t.Fatalf("center should be the selected project, got %d", sys.Center.Project.ID)
	}
	if len(sys.Children) != 1 || sys.Children[0].Project.ID != sub.ID {
		t.Fatalf("solar system must include child projects; got %+v", sys.Children)
	}
	// Solar route is built from the child projects.
	if len(sys.Route.Nodes) != 1 || sys.Route.Mode != "solar" {
		t.Fatalf("solar route should hold the child; got mode=%q nodes=%+v", sys.Route.Mode, sys.Route.Nodes)
	}
}

func TestBuildRocketRouteCurrentAndPathSpread(t *testing.T) {
	route := BuildRocketRoute("galaxy", "Galaxy", "galaxy", []models.RouteNode{
		{Name: "Done", State: "done"},
		{Name: "Active", State: "active"},
		{Name: "Locked", State: "locked"},
	})
	if route.ID != "galaxy" || route.Mode != "galaxy" {
		t.Fatalf("id/mode not set: %+v", route)
	}
	// Current marks the first active node.
	if !route.Nodes[1].Current || route.Nodes[0].Current || route.Nodes[2].Current {
		t.Fatalf("current should be the first active node; got %+v", route.Nodes)
	}
	// PathPct spreads evenly from 6%..94% so the ship has runway at both ends.
	if route.Nodes[0].PathPct != 6 || route.Nodes[2].PathPct != 94 {
		t.Fatalf("path spread wrong: first=%v last=%v", route.Nodes[0].PathPct, route.Nodes[2].PathPct)
	}
}
