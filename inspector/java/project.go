package java

import (
	"github.com/viant/linager/inspector/info"
	"github.com/viant/linager/inspector/repository"
)

// InspectProject parses a Go source file and extracts types
func (i *Inspector) InspectProject(location string) (*info.Project, error) {
	detector := repository.New()
	project := &info.Project{}
	if info, err := detector.DetectProject(location); err == nil {
		project.Name = info.Name
		project.Type = info.Type
		project.RootPath = info.RootPath
		if info.RootPath != "" {
			location = info.RootPath
		}
	}
	if info, err := detector.DetectRepository(location); err == nil {
		project.RepositoryURL = info.Origin

	}
	var err error
	if project.Packages, err = i.InspectPackages(location); err != nil {
		return nil, err
	}

	project.Init()
	return project, nil
}
