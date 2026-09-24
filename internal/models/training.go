package models

import "time"

// Training entries are normalized snapshots, independent of provider payloads.
type TrainingSet struct {
	Kind                                                       string
	LoadKG, Reps, MinReps, MaxReps, DistanceM, Seconds, Effort *float64
}
type TrainingExercise struct {
	Name, PrimaryMuscle string
	SecondaryMuscles    []string
	Sets                []TrainingSet
}
type TrainingEntry struct {
	ID, Kind, Title string
	StartedAt       time.Time
	Exercises       []TrainingExercise
}
type TrainingCache struct {
	Entries  []TrainingEntry
	SyncedAt time.Time
	Error    string
}
