package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/repository"
	"lifeops-tracker/internal/services"
)

// dumpAllTables returns every row of every table (except schema_migrations),
// so a test can prove a request changed nothing, columns like updated_at and
// sync_version included.
func dumpAllTables(t *testing.T, database *sql.DB) string {
	t.Helper()
	names, err := database.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for names.Next() {
		var n string
		if err := names.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	names.Close()
	var out strings.Builder
	for _, table := range tables {
		rows, err := database.Query(fmt.Sprintf(`SELECT * FROM "%s" ORDER BY 1, 2`, table))
		if err != nil {
			t.Fatalf("dump %s: %v", table, err)
		}
		cols, _ := rows.Columns()
		fmt.Fprintf(&out, "== %s %v\n", table, cols)
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&out, "%v\n", vals)
		}
		rows.Close()
	}
	return out.String()
}

func readonlyHandler(t *testing.T) (*Handler, *sql.DB) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if err := db.SeedIfEmpty(database, filepath.Join("..", "..", "seeds", "example.sql")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	h := New(services.New(repository.New(database)), "../../web/templates")
	h.AllowPublicFeeds = true
	return h, database
}

// GYST-VN-028: the GET routes that used to write (galaxy positions, orphan
// adoption) change no row in any table, including a raw-SQL orphan, projects
// with NULL positions, a stored position, a moon and a done project.
func TestGETRoutesChangeNoRows(t *testing.T) {
	h, database := readonlyHandler(t)
	for _, stmt := range []string{
		`INSERT INTO projects (name, project_kind, status, updated_at) VALUES ('Fictional Raw Orphan', 'main', 'active', '2026-01-01 00:00:00')`,
		`INSERT INTO projects (name, project_kind, status, pos_x, pos_y, pos_z, updated_at) VALUES ('Fictional Pinned', 'main', 'active', 4, 5, 6, '2026-01-01 00:00:00')`,
		`INSERT INTO projects (name, project_kind, status, updated_at) VALUES ('Fictional Finished', 'main', 'done', '2026-01-01 00:00:00')`,
		`INSERT INTO projects (name, project_kind, status, parent_project_id, updated_at)
			SELECT 'Fictional Moon', 'phase', 'active', id, '2026-01-01 00:00:00' FROM projects WHERE name = 'Fictional Pinned'`,
	} {
		if _, err := database.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	var orphans int
	if err := database.QueryRow(`SELECT COUNT(*) FROM projects WHERE roadmap_id IS NULL`).Scan(&orphans); err != nil || orphans == 0 {
		t.Fatalf("fixture has no orphans (%d, %v)", orphans, err)
	}
	dash, err := h.Service.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{
		"/galaxy/data", "/galaxy", "/projects",
		fmt.Sprintf("/projects?roadmap=%d", dash.ID), "/projects?status=active", "/analytics",
	}
	before := dumpAllTables(t, database)
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status %d", path, rec.Code)
		}
		if after := dumpAllTables(t, database); after != before {
			t.Fatalf("GET %s changed the database", path)
		}
	}
	var stored int
	if err := database.QueryRow(`SELECT COUNT(*) FROM projects WHERE pos_x IS NOT NULL`).Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("stored positions = %d, %v; want only the pinned project", stored, err)
	}
}

// An Unassigned project is grouped by the write itself, with no GET-time help.
func TestCreateProjectUnassignedAppearsInDashboardGroup(t *testing.T) {
	h := testHandler(t)
	rec := serve(h, http.MethodPost, "/projects/new", url.Values{
		"name": {"Fictional Fresh Trip"}, "status": {"active"}, "project_kind": {"main"}, "roadmap_id": {""},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	dash, err := h.Service.Repo.GetDashboardRoadmap()
	if err != nil {
		t.Fatal(err)
	}
	projects, err := h.Service.Repo.ListProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range projects {
		if p.Project.Name == "Fictional Fresh Trip" {
			found = true
			if p.Project.RoadmapID != dash.ID {
				t.Fatalf("roadmap = %d, want dashboard %d", p.Project.RoadmapID, dash.ID)
			}
		}
	}
	if !found {
		t.Fatal("project not created")
	}
	tree, err := h.Service.Repo.ProjectTreeByRoadmap(dash.ID)
	if err != nil {
		t.Fatal(err)
	}
	inGroup := false
	for _, p := range append(tree.Main, tree.Upcoming...) {
		if p.Project.Name == "Fictional Fresh Trip" {
			inGroup = true
		}
	}
	if !inGroup {
		t.Fatal("project missing from the dashboard roadmap group")
	}
	if body := getProjects(t, h, "/projects"); !strings.Contains(body, "Fictional Fresh Trip") {
		t.Fatal("GET /projects does not show the new project")
	}
}
