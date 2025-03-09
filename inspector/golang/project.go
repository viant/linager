package golang

import (
	"github.com/viant/linager/inspector/info"
	"github.com/viant/linager/inspector/repository"
)

// InspectFile parses a Go source file and extracts types
func (i *Inspector) InspectProject(location string) (*info.Project, error) {
	detector := repository.New()
	project := &info.Project{}
	if info, err := detector.DetectRepository(location); err == nil {
		project.Name = info.Info.Name
		project.Type = info.Info.Type
		project.RootDir = info.Root
		project.RepositoryURL = info.Origin
		if info.Root != "" {
			location = info.Root
		}
	}

	var err error
	if project.Packages, err = i.InspectPackages(location); err != nil {
		return nil, err
	}

	return project, nil
}
