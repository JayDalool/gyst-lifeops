package services

import (
	"context"
	"errors"
	"lifeops-tracker/internal/models"
	"testing"
	"time"
)

type trainingFixture struct{ fail bool }

func (f *trainingFixture) Fetch(context.Context) ([]models.TrainingEntry, error) {
	if f.fail {
		return nil, errors.New("Unavailable")
	}
	return []models.TrainingEntry{{ID: "one", Kind: "routine", Title: "Real cached title"}}, nil
}
func TestTrainingFailurePreservesSnapshot(t *testing.T) {
	s := testService(t)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	f := &trainingFixture{}
	s.Training = &TrainingSync{Provider: f}
	if err := s.SyncTraining(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncTraining(context.Background()); err == nil {
		t.Fatal("no cooldown")
	}
	now = now.Add(2 * time.Minute)
	f.fail = true
	if err := s.SyncTraining(context.Background()); err == nil {
		t.Fatal("wanted failure")
	}
	c, err := s.Repo.TrainingCache("hevy")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Entries) != 1 || c.Entries[0].Title != "Real cached title" || c.SyncedAt.Equal(now) || c.Error == "" {
		t.Fatalf("cache damaged: %+v", c)
	}
}
