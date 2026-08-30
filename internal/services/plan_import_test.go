package services

import (
	"strings"
	"testing"
	"time"
)

func TestParseAIPlanMarkdownPreservesHierarchyAndChecklist(t *testing.T) {
	source := `# Plan: Ship the widget
Description: A usable first release

## Project: Widget MVP
Category: coding
Target date: 2026-09-30

### Task: Build the API
Priority: high
Due: 2026-09-10
#### Checklist
- [x] Define schema
- [ ] Add endpoint`
	plan := ParseAIPlanMarkdown(source, time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC))
	if plan.Title != "Ship the widget" || len(plan.Projects) != 1 || len(plan.Projects[0].Tasks) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	task := plan.Projects[0].Tasks[0]
	if task.Priority != "high" || task.DueDate != "2026-09-10" || len(task.Checklist) != 2 {
		t.Fatalf("unexpected task: %#v", task)
	}
	if task.Checklist[0].Status != "completed" || task.Checklist[1].Title != "Add endpoint" {
		t.Fatalf("unexpected checklist: %#v", task.Checklist)
	}
}

func TestAIPlanPromptTemplateMatchesParserHeadings(t *testing.T) {
	for _, want := range []string{"# Plan:", "## Project:", "### Task:", "#### Checklist", "- [ ]"} {
		if !strings.Contains(AIPlanPromptTemplate, want) {
			t.Errorf("template missing %q", want)
		}
	}
}
