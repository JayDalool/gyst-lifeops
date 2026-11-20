package services

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"lifeops-tracker/internal/models"
)

func refForm() HealthReferenceForm {
	return HealthReferenceForm{SourceLabel: "Example dietitian", SourceKind: "dietitian", IssuedOn: "2026-11-01", Kcal: "1900", Protein: "120"}
}

func TestParseHealthReference(t *testing.T) {
	got, err := ParseHealthReference(refForm())
	if err != nil || got.Kcal == nil || *got.Kcal != 1900 {
		t.Fatalf("parse: %+v %v", got, err)
	}
	// Blank numbers stay nil and never become 0.
	if got.CarbsG != nil || got.FatG != nil || got.FiberG != nil || got.ProteinG == nil || *got.ProteinG != 120 {
		t.Fatalf("blank numbers must stay nil: %+v", got)
	}
	for name, mutate := range map[string]func(*HealthReferenceForm){
		"no source":     func(f *HealthReferenceForm) { f.SourceLabel = "  " },
		"long source":   func(f *HealthReferenceForm) { f.SourceLabel = strings.Repeat("x", 201) },
		"bad kind":      func(f *HealthReferenceForm) { f.SourceKind = "guess" },
		"bad date":      func(f *HealthReferenceForm) { f.IssuedOn = "2026-02-30" },
		"negative":      func(f *HealthReferenceForm) { f.Protein = "-1" },
		"not a number":  func(f *HealthReferenceForm) { f.Fat = "lots" },
		"too big":       func(f *HealthReferenceForm) { f.Fiber = "10001" },
		"bad cycle":     func(f *HealthReferenceForm) { f.CycleID = "abc" },
		"long notes":    func(f *HealthReferenceForm) { f.Notes = strings.Repeat("n", 401) },
		"infinity text": func(f *HealthReferenceForm) { f.Kcal = "Inf" },
	} {
		f := refForm()
		mutate(&f)
		_, err := ParseHealthReference(f)
		var invalid *HealthValidationError
		if !errors.As(err, &invalid) {
			t.Errorf("%s: want validation error, got %v", name, err)
		}
	}
}

func TestHealthReferenceTargetLifecycleLeavesCyclesUnchanged(t *testing.T) {
	s := testService(t)
	s.Now = func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }
	c := cycleFixture()
	if err := s.CreateHealthCycle(&c); err != nil {
		t.Fatal(err)
	}
	before, err := s.ListHealthCycles()
	if err != nil {
		t.Fatal(err)
	}
	global, err := ParseHealthReference(refForm())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateHealthReferenceTarget(&global); err != nil {
		t.Fatal(err)
	}
	tied := models.HealthReferenceTarget{CycleID: &c.ID, SourceLabel: "Example device", SourceKind: "device"}
	if err := s.CreateHealthReferenceTarget(&tied); err != nil {
		t.Fatal(err)
	}
	missing := int64(9999)
	if err := s.CreateHealthReferenceTarget(&models.HealthReferenceTarget{CycleID: &missing, SourceLabel: "x", SourceKind: "other"}); err == nil {
		t.Fatal("unknown cycle accepted")
	}
	list, err := s.ListHealthReferenceTargets()
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	for _, got := range list {
		if got.SourceLabel == "Example device" && (got.Kcal != nil || got.ProteinG != nil || got.CycleID == nil || *got.CycleID != c.ID) {
			t.Fatalf("tied target: %+v", got)
		}
		if got.CreatedAt != "2026-11-16T12:00:00.000000000Z" {
			t.Fatalf("created_at not an app-written instant: %q", got.CreatedAt)
		}
	}
	if err := s.ArchiveHealthReferenceTarget(global.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ArchiveHealthReferenceTarget(global.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second archive: %v", err)
	}
	list, _ = s.ListHealthReferenceTargets()
	if len(list) != 1 || list[0].SourceLabel != "Example device" {
		t.Fatalf("archived target still listed: %+v", list)
	}
	after, err := s.ListHealthCycles()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("health_cycles changed:\n%+v\n%+v", before, after)
	}
}

// GYST-154: a thousands comma was read as a decimal point ("2,000" became 2).
func TestParseHealthReferenceThousandsComma(t *testing.T) {
	set := map[string]func(*HealthReferenceForm, string){
		"kcal": func(f *HealthReferenceForm, v string) { f.Kcal = v }, "protein": func(f *HealthReferenceForm, v string) { f.Protein = v },
		"carbs": func(f *HealthReferenceForm, v string) { f.Carbs = v }, "fat": func(f *HealthReferenceForm, v string) { f.Fat = v },
		"fiber": func(f *HealthReferenceForm, v string) { f.Fiber = v },
	}
	get := func(t models.HealthReferenceTarget, name string) *float64 {
		return map[string]*float64{"kcal": t.Kcal, "protein": t.ProteinG, "carbs": t.CarbsG, "fat": t.FatG, "fiber": t.FiberG}[name]
	}
	for name, setter := range set {
		for raw, want := range map[string]float64{"2,000": 2000, "1,500": 1500, "2.5": 2.5, "2": 2, " 2 ": 2, "2,5": 2.5} {
			f := refForm()
			f.Kcal, f.Protein = "", ""
			setter(&f, raw)
			got, err := ParseHealthReference(f)
			if v := get(got, name); err != nil || v == nil || *v != want {
				t.Errorf("%s %q: got %v %v, want %v", name, raw, v, err, want)
			}
		}
		for _, raw := range []string{"2,00,0", "abc", "2,0000", "1,5,0"} {
			f := refForm()
			setter(&f, raw)
			_, err := ParseHealthReference(f)
			var invalid *HealthValidationError
			if !errors.As(err, &invalid) || !strings.Contains(err.Error(), "must be a number") {
				t.Errorf("%s %q: want plain refusal, got %v", name, raw, err)
			}
		}
	}
}

func TestParseTypedNumberMatchesDiaryRuleWithoutThousands(t *testing.T) {
	for _, raw := range []string{"1,000", "2,000", "1,234"} {
		if _, ok := ParseTypedNumber(raw, false); ok {
			t.Errorf("%q must stay refused for the diary", raw)
		}
	}
}
