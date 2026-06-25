package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"lifeops-tracker/internal/models"
)

var (
	markdownLinkRE = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)
	bareURLRE      = regexp.MustCompile(`https?://[^\s)]+`)
	isoDateRE      = regexp.MustCompile(`\b20\d{2}-\d{2}-\d{2}\b`)
	hashTagRE      = regexp.MustCompile(`#([A-Za-z0-9_-]+)`)
	metaLineRE     = regexp.MustCompile(`^([A-Za-z][A-Za-z /_-]*):\s*(.+)$`)
)

func (s *Service) Roadmap() (models.RoadmapData, error) {
	today := s.Today()
	weekStart, weekEnd := weekBounds(today)

	projects, err := s.Repo.ListRoadmapProjects(false)
	if err != nil {
		return models.RoadmapData{}, err
	}
	latestImport, err := s.Repo.LatestRoadmapImport()
	if err != nil {
		return models.RoadmapData{}, err
	}
	globalResources, err := s.Repo.GlobalRoadmapResources()
	if err != nil {
		return models.RoadmapData{}, err
	}
	reviews, err := s.Repo.ListRoadmapWeeklyReviews(8)
	if err != nil {
		return models.RoadmapData{}, err
	}

	totalWork := 0
	completedWork := 0
	weeklyTotal := 0
	weeklyCompleted := 0
	var resources []models.RoadmapResource
	resources = append(resources, globalResources...)
	var missed []models.RoadmapDeadline

	for index := range projects {
		items, err := s.Repo.ProjectItems(projects[index].Project.Project.ID)
		if err != nil {
			return models.RoadmapData{}, err
		}
		milestones, err := s.Repo.RoadmapMilestones(projects[index].Project.Project.ID)
		if err != nil {
			return models.RoadmapData{}, err
		}
		projectResources, err := s.Repo.RoadmapResources(projects[index].Project.Project.ID)
		if err != nil {
			return models.RoadmapData{}, err
		}

		projects[index].Items = items
		projects[index].Milestones = milestones
		projects[index].Resources = projectResources
		resources = append(resources, projectResources...)

		projectTotal, projectCompleted := roadmapWorkCounts(items, milestones)
		projects[index].TotalWork = projectTotal
		projects[index].CompletedWork = projectCompleted
		projects[index].Progress = percentInt(projectCompleted, projectTotal)
		projects[index].Project.Momentum = trendChart(projects[index].Progress)
		totalWork += projectTotal
		completedWork += projectCompleted

		projectWeeklyTotal, projectWeeklyCompleted := roadmapWeeklyCounts(items, milestones, weekStart, weekEnd)
		projects[index].WeeklyCompletion = percentInt(projectWeeklyCompleted, projectWeeklyTotal)
		weeklyTotal += projectWeeklyTotal
		weeklyCompleted += projectWeeklyCompleted

		missed = append(missed, missedRoadmapDeadlines(projects[index], today)...)
	}

	currentFocus := currentRoadmapFocus(projects)
	goalTitle := latestImport.GoalTitle
	if goalTitle == "" {
		goalTitle = "Become an AI Engineer"
	}
	goalDeadline := latestImport.GoalDeadline
	if goalDeadline == "" {
		goalDeadline = time.Date(today.Year(), time.November, 1, 0, 0, 0, 0, today.Location()).Format("2006-01-02")
	}

	return models.RoadmapData{
		Page:               "roadmap",
		Today:              formatDate(today),
		WeekStart:          formatDate(weekStart),
		WeekEnd:            formatDate(weekEnd),
		GoalTitle:          goalTitle,
		GoalDeadline:       goalDeadline,
		LatestImport:       latestImport,
		Projects:           projects,
		OverallProgress:    percentInt(completedWork, totalWork),
		WeeklyCompletion:   percentInt(weeklyCompleted, weeklyTotal),
		MissedDeadlines:    missed,
		CurrentFocus:       currentFocus,
		Daily:              roadmapDailyTasks(projects, today),
		Resources:          resources,
		WeeklyReviews:      reviews,
		ProjectTypeOptions: models.RoadmapProjectTypeOptions(),
	}, nil
}

func (s *Service) ImportRoadmapMarkdown(source string) (int64, error) {
	plan := ParseRoadmapMarkdown(source, s.Today())
	return s.Repo.ImportRoadmapPlan(plan)
}

