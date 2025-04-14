package graph

import (
	"path/filepath"
	"strings"
)

// Project represents a code project with multiple packages
type Project struct {
	Name          string
	Type          string
	RootPath      string
	RepositoryURL string
	Packages      []*Package
	packageMap    map[string]int //position
}

// GetPackage retrieves a constant by name from the file
func (p *Project) GetPackage(name string) *Package {
	if p.Packages == nil {
		return nil
	}
	if idx, ok := p.packageMap[name]; ok && idx < len(p.Packages) {
		return p.Packages[idx]
	}
	return nil
}

func (p *Project) Init() {
	p.adjustRelativePath()
	p.adjustPackageTypes()
}

// AdjustRelativePath initializes project-related properties, including updating file paths to be relative to project root
func (p *Project) adjustRelativePath() {
	if p.RootPath == "" {
		return
	}

	// Update all file paths to be relative to project root
	for _, pkg := range p.Packages {

		for _, file := range pkg.FileSet {
			if file.ImportPath == "" {
				file.ImportPath = pkg.ImportPath
			}

			// Make file name relative to project root
			if file.Path != "" {
				relPath, err := filepath.Rel(p.RootPath, file.Path)
				if err == nil {
					file.Name = filepath.Base(file.Path)
					file.Path = relPath
					if strings.HasSuffix(file.ImportPath, file.Name) {
						file.ImportPath, _ = filepath.Split(file.ImportPath)
						file.ImportPath = strings.TrimSuffix(file.ImportPath, "/")
					}
				}
			}
			for _, asset := range pkg.Assets {
				if asset.Path != "" {
					relPath, err := filepath.Rel(p.RootPath, asset.Path)
					if err == nil {
						asset.Name = filepath.Base(asset.Path)
						asset.Path = relPath
					}
				}
			}

			// Update type information with full package paths
			for _, t := range file.Types {
				// If package info is missing but we have import path, set it
				if t.Package == "" && pkg.ImportPath != "" {
					t.Package = pkg.Name
					t.PackagePath = pkg.ImportPath
				}
			}
		}
	}
}

// AdjustPackageTypes initializes project-related properties, including updating file paths to be relative to project root
func (p *Project) adjustPackageTypes() {
	if p.RootPath == "" {
		return
	}
	// Update all file paths to be relative to project root
	for _, pkg := range p.Packages {
		for _, file := range pkg.FileSet {
			// Update type information with full package paths
			for _, t := range file.Types {
				// If package info is missing but we have import path, set it
				if t.Package == "" && pkg.ImportPath != "" {
					t.Package = pkg.Name
					t.PackagePath = pkg.ImportPath
				}
			}
		}
	}
}
