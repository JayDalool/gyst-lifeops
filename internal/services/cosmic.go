package services

import (
	"fmt"
	"strconv"

	"lifeops-tracker/internal/models"
)

// projectNode builds a galaxy/solar route node from a project.
func projectNode(p models.ProjectWithProgress) models.RouteNode {
	return models.RouteNode{
		Name:  p.Project.Name,
		URL:   "/projects/" + strconv.FormatInt(p.Project.ID, 10),
		State: RouteState(p.Project.Status, p.Progress),
	}
}

// TopLevelProjectsByRoadmap exposes the roadmap's top-level destinations
// (parent_project_id IS NULL). Subprojects are excluded by design.
func (s *Service) TopLevelProjectsByRoadmap(roadmapID int64) ([]models.ProjectWithProgress, error) {
	return s.Repo.TopLevelProjectsByRoadmap(roadmapID)
}

// RocketRouteForRoadmap builds the galaxy route for a roadmap: top-level
// destinations only. Subprojects never appear here.
func (s *Service) RocketRouteForRoadmap(roadmapID int64, title string) (models.RocketRoute, error) {
	projects, err := s.Repo.TopLevelProjectsByRoadmap(roadmapID)
	if err != nil {
		return models.RocketRoute{}, err
	}
	nodes := make([]models.RouteNode, 0, len(projects))
	for _, p := range projects {
		nodes = append(nodes, projectNode(p))
	}
	return BuildRocketRoute("galaxy", title, "galaxy", nodes), nil
}

// RocketRouteForProject builds the solar route for one project: its child
// projects as orbiting stations, falling back to its missions when it has no
// children, so a leaf project still gets a route.
func (s *Service) RocketRouteForProject(projectID int64) (models.RocketRoute, error) {
	children, err := s.Repo.SubProjects(projectID)
	if err != nil {
		return models.RocketRoute{}, err
	}
	nodes := make([]models.RouteNode, 0, len(children))
	for _, p := range children {
		nodes = append(nodes, projectNode(p))
	}
	if len(nodes) == 0 {
		items, err := s.Repo.ProjectItems(projectID)
		if err != nil {
			return models.RocketRoute{}, err
		}
		for _, item := range items {
			state := "active"
			if item.Status == "completed" {
				state = "done"
			}
			nodes = append(nodes, models.RouteNode{
				Name:  item.Title,
				URL:   fmt.Sprintf("/projects/%d#mission-%d", projectID, item.ID),
				State: state,
			})
		}
	}
	return BuildRocketRoute("solar", "Solar System", "solar", nodes), nil
}

// ProjectSolarSystem assembles a project's system view: the project at the
// centre, its child projects orbiting, its missions, and the solar route.
func (s *Service) ProjectSolarSystem(projectID int64) (models.SolarSystem, error) {
	center, err := s.Repo.ProjectWithProgress(projectID)
	if err != nil {
		return models.SolarSystem{}, err
	}
	children, err := s.Repo.SubProjects(projectID)
	if err != nil {
		return models.SolarSystem{}, err
	}
	missions, err := s.Repo.ProjectItems(projectID)
	if err != nil {
		return models.SolarSystem{}, err
	}
	route, err := s.RocketRouteForProject(projectID)
	if err != nil {
		return models.SolarSystem{}, err
	}
	var roadmap models.Roadmap
	hasRoadmap := false
	if center.Project.RoadmapID != 0 {
		if found, rmErr := s.Repo.GetRoadmap(center.Project.RoadmapID); rmErr == nil {
			roadmap = found
			hasRoadmap = true
		}
	}
	return models.SolarSystem{
		Center:     center,
		Roadmap:    roadmap,
		HasRoadmap: hasRoadmap,
		Children:   children,
		Missions:   missions,
		Route:      route,
	}, nil
}
