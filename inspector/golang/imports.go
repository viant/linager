package golang

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// ImportSpec represents a parsed import statement
type ImportSpec struct {
	Name string // Local package name (may be empty for default)
	Path string // Full import path
}

// ParseImports extracts import information from a Go file
func ParseImports(file *ast.File) []ImportSpec {
	imports := make([]ImportSpec, 0, len(file.Imports))

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		name := ""

		// If import has an explicit name
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Otherwise, use the last segment of the path
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}

		imports = append(imports, ImportSpec{
			Name: name,
			Path: path,
		})
	}

	return imports
}

// FindPackageDir attempts to locate a package directory by its import path
func FindPackageDir(importPath string) (string, error) {
	// Check in GOROOT first
	goRoot := os.Getenv("GOROOT")
	if goRoot != "" {
		dir := filepath.Join(goRoot, "src", importPath)
		if dirExists(dir) {
			return dir, nil
		}
	}

	// Then check in GOPATH
	goPath := os.Getenv("GOPATH")
	if goPath != "" {
		dir := filepath.Join(goPath, "src", importPath)
		if dirExists(dir) {
			return dir, nil
		}
	}

	// Try in the current module
	goModCache := filepath.Join(os.Getenv("HOME"), "go", "pkg", "mod")
	if dirExists(goModCache) {
		// This is a simplified approach; real implementation would parse go.mod
		entries, err := os.ReadDir(goModCache)
		if err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), importPath) {
					return filepath.Join(goModCache, entry.Name()), nil
				}
			}
		}
	}

	return "", os.ErrNotExist
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
