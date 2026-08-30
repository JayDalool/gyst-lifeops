// Package xp defines the LifeOps reward economy. Award persistence remains in
// the repository; this package keeps reward sizes consistent across creation
// paths without introducing a second XP ledger.
package xp

const (
	ProofBonus             = 5
	ProjectCompletionBonus = 250
	XPPerLevel             = 100
)

func TaskReward(priority string) int {
	switch priority {
	case "low":
		return 5
	case "high":
		return 20
	default:
		return 10
	}
}

func MissionReward(priority string) int {
	switch priority {
	case "low":
		return 10
	case "high":
		return 35
	default:
		return 20
	}
}
