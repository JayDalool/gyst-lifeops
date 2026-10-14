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
	// Migration 006 makes the default roadmap private; this test covers
	// project-level flags, so make that roadmap public first.
	if _, err := repo.db.Exec(`UPDATE roadmaps SET privacy = 'public'`); err != nil {
		t.Fatal(err)
	}
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

func privTestRoadmap(t *testing.T, repo *Repository, name, privacy string) int64 {
	t.Helper()
	rm := models.Roadmap{Name: name, Status: "active", Privacy: privacy}
	if err := repo.CreateRoadmap(&rm); err != nil {
		t.Fatalf("create roadmap %s: %v", name, err)
	}
	return rm.ID
}

// privTestIn creates a project (no meta flags) in the given roadmap (0 = orphan).
func privTestIn(t *testing.T, repo *Repository, name string, parent, roadmapID int64) int64 {
	t.Helper()
	p := models.Project{Name: name, ProjectKind: "phase", Status: "active", ParentProjectID: parent}
	if err := repo.CreateProject(&p); err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	var rid any
	if roadmapID != 0 {
		rid = roadmapID
	}
	if _, err := repo.db.Exec(`UPDATE projects SET roadmap_id = ? WHERE id = ?`, rid, p.ID); err != nil {
		t.Fatalf("assign %s: %v", name, err)
	}
	return p.ID
}

func TestEffectiveProjectPrivacyFromRoadmap(t *testing.T) {
	repo := testRepository(t)
	// Take the migrated default roadmap out of play.
	if _, err := repo.db.Exec(`UPDATE roadmaps SET status = 'archived', archived_at = CURRENT_TIMESTAMP, featured = 0`); err != nil {
		t.Fatal(err)
	}
	pubRM := privTestRoadmap(t, repo, "Fictional Public Trips", "public")
	privRM := privTestRoadmap(t, repo, "Fictional Private Trips", "private")
	locRM := privTestRoadmap(t, repo, "Fictional Local Trips", "local_only")
	pubMain := privTestIn(t, repo, "Public Main", 0, pubRM)
	pubPhase := privTestIn(t, repo, "Public Phase", pubMain, pubRM)
	privMain := privTestIn(t, repo, "Private Main", 0, privRM)
	privPhase := privTestIn(t, repo, "Private Phase", privMain, privRM)
	locMain := privTestIn(t, repo, "Local Main", 0, locRM)
	locPhase := privTestIn(t, repo, "Local Phase", locMain, locRM)
	// Phase points at a public roadmap but its parent is in a private one.
	crossPhase := privTestIn(t, repo, "Cross Phase", privMain, pubRM)

	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{privMain, privPhase, crossPhase} {
		if !got[id].Private {
			t.Errorf("project %d should be Private: %+v", id, got[id])
		}
	}
	for _, id := range []int64{locMain, locPhase} {
		if !got[id].LocalOnly || got[id].Private {
			t.Errorf("project %d should be LocalOnly: %+v", id, got[id])
		}
	}
	for _, id := range []int64{pubMain, pubPhase} {
		if _, ok := got[id]; ok {
			t.Errorf("project %d should be public", id)
		}
	}
}

func TestEffectiveProjectPrivacyOrphanFollowsFeaturedRoadmap(t *testing.T) {
	repo := testRepository(t)
	if _, err := repo.db.Exec(`UPDATE roadmaps SET featured = 0`); err != nil {
		t.Fatal(err)
	}
	rm := models.Roadmap{Name: "Fictional Featured Trips", Status: "active", Privacy: "private", Featured: true}
	if err := repo.CreateRoadmap(&rm); err != nil {
		t.Fatal(err)
	}
	orphan := privTestIn(t, repo, "Fictional Orphan", 0, 0)
	orphanPhase := privTestIn(t, repo, "Fictional Orphan Phase", orphan, 0)

	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	if !got[orphan].Private || !got[orphanPhase].Private {
		t.Errorf("orphans should follow the private featured roadmap: %+v %+v", got[orphan], got[orphanPhase])
	}

	rm.Privacy = "public"
	if err := repo.UpdateRoadmap(&rm); err != nil {
		t.Fatal(err)
	}
	// The default roadmap is still private but not featured; the featured one wins.
	got, err = repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{orphan, orphanPhase} {
		if _, ok := got[id]; ok {
			t.Errorf("orphan %d should be public once its roadmap is public", id)
		}
	}
}

func TestEffectiveProjectPrivacyOrphanWithoutActiveRoadmap(t *testing.T) {
	repo := testRepository(t)
	orphan := privTestIn(t, repo, "Fictional Orphan", 0, 0)
	if _, err := repo.db.Exec(`UPDATE roadmaps SET status = 'archived', archived_at = CURRENT_TIMESTAMP, featured = 0`); err != nil {
		t.Fatal(err)
	}
	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[orphan]; ok {
		t.Errorf("orphan should not be hidden when no roadmap is active: %+v", got[orphan])
	}
}

func TestEffectiveProjectPrivacyArchivedPrivateRoadmapStillHides(t *testing.T) {
	repo := testRepository(t)
	rm := privTestRoadmap(t, repo, "Fictional Old Trips", "private")
	main := privTestIn(t, repo, "Old Main", 0, rm)
	if err := repo.ArchiveRoadmap(rm); err != nil {
		t.Fatal(err)
	}
	got, err := repo.EffectiveProjectPrivacy()
	if err != nil {
		t.Fatal(err)
	}
	if !got[main].Private {
		t.Errorf("archived private roadmap must still hide its project: %+v", got[main])
	}
}
