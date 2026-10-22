package repository

import (
	"errors"
	"reflect"
	"testing"

	"lifeops-tracker/internal/models"
)

func cycleProject(t *testing.T, repo *Repository, name string, parent int64) models.Project {
	t.Helper()
	p := models.Project{Name: name, ProjectKind: "main", Status: "active", ParentProjectID: parent, AutoUnlock: true}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return p
}

func parentOfProject(t *testing.T, repo *Repository, id int64) int64 {
	t.Helper()
	p, err := repo.ProjectByID(id)
	if err != nil {
		t.Fatalf("project %d: %v", id, err)
	}
	return p.ParentProjectID
}

func setParent(t *testing.T, repo *Repository, id, parent int64) error {
	t.Helper()
	p, err := repo.ProjectByID(id)
	if err != nil {
		t.Fatalf("project %d: %v", id, err)
	}
	p.ParentProjectID = parent
	return repo.UpdateProject(&p)
}

func TestUpdateProjectRefusesSelfParentAndLeavesRowUntouched(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	before, _ := repo.ProjectByID(a.ID)
	versionOf := func() int {
		var v int
		if err := repo.db.QueryRow(`SELECT sync_version FROM projects WHERE id = ?`, a.ID).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	versionBefore := versionOf()

	err := setParent(t, repo, a.ID, a.ID)
	if !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("want ErrProjectParentCycle, got %v", err)
	}
	after, _ := repo.ProjectByID(a.ID)
	if after.ParentProjectID != 0 || versionOf() != versionBefore || after.UpdatedAt != before.UpdatedAt {
		t.Fatalf("refused write changed the row: before %#v after %#v", before, after)
	}
}

func TestUpdateProjectRefusesDescendantAsParent(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	b := cycleProject(t, repo, "Orbit Alpha", a.ID)
	c := cycleProject(t, repo, "Orbit Beta", b.ID)
	d := cycleProject(t, repo, "Orbit Gamma", 0)

	if err := setParent(t, repo, a.ID, b.ID); !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("A under its child: got %v", err)
	}
	if err := setParent(t, repo, a.ID, c.ID); !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("A under its grandchild: got %v", err)
	}
	if got := parentOfProject(t, repo, a.ID); got != 0 {
		t.Fatalf("A parent changed to %d", got)
	}
	// Moving B from A to D is fine, and clearing a parent is fine.
	if err := setParent(t, repo, b.ID, d.ID); err != nil {
		t.Fatalf("move B to D: %v", err)
	}
	if err := setParent(t, repo, b.ID, 0); err != nil {
		t.Fatalf("clear parent: %v", err)
	}
	if got := parentOfProject(t, repo, b.ID); got != 0 {
		t.Fatalf("B parent = %d, want none", got)
	}
}

func TestExistingCycleDoesNotHangAndCanBeRepaired(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	b := cycleProject(t, repo, "Orbit Alpha", a.ID)
	c := cycleProject(t, repo, "Orbit Beta", 0)
	if _, err := repo.db.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, b.ID, a.ID); err != nil {
		t.Fatalf("seed cycle: %v", err)
	}

	// Attaching under an existing cycle is not a new cycle and must return.
	if err := setParent(t, repo, c.ID, a.ID); err != nil {
		t.Fatalf("attach under existing cycle: %v", err)
	}
	desc, err := repo.ProjectDescendantIDs(a.ID)
	if err != nil {
		t.Fatalf("descendants: %v", err)
	}
	if !desc[b.ID] || !desc[c.ID] {
		t.Fatalf("descendants of A = %v, want B and C", desc)
	}
	// Setting the surfaced project's parent to None repairs it.
	if err := setParent(t, repo, a.ID, 0); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if got := parentOfProject(t, repo, a.ID); got != 0 {
		t.Fatalf("A parent = %d after repair", got)
	}
}

func TestMoveProjectRefusesCycles(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	b := cycleProject(t, repo, "Orbit Alpha", a.ID)
	if err := repo.MoveProject(a.ID, a.ID, "subproject"); !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("self move: %v", err)
	}
	if err := repo.MoveProject(a.ID, b.ID, "subproject"); !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("cycle move: %v", err)
	}
	got, _ := repo.ProjectByID(a.ID)
	if got.ParentProjectID != 0 || got.ProjectKind != "main" {
		t.Fatalf("refused move changed the row: %#v", got)
	}
	if err := repo.MoveProject(b.ID, 0, "main"); err != nil {
		t.Fatalf("valid move: %v", err)
	}
}

func projectsWithParents(pairs ...[2]int64) []models.ProjectWithProgress {
	out := make([]models.ProjectWithProgress, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, models.ProjectWithProgress{Project: models.Project{ID: p[0], ParentProjectID: p[1], Status: "active"}})
	}
	return out
}