func (s *Service) CreateRoadmapWeeklyReview(review models.RoadmapWeeklyReview) error {
	if strings.TrimSpace(review.WeekStart) == "" {
		weekStart, _ := weekBounds(s.Today())
		review.WeekStart = formatDate(weekStart)
	}
	return s.Repo.CreateRoadmapWeeklyReview(&review)
}

func (s *Service) ExportRoadmap(public bool) (string, error) {
	data, err := s.Roadmap()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if public {
		fmt.Fprintf(&b, "# AI Engineer Portfolio Summary\n\n")
	} else {
		fmt.Fprintf(&b, "# AI Engineer Roadmap Full Plan\n\n")
	}
	fmt.Fprintf(&b, "Goal: %s\n", data.GoalTitle)
	fmt.Fprintf(&b, "Deadline: %s\n", data.GoalDeadline)
	fmt.Fprintf(&b, "Overall progress: %d%%\n", data.OverallProgress)
	fmt.Fprintf(&b, "Weekly completion: %d%%\n\n", data.WeeklyCompletion)

	for _, project := range data.Projects {
		if public && (project.Meta.IsPrivate || project.Meta.LocalOnly) {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", project.Project.Project.Name)
		fmt.Fprintf(&b, "Type: %s\n", project.Meta.ProjectTypeLabel)
		fmt.Fprintf(&b, "Status: %s\n", project.Project.Project.Status)
		fmt.Fprintf(&b, "Progress: %d%%\n", project.Progress)
		if len(project.Meta.Tags) > 0 {
			fmt.Fprintf(&b, "Tags: %s\n", strings.Join(project.Meta.Tags, ", "))
		}
		if !public {
			fmt.Fprintf(&b, "Private: %t\n", project.Meta.IsPrivate)
			fmt.Fprintf(&b, "Local only: %t\n", project.Meta.LocalOnly)
			fmt.Fprintf(&b, "Current focus: %t\n", project.Meta.CurrentFocus)
		}
		summary := strings.TrimSpace(project.Meta.PublicSummary)
		if summary == "" {
			summary = project.Project.Project.Description
		}
		if summary != "" {
			fmt.Fprintf(&b, "\n%s\n", summary)
		}

		if !public && len(project.Milestones) > 0 {
			fmt.Fprintf(&b, "\n### Milestones\n")
			for _, milestone := range project.Milestones {
				fmt.Fprintf(&b, "- [%s] %s", checkboxMark(milestone.Status), milestone.Title)
				if milestone.DueDate != "" {
					fmt.Fprintf(&b, " (due %s)", milestone.DueDate)
				}
				fmt.Fprintf(&b, "\n")
			}
		}
		if !public && len(project.Items) > 0 {
			fmt.Fprintf(&b, "\n### Tasks\n")
			for _, item := range project.Items {
				fmt.Fprintf(&b, "- [%s] %s", checkboxMark(item.Status), item.Title)
				if item.DueDate != "" {
					fmt.Fprintf(&b, " (due %s)", item.DueDate)
				}
				fmt.Fprintf(&b, "\n")
			}
		}
		if len(project.Resources) > 0 {
			fmt.Fprintf(&b, "\n### Resources\n")
			for _, resource := range project.Resources {
				if public && resource.IsPrivate {
					continue
				}
				if resource.URL != "" {
					fmt.Fprintf(&b, "- [%s](%s) (%s)\n", resource.Title, resource.URL, resource.ResourceType)
				} else {
					fmt.Fprintf(&b, "- %s (%s)\n", resource.Title, resource.ResourceType)
				}
			}
		}
		fmt.Fprintf(&b, "\n")
	}

	if !public && len(data.WeeklyReviews) > 0 {
		fmt.Fprintf(&b, "## Weekly Reviews\n\n")
		for _, review := range data.WeeklyReviews {
			fmt.Fprintf(&b, "### Week of %s\n", review.WeekStart)
			fmt.Fprintf(&b, "Finished: %s\n", review.Finished)
			fmt.Fprintf(&b, "Blocked: %s\n", review.Blocked)
			fmt.Fprintf(&b, "Next: %s\n\n", review.NextSteps)
		}
	}

	return b.String(), nil
}

func ParseRoadmapMarkdown(source string, today time.Time) models.RoadmapImportPlan {
	plan := models.RoadmapImportPlan{
		SourceMarkdown: source,
		ImportedDate:   formatDate(today),
	}
	currentProject := -1
	currentMainProject := -1
	section := ""

	for _, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if level, title, ok := markdownHeading(line); ok {
			section = applyRoadmapHeading(&plan, level, title, today, &currentProject, &currentMainProject)
			continue
		}

		if key, value, ok := metadataLine(line); ok {
			applyRoadmapMetadata(&plan, currentProject, key, value, today)
			continue
		}

		if bullet, ok := markdownBullet(line); ok {
			handleRoadmapBullet(&plan, &currentProject, section, bullet, today)
			continue
		}

		if currentProject >= 0 && len(extractResources(line)) > 0 {
			plan.Projects[currentProject].Resources = append(plan.Projects[currentProject].Resources, extractResources(line)...)
			continue
		}
		if plan.Goal.Description == "" && currentProject < 0 {
			plan.Goal.Description = line
		}
	}

	if strings.TrimSpace(plan.Goal.Title) == "" {
		plan.Goal.Title = "Become an AI Engineer"
	}
	if strings.TrimSpace(plan.Goal.Deadline) == "" {
		plan.Goal.Deadline = inferDateFromText(source, today)
	}
	if len(plan.Projects) == 0 {
		plan.Projects = append(plan.Projects, models.RoadmapProjectPlan{
			Name:        "AI Engineer Roadmap Execution",
			ProjectType: "learning",
			ProjectKind: "main",
			Status:      "active",
			Deadline:    plan.Goal.Deadline,
			AutoUnlock:  true,
		})
	}

	for index := range plan.Projects {
		if plan.Projects[index].ProjectType == "" {
			plan.Projects[index].ProjectType = inferProjectType(plan.Projects[index].Name + " " + plan.Projects[index].Description)
		}
		if plan.Projects[index].Status == "" {
			plan.Projects[index].Status = "active"
		}
		if plan.Projects[index].ProjectKind == "" {
			plan.Projects[index].ProjectKind = "main"
		}
		for itemIndex := range plan.Projects[index].Tasks {
			if plan.Projects[index].Tasks[itemIndex].SortOrder == 0 {
				plan.Projects[index].Tasks[itemIndex].SortOrder = itemIndex + 1
			}
		}
		for milestoneIndex := range plan.Projects[index].Milestones {
			if plan.Projects[index].Milestones[milestoneIndex].SortOrder == 0 {
				plan.Projects[index].Milestones[milestoneIndex].SortOrder = milestoneIndex + 1
			}
		}
	}

	return plan
}

