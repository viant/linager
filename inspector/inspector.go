package inspector

import (
	"fmt"
	"github.com/viant/linager/inspector/repository"
	"path/filepath"
	"strings"

	"github.com/viant/linager/inspector/golang"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/java"
	"github.com/viant/linager/inspector/javascript"
)

// Inspector provides an interface for inspecting source code
type Inspector interface {
	// InspectSource parses source code from a byte slice and extracts type information
	InspectSource(src []byte) (*graph.File, error)

	// InspectFile parses a source file and extracts type information
	InspectFile(filename string) (*graph.File, error)

	// InspectPackage inspects a package directory and extracts all type information
	InspectPackage(packagePath string) (*graph.Package, error)

	// InspectProject inspects a project directory and extracts all type information
	InspectProject(location string) (*graph.Project, error)
}

// Factory creates appropriate inspectors based on language
type Factory struct {
	config *graph.Config
}

// NewFactory creates a new inspector factory with the given config
func NewFactory(config *graph.Config) *Factory {
	if config == nil {
		config = &graph.Config{
			IncludeUnexported: true,
			SkipTests:         true,
		}
	}
	return &Factory{
		config: config,
	}
}

// GetInspector returns an appropriate inspector based on file extension
func (f *Factory) GetInspector(filename string) (Inspector, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".go":
		return golang.NewInspector(f.config), nil
	case ".java":
		return java.NewInspector(f.config), nil
	case ".js", ".jsx":
		return javascript.NewInspector(f.config), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// InspectFile is a convenience method that gets the appropriate inspector and inspects the file
func (f *Factory) InspectFile(filename string) (*graph.File, error) {
	inspector, err := f.GetInspector(filename)
	if err != nil {
		return nil, err
	}

	return inspector.InspectFile(filename)
}

// InspectPackage is a convenience method that gets the appropriate inspector for a package
func (f *Factory) InspectPackage(packagePath string) (*graph.Package, error) {
	// Try to determine language from files in the directory
	entries, err := filepath.Glob(filepath.Join(packagePath, "*"))
	if err != nil {
		return nil, fmt.Errorf("failed to read package directory: %w", err)
	}

	// Look for source files to determine language
	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry))
		switch ext {
		case ".go":
			inspector := golang.NewInspector(f.config)
			return inspector.InspectPackage(packagePath)
		case ".java":
			inspector := java.NewInspector(f.config)
			return inspector.InspectPackage(packagePath)
		case ".js", ".jsx":
			inspector := javascript.NewInspector(f.config)
			return inspector.InspectPackage(packagePath)
		}
	}

	return nil, fmt.Errorf("unable to determine language for package: %s", packagePath)
}

// InspectProject is a convenience method that gets the appropriate inspector for a package
func (f *Factory) InspectProject(project *repository.Project) (*graph.Project, error) {
	switch project.Type {
	case "go":
		return golang.NewInspector(f.config).InspectProject(project.RootPath)
	case "java":
		return java.NewInspector(f.config).InspectProject(project.RootPath)
	case "javascript":
		return javascript.NewInspector(f.config).InspectProject(project.RootPath)
	}
	return nil, nil
}
