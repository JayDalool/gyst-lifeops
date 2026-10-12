package repository

import (
	"testing"

	"lifeops-tracker/internal/models"
)

func privTestProject(t *testing.T, repo *Repository, name string, parent int64, private, localOnly bool) int64 {
	t.Helper()
	p := models.Project{Name: name, ProjectKind: "phase", Status: "active", ParentProjectID: parent}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := repo.db.Exec(`INSERT INTO roadmap_project_meta (project_id, is_private, local_only) VALUES (?, ?, ?)`,
		p.ID, boolFlag(private), boolFlag(localOnly)); err != nil {
		t.Fatalf("meta %s: %v", name, err)
	}
	return p.ID
}

func TestEffectiveProjectPrivacyInheritsFromAncestors(t *testing.T) {
	repo := testRepository(t)
	priv := privTestProject(t, repo, "Fictional Garden Planner", 0, true, false)
	phase := privTestProject(t, repo, "Seed Catalog", priv, false, false)
	sub := privTestProject(t, repo, "Seed Labels", phase, false, false)
	local := privTestProject(t, repo, "Fictional Kite Log", 0, false, true)
	localChild := privTestProject(t, repo, "Kite Phase", local, false, false)
	pub := privTestProject(t, repo, "Fictional Open Book", 0, false, false)
	pubChild := privTestProject(t, repo, "Open Phase", pub, false, false)
	archParent := privTestProject(t, repo, "Fictional Old Plan", 0, true, false)
	if _, err := repo.db.Exec(`UPDATE projects SET archived_at = CURRENT_TIMESTAMP WHERE id = ?`, archParent); err != nil {
		t.Fatal(err)
	}
	archChild := privTestProject(t, repo, "Old Plan Phase", archParent, false, false)
	// Child without any meta row under a private parent.
	noMeta := models.Project{Name: "No Meta Phase", ProjectKind: "phase", Status: "active", ParentProjectID: priv}
	if err := repo.CreateProject(&noMeta); err != nil {
		t.Fatal(err)
	}

	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{priv, phase, sub, archParent, archChild, noMeta.ID} {
		if !got[id].Private || !got[id].Hidden() {
			t.Errorf("project %d should be private-hidden: %+v", id, got[id])
		}
	}
	for _, id := range []int64{local, localChild} {
		if !got[id].LocalOnly || !got[id].Hidden() {
			t.Errorf("project %d should be local-only hidden: %+v", id, got[id])
		}
	}
	for _, id := range []int64{pub, pubChild} {
		if _, ok := got[id]; ok {
			t.Errorf("project %d should be public", id)
		}
	}
}

func TestEffectiveProjectPrivacyTerminatesOnCycles(t *testing.T) {
	repo := testRepository(t)
	self := privTestProject(t, repo, "Fictional Self Loop", 0, true, false)
	a := privTestProject(t, repo, "Fictional Cycle A", 0, true, false)
	b := privTestProject(t, repo, "Fictional Cycle B", 0, false, false)
	for _, q := range [][2]int64{{self, self}, {a, b}, {b, a}} {
		if _, err := repo.db.Exec(`UPDATE projects SET parent_project_id = ? WHERE id = ?`, q[1], q[0]); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{self, a, b} {
		if !got[id].Hidden() {
			t.Errorf("project %d should be hidden", id)
		}
	}
}