func TestParentCycleRoots(t *testing.T) {
	cases := []struct {
		name string
		in   []models.ProjectWithProgress
		want map[int64]bool
	}{
		{"acyclic", projectsWithParents([2]int64{1, 0}, [2]int64{2, 1}, [2]int64{3, 2}), map[int64]bool{}},
		{"self loop", projectsWithParents([2]int64{1, 0}, [2]int64{4, 4}), map[int64]bool{4: true}},
		{"three cycle with tail", projectsWithParents([2]int64{9, 7}, [2]int64{7, 5}, [2]int64{5, 9}, [2]int64{8, 7}, [2]int64{1, 0}), map[int64]bool{5: true}},
		{"missing parent is not a cycle", projectsWithParents([2]int64{2, 99}, [2]int64{3, 2}), map[int64]bool{}},
		{"two separate cycles", projectsWithParents([2]int64{1, 2}, [2]int64{2, 1}, [2]int64{3, 4}, [2]int64{4, 3}), map[int64]bool{1: true, 3: true}},
	}
	for _, tc := range cases {
		if got := ParentCycleRoots(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func ids(list []models.ProjectWithProgress) []int64 {
	out := []int64{}
	for _, p := range list {
		out = append(out, p.Project.ID)
	}
	return out
}

func TestProjectTreeAcyclicAndCycles(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	b := cycleProject(t, repo, "Orbit Alpha", a.ID)
	tree, err := repo.ProjectTree()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(tree.Main), []int64{a.ID}) || !reflect.DeepEqual(ids(tree.Children[a.ID]), []int64{b.ID}) || len(tree.Upcoming) != 0 {
		t.Fatalf("acyclic tree changed: main %v children %v", ids(tree.Main), ids(tree.Children[a.ID]))
	}

	// Two-cycle: the lowest ID becomes a root, the other stays its child.
	if _, err := repo.db.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	tree, err = repo.ProjectTree()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids(tree.Main), []int64{a.ID}) || !reflect.DeepEqual(ids(tree.Children[a.ID]), []int64{b.ID}) {
		t.Fatalf("two-cycle tree: main %v, children of A %v", ids(tree.Main), ids(tree.Children[a.ID]))
	}
	// A self-parent shows once, in Main only.
	s := cycleProject(t, repo, "Orbit Gamma", 0)
	if _, err := repo.db.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, s.ID, s.ID); err != nil {
		t.Fatal(err)
	}
	tree, _ = repo.ProjectTree()
	count := 0
	for _, p := range tree.Main {
		if p.Project.ID == s.ID {
			count++
		}
	}
	if count != 1 || len(tree.Children[s.ID]) != 0 {
		t.Fatalf("self loop: in Main %d times, children %v", count, ids(tree.Children[s.ID]))
	}
}

func TestProjectDescendantIDsIncludesGrandchildrenAndArchived(t *testing.T) {
	repo := testRepository(t)
	a := cycleProject(t, repo, "Fictional Atlas", 0)
	b := cycleProject(t, repo, "Orbit Alpha", a.ID)
	c := cycleProject(t, repo, "Orbit Beta", b.ID)
	other := cycleProject(t, repo, "Orbit Gamma", 0)
	if err := repo.ArchiveProject(c.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ProjectDescendantIDs(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[int64]bool{b.ID: true, c.ID: true}) || got[other.ID] {
		t.Fatalf("descendants = %v", got)
	}
}

func TestImportRoadmapPlanRefusesParentCycleAndRollsBack(t *testing.T) {
	repo := testRepository(t)
	var before int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	var importsBefore int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM roadmap_imports`).Scan(&importsBefore); err != nil {
		t.Fatal(err)
	}
	plan := models.RoadmapImportPlan{
		SourceMarkdown: "fictional",
		ImportedDate:   "2026-01-01",
		Projects: []models.RoadmapProjectPlan{
			{Name: "Orbit Alpha", ProjectKind: "main", Status: "active", ParentName: "Orbit Beta"},
			{Name: "Orbit Beta", ProjectKind: "main", Status: "active", ParentName: "Orbit Alpha"},
		},
	}
	if _, err := repo.ImportRoadmapPlan(plan); !errors.Is(err, ErrProjectParentCycle) {
		t.Fatalf("want ErrProjectParentCycle, got %v", err)
	}
	var after, importsAfter int
	repo.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&after)
	repo.db.QueryRow(`SELECT COUNT(*) FROM roadmap_imports`).Scan(&importsAfter)
	if after != before || importsAfter != importsBefore {
		t.Fatalf("import left rows behind: projects %d->%d, imports %d->%d", before, after, importsBefore, importsAfter)
	}
}

func TestImportRoadmapPlanKeepsFirstPassParentWhenNamesCollide(t *testing.T) {
	repo := testRepository(t)
	plan := models.RoadmapImportPlan{
		SourceMarkdown: "fictional",
		ImportedDate:   "2026-01-01",
		Projects: []models.RoadmapProjectPlan{
			{Name: "Fictional Atlas", ProjectKind: "main", Status: "active"},
			{Name: "Fictional Atlas", ProjectKind: "phase", Status: "active", ParentName: "Fictional Atlas"},
		},
	}
	if _, err := repo.ImportRoadmapPlan(plan); err != nil {
		t.Fatalf("import: %v", err)
	}
	rows, err := repo.db.Query(`SELECT id, COALESCE(parent_project_id, 0) FROM projects WHERE name = 'Fictional Atlas' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var idsSeen, parents []int64
	for rows.Next() {
		var id, parent int64
		if err := rows.Scan(&id, &parent); err != nil {
			t.Fatal(err)
		}
		idsSeen = append(idsSeen, id)
		parents = append(parents, parent)
	}
	if len(idsSeen) != 2 || parents[0] != 0 || parents[1] != idsSeen[0] {
		t.Fatalf("ids %v parents %v: phase must sit under the main, not itself", idsSeen, parents)
	}
}
