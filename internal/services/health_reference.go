package services

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"lifeops-tracker/internal/clock"
	"lifeops-tracker/internal/models"
)

// Reference targets (#205): outside recommendations kept beside the active
// plan. Creating or archiving one never touches health_cycles.

// HealthReferenceForm holds the submitted strings so an invalid form can be
// shown again as typed.
type HealthReferenceForm struct {
	CycleID, SourceLabel, SourceKind, IssuedOn string
	Kcal, Protein, Carbs, Fat, Fiber, Notes    string
}

var healthReferenceKinds = map[string]bool{"dietitian": true, "device": true, "ai": true, "other": true}

// ParseHealthReference validates the form. Blank numbers stay unknown (nil).
// Limits are sized so the worst-case percent-encoded form (every character
// four bytes, 12 bytes each once encoded: 200 source + 400 notes) stays under
// the 8 KB body cap in postGuard, so a long note gets this message, not a bare 400.
func ParseHealthReference(f HealthReferenceForm) (models.HealthReferenceTarget, error) {
	t := models.HealthReferenceTarget{SourceLabel: strings.TrimSpace(f.SourceLabel), SourceKind: f.SourceKind, IssuedOn: strings.TrimSpace(f.IssuedOn), Notes: strings.TrimSpace(f.Notes)}
	switch {
	case t.SourceLabel == "":
		return t, healthInvalid("Say where the reference comes from")
	case utf8.RuneCountInString(t.SourceLabel) > 200:
		return t, healthInvalid("The source can be at most 200 characters")
	case !healthReferenceKinds[t.SourceKind]:
		return t, healthInvalid("Choose a source type")
	case utf8.RuneCountInString(t.Notes) > 400:
		return t, healthInvalid("Notes can be at most 400 characters")
	}
	if t.IssuedOn != "" {
		if err := ValidateHealthDate(t.IssuedOn); err != nil {
			return t, healthInvalid("Issued date must be YYYY-MM-DD")
		}
	}
	if raw := strings.TrimSpace(f.CycleID); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return t, healthInvalid("Choose a valid cycle")
		}
		t.CycleID = &id
	}
	fields := []struct {
		label, raw string
		max        float64
		dst        **float64
	}{
		{"Calories", f.Kcal, 100000, &t.Kcal}, {"Protein", f.Protein, 10000, &t.ProteinG},
		{"Carbs", f.Carbs, 10000, &t.CarbsG}, {"Fat", f.Fat, 10000, &t.FatG}, {"Fiber", f.Fiber, 10000, &t.FiberG},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.raw) == "" {
			continue
		}
		v, ok := ParseTypedNumber(field.raw, true)
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > field.max {
			return t, healthInvalid(field.label + " must be a number from 0 to " + strconv.FormatFloat(field.max, 'f', -1, 64))
		}
		v = math.Round(v*10) / 10
		*field.dst = &v
	}
	return t, nil
}

// CreateHealthReferenceTarget stores a validated target. A cycle, when given,
// must exist.
func (s *Service) CreateHealthReferenceTarget(t *models.HealthReferenceTarget) error {
	if t.CycleID != nil {
		if _, err := s.Repo.GetHealthCycle(*t.CycleID); errors.Is(err, sql.ErrNoRows) {
			return healthInvalid("Choose a valid cycle")
		} else if err != nil {
			return err
		}
	}
	t.CreatedAt = clock.FormatInstant(s.Now())
	return s.Repo.CreateHealthReferenceTarget(t)
}

func (s *Service) ListHealthReferenceTargets() ([]models.HealthReferenceTarget, error) {
	return s.Repo.ListHealthReferenceTargets()
}

// ArchiveHealthReferenceTarget returns sql.ErrNoRows for an unknown or
// already archived target.
func (s *Service) ArchiveHealthReferenceTarget(id int64) error {
	return s.Repo.ArchiveHealthReferenceTarget(id, clock.FormatInstant(s.Now()))
}
