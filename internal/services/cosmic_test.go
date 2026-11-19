package services

import (
	"database/sql"
	"encoding/json"
	"math"
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
	for _, project := range projects {
		if project.Project.PosX != nil || project.Project.PosY != nil || project.Project.PosZ != nil {
			t.Fatalf("position stored by a read for %q", project.Project.Name)
		}
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

// GYST-076 review: parity with the base branch. The expected coordinates are
// literals captured from master 0b83127 before this PR (where GalaxyData stored the
// hash position on read) for fixed fictional UUIDs, so a change to the hash or
// coordinate math fails here instead of passing through the helper under test.
func TestGalaxyDataCoordinatesMatchBaseBranch(t *testing.T) {
	svc, database := testServiceDB(t)
	mk := func(name, kind, status, uuid string, parent int64) int64 {
		p := models.Project{Name: name, ProjectKind: kind, Status: status, ParentProjectID: parent, AutoUnlock: true}
		if err := svc.Repo.CreateProject(&p); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`UPDATE projects SET uuid = ? WHERE id = ?`, uuid, p.ID); err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	a := mk("Fictional Planet A", "main", "active", "00000000-0000-4000-8000-00000000000a", 0)
	b := mk("Fictional Planet B", "main", "active", "00000000-0000-4000-8000-00000000000b", 0)
	m1 := mk("Fictional Moon A1", "phase", "locked", "00000000-0000-4000-8000-0000000000a1", a)
	m2 := mk("Fictional Moon A2", "phase", "active", "00000000-0000-4000-8000-0000000000a2", a)
	c := mk("Fictional Planet C", "main", "active", "00000000-0000-4000-8000-00000000000c", 0)
	c1 := mk("Fictional Moon C1", "phase", "active", "00000000-0000-4000-8000-0000000000c1", c)
	b1 := mk("Fictional Moon B1", "phase", "upcoming", "00000000-0000-4000-8000-0000000000b1", b)
	// Stored positions (C, C1) are returned as saved; the rest are hashed.
	if _, err := database.Exec(`UPDATE projects SET pos_x = 1.5, pos_y = -2.25, pos_z = 3 WHERE id = ?`, c); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`UPDATE projects SET pos_x = -4, pos_y = 5.5, pos_z = 0.25 WHERE id = ?`, c1); err != nil {
		t.Fatal(err)
	}
	want := map[int64][3]float64{
		a:  {22.996171905306085, 10.444841325729433, 3.822785837587333},
		b:  {-48.650702651082575, -7.552110131085014, -13.134201526790411},
		m1: {-54.84498724075655, 16.210117896720217, 4.384637054689912},
		m2: {34.4250004295751, 28.29543142812656, -9.68551338666482},
		b1: {13.248659230553349, -50.65871346057022, -10.811759680530022},
		c:  {1.5, -2.25, 3},
		c1: {-4, 5.5, 0.25},
	}
	data, err := svc.GalaxyData()
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]models.GalaxyPlanet{}
	for _, planet := range data {
		got[planet.ID] = planet
		for _, moon := range planet.Moons {
			got[moon.ID] = moon
		}
	}
	if len(got) != len(want) {
		t.Fatalf("galaxy has %d bodies, want %d", len(got), len(want))
	}
	for id, w := range want {
		g, ok := got[id]
		if !ok {
			t.Fatalf("body %d missing", id)
		}
		for i, pair := range [][2]float64{{g.X, w[0]}, {g.Y, w[1]}, {g.Z, w[2]}} {
			if math.Abs(pair[0]-pair[1]) > 1e-9 {
				t.Errorf("body %d axis %d = %v, want %v", id, i, pair[0], pair[1])
			}
		}
	}
}
