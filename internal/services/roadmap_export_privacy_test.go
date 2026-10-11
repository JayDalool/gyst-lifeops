package services

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/db"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/repository"
)

const exportPrivacySource = `# Fictional Campaign

## Project: Fictional Garden Planner
Private: yes
Description: Secret garden description.
- [x] Garden root task

### Phase 1: Seed Catalog
Description: Seed catalog private words.
- [ ] Sort seeds

### Phase 1a: Seed Labels
Description: Label private words.
- [ ] Print labels

## Project: Fictional Open Book
Description: Open book words.
- [ ] Write chapter

### Phase 1: Open Chapter
Description: Open chapter words.
- [x] Outline
`

func exportPrivacyService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "lifeops.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(database, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	svc := New(repository.New(database))
	svc.Now = func() time.Time { return time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC) }
	if _, err := svc.ImportRoadmapMarkdown(exportPrivacySource); err != nil {
		t.Fatalf("import: %v", err)
	}
	// The parser only builds two levels; nest Seed Labels under Seed Catalog.
	if _, err := database.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`,
		projectIDByName(t, svc, "Phase 1: Seed Catalog"), projectIDByName(t, svc, "Phase 1a: Seed Labels")); err != nil {
		t.Fatalf("nest: %v", err)
	}
	return svc, database
}

func projectIDByName(t *testing.T, svc *Service, name string) int64 {
	t.Helper()
	projects, err := svc.Repo.ListProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projects {
		if p.Project.Name == name {
			return p.Project.ID
		}
	}
	t.Fatalf("project %q not found", name)
	return 0
}

func TestPublicExportOmitsPhasesOfPrivateProject(t *testing.T) {
	svc, _ := exportPrivacyService(t)
	pub, err := svc.ExportRoadmap(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"Fictional Garden Planner", "Seed Catalog", "Seed Labels", "Seed catalog private words", "Label private words", "Secret garden description"} {
		if strings.Contains(pub, leak) {
			t.Errorf("public export leaks %q:\n%s", leak, pub)
		}
	}
	for _, want := range []string{"Fictional Open Book", "Open Chapter", "Open chapter words"} {
		if !strings.Contains(pub, want) {
			t.Errorf("public export missing %q:\n%s", want, pub)
		}
	}
	priv, err := svc.ExportRoadmap(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Fictional Garden Planner", "Seed Catalog", "Seed Labels", "Fictional Open Book", "Private: true"} {
		if !strings.Contains(priv, want) {
			t.Errorf("private export missing %q:\n%s", want, priv)
		}
	}
}

func TestPublicExportTotalsCountVisibleProjectsOnly(t *testing.T) {
	svc, _ := exportPrivacyService(t)
	pub, err := svc.ExportRoadmap(true)
	if err != nil {
		t.Fatal(err)
	}
	// Visible work: 2 tasks in the open book, 1 done = 50%.
	if !strings.Contains(pub, "Overall progress: 50%") {
		t.Errorf("public overall should be 50%%:\n%s", pub)
	}
	priv, err := svc.ExportRoadmap(false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(priv, "Overall progress: 50%") {
		t.Errorf("private overall should count all projects:\n%s", priv)
	}
}

func TestPublicExportHidesPhasesAfterParentFlippedPrivate(t *testing.T) {
	for _, mode := range []string{"private", "local"} {
		svc, _ := exportPrivacyService(t)
		id := projectIDByName(t, svc, "Fictional Open Book")
		set := func(on bool) {
			meta := models.RoadmapProjectMeta{ProjectID: id, ProjectType: "learning"}
			if mode == "private" {
				meta.IsPrivate = on
			} else {
				meta.LocalOnly = on
			}
			if err := svc.Repo.SetRoadmapProjectMeta(meta); err != nil {
				t.Fatal(err)
			}
		}
		export := func() string {
			out, err := svc.ExportRoadmap(true)
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		if !strings.Contains(export(), "Open Chapter") {
			t.Fatalf("%s: phase should be listed while public", mode)
		}
		set(true)
		if out := export(); strings.Contains(out, "Open Chapter") || strings.Contains(out, "Fictional Open Book") {
			t.Fatalf("%s: parent/phase should be hidden:\n%s", mode, out)
		}
		set(false)
		if !strings.Contains(export(), "Open Chapter") {
			t.Fatalf("%s: phase should return after parent is public again", mode)
		}
	}
}

func TestPublicExportResourcesUnchanged(t *testing.T) {
	svc := testService(t)
	svc.Now = func() time.Time { return time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC) }
	src := "# Fictional Campaign\n\n## Project: Fictional Open Book\n- [ ] Write chapter\n[Fictional Guide](https://example.com/guide)\n"
	if _, err := svc.ImportRoadmapMarkdown(src); err != nil {
		t.Fatal(err)
	}
	out, err := svc.ExportRoadmap(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "### Resources") || !strings.Contains(out, "[Fictional Guide](https://example.com/guide)") {
		t.Errorf("resources should still be exported:\n%s", out)
	}
}

func TestPublicExportFailsClosedOnPrivacyError(t *testing.T) {
	svc, database := exportPrivacyService(t)
	if _, err := database.Exec(`DROP TABLE roadmap_project_meta`); err != nil {
		t.Fatal(err)
	}
	if out, err := svc.ExportRoadmap(true); err == nil {
		t.Fatalf("expected error, got export:\n%s", out)
	}
}
