package handlers

import (
	"fmt"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
	"net/http"
	"sort"
	"strings"
	"time"
)

type trainingExerciseView struct {
	Name, Muscles string
	Sets          []string
}
type trainingEntryView struct {
	ID, Title, Date string
	Exercises       []trainingExerciseView
}
type trainingData struct {
	HealthData
	Connected, HasData, Stale      bool
	LastSync, Error, Mode, Routine string
	Routines, Workouts             []trainingEntryView
	Summary                        services.TrainingSummary
}

func trainingView(e models.TrainingEntry) trainingEntryView {
	v := trainingEntryView{ID: e.ID, Title: e.Title}
	if !e.StartedAt.IsZero() {
		v.Date = e.StartedAt.Format("Jan 2, 2006 · 15:04 MST")
	}
	for _, ex := range e.Exercises {
		x := trainingExerciseView{Name: ex.Name, Muscles: services.MuscleName(ex.PrimaryMuscle)}
		for _, s := range ex.Sets {
			parts := []string{s.Kind}
			if s.Reps != nil {
				parts = append(parts, fmt.Sprintf("%g reps", *s.Reps))
			} else if s.MinReps != nil && s.MaxReps != nil {
				parts = append(parts, fmt.Sprintf("%g–%g reps", *s.MinReps, *s.MaxReps))
			}
			if s.LoadKG != nil {
				parts = append(parts, fmt.Sprintf("%g kg", *s.LoadKG))
			}
			if s.DistanceM != nil {
				parts = append(parts, fmt.Sprintf("%g m", *s.DistanceM))
			}
			if s.Seconds != nil {
				parts = append(parts, fmt.Sprintf("%g sec", *s.Seconds))
			}
			if s.Effort != nil {
				parts = append(parts, fmt.Sprintf("RPE %g", *s.Effort))
			}
			x.Sets = append(x.Sets, strings.Join(parts, " · "))
		}
		v.Exercises = append(v.Exercises, x)
	}
	return v
}
func (h *Handler) trainingSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		methodNotAllowed(w)
		return
	}
	if requireSameOrigin(w, r) {
		return
	}
	if err := h.Service.SyncTraining(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, "/health/training", http.StatusSeeOther)
}
func (h *Handler) training(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	mission, err := h.Service.DailyMission()
	if err != nil {
		serverError(w, err)
		return
	}
	cache, err := h.Service.Repo.TrainingCache("hevy")
	if err != nil {
		serverError(w, err)
		return
	}
	data := trainingData{HealthData: HealthData{Page: "health", Mission: mission}, Connected: h.Service.Training != nil, HasData: !cache.SyncedAt.IsZero(), Error: cache.Error, Mode: r.URL.Query().Get("mode"), Routine: r.URL.Query().Get("routine")}
	if data.Mode != "7" && data.Mode != "30" {
		data.Mode = "planned"
	}
	if data.HasData {
		data.LastSync = cache.SyncedAt.UTC().Format("Jan 2, 15:04 UTC")
		data.Stale = h.Service.Now().Sub(cache.SyncedAt) > 2*time.Hour
	}
	valid := data.Routine == ""
	sort.SliceStable(cache.Entries, func(i, j int) bool { return cache.Entries[i].StartedAt.After(cache.Entries[j].StartedAt) })
	for _, e := range cache.Entries {
		if e.Kind == "routine" {
			data.Routines = append(data.Routines, trainingView(e))
			if e.ID == data.Routine {
				valid = true
			}
		} else if len(data.Workouts) < 30 {
			data.Workouts = append(data.Workouts, trainingView(e))
		}
	}
	if !valid {
		http.NotFound(w, r)
		return
	}
	data.Summary = services.SummarizeTraining(cache.Entries, data.Mode, data.Routine, h.Service.Now())
	h.render(w, "training.html", data)
}
