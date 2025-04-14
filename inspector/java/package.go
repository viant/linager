package java

import (
	"fmt"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/repository"
	"os"
	"path/filepath"
)

// InspectPackages inspects multiple Go package directories recursively
func (i *Inspector) InspectPackages(rootPath string) ([]*graph.Package, error) {
	// Get the absolute path of the root directory
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	var packages []*graph.Package

	// Walk the directory tree to find all potential package directories
	err = filepath.Walk(absPath, func(aPath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			return nil
		}
		var exclusion []string
		hasJavaFiles, err := repository.HasFileWithSuffixes(aPath, []string{".java"}, exclusion)
		if err != nil {
			return err
		}
		if hasJavaFiles {
			pkg, err := i.InspectPackage(aPath)
			if err != nil {
				return fmt.Errorf("error inspecting package in %s: %w", aPath, err)
			}
			packages = append(packages, pkg)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking package directories: %w", err)
	}
	return packages, nil
}
