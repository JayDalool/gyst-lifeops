package xp

import "testing"

func TestPriorityRewards(t *testing.T) {
	tests := []struct {
		priority string
		task     int
		mission  int
	}{
		{"low", 5, 10},
		{"medium", 10, 20},
		{"high", 20, 35},
		{"", 10, 20},
	}
	for _, test := range tests {
		if got := TaskReward(test.priority); got != test.task {
			t.Errorf("TaskReward(%q) = %d, want %d", test.priority, got, test.task)
		}
		if got := MissionReward(test.priority); got != test.mission {
			t.Errorf("MissionReward(%q) = %d, want %d", test.priority, got, test.mission)
		}
	}
}
