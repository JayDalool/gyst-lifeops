package handlers

import (
	"fmt"
	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
	"math"
	"net/http"
	"sort"
	"strconv"
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
	Week                           trainingWeekView
	Progress                       []trainingTrendView
	Groups                         []services.MuscleGroupShare
}

type trainingDayView struct {
	Letter, Name  string
	Done, IsToday bool
	Future        bool
}
type trainingWeekView struct {
	Workouts, Target int
	Days             []trainingDayView
	Volume, Time     string
	UnknownTime      int
}
type trainingTrendView struct {
	Name, Change, BestSet, Points string
	Sessions                      int
	Up                            bool
}

// thousands formats a whole number with comma separators.
func thousands(v float64) string {
	digits := strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return sign + digits
}

func trainingWeekViewOf(w services.TrainingWeek, target int) trainingWeekView {
	v := trainingWeekView{Workouts: w.Workouts, Target: target, UnknownTime: w.UnknownTime, Volume: thousands(w.VolumeKG) + " kg"}
	names := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for i, name := range names {
		v.Days = append(v.Days, trainingDayView{Letter: name[:1], Name: name, Done: w.Days[i], IsToday: i == w.Today, Future: i > w.Today})
	}
	switch {
	case w.Workouts == 0:
		v.Time = "0 min"
	case w.Minutes == 0 && w.UnknownTime > 0:
		v.Time = ""
	case w.Minutes >= 60:
		v.Time = fmt.Sprintf("%d h %d min", w.Minutes/60, w.Minutes%60)
	default:
		v.Time = fmt.Sprintf("%d min", w.Minutes)
	}
	return v
}

// sparkPoints maps a series onto a 100×32 SVG polyline.
func sparkPoints(series []float64) string {
	low, high := series[0], series[0]
	for _, v := range series {
		low, high = math.Min(low, v), math.Max(high, v)
	}
	points := make([]string, len(series))
	for i, v := range series {
		y := 16.0
		if high > low {
			y = 28 - 24*(v-low)/(high-low)
		}
		points[i] = fmt.Sprintf("%.1f,%.1f", 100*float64(i)/float64(len(series)-1), y)
	}
	return strings.Join(points, " ")
}

func trainingTrendViews(trends []services.ExerciseTrend, limit int) []trainingTrendView {
	out := []trainingTrendView{}
	for _, t := range trends {
		if len(out) == limit {
			break
		}
		change := fmt.Sprintf("%+d%%", t.ChangePct)
		if t.ChangePct < 0 {
			change = fmt.Sprintf("−%d%%", -t.ChangePct)
		}
		out = append(out, trainingTrendView{Name: t.Name, Change: change, BestSet: t.BestSet, Points: sparkPoints(t.Series), Sessions: t.Sessions, Up: t.ChangePct > 0})
	}
	return out
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
	now, location := h.Service.Now(), h.Service.Location()
	data.Summary = services.SummarizeTraining(cache.Entries, data.Mode, data.Routine, now)
	target := 0
	if date, err := h.Service.HealthTodayDate(); err == nil {
		if cycle, err := h.Service.GetHealthCycleForDate(date); err == nil {
			target = int(cycle.StrengthSessionsPerWeek)
		}
	}
	data.Week = trainingWeekViewOf(services.SummarizeTrainingWeek(cache.Entries, now, location), target)
	data.Progress = trainingTrendViews(services.ExerciseProgress(cache.Entries, now, 84), 3)
	data.Groups = services.MuscleGroupShares(cache.Entries, now, 30)
	h.render(w, "training.html", data)
}
