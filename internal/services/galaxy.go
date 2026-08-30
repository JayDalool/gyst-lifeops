package services

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"strconv"

	"lifeops-tracker/internal/models"
)

func (s *Service) GalaxyData() ([]models.GalaxyPlanet, error) {
	projects, err := s.Repo.ListProjects(false)
	if err != nil {
		return nil, err
	}
	children := make(map[int64][]models.ProjectWithProgress)
	for _, project := range projects {
		if project.Project.ParentProjectID != 0 && !galaxyProjectComplete(project.Project.Status) {
			children[project.Project.ParentProjectID] = append(children[project.Project.ParentProjectID], project)
		}
	}
	planets := make([]models.GalaxyPlanet, 0)
	for _, project := range projects {
		if project.Project.ParentProjectID != 0 || galaxyProjectComplete(project.Project.Status) {
			continue
		}
		planet, err := s.galaxyPlanet(project, children[project.Project.ID])
		if err != nil {
			return nil, err
		}
		planets = append(planets, planet)
	}
	return planets, nil
}

func galaxyProjectComplete(status string) bool {
	return status == "done" || status == "completed" || status == "archived"
}

func (s *Service) galaxyPlanet(project models.ProjectWithProgress, moons []models.ProjectWithProgress) (models.GalaxyPlanet, error) {
	x, y, z := galaxyPosition(project.Project)
	if project.Project.PosX == nil || project.Project.PosY == nil || project.Project.PosZ == nil {
		if err := s.Repo.SetGalaxyPosition(project.Project.ID, x, y, z); err != nil {
			return models.GalaxyPlanet{}, err
		}
	}
	result := models.GalaxyPlanet{ID: project.Project.ID, Name: project.Project.Name, Category: project.Project.Category, Color: project.Project.Color, Summary: project.Project.Description, Target: project.Project.TargetDate, Next: project.NextDueItem, Missions: project.TotalItems, URL: "/projects/" + strconv.FormatInt(project.Project.ID, 10), State: RouteState(project.Project.Status, project.Progress), Size: math.Min(3, 1+float64(project.TotalItems)/8), Progress: project.Progress, X: x, Y: y, Z: z, Moons: []models.GalaxyPlanet{}}
	for _, moon := range moons {
		m, err := s.galaxyPlanet(moon, nil)
		if err != nil {
			return models.GalaxyPlanet{}, err
		}
		m.Size = math.Min(1.2, .45+float64(moon.TotalItems)/12)
		result.Moons = append(result.Moons, m)
	}
	return result, nil
}

func galaxyPosition(project models.Project) (float64, float64, float64) {
	if project.PosX != nil && project.PosY != nil && project.PosZ != nil {
		return *project.PosX, *project.PosY, *project.PosZ
	}
	hash := sha256.Sum256([]byte(project.UUID))
	unit := func(offset int) float64 {
		return float64(binary.BigEndian.Uint64(hash[offset:offset+8])) / float64(^uint64(0))
	}
	radius, theta, phi := 18+unit(0)*72, unit(8)*2*math.Pi, math.Acos(2*unit(16)-1)
	return radius * math.Sin(phi) * math.Cos(theta), radius * math.Sin(phi) * math.Sin(theta), radius * math.Cos(phi)
}
