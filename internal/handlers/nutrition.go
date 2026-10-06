package handlers

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

// Nutrition v3 (#55): today's intake against the plan, a seven-day calorie
// view, and where the numbers come from. GYST doesn't log food; MyFitnessPal
// does, and Apple Health carries daily totals here. Totals only: no meals.
type nutritionDay struct {
	Letter, Name, Value string
	Present, Over       bool
	IsToday             bool
	Height              int // bar height, % of the chart
}

type nutritionData struct {
	Page         string
	Today        HealthTodayView
	HasNutrition bool
	Week         []nutritionDay
	WeekLabel    string
	TargetLine   int    // target as % of chart height; 0 when no target
	WeekAvg      string // average of logged days before today; empty when none
	WeekLogged   int
	WeekWithin   int
	// Close-the-gap suggestions (#62): shown only when protein eaten and a
	// protein target are both known and eaten is below target.
	ShowGap  bool
	GapLeft  string
	GapFoods []gapFood
	HasFoods bool
}

type gapFood struct {
	Name, Serving string
	Protein, Kcal string
	Fits          bool // kcal known and within today's budget left
	BudgetKnown   bool
}

// buildProteinGap picks up to three active foods with at least 5 g protein
// per serving, best protein per kcal first (unknown kcal last).
func buildProteinGap(data *nutritionData, foods []models.HealthFood) {
	data.HasFoods = len(foods) > 0
	var protein HealthMacroView
	for _, m := range data.Today.Nutrients {
		if m.Label == "Protein" {
			protein = m
		}
	}
	if !protein.HasBar || protein.State == "ok" || protein.Over {
		return
	}
	data.ShowGap, data.GapLeft = true, strings.TrimPrefix(protein.Remaining.Value, "−")
	budget, budgetKnown := 0.0, data.Today.Calories.HasBar && !data.Today.CaloriesOver
	if budgetKnown {
		budget, _ = strconv.ParseFloat(strings.ReplaceAll(data.Today.BudgetAbs, ",", ""), 64)
	}
	candidates := []models.HealthFood{}
	for _, f := range foods {
		if f.Active && f.ProteinG != nil && *f.ProteinG >= 5 {
			candidates = append(candidates, f)
		}
	}
	ratio := func(f models.HealthFood) float64 {
		if f.Kcal == nil || *f.Kcal <= 0 {
			return -1
		}
		return *f.ProteinG / *f.Kcal
	}
	sort.SliceStable(candidates, func(i, j int) bool { return ratio(candidates[i]) > ratio(candidates[j]) })
	for _, f := range candidates {
		if len(data.GapFoods) == 3 {
			break
		}
		g := gapFood{Name: f.Name, Serving: f.Serving, Protein: niceNumber(*f.ProteinG), BudgetKnown: budgetKnown}
		// 0 kcal is treated as unknown, as in the ordering above.
		if f.Kcal != nil && *f.Kcal > 0 {
			g.Kcal = formatThousands(*f.Kcal)
			g.Fits = budgetKnown && *f.Kcal <= budget
		}
		data.GapFoods = append(data.GapFoods, g)
	}
}

// buildNutritionWeek lays out the six days before today plus today, from
// stored daily totals. A day without a dietary observation is "not logged",
// never zero.
func buildNutritionWeek(data *nutritionData, history []models.DailyHealthSnapshot, today string, target float64) {
	day, err := time.Parse("2006-01-02", today)
	if err != nil {
		return
	}
	byDate := map[string]*float64{}
	for _, s := range history {
		byDate[s.Date] = s.DietaryCalories
	}
	values := make([]*float64, 7)
	top := target
	for i := range values {
		values[i] = byDate[day.AddDate(0, 0, i-6).Format("2006-01-02")]
		if values[i] != nil {
			top = math.Max(top, *values[i])
		}
	}
	if top <= 0 {
		top = 1
	}
	top *= 1.1
	var sum float64
	data.WeekLabel = "Calories by day, oldest first:"
	for i, v := range values {
		d := day.AddDate(0, 0, i-6)
		nd := nutritionDay{Letter: d.Format("Mon")[:1], Name: d.Format("Mon Jan 2"), IsToday: i == 6}
		shown := "not logged"
		if v != nil {
			nd.Present, nd.Value = true, formatThousands(*v)
			shown = nd.Value
			nd.Height = max(2, int(math.Round(*v/top*100)))
			// Same rule as the Today card: compare at displayed precision.
			nd.Over = target > 0 && math.Round(*v) > math.Round(target)
			if i < 6 {
				data.WeekLogged++
				sum += *v
				if target > 0 && math.Round(*v) <= math.Round(target) {
					data.WeekWithin++
				}
			}
		}
		data.WeekLabel += fmt.Sprintf(" %s %s;", nd.Name, shown)
		data.Week = append(data.Week, nd)
	}
	if data.WeekLogged > 0 {
		data.WeekAvg = formatThousands(sum / float64(data.WeekLogged))
	}
	if target > 0 {
		data.TargetLine = max(1, int(math.Round(target/top*100)))
		data.WeekLabel += fmt.Sprintf(" target %s.", formatThousands(target))
	}
}

func buildNutritionData(today HealthTodayView) nutritionData {
	data := nutritionData{Page: "health", Today: today}
	for _, macro := range today.Macros {
		data.HasNutrition = data.HasNutrition || macro.Consumed.Present
	}
	return data
}

func (h *Handler) nutrition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		methodNotAllowed(w)
		return
	}
	today, err := h.Service.HealthToday()
	if err != nil {
		serverError(w, err)
		return
	}
	data := buildNutritionData(buildHealthTodayView(today, h.Service.Now()))
	if day, err := time.Parse("2006-01-02", today.Date); err == nil {
		history, err := h.Service.Repo.ListDailyHealthSnapshots(day.AddDate(0, 0, -6).Format("2006-01-02"), today.Date)
		if err != nil {
			serverError(w, err)
			return
		}
		target := 0.0
		if today.HealthCycle != nil {
			target = today.HealthCycle.DailyCalorieTarget
		}
		buildNutritionWeek(&data, history, today.Date, target)
	}
	foods, err := h.Service.Repo.ListHealthFoods(false)
	if err != nil {
		serverError(w, err)
		return
	}
	buildProteinGap(&data, foods)
	h.render(w, "nutrition.html", data)
}