func applyRoadmapHeading(plan *models.RoadmapImportPlan, level int, title string, today time.Time, currentProject, currentMainProject *int) string {
	lower := strings.ToLower(title)
	cleanTitle := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(title, "Project:"), "project:"))
	switch {
	case strings.Contains(lower, "resource"):
		return "resources"
	case strings.Contains(lower, "milestone"):
		return "milestones"
	case strings.Contains(lower, "task") || strings.Contains(lower, "daily") || strings.Contains(lower, "execution"):
		return "tasks"
	case strings.Contains(lower, "goal"):
		plan.Goal.Title = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(title, "Goal:"), "goal:"))
		if plan.Goal.Title == "" {
			plan.Goal.Title = "Become an AI Engineer"
		}
		if deadline := inferDateFromText(title, today); deadline != "" {
			plan.Goal.Deadline = deadline
		}
		*currentProject = -1
		return "goal"
	case lower == "projects" || lower == "project plan":
		return "projects"
	case level == 1 && plan.Goal.Title == "":
		plan.Goal.Title = cleanTitle
		if deadline := inferDateFromText(title, today); deadline != "" {
			plan.Goal.Deadline = deadline
		}
		*currentProject = -1
		return "goal"
	case level == 2 || strings.Contains(lower, "project"):
		project := projectPlanFromTitle(cleanTitle, today)
		project.ProjectKind = "main"
		plan.Projects = append(plan.Projects, project)
		*currentProject = len(plan.Projects) - 1
		*currentMainProject = *currentProject
		return "project"
	case level == 3:
		project := projectPlanFromTitle(cleanTitle, today)
		project.ProjectKind = "phase"
		if *currentMainProject >= 0 {
			project.ParentName = plan.Projects[*currentMainProject].Name
		}
		plan.Projects = append(plan.Projects, project)
		*currentProject = len(plan.Projects) - 1
		return "project"
	default:
		return ""
	}
}

