package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

const AIPlanPromptTemplate = `You are planning work for my LifeOps app. Return ONLY one Markdown document using the exact structure below. Be specific, split work into small actionable tasks, and include acceptance-oriented checklist items. Do not use tables or fenced code blocks.

# Plan: <short plan title>
Description: <what success looks like and important context>

## Project: <project name>
Description: <project outcome>
Category: coding|business|health|personal|other
Status: active|upcoming|locked
Target date: YYYY-MM-DD

### Task: <actionable task title>
Description: <details needed to complete it>
Priority: low|medium|high
Due: YYYY-MM-DD
Status: pending|completed

#### Checklist
- [ ] <concrete subtask or acceptance check>
- [ ] <concrete subtask or acceptance check>

Repeat the Project and Task sections as needed. Omit unknown optional metadata instead of guessing dates. Every checklist item must belong to the Task directly above it.`

func ParseAIPlanMarkdown(source string, today time.Time) models.AIPlan {
	var plan models.AIPlan
	projectIndex, taskIndex := -1, -1
	inChecklist := false
	for _, raw := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(strings.ToLower(line), "# plan:"):
			plan.Title = strings.TrimSpace(line[len("# Plan:"):])
			projectIndex, taskIndex, inChecklist = -1, -1, false
		case strings.HasPrefix(strings.ToLower(line), "## project:"):
			plan.Projects = append(plan.Projects, models.AIPlanProject{Name: strings.TrimSpace(line[len("## Project:"):]), Category: "other", Status: "active"})
			projectIndex, taskIndex, inChecklist = len(plan.Projects)-1, -1, false
		case strings.HasPrefix(strings.ToLower(line), "### task:"):
			task := models.AIPlanTask{Title: strings.TrimSpace(line[len("### Task:"):]), Category: "personal", Priority: "medium", Status: "pending"}
			if projectIndex >= 0 {
				plan.Projects[projectIndex].Tasks = append(plan.Projects[projectIndex].Tasks, task)
				taskIndex = len(plan.Projects[projectIndex].Tasks) - 1
			} else {
				plan.Tasks = append(plan.Tasks, task)
				taskIndex = len(plan.Tasks) - 1
			}
			inChecklist = false
		case strings.HasPrefix(strings.ToLower(line), "#### checklist"):
			inChecklist = true
		case inChecklist && (strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(strings.ToLower(line), "- [x]")):
			item := models.TaskChecklistItem{Title: strings.TrimSpace(line[5:]), Status: "pending"}
			if strings.HasPrefix(strings.ToLower(line), "- [x]") {
				item.Status = "completed"
			}
			appendPlanChecklist(&plan, projectIndex, taskIndex, item)
		default:
			key, value, ok := metadataLine(line)
			if !ok {
				continue
			}
			applyAIPlanMetadata(&plan, projectIndex, taskIndex, strings.ToLower(key), value, today)
		}
	}
	if plan.Title == "" {
		plan.Title = "Imported plan"
	}
	return plan
}

func appendPlanChecklist(plan *models.AIPlan, pi, ti int, item models.TaskChecklistItem) {
	if ti < 0 {
		return
	}
	if pi >= 0 {
		plan.Projects[pi].Tasks[ti].Checklist = append(plan.Projects[pi].Tasks[ti].Checklist, item)
	} else {
		plan.Tasks[ti].Checklist = append(plan.Tasks[ti].Checklist, item)
	}
}

func applyAIPlanMetadata(plan *models.AIPlan, pi, ti int, key, value string, today time.Time) {
	if ti >= 0 {
		var task *models.AIPlanTask
		if pi >= 0 {
			task = &plan.Projects[pi].Tasks[ti]
		} else {
			task = &plan.Tasks[ti]
		}
		switch key {
		case "description", "details":
			task.Description = value
		case "category":
			task.Category = value
		case "priority":
			task.Priority = value
		case "status":
			task.Status = value
		case "due", "due date":
			task.DueDate = parseFlexibleDate(value, today)
		}
		return
	}
	if pi >= 0 {
		p := &plan.Projects[pi]
		switch key {
		case "description":
			p.Description = value
		case "category":
			p.Category = value
		case "status":
			p.Status = value
		case "target", "target date", "due":
			p.TargetDate = parseFlexibleDate(value, today)
		}
		return
	}
	if key == "description" {
		plan.Description = value
	}
}

