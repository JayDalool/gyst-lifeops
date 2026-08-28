package services

import (
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func TestParseRoadmapMarkdownQuestHierarchy(t *testing.T) {
	today := time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC)
	source := `# Launch Career Campaign

## Portfolio Quest
Status: active
Notes: Keep this visible.

### MVP Chapter
Status: locked
Depends On: Portfolio Quest
Auto Unlock: true
Unlock Notes: Start after portfolio shell is done.
- [ ] Publish demo Stars: 5 Proof Required: link
`

	plan := ParseRoadmapMarkdown(source, today)

	if plan.Goal.Title != "Launch Career Campaign" {
		t.Fatalf("goal title = %q", plan.Goal.Title)
	}
	if len(plan.Projects) != 2 {
		t.Fatalf("projects length = %d, want 2", len(plan.Projects))
	}
	if plan.Projects[0].ProjectKind != "main" {
		t.Errorf("main project kind = %q", plan.Projects[0].ProjectKind)
	}
	chapter := plan.Projects[1]
	if chapter.ProjectKind != "phase" || chapter.ParentName != "Portfolio Quest" {
		t.Errorf("chapter kind/parent = %q/%q", chapter.ProjectKind, chapter.ParentName)
	}
	if chapter.Status != "locked" || chapter.DependsOn != "Portfolio Quest" || !chapter.AutoUnlock {
		t.Errorf("chapter lock metadata = status %q depends %q auto %t", chapter.Status, chapter.DependsOn, chapter.AutoUnlock)
	}
	if chapter.UnlockNotes == "" {
		t.Error("expected unlock notes")
	}
	if len(chapter.Tasks) != 1 {
		t.Fatalf("chapter tasks = %d, want 1", len(chapter.Tasks))
	}
	if chapter.Tasks[0].StarsReward != 5 || chapter.Tasks[0].ProofType != "link" || !chapter.Tasks[0].ProofRequired {
		t.Errorf("task rewards = stars %d proof %q required %t", chapter.Tasks[0].StarsReward, chapter.Tasks[0].ProofType, chapter.Tasks[0].ProofRequired)
	}
}

func TestImportResolvesForwardDependencyByProjectName(t *testing.T) {
	service := testService(t)
	service.Now = func() time.Time { return time.Date(2026, time.June, 24, 12, 0, 0, 0, time.UTC) }
	source := `# Campaign

## First Quest
Status: locked
Depends On: Later Quest
Auto Unlock: true

## Later Quest
Status: active
`
	if _, err := service.ImportRoadmapMarkdown(source); err != nil {
		t.Fatalf("import roadmap: %v", err)
	}
	projects, err := service.Repo.ListProjects(false)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]models.Project)
	for _, project := range projects {
		byName[project.Project.Name] = project.Project
	}
	if byName["First Quest"].DependsOnProjectID != byName["Later Quest"].ID {
		t.Fatalf("forward dependency = %d, want %d", byName["First Quest"].DependsOnProjectID, byName["Later Quest"].ID)
	}
}