func applyRoadmapMetadata(plan *models.RoadmapImportPlan, currentProject int, key, value string, today time.Time) {
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if currentProject < 0 {
		switch key {
		case "goal", "title":
			plan.Goal.Title = value
		case "description", "why":
			plan.Goal.Description = value
		case "deadline", "due", "target", "target date":
			plan.Goal.Deadline = firstNonEmpty(parseFlexibleDate(value, today), inferDateFromText(value, today))
		case "status":
			plan.Goal.Status = value
		case "tags":
			plan.Goal.Tags = append(plan.Goal.Tags, parseTagList(value)...)
		}
		return
	}

	project := &plan.Projects[currentProject]
	switch key {
	case "project", "name", "title":
		project.Name = value
	case "description", "notes":
		project.Notes = value
		if project.Description == "" {
			project.Description = value
		}
	case "type", "project type":
		project.ProjectType = models.NormalizeRoadmapProjectType(value)
	case "kind", "project kind":
		project.ProjectKind = value
	case "deadline", "due", "target", "target date":
		project.Deadline = firstNonEmpty(parseFlexibleDate(value, today), inferDateFromText(value, today))
	case "status":
		project.Status = value
	case "depends on", "depends-on", "dependency":
		project.DependsOn = value
	case "auto unlock", "auto-unlock":
		project.AutoUnlock = truthy(value)
	case "unlock notes", "unlock-notes":
		project.UnlockNotes = value
	case "tags":
		project.Tags = append(project.Tags, parseTagList(value)...)
	case "private":
		project.IsPrivate = truthy(value)
	case "local only", "local-only", "local/private", "local private":
		project.LocalOnly = truthy(value)
		project.IsPrivate = true
	case "current focus", "focus":
		project.CurrentFocus = truthy(value)
	case "public summary", "summary":
		project.PublicSummary = value
	}
}

func handleRoadmapBullet(plan *models.RoadmapImportPlan, currentProject *int, section, bullet string, today time.Time) {
	entry := parseRoadmapEntry(bullet, today)
	lowerTitle := strings.ToLower(entry.Title)

	if strings.HasPrefix(lowerTitle, "project:") || section == "projects" {
		project := projectPlanFromTitle(strings.TrimSpace(strings.TrimPrefix(entry.Title, "Project:")), today)
		project.Description = entry.Description
		project.ProjectType = firstNonEmpty(entry.ProjectType, project.ProjectType)
		project.Status = entry.Status
		project.Deadline = entry.DueDate
		project.Tags = entry.Tags
		project.IsPrivate = entry.IsPrivate
		project.LocalOnly = entry.LocalOnly
		project.CurrentFocus = entry.CurrentFocus
		project.Resources = entry.Resources
		project.AutoUnlock = true
		plan.Projects = append(plan.Projects, project)
		*currentProject = len(plan.Projects) - 1
		return
	}

	if *currentProject < 0 {
		plan.Projects = append(plan.Projects, models.RoadmapProjectPlan{
			Name:        "AI Engineer Roadmap Execution",
			ProjectType: "learning",
			Status:      "active",
			Deadline:    plan.Goal.Deadline,
		})
		*currentProject = len(plan.Projects) - 1
	}
	project := &plan.Projects[*currentProject]

	switch section {
	case "milestones":
		project.Milestones = append(project.Milestones, models.RoadmapMilestonePlan{
			Title:       entry.Title,
			Description: entry.Description,
			Status:      entry.Status,
			DueDate:     entry.DueDate,
		})
	case "resources":
		if len(entry.Resources) > 0 {
			project.Resources = append(project.Resources, entry.Resources...)
		} else {
			project.Resources = append(project.Resources, models.RoadmapResourcePlan{
				Title:        entry.Title,
				ResourceType: models.NormalizeRoadmapResourceType(entry.Title),
				Notes:        entry.Description,
				IsPrivate:    entry.IsPrivate,
			})
		}
	default:
		if len(entry.Resources) > 0 && looksLikeResourceOnly(entry.Title) {
			project.Resources = append(project.Resources, entry.Resources...)
			return
		}
		project.Tasks = append(project.Tasks, models.RoadmapTaskPlan{
			Title:         entry.Title,
			Description:   entry.Description,
			Status:        entry.Status,
			Priority:      entry.Priority,
			DueDate:       entry.DueDate,
			StarsReward:   entry.StarsReward,
			ProofRequired: entry.ProofRequired,
			ProofType:     entry.ProofType,
		})
		project.Resources = append(project.Resources, entry.Resources...)
		project.Tags = append(project.Tags, entry.Tags...)
		if entry.IsPrivate {
			project.IsPrivate = true
		}
		if entry.LocalOnly {
			project.LocalOnly = true
			project.IsPrivate = true
		}
		if entry.CurrentFocus {
			project.CurrentFocus = true
		}
	}
}

