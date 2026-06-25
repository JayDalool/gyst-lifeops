package repository

import (
	"database/sql"
	"strings"

	"lifeops-tracker/internal/models"
)

type Repository struct {
	db *sql.DB
}

type scanner interface {
	Scan(dest ...any) error
}

func New(database *sql.DB) *Repository {
	return &Repository{db: database}
}

func nullString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func clean(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func progressPercent(current, target float64) int {
	if target <= 0 {
		return 0
	}
	progress := int((current / target) * 100)
	if progress > 100 {
		return 100
	}
	if progress < 0 {
		return 0
	}
	return progress
}

func itemProgress(completed, total int) int {
	if total == 0 {
		return 0
	}
	progress := (completed * 100) / total
	if progress > 100 {
		return 100
	}
	return progress
}

func scanTask(s scanner) (models.Task, error) {
	var task models.Task
	var proofRequired int
	err := s.Scan(
		&task.ID,
		&task.UUID,
		&task.Title,
		&task.Description,
		&task.Category,
		&task.Priority,
		&task.Status,
		&task.DueDate,
		&task.RepeatType,
		&task.ReminderTime,
		&task.StarsReward,
		&proofRequired,
		&task.ProofType,
		&task.ProofURL,
		&task.ProofPath,
		&task.StarsAwarded,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.CompletedAt,
		&task.ArchivedAt,
	)
	task.ProofRequired = proofRequired == 1
	return task, err
}

func scanProject(s scanner) (models.Project, error) {
	var project models.Project
	var autoUnlock int
	err := s.Scan(
		&project.ID,
		&project.UUID,
		&project.ParentProjectID,
		&project.Name,
		&project.Description,
		&project.Category,
		&project.ProjectKind,
		&project.Status,
		&project.StartDate,
		&project.TargetDate,
		&project.Notes,
		&project.UnlockNotes,
		&project.DependsOnProjectID,
		&autoUnlock,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.ArchivedAt,
	)
	project.AutoUnlock = autoUnlock == 1
	return project, err
}

func scanGoal(s scanner) (models.Goal, error) {
	var goal models.Goal
	err := s.Scan(
		&goal.ID,
		&goal.UUID,
		&goal.Title,
		&goal.Description,
		&goal.Type,
		&goal.Category,
		&goal.TargetValue,
		&goal.CurrentValue,
		&goal.Unit,
		&goal.StartDate,
		&goal.EndDate,
		&goal.Status,
		&goal.CreatedAt,
		&goal.UpdatedAt,
		&goal.ArchivedAt,
	)
	return goal, err
}
