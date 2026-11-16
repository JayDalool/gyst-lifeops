package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseCardioNotes(t *testing.T) {
	type r = [2]float64 // incline, speed; -1 means absent
	const none = -1
	for _, tc := range []struct {
		name, notes string
		timed       int
		want        []r
	}{
		// Keyword form: exercise-level, every timed set.
		{"dot separated", "incline 8% · 5.0 km/h", 1, []r{{8, 5}}},
		{"bare numbers", "incline 8 5 kmh", 1, []r{{8, 5}}},
		{"mph converted", "incline 10% 3.1 mph", 1, []r{{10, 5}}},
		{"mph no space", "incline 10 3.1mph", 1, []r{{10, 5}}},
		{"upper case", "INCLINE 7.5% · 6 KM/H", 1, []r{{7.5, 6}}},
		{"colon and kph", "Incline: 12, Speed 4.5 kph", 1, []r{{12, 4.5}}},
		{"applies to all timed sets", "incline 8% · 5 km/h", 3, []r{{8, 5}, {8, 5}, {8, 5}}},
		{"zero incline is real", "incline 0% · 6 km/h", 1, []r{{0, 6}}},
		{"incline only", "incline 5%", 1, []r{{5, none}}},
		{"speed only", "walked 5.5 km/h", 1, []r{{none, 5.5}}},
		{"comma decimal", "incline 7,5% · 5,5 km/h", 1, []r{{7.5, 5.5}}},
		{"same keyword value repeated", "incline 8% ... incline 8%", 1, []r{{8, none}}},
		// Per-set tokens.
		{"single token", "8%/5.0", 1, []r{{8, 5}}},
		{"token mph", "6% / 3 mph", 1, []r{{6, 4.8}}},
		{"tokens map to timed sets", "6%/4.5 8%/5.0 10%/5.5", 3, []r{{6, 4.5}, {8, 5}, {10, 5.5}}},
		{"token count differs", "6%/4.5 8%/5.0", 3, []r{{none, none}, {none, none}, {none, none}}},
		{"one token many sets", "8%/5.0", 2, []r{{none, none}, {none, none}}},
		{"keyword and tokens agree", "incline 8% 8%/5 8%/5", 2, []r{{8, 5}, {8, 5}}},
		{"keyword and tokens disagree", "incline 8% 6%/5 8%/5", 2, []r{{none, none}, {none, none}}},
		{"keyword speed and token speed disagree", "incline 8% · 5 km/h, 8%/6", 1, []r{{none, none}}},
		{"keyword speed and token speed agree", "incline 8% · 5 km/h, 8%/5", 1, []r{{8, 5}}},
		{"degrees are not percent", "incline 30 degrees, 5 km/h", 1, []r{{none, 5}}},
		{"keyword values differ", "incline 8% then incline 10%", 1, []r{{none, none}}},
		{"speeds differ", "5 km/h then 6 km/h", 1, []r{{none, none}}},
		// Absent, never 0.
		{"bare percent is not incline", "felt 20% harder", 1, []r{{none, none}}},
		{"effort percent", "30% effort, 5 km/h", 1, []r{{none, 5}}},
		{"empty", "", 1, []r{{none, none}}},
		{"garbage", "felt great, legs tired", 1, []r{{none, none}}},
		{"no timed sets", "incline 8% · 5 km/h", 0, []r{}},
		// Bounds.
		{"incline above 40", "incline 45% · 5 km/h", 1, []r{{none, 5}}},
		{"incline 40 allowed", "incline 40% · 5 km/h", 1, []r{{40, 5}}},
		{"speed above 25", "incline 5% · 30 km/h", 1, []r{{5, none}}},
		{"speed below 0.5", "incline 5% · 0.2 km/h", 1, []r{{5, none}}},
		{"negative incline ignored", "incline -2 5 km/h", 1, []r{{none, 5}}},
		{"distance is not speed", "incline 8%, 3 km", 1, []r{{8, none}}},
		{"token incline out of bounds", "90%/5", 1, []r{{none, 5}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseCardioNotes(tc.notes, tc.timed)
			if len(got) != len(tc.want) {
				t.Fatalf("%d readings, want %d", len(got), len(tc.want))
			}
			show := func(p *float64) float64 {
				if p == nil {
					return none
				}
				return *p
			}
			for i, w := range tc.want {
				if g := (r{show(got[i].InclinePct), show(got[i].SpeedKmh)}); g != w {
					t.Errorf("set %d of %q: got %v want %v", i, tc.notes, g, w)
				}
			}
		})
	}
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
		t.Fatal(fmt.Sprintf("%s %v", out, err))
	}
}