type roadmapEntry struct {
	Title         string
	Description   string
	Status        string
	Priority      string
	DueDate       string
	Tags          []string
	Resources     []models.RoadmapResourcePlan
	IsPrivate     bool
	LocalOnly     bool
	CurrentFocus  bool
	ProjectType   string
	StarsReward   int
	ProofRequired bool
	ProofType     string
}

func parseRoadmapEntry(value string, today time.Time) roadmapEntry {
	entry := roadmapEntry{
		Status:      "pending",
		Priority:    "medium",
		StarsReward: 1,
		ProofType:   "none",
	}
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "[x]") || strings.HasPrefix(lower, "[done]") {
		entry.Status = "completed"
		value = strings.TrimSpace(value[strings.Index(value, "]")+1:])
	} else if strings.HasPrefix(lower, "[ ]") {
		value = strings.TrimSpace(value[3:])
	}

	entry.Resources = extractResources(value)
	entry.Tags = parseTags(value)
	entry.DueDate = inferDateFromText(value, today)
	entry.ProjectType = inferProjectType(value)
	entry.IsPrivate = containsPrivacySignal(value)
	entry.LocalOnly = containsLocalOnlySignal(value)
	entry.CurrentFocus = strings.Contains(lower, "current focus") || strings.Contains(lower, "focus: true")
	entry.Status = statusFromText(value, entry.Status)
	entry.Priority = priorityFromText(value)
	entry.StarsReward = starsFromText(value)
	entry.ProofType = proofTypeFromText(value)
	entry.ProofRequired = entry.ProofType != "none" || strings.Contains(lower, "proof required")

	title := cleanRoadmapTitle(value)
	if strings.Contains(title, " - ") {
		parts := strings.SplitN(title, " - ", 2)
		title = strings.TrimSpace(parts[0])
		entry.Description = strings.TrimSpace(parts[1])
	}
	entry.Title = strings.TrimSpace(title)
	if entry.Title == "" && len(entry.Resources) > 0 {
		entry.Title = entry.Resources[0].Title
	}
	return entry
}

func markdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level:]), true
}

func markdownBullet(line string) (string, bool) {
	for _, prefix := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", false
}

func metadataLine(line string) (string, string, bool) {
	matches := metaLineRE.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
}

func projectPlanFromTitle(title string, today time.Time) models.RoadmapProjectPlan {
	entry := parseRoadmapEntry(title, today)
	name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(entry.Title, "Project:"), "project:"))
	if name == "" {
		name = title
	}
	return models.RoadmapProjectPlan{
		Name:          name,
		Description:   entry.Description,
		ProjectType:   firstNonEmpty(entry.ProjectType, inferProjectType(name)),
		ProjectKind:   "main",
		Status:        entry.Status,
		Deadline:      entry.DueDate,
		Tags:          entry.Tags,
		IsPrivate:     entry.IsPrivate,
		LocalOnly:     entry.LocalOnly,
		CurrentFocus:  entry.CurrentFocus,
		PublicSummary: entry.Description,
		Resources:     entry.Resources,
		AutoUnlock:    true,
	}
}

func extractResources(value string) []models.RoadmapResourcePlan {
	var resources []models.RoadmapResourcePlan
	seen := map[string]bool{}
	for _, match := range markdownLinkRE.FindAllStringSubmatch(value, -1) {
		if len(match) != 3 {
			continue
		}
		url := strings.TrimRight(match[2], ".,;")
		seen[url] = true
		resources = append(resources, models.RoadmapResourcePlan{
			Title:        strings.TrimSpace(match[1]),
			URL:          url,
			ResourceType: models.NormalizeRoadmapResourceType(url + " " + match[1]),
			IsPrivate:    containsPrivacySignal(value),
		})
	}
	for _, url := range bareURLRE.FindAllString(value, -1) {
		url = strings.TrimRight(url, ".,;")
		if seen[url] {
			continue
		}
		resources = append(resources, models.RoadmapResourcePlan{
			Title:        url,
			URL:          url,
			ResourceType: models.NormalizeRoadmapResourceType(url),
			IsPrivate:    containsPrivacySignal(value),
		})
	}
	return resources
}

