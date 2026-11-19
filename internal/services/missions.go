package services

// Mission and project-status use cases (GYST-VN-038).
//
// Each method is a single entry point for a multi-write operation. The
// repository method it calls owns the database transaction, so the item,
// stars, proof achievement, project status, "Quest complete" bonus and
// dependent unlock either all commit or all roll back. Handlers (and any future
// non-HTML client) call these methods rather than the repository directly.

// CompleteMission marks a project mission completed and applies every
// follow-on write atomically.
func (s *Service) CompleteMission(projectID, itemID int64) error {
	return s.Repo.SetProjectItemStatus(projectID, itemID, "completed")
}

// ReopenMission returns a mission to pending and reconciles the project status
// atomically.
func (s *Service) ReopenMission(projectID, itemID int64) error {
	return s.Repo.SetProjectItemStatus(projectID, itemID, "pending")
}

// SetProjectStatus changes a project's status; marking it done also awards the
// one-time Quest bonus and unlocks dependents in the same transaction.
func (s *Service) SetProjectStatus(id int64, status string) error {
	return s.Repo.SetProjectStatus(id, status)
}

// SetTaskOccurrenceStatus sets a task occurrence's status. The repository
// already runs the occurrence, stars and recurrence writes in one transaction.
func (s *Service) SetTaskOccurrenceStatus(taskID int64, occurrenceDate, status string) error {
	return s.Repo.SetTaskOccurrenceStatus(taskID, occurrenceDate, status)
}
