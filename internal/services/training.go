package services

import (
	"context"
	"lifeops-tracker/internal/models"
	"log/slog"
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
		return userError("Hevy is not connected")
	}
	t := s.Training
	if !t.mu.TryLock() {
		return userError("A training sync is already running")
	}
	defer t.mu.Unlock()
	if !t.lastAttempt.IsZero() && s.Now().Sub(t.lastAttempt) < time.Minute {
		return userError("Please wait one minute before syncing again")
	}
	t.lastAttempt = s.Now()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	entries, err := t.Provider.Fetch(ctx)
	if err != nil {
		// The Hevy client returns only fixed, curated text (never the request,
		// the key or the response body), so it is safe to store, log and show.
		_ = s.Repo.TrainingFailure("hevy", err.Error())
		slog.Warn("provider failure", "event", "provider.failed", "provider", "hevy", "error", err.Error())
		return userError(err.Error())
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
