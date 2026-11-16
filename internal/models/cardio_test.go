package models

import (
	"encoding/json"
	"testing"
)

func TestParseCardioNotes(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	for _, tc := range []struct {
		name, notes    string
		incline, speed *float64
	}{
		{"dot separated", "incline 8% · 5.0 km/h", f(8), f(5)},
		{"bare numbers", "incline 8 5 kmh", f(8), f(5)},
		{"mph converted", "incline 10% 3.1 mph", f(10), f(5)},
		{"mph no space", "incline 10 3.1mph", f(10), f(5)},
		{"per set short", "8%/5.0", f(8), f(5)},
		{"per set short mph", "6% / 3 mph", f(6), f(4.8)},
		{"upper case", "INCLINE 7.5% · 6 KM/H", f(7.5), f(6)},
		{"colon", "Incline: 12, Speed 4.5 kph", f(12), f(4.5)},
		{"zero incline is real", "incline 0% · 6 km/h", f(0), f(6)},
		{"incline only", "incline 5%", f(5), nil},
		{"speed only", "walked 5.5 km/h", nil, f(5.5)},
		{"percent without keyword", "10% and 4 km/h", f(10), f(4)},
		{"empty", "", nil, nil},
		{"garbage", "felt great, legs tired", nil, nil},
		{"incline above 40", "incline 45% · 5 km/h", nil, f(5)},
		{"speed above 25", "incline 5% · 30 km/h", f(5), nil},
		{"speed below 0.5", "incline 5% · 0.2 km/h", f(5), nil},
		{"negative percent ignored", "-5% 5 km/h", nil, f(5)},
		{"distance is not speed", "incline 8%, 3 km", f(8), nil},
		{"bad short form", "90%/5", nil, f(5)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			incline, speed := ParseCardioNotes(tc.notes)
			eq := func(got, want *float64) bool {
				return (got == nil) == (want == nil) && (got == nil || *got == *want)
			}
			if !eq(incline, tc.incline) || !eq(speed, tc.speed) {
				t.Fatalf("%q: got %v %v", tc.notes, deref(incline), deref(speed))
			}
		})
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// Snapshots cached before incline and speed existed must still decode, and
// absent values must stay absent rather than become 0.
func TestTrainingSetDecodesOldSnapshot(t *testing.T) {
	var s TrainingSet
	if err := json.Unmarshal([]byte(`{"Kind":"normal","Seconds":1200}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.InclinePct != nil || s.SpeedKmh != nil || *s.Seconds != 1200 {
		t.Fatalf("%+v", s)
	}
	out, _ := json.Marshal(s)
	var back TrainingSet
	if err := json.Unmarshal(out, &back); err != nil || back.InclinePct != nil {
		t.Fatalf("%s %v", out, err)
	}
}
