package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"lifeops-tracker/internal/models"
	"lifeops-tracker/internal/services"
)

// Reference targets and cycle comparison on the Plan page (#205). Reference
// targets are outside recommendations; nothing here reads them as active
// targets or writes to health_cycles.

type refTargetView struct {
	ID                         int64
	Label, Kind, Issued, Cycle string
	Kcal, Protein, Carbs       string
	Fat, Fiber, Note           string
}

const missingValue = "—" // NULL is "not given", never 0

func refNumber(v *float64, unit string) string {
	if v == nil {
		return missingValue
	}
	return niceNumber(*v) + unit
}

var refKindLabels = map[string]string{"dietitian": "Dietitian", "device": "Device", "ai": "AI", "other": "Other"}

func buildRefViews(targets []models.HealthReferenceTarget, cycles []models.HealthCycle) []refTargetView {
	titles := map[int64]string{}
	for _, c := range cycles {
		titles[c.ID] = c.Title
	}
	out := make([]refTargetView, 0, len(targets))
	for _, t := range targets {
		v := refTargetView{ID: t.ID, Label: t.SourceLabel, Kind: refKindLabels[t.SourceKind], Issued: t.IssuedOn, Cycle: "Not tied to a cycle", Note: t.Notes,
			Kcal: refNumber(t.Kcal, " kcal"), Protein: refNumber(t.ProteinG, " g"), Carbs: refNumber(t.CarbsG, " g"), Fat: refNumber(t.FatG, " g"), Fiber: refNumber(t.FiberG, " g")}
		if t.CycleID != nil {
			v.Cycle = titles[*t.CycleID]
		}
		out = append(out, v)
	}
	return out
}

// fillPlanExtras adds the reference targets and the cycle comparison.
func (h *Handler) fillPlanExtras(data *healthPlanData) error {
	targets, err := h.Service.ListHealthReferenceTargets()
	if err != nil {
		return err
	}
	data.References = buildRefViews(targets, data.Cycles)
	today, err := h.Service.HealthTodayDate()
	if err != nil {
		return err
	}
	entries, err := h.Service.ListHealthWeightEntries()
	if err != nil {
		return err
	}
	weights := groupWeightDays(entries, h.Service.Now(), h.Service.Location())
	for _, c := range data.Cycles {
		to := c.EndDate
		if today < to {
			to = today
		}
		days, err := h.dayIntakeSpan(c.StartDate, to)
		if err != nil {
			return err
		}
		data.Compare = append(data.Compare, buildCycleComparison(c, days, weights, today, h.Service.Location()))
	}
	return nil
}

// dayIntakeSpan resolves each day's one counted source (GYST diary or Apple
// Health) over a date span, both ends included. DayIntakeRange caps a call at
// 400 days, so a long cycle is read in chunks.
func (h *Handler) dayIntakeSpan(start, end string) ([]models.DayIntake, error) {
	var out []models.DayIntake
	for start <= end {
		first, err := time.Parse("2006-01-02", start)
		if err != nil {
			return nil, err
		}
		last := first.AddDate(0, 0, 399).Format("2006-01-02")
		if last > end {
			last = end
		}
		days, err := h.Service.DayIntakeRange(start, last)
		if err != nil {
			return nil, err
		}
		out = append(out, days...)
		lastDay, _ := time.Parse("2006-01-02", last)
		start = lastDay.AddDate(0, 0, 1).Format("2006-01-02")
	}
	return out, nil
}

// planFormForNow is the form the Plan page shows when no cycle is picked.
func (h *Handler) planFormForNow() (healthPlanForm, error) {
	date, err := h.Service.HealthTodayDate()
	if err != nil {
		return healthPlanForm{}, err
	}
	start, _ := time.Parse("2006-01-02", date)
	cycle := models.HealthCycle{StartDate: date, EndDate: start.AddDate(0, 0, 84).Format("2006-01-02")}
	if current, err := h.Service.GetHealthCycleForDate(date); err == nil {
		cycle = current
	} else if !errors.Is(err, sql.ErrNoRows) {
		return healthPlanForm{}, err
	}
	return healthPlanValues(cycle), nil
}

// healthReferenceAdd handles POST /health/plan/reference.
func (h *Handler) healthReferenceAdd(w http.ResponseWriter, r *http.Request) {
	if !postGuard(w, r, "POST") {
		return
	}
	form := services.HealthReferenceForm{CycleID: r.PostForm.Get("cycle_id"), SourceLabel: r.PostForm.Get("source_label"), SourceKind: r.PostForm.Get("source_kind"),
		IssuedOn: r.PostForm.Get("issued_on"), Kcal: r.PostForm.Get("kcal"), Protein: r.PostForm.Get("protein_g"), Carbs: r.PostForm.Get("carbs_g"),
		Fat: r.PostForm.Get("fat_g"), Fiber: r.PostForm.Get("fiber_g"), Notes: r.PostForm.Get("notes")}
	target, err := services.ParseHealthReference(form)
	if err == nil {
		err = h.Service.CreateHealthReferenceTarget(&target)
	}
	if err == nil {
		http.Redirect(w, r, "/health/plan?ref=added#reference-targets", http.StatusSeeOther)
		return
	}
	var invalid *services.HealthValidationError
	if !errors.As(err, &invalid) {
		serverError(w, err)
		return
	}
	planForm, err := h.planFormForNow()
	if err != nil {
		serverError(w, err)
		return
	}
	data := healthPlanData{Page: "health", Form: planForm, RefForm: form, RefError: invalid.Error()}
	if data.Cycles, err = h.Service.ListHealthCycles(); err == nil {
		err = h.fillPlanExtras(&data)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	h.render(w, "health_plan.html", data)
}

// healthReferenceArchive handles POST /health/plan/reference/{id}/archive.
func (h *Handler) healthReferenceArchive(w http.ResponseWriter, r *http.Request, rawID string) {
	if !postGuard(w, r, "POST") {
		return
	}
	id, ok := parseID(rawID)
	if !ok {
		notFound(w, r)
		return
	}
	if err := h.Service.ArchiveHealthReferenceTarget(id); errors.Is(err, sql.ErrNoRows) {
		notFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/health/plan?ref=archived#reference-targets", http.StatusSeeOther)
}

// coachReferenceTargets lists the active reference targets that apply to the
// current cycle (or to no cycle) for the Coach, each labelled with its source
// and "not_active". Missing numbers stay null, never 0.
func (h *Handler) coachReferenceTargets(cycleID *int64) ([]map[string]any, error) {
	targets, err := h.Service.ListHealthReferenceTargets()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, t := range targets {
		if t.CycleID != nil && (cycleID == nil || *t.CycleID != *cycleID) {
			continue
		}
		out = append(out, map[string]any{"not_active": true, "source_kind": t.SourceKind, "source_label": t.SourceLabel,
			"issued_on": t.IssuedOn, "calories": t.Kcal, "protein_g": t.ProteinG, "carbs_g": t.CarbsG, "fat_g": t.FatG, "fiber_g": t.FiberG, "notes": t.Notes})
	}
	return out, nil
}
