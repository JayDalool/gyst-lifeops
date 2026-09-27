package services

import (
	"context"
	"errors"
	"lifeops-tracker/internal/models"
	"sync"
	"time"
)

type TrainingProvider interface {
	Fetch(context.Context) ([]models.TrainingEntry, error)
}
type TrainingSync struct {
	Provider    TrainingProvider
	mu          sync.Mutex
	lastAttempt time.Time
}

func (s *Service) SyncTraining(ctx context.Context) error {
	if s.Training == nil || s.Training.Provider == nil {
		return errors.New("Hevy is not connected")
	}
	t := s.Training
	if !t.mu.TryLock() {
		return errors.New("A training sync is already running")
	}
	defer t.mu.Unlock()
	if !t.lastAttempt.IsZero() && s.Now().Sub(t.lastAttempt) < time.Minute {
		return errors.New("Please wait one minute before syncing again")
	}
	t.lastAttempt = s.Now()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	entries, err := t.Provider.Fetch(ctx)
	if err != nil {
		_ = s.Repo.TrainingFailure("hevy", err.Error())
		return err
	}
	return s.Repo.SaveTraining("hevy", entries, s.Now())
}
func (s *Service) RunTrainingSync(ctx context.Context) {
	if s.Training == nil {
		return
	}
	ticker := time.NewTicker(37 * time.Minute)
	defer ticker.Stop()
	for {
		_ = s.SyncTraining(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
