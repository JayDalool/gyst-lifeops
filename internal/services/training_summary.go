package services

import (
	"lifeops-tracker/internal/models"
	"sort"
	"strings"
	"time"
)

type MuscleCount struct {
	ID, Name                  string
	Primary, Secondary, Level int
}
type TrainingSummary struct {
	Muscles                     []MuscleCount
	Heat                        map[string]int
	Sets, Unclassified, Entries int
}

var muscleNames = map[string]string{"abdominals": "Abdominals", "shoulders": "Shoulders", "biceps": "Biceps", "triceps": "Triceps", "forearms": "Forearms", "quadriceps": "Quadriceps", "hamstrings": "Hamstrings", "calves": "Calves", "glutes": "Glutes", "abductors": "Abductors", "adductors": "Adductors", "lats": "Lats", "upper_back": "Upper back", "traps": "Traps", "lower_back": "Lower back", "chest": "Chest", "cardio": "Cardio", "neck": "Neck", "full_body": "Full body", "other": "Other"}

func MuscleName(id string) string {
	if name, ok := muscleNames[id]; ok {
		return name
	}
	return "Unclassified"
}
func SummarizeTraining(entries []models.TrainingEntry, mode, routine string, now time.Time) TrainingSummary {
	result := TrainingSummary{Heat: map[string]int{}}
	counts := map[string]*MuscleCount{}
	for id, name := range muscleNames {
		counts[id] = &MuscleCount{ID: id, Name: name}
	}
	for _, entry := range entries {
		if mode == "planned" {
			if entry.Kind != "routine" || (routine != "" && routine != entry.ID) {
				continue
			}
		} else {
			days := 7
			if mode == "30" {
				days = 30
			}
			if entry.Kind != "workout" || entry.StartedAt.Before(now.AddDate(0, 0, -days)) || entry.StartedAt.After(now) {
				continue
			}
		}
		result.Entries++
		for _, ex := range entry.Exercises {
			for _, set := range ex.Sets {
				if set.Kind != "normal" && set.Kind != "failure" && set.Kind != "dropset" {
					continue
				}
				result.Sets++
				if c := counts[ex.PrimaryMuscle]; c != nil {
					c.Primary++
				} else {
					result.Unclassified++
				}
				seen := map[string]bool{ex.PrimaryMuscle: true}
				for _, id := range ex.SecondaryMuscles {
					if !seen[id] && counts[id] != nil {
						counts[id].Secondary++
						seen[id] = true
					}
				}
			}
		}
	}
	max := 0
	for _, c := range counts {
		if c.Primary > max {
			max = c.Primary
		}
	}
	for _, c := range counts {
		if c.Primary > 0 {
			c.Level = 1 + (c.Primary*3-1)/max
		}
		result.Heat[c.ID] = c.Level
		if c.Primary > 0 || c.Secondary > 0 {
			result.Muscles = append(result.Muscles, *c)
		}
	}
	sort.Slice(result.Muscles, func(i, j int) bool {
		a, b := result.Muscles[i], result.Muscles[j]
		if a.Primary != b.Primary {
			return a.Primary > b.Primary
		}
		if a.Secondary != b.Secondary {
			return a.Secondary > b.Secondary
		}
		return strings.Compare(a.Name, b.Name) < 0
	})
	return result
}