func parseTags(value string) []string {
	var tags []string
	for _, match := range hashTagRE.FindAllStringSubmatch(value, -1) {
		if len(match) == 2 {
			tags = append(tags, match[1])
		}
	}
	if key, raw, ok := metadataLine(value); ok && strings.EqualFold(strings.TrimSpace(key), "tags") {
		tags = append(tags, parseTagList(raw)...)
	}
	return uniqueText(tags)
}

func parseTagList(value string) []string {
	parts := strings.Split(value, ",")
	if len(parts) == 1 {
		parts = strings.Fields(value)
	}
	var tags []string
	for _, part := range parts {
		part = strings.TrimSpace(strings.TrimPrefix(part, "#"))
		if part != "" {
			tags = append(tags, part)
		}
	}
	return uniqueText(tags)
}

func inferDateFromText(value string, today time.Time) string {
	if match := isoDateRE.FindString(value); match != "" {
		return match
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "before november") || strings.Contains(lower, "by november") {
		year := today.Year()
		if today.Month() > time.November {
			year++
		}
		return time.Date(year, time.November, 1, 0, 0, 0, 0, today.Location()).Format("2006-01-02")
	}

	for _, key := range []string{"deadline:", "due:", "target:", "by:"} {
		index := strings.Index(lower, key)
		if index < 0 {
			continue
		}
		raw := strings.TrimSpace(value[index+len(key):])
		raw = strings.TrimSpace(strings.Split(raw, "|")[0])
		raw = strings.TrimSpace(strings.Split(raw, ";")[0])
		if parsed := parseFlexibleDate(raw, today); parsed != "" {
			return parsed
		}
	}
	return ""
}

func parseFlexibleDate(value string, today time.Time) string {
	value = strings.TrimSpace(strings.Trim(value, ".,()[]"))
	if value == "" {
		return ""
	}
	for _, layout := range []string{
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"1/2/2006",
		"01/02/2006",
		"1/2/06",
	} {
		if parsed, err := time.ParseInLocation(layout, value, today.Location()); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	for _, layout := range []string{"Jan 2", "January 2", "1/2", "01/02"} {
		if parsed, err := time.ParseInLocation(layout, value, today.Location()); err == nil {
			parsed = time.Date(today.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, today.Location())
			return parsed.Format("2006-01-02")
		}
	}
	month := monthFromName(value)
	if month != 0 {
		return time.Date(today.Year(), month, 1, 0, 0, 0, 0, today.Location()).Format("2006-01-02")
	}
	return ""
}

func monthFromName(value string) time.Month {
	value = strings.ToLower(strings.TrimSpace(value))
	for month := time.January; month <= time.December; month++ {
		if value == strings.ToLower(month.String()) || value == strings.ToLower(month.String()[:3]) {
			return month
		}
	}
	return 0
}

func inferProjectType(value string) string {
	return models.NormalizeRoadmapProjectType(value)
}

func statusFromText(value, fallback string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "status: locked"):
		return "locked"
	case strings.Contains(lower, "status: upcoming"):
		return "upcoming"
	case strings.Contains(lower, "status: active"):
		return "active"
	case strings.Contains(lower, "status: completed"), strings.Contains(lower, "status: done"), strings.Contains(lower, "done: true"):
		if fallback == "pending" {
			return "completed"
		}
		return "done"
	default:
		return fallback
	}
}

func starsFromText(value string) int {
	lower := strings.ToLower(value)
	for _, key := range []string{"stars reward:", "stars:"} {
		index := strings.Index(lower, key)
		if index < 0 {
			continue
		}
		raw := strings.TrimSpace(value[index+len(key):])
		parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '|' || r == ';' || r == ',' })
		if len(parts) == 0 {
			continue
		}
		var stars int
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &stars); err == nil && stars > 0 {
			return stars
		}
	}
	return 1
}

func proofTypeFromText(value string) string {
	lower := strings.ToLower(value)
	for _, key := range []string{"proof required:", "proof:"} {
		index := strings.Index(lower, key)
		if index < 0 {
			continue
		}
		raw := strings.TrimSpace(value[index+len(key):])
		parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '|' || r == ';' || r == ',' })
		if len(parts) == 0 {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(parts[0])) {
		case "photo", "file", "link", "text":
			return strings.TrimSpace(strings.ToLower(parts[0]))
		}
	}
	return "none"
}

