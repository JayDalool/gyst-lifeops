package models

import "strings"

func RoadmapProjectTypeOptions() []RoadmapProjectTypeOption {
	return []RoadmapProjectTypeOption{
		{Value: "real_estate_ai", Label: "Real Estate AI"},
		{Value: "healthcare_ai", Label: "Healthcare AI"},
		{Value: "ml_model", Label: "ML Model"},
		{Value: "certification", Label: "Certification"},
		{Value: "portfolio", Label: "Portfolio"},
		{Value: "learning", Label: "Learning"},
	}
}

func RoadmapProjectTypeLabel(value string) string {
	value = NormalizeRoadmapProjectType(value)
	for _, option := range RoadmapProjectTypeOptions() {
		if option.Value == value {
			return option.Label
		}
	}
	return "Learning"
}

func NormalizeRoadmapProjectType(value string) string {
	normalized := normalizeToken(value)
	switch normalized {
	case "real_estate_ai", "real_estate", "realestate", "real_estate_project":
		return "real_estate_ai"
	case "healthcare_ai", "healthcare", "health_ai", "medical_ai":
		return "healthcare_ai"
	case "ml_model", "machine_learning", "machine_learning_model", "model":
		return "ml_model"
	case "certification", "cert", "exam":
		return "certification"
	case "portfolio", "public_portfolio":
		return "portfolio"
	case "learning", "course", "courses", "study":
		return "learning"
	default:
		lower := strings.ToLower(value)
		switch {
		case strings.Contains(lower, "real estate"):
			return "real_estate_ai"
		case strings.Contains(lower, "healthcare") || strings.Contains(lower, "medical"):
			return "healthcare_ai"
		case strings.Contains(lower, "ml model") || strings.Contains(lower, "machine learning"):
			return "ml_model"
		case strings.Contains(lower, "cert"):
			return "certification"
		case strings.Contains(lower, "portfolio"):
			return "portfolio"
		default:
			return "learning"
		}
	}
}

func NormalizeRoadmapResourceType(value string) string {
	normalized := normalizeToken(value)
	switch normalized {
	case "youtube", "video", "videos", "yt":
		return "youtube"
	case "course", "courses", "class":
		return "course"
	case "github", "repo", "repos", "repository":
		return "github"
	case "documentation", "docs", "doc":
		return "documentation"
	case "notes", "note":
		return "notes"
	default:
		lower := strings.ToLower(value)
		switch {
		case strings.Contains(lower, "youtube") || strings.Contains(lower, "youtu.be"):
			return "youtube"
		case strings.Contains(lower, "github.com"):
			return "github"
		case strings.Contains(lower, "docs") || strings.Contains(lower, "documentation"):
			return "documentation"
		case strings.Contains(lower, "course"):
			return "course"
		case strings.Contains(lower, "note"):
			return "notes"
		default:
			return "other"
		}
	}
}

func normalizeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer("-", "_", " ", "_", "/", "_", "&", "and")
	value = replacer.Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}
