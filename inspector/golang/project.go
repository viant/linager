package golang

import (
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/repository"
)

// InspectProject parses a Go source file and extracts types
func (i *Inspector) InspectProject(location string) (*graph.Project, error) {
	detector := repository.New()
	project := &graph.Project{}
	if info, err := detector.DetectProject(location); err == nil {
		project.Name = info.Name
		project.Type = info.Type
		project.RootPath = info.RootPath
	}
	if info, err := detector.DetectRepository(location); err == nil {
		project.RepositoryURL = info.Origin
		if info.Root != "" {
			location = info.Root
		}
	}

	var err error
	if project.Packages, err = i.InspectPackages(location); err != nil {
		return nil, err
	}
	project.Init()

	return project, nil
}