func priorityFromText(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "priority: high") || strings.Contains(lower, "high priority"):
		return "high"
	case strings.Contains(lower, "priority: low") || strings.Contains(lower, "low priority"):
		return "low"
	default:
		return "medium"
	}
}

func cleanRoadmapTitle(value string) string {
	title := markdownLinkRE.ReplaceAllString(value, "$1")
	title = bareURLRE.ReplaceAllString(title, "")
	title = hashTagRE.ReplaceAllString(title, "")
	for _, marker := range []string{"Deadline:", "Due:", "Target:", "By:", "Status:", "Priority:", "Type:", "Tags:", "Private:", "Local-only:", "Local only:", "Focus:", "Current focus:", "Stars:", "Stars Reward:", "Proof:", "Proof Required:", "Privacy:", "Depends On:", "Auto Unlock:", "Unlock Notes:", "Notes:"} {
		title = stripMarkerValue(title, marker)
		title = stripMarkerValue(title, strings.ToLower(marker))
	}
	title = isoDateRE.ReplaceAllString(title, "")
	title = strings.ReplaceAll(title, "()", "")
	title = strings.TrimSpace(strings.Trim(title, "-|;,."))
	return title
}

func stripMarkerValue(value, marker string) string {
	index := strings.Index(value, marker)
	if index < 0 {
		return value
	}
	tail := value[index+len(marker):]
	stop := len(tail)
	for _, sep := range []string{"|", ";"} {
		if sepIndex := strings.Index(tail, sep); sepIndex >= 0 && sepIndex < stop {
			stop = sepIndex
		}
	}
	return strings.TrimSpace(value[:index] + tail[stop:])
}

func containsPrivacySignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "private") || strings.Contains(lower, "not public")
}

func containsLocalOnlySignal(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "local-only") || strings.Contains(lower, "local only") || strings.Contains(lower, "local/private")
}

func truthy(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "yes", "true", "on", "private", "local", "local only", "local-only":
		return true
	default:
		return false
	}
}

func looksLikeResourceOnly(title string) bool {
	lower := strings.ToLower(title)
	return strings.Contains(lower, "youtube") || strings.Contains(lower, "github") || strings.Contains(lower, "course") || strings.Contains(lower, "docs")
}

func roadmapWorkCounts(items []models.ProjectItem, milestones []models.RoadmapMilestone) (int, int) {
	total := len(items) + len(milestones)
	completed := 0
	for _, item := range items {
		if item.Status == "completed" {
			completed++
		}
	}
	for _, milestone := range milestones {
		if milestone.Status == "completed" {
			completed++
		}
	}
	return total, completed
}

func roadmapWeeklyCounts(items []models.ProjectItem, milestones []models.RoadmapMilestone, start, end time.Time) (int, int) {
	total := 0
	completed := 0
	for _, item := range items {
		if !dateInRange(item.DueDate, start, end) {
			continue
		}
		total++
		if item.Status == "completed" {
			completed++
		}
	}
	for _, milestone := range milestones {
		if !dateInRange(milestone.DueDate, start, end) {
			continue
		}
		total++
		if milestone.Status == "completed" {
			completed++
		}
	}
	return total, completed
}

func missedRoadmapDeadlines(project models.RoadmapProject, today time.Time) []models.RoadmapDeadline {
	var missed []models.RoadmapDeadline
	if project.Project.Project.TargetDate != "" && project.Progress < 100 && dateBefore(project.Project.Project.TargetDate, today) {
		missed = append(missed, models.RoadmapDeadline{
			Kind:        "project",
			Title:       "Project target",
			ProjectName: project.Project.Project.Name,
			DueDate:     project.Project.Project.TargetDate,
			ProjectID:   project.Project.Project.ID,
		})
	}
	for _, item := range project.Items {
		if item.Status == "completed" || !dateBefore(item.DueDate, today) {
			continue
		}
		missed = append(missed, models.RoadmapDeadline{
			Kind:        "task",
			Title:       item.Title,
			ProjectName: project.Project.Project.Name,
			DueDate:     item.DueDate,
			ProjectID:   project.Project.Project.ID,
			ItemID:      item.ID,
		})
	}
	for _, milestone := range project.Milestones {
		if milestone.Status == "completed" || !dateBefore(milestone.DueDate, today) {
			continue
		}
		missed = append(missed, models.RoadmapDeadline{
			Kind:        "milestone",
			Title:       milestone.Title,
			ProjectName: project.Project.Project.Name,
			DueDate:     milestone.DueDate,
			ProjectID:   project.Project.Project.ID,
			ItemID:      milestone.ID,
		})
	}
	return missed
}

