package services

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

// Work sets count toward volume, progress and muscle shares. Warmups don't.
func isWorkSet(s models.TrainingSet) bool {
	return s.Kind == "normal" || s.Kind == "failure" || s.Kind == "dropset"
}

// dayIndex is the number of local calendar days from start to t. Rounding
// absorbs the 23- and 25-hour days at DST changes.
func dayIndex(start, t time.Time, loc *time.Location) int {
	local := t.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return int(day.Sub(start).Round(24*time.Hour) / (24 * time.Hour))
}

// weekStart returns local Monday 00:00 of the week containing t.
func weekStart(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

func workoutsBetween(entries []models.TrainingEntry, from, to time.Time) []models.TrainingEntry {
	out := []models.TrainingEntry{}
	for _, e := range entries {
		if e.Kind == "workout" && !e.StartedAt.Before(from) && e.StartedAt.Before(to) {
			out = append(out, e)
		}
	}
	return out
}

// TrainingWeek is this local week, Monday to Sunday.
type TrainingWeek struct {
	Workouts int
	// Days[0] is Monday. Today is the index of the current weekday.
	Days  [7]bool
	Today int
	// VolumeKG sums load × reps over work sets that have both.
	VolumeKG float64
	// Minutes sums workouts with a known end time; UnknownTime counts the rest.
	Minutes, UnknownTime int
}

func SummarizeTrainingWeek(entries []models.TrainingEntry, now time.Time, loc *time.Location) TrainingWeek {
	start := weekStart(now, loc)
	week := TrainingWeek{Today: dayIndex(start, now, loc)}
	var seconds float64
	for _, e := range workoutsBetween(entries, start, start.AddDate(0, 0, 7)) {
		if e.StartedAt.After(now) {
			continue
		}
		week.Workouts++
		week.Days[dayIndex(start, e.StartedAt, loc)] = true
		if e.EndedAt.IsZero() {
			week.UnknownTime++
		} else {
			seconds += e.EndedAt.Sub(e.StartedAt).Seconds()
		}
		for _, ex := range e.Exercises {
			for _, s := range ex.Sets {
				if isWorkSet(s) && s.LoadKG != nil && s.Reps != nil && *s.LoadKG > 0 && *s.Reps > 0 {
					week.VolumeKG += *s.LoadKG * *s.Reps
				}
			}
		}
	}
	week.Minutes = int(math.Round(seconds / 60))
	week.VolumeKG = math.Round(week.VolumeKG)
	return week
}

// WorkoutsPerWeek counts workouts in each of the last n local weeks, oldest
// first; the last value is the current week so far.
func WorkoutsPerWeek(entries []models.TrainingEntry, now time.Time, loc *time.Location, n int) []int {
	counts := make([]int, n)
	current := weekStart(now, loc)
	for i := range counts {
		from := current.AddDate(0, 0, -7*(n-1-i))
		for _, e := range workoutsBetween(entries, from, from.AddDate(0, 0, 7)) {
			if !e.StartedAt.After(now) {
				counts[i]++
			}
		}
	}
	return counts
}

// ExerciseTrend is one exercise's best estimated 1-rep max per session.
type ExerciseTrend struct {
	Name      string
	Sessions  int
	Series    []float64 // per session, oldest first, kg
	ChangePct int       // first to last session
	BestSet   string    // heaviest-e1RM set in the window, e.g. "150 kg × 10"
}

// estimatedOneRepMax uses Epley for 1–12 reps; other sets give no estimate.
func estimatedOneRepMax(s models.TrainingSet) (float64, bool) {
	if !isWorkSet(s) || s.LoadKG == nil || s.Reps == nil || *s.LoadKG <= 0 || *s.Reps < 1 || *s.Reps > 12 {
		return 0, false
	}
	if *s.Reps == 1 {
		return *s.LoadKG, true
	}
	return *s.LoadKG * (1 + *s.Reps/30), true
}

func formatSet(s models.TrainingSet) string {
	return trimFloat(*s.LoadKG) + " kg × " + trimFloat(*s.Reps)
}

func trimFloat(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// ExerciseProgress returns exercises logged in 2+ sessions within the last
// `days`, sorted by change (largest first), then name.
func ExerciseProgress(entries []models.TrainingEntry, now time.Time, days int) []ExerciseTrend {
	type point struct {
		at   time.Time
		best float64
		set  models.TrainingSet
	}
	byName := map[string][]point{}
	display := map[string]string{}
	for _, e := range workoutsBetween(entries, now.AddDate(0, 0, -days), now.Add(time.Second)) {
		for _, ex := range e.Exercises {
			key := strings.ToLower(strings.TrimSpace(ex.Name))
			if key == "" {
				continue
			}
			best, found := point{at: e.StartedAt}, false
			for _, s := range ex.Sets {
				if v, ok := estimatedOneRepMax(s); ok && v > best.best {
					best.best, best.set, found = v, s, true
				}
			}
			if found {
				byName[key] = append(byName[key], best)
				display[key] = strings.TrimSpace(ex.Name)
			}
		}
	}
	trends := []ExerciseTrend{}
	for key, points := range byName {
		if len(points) < 2 {
			continue
		}
		sort.Slice(points, func(i, j int) bool { return points[i].at.Before(points[j].at) })
		t := ExerciseTrend{Name: display[key], Sessions: len(points)}
		top := points[0]
		for _, p := range points {
			t.Series = append(t.Series, math.Round(p.best*10)/10)
			if p.best > top.best {
				top = p
			}
		}
		first, last := points[0].best, points[len(points)-1].best
		t.ChangePct = int(math.Round((last - first) / first * 100))
		t.BestSet = formatSet(top.set)
		trends = append(trends, t)
	}
	sort.Slice(trends, func(i, j int) bool {
		if trends[i].ChangePct != trends[j].ChangePct {
			return trends[i].ChangePct > trends[j].ChangePct
		}
		return trends[i].Name < trends[j].Name
	})
	return trends
}

// MuscleGroupShare is one body area's share of primary work sets.
type MuscleGroupShare struct {
	Name     string
	Sets     int
	Percent  int
	Muscles  []string
	fraction float64
}

var muscleGroups = []struct {
	name    string
	muscles []string
}{
	{"Legs", []string{"quadriceps", "hamstrings", "glutes", "calves", "adductors", "abductors"}},
	{"Back", []string{"lats", "upper_back", "lower_back", "traps"}},
	{"Chest", []string{"chest"}},
	{"Shoulders", []string{"shoulders"}},
	{"Arms", []string{"biceps", "triceps", "forearms"}},
	{"Core", []string{"abdominals"}},
}

// MuscleGroupShares groups primary work sets from the last `days` into body
// areas. Unmapped or other muscles go to "Other". Groups with no sets are
// left out; shares add up to 100 (largest remainder).
func MuscleGroupShares(entries []models.TrainingEntry, now time.Time, days int) []MuscleGroupShare {
	groupOf := map[string]int{}
	for i, g := range muscleGroups {
		for _, m := range g.muscles {
			groupOf[m] = i
		}
	}
	counts := make([]int, len(muscleGroups)+1)
	total := 0
	for _, e := range workoutsBetween(entries, now.AddDate(0, 0, -days), now.Add(time.Second)) {
		for _, ex := range e.Exercises {
			for _, s := range ex.Sets {
				if !isWorkSet(s) {
					continue
				}
				i, ok := groupOf[ex.PrimaryMuscle]
				if !ok {
					i = len(muscleGroups)
				}
				counts[i]++
				total++
			}
		}
	}
	if total == 0 {
		return nil
	}
	shares := []MuscleGroupShare{}
	for i, c := range counts {
		if c == 0 {
			continue
		}
		name, muscles := "Other", []string(nil)
		if i < len(muscleGroups) {
			name, muscles = muscleGroups[i].name, muscleGroups[i].muscles
		}
		f := float64(c) * 100 / float64(total)
		shares = append(shares, MuscleGroupShare{Name: name, Sets: c, Percent: int(f), Muscles: muscles, fraction: f - math.Floor(f)})
	}
	sum := 0
	for _, s := range shares {
		sum += s.Percent
	}
	order := make([]int, len(shares))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return shares[order[a]].fraction > shares[order[b]].fraction })
	for k := 0; sum < 100 && k < len(order); k++ {
		shares[order[k]].Percent++
		sum++
	}
	sort.SliceStable(shares, func(i, j int) bool {
		if shares[i].Sets != shares[j].Sets {
			return shares[i].Sets > shares[j].Sets
		}
		return shares[i].Name < shares[j].Name
	})
	return shares
}