func (s *Service) ImportAIPlan(plan models.AIPlan, destination string, targetID int64) (string, error) {
	switch destination {
	case "project":
		if _, err := s.Repo.ProjectByID(targetID); err != nil {
			return "", err
		}
		order := 0
		for _, task := range allAIPlanTasks(plan) {
			order++
			if err := s.createPlanProjectItem(targetID, order, task); err != nil {
				return "", err
			}
		}
		return "/projects/" + strconv.FormatInt(targetID, 10), nil
	case "task":
		if _, err := s.Repo.TaskByID(targetID); err != nil {
			return "", err
		}
		items := planChecklistItems(plan)
		if len(items) == 0 {
			for _, task := range allAIPlanTasks(plan) {
				items = append(items, models.TaskChecklistItem{Title: task.Title, Status: task.Status})
			}
		}
		if err := s.Repo.AddTaskChecklistItems(targetID, items); err != nil {
			return "", err
		}
		return "/tasks/" + strconv.FormatInt(targetID, 10), nil
	case "standalone":
		var firstID int64
		for _, input := range allAIPlanTasks(plan) {
			task := models.Task{Title: input.Title, Description: input.Description, Category: input.Category, Priority: input.Priority, Status: input.Status, DueDate: input.DueDate, RepeatType: "none", ProofType: "none"}
			if err := s.Repo.CreateTask(&task); err != nil {
				return "", err
			}
			if firstID == 0 {
				firstID = task.ID
			}
			if err := s.Repo.AddTaskChecklistItems(task.ID, input.Checklist); err != nil {
				return "", err
			}
		}
		if firstID == 0 {
			return "", fmt.Errorf("the plan contains no tasks")
		}
		return "/tasks/" + strconv.FormatInt(firstID, 10), nil
	default:
		var firstID int64
		projects := plan.Projects
		if len(projects) == 0 {
			projects = []models.AIPlanProject{{Name: plan.Title, Description: plan.Description, Category: "other", Status: "active", Tasks: plan.Tasks}}
		}
		for _, input := range projects {
			if strings.TrimSpace(input.Name) == "" {
				continue
			}
			project := models.Project{Name: input.Name, Description: input.Description, Category: input.Category, Status: input.Status, ProjectKind: "main", TargetDate: input.TargetDate}
			if err := s.Repo.CreateProject(&project); err != nil {
				return "", err
			}
			if firstID == 0 {
				firstID = project.ID
			}
			for i, task := range input.Tasks {
				if err := s.createPlanProjectItem(project.ID, i+1, task); err != nil {
					return "", err
				}
			}
		}
		if firstID == 0 {
			return "", fmt.Errorf("the plan contains no project")
		}
		return "/projects/" + strconv.FormatInt(firstID, 10), nil
	}
}

func (s *Service) createPlanProjectItem(projectID int64, order int, input models.AIPlanTask) error {
	description := input.Description
	if len(input.Checklist) > 0 {
		if description != "" {
			description += "\n\n"
		}
		description += "Checklist:\n"
		for _, item := range input.Checklist {
			mark := " "
			if item.Status == "completed" {
				mark = "x"
			}
			description += fmt.Sprintf("- [%s] %s\n", mark, item.Title)
		}
	}
	return s.Repo.CreateProjectItem(&models.ProjectItem{ProjectID: projectID, Title: input.Title, Description: strings.TrimSpace(description), Status: input.Status, Priority: input.Priority, DueDate: input.DueDate, SortOrder: order})
}

func allAIPlanTasks(plan models.AIPlan) []models.AIPlanTask {
	tasks := append([]models.AIPlanTask{}, plan.Tasks...)
	for _, p := range plan.Projects {
		tasks = append(tasks, p.Tasks...)
	}
	return tasks
}
func planChecklistItems(plan models.AIPlan) []models.TaskChecklistItem {
	var items []models.TaskChecklistItem
	for _, task := range allAIPlanTasks(plan) {
		items = append(items, task.Checklist...)
	}
	return items
}