func currentRoadmapFocus(projects []models.RoadmapProject) []models.RoadmapProject {
	var focus []models.RoadmapProject
	for _, project := range projects {
		if project.Meta.CurrentFocus {
			focus = append(focus, project)
		}
	}
	if len(focus) > 0 {
		return focus
	}
	for _, project := range projects {
		if project.Project.Project.Status == "active" {
			return []models.RoadmapProject{project}
		}
	}
	return nil
}

func roadmapDailyTasks(projects []models.RoadmapProject, today time.Time) models.RoadmapDailyTasks {
	return models.RoadmapDailyTasks{
		AITask:     pickRoadmapTask(projects, today, func(string) bool { return true }),
		CourseTask: pickRoadmapTask(projects, today, func(projectType string) bool { return projectType == "learning" || projectType == "certification" }),
		CodingTask: pickRoadmapTask(projects, today, func(projectType string) bool {
			return projectType == "real_estate_ai" || projectType == "healthcare_ai" || projectType == "ml_model"
		}),
		PortfolioTask: pickRoadmapTask(projects, today, func(projectType string) bool { return projectType == "portfolio" }),
	}
}

type roadmapTaskCandidate struct {
	Task        models.ProjectItem
	ProjectName string
	ProjectID   int64
	ProjectType string
	DueRank     int
}

func pickRoadmapTask(projects []models.RoadmapProject, today time.Time, matches func(string) bool) models.RoadmapDailyTask {
	var candidates []roadmapTaskCandidate
	for _, project := range projects {
		projectType := project.Meta.ProjectType
		if !matches(projectType) {
			continue
		}
		for _, item := range project.Items {
			if item.Status == "completed" {
				continue
			}
			candidates = append(candidates, roadmapTaskCandidate{
				Task:        item,
				ProjectName: project.Project.Project.Name,
				ProjectID:   project.Project.Project.ID,
				ProjectType: project.Meta.ProjectTypeLabel,
				DueRank:     dueRank(item.DueDate, today),
			})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DueRank != candidates[j].DueRank {
			return candidates[i].DueRank < candidates[j].DueRank
		}
		if candidates[i].Task.DueDate != candidates[j].Task.DueDate {
			if candidates[i].Task.DueDate == "" {
				return false
			}
			if candidates[j].Task.DueDate == "" {
				return true
			}
			return candidates[i].Task.DueDate < candidates[j].Task.DueDate
		}
		return candidates[i].Task.SortOrder < candidates[j].Task.SortOrder
	})
	if len(candidates) == 0 {
		return models.RoadmapDailyTask{}
	}
	first := candidates[0]
	return models.RoadmapDailyTask{
		Title:       first.Task.Title,
		ProjectName: first.ProjectName,
		ProjectType: first.ProjectType,
		DueDate:     first.Task.DueDate,
		ProjectID:   first.ProjectID,
		ItemID:      first.Task.ID,
	}
}

func dueRank(dueDate string, today time.Time) int {
	if dueDate == "" {
		return 2
	}
	parsed, ok := parseDate(dueDate, today.Location())
	if !ok {
		return 2
	}
	if !parsed.After(today) {
		return 0
	}
	return 1
}

func dateInRange(value string, start, end time.Time) bool {
	parsed, ok := parseDate(value, start.Location())
	if !ok {
		return false
	}
	return !parsed.Before(start) && !parsed.After(end)
}

func dateBefore(value string, today time.Time) bool {
	parsed, ok := parseDate(value, today.Location())
	if !ok {
		return false
	}
	return parsed.Before(today)
}

func weekBounds(today time.Time) (time.Time, time.Time) {
	weekday := int(today.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := today.AddDate(0, 0, -(weekday - 1))
	end := start.AddDate(0, 0, 6)
	return start, end
}

func percentInt(completed, total int) int {
	if total <= 0 {
		return 0
	}
	percent := (completed * 100) / total
	if percent > 100 {
		return 100
	}
	if percent < 0 {
		return 0
	}
	return percent
}

func checkboxMark(status string) string {
	if status == "completed" {
		return "x"
	}
	return " "
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueText(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}
