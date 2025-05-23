package golang

import (
	"fmt"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/repository"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
)

// InspectPackage inspects a Go package directory and extracts all types
// This version loads only one package per folder (no recursive option)
func (i *Inspector) InspectPackage(packagePath string) (*graph.Package, error) {
	// Get the absolute path of the package
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create the Package to hold all discovered files and types
	pkg := &graph.Package{
		ImportPath: getImportPath(absPath),
	}

	// Process the single package directory
	pkgFiles, assets, err := i.inspectSinglePackage(absPath)
	if err != nil {
		return nil, fmt.Errorf("error processing package in %s: %w", absPath, err)
	}

	// Use the most common package name if multiple are found
	if pkg.Name == "" && len(pkgFiles) > 0 {
		pkg.Name = pkgFiles[0].Package
	}
	pkg.FileSet = pkgFiles
	pkg.Assets = assets

	if len(pkg.FileSet) == 0 {
		return nil, fmt.Errorf("no Go files found in package: %s", packagePath)
	}

	return pkg, nil
}

// InspectPackages inspects multiple Go package directories recursively
func (i *Inspector) InspectPackages(rootPath string) ([]*graph.Package, error) {
	// Get the absolute path of the root directory
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	var packags []*graph.Package

	// Walk the directory tree to find all potential package directories
	err = filepath.Walk(absPath, func(aPath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			return nil
		}

		var exclusion []string
		if i.config.SkipTests {
			exclusion = []string{"_test.go"}
		}
		// Check if directory has Go files
		hasGoFiles, err := repository.HasFileWithSuffixes(aPath, []string{".go"}, exclusion)
		if err != nil {
			return err
		}
		if hasGoFiles {
			pkg, err := i.InspectPackage(aPath)
			if err != nil {
				return fmt.Errorf("error inspecting package in %s: %w", aPath, err)
			}
			packags = append(packags, pkg)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking package directories: %w", err)
	}
	return packags, nil
}

// inspectSinglePackage processes a single directory as a Go package
func (i *Inspector) inspectSinglePackage(packageDir string) ([]*graph.File, []*graph.Asset, error) {
	var files []*graph.File
	var assets []*graph.Asset

	// Process Go files
	pkgs, err := parser.ParseDir(i.fset, packageDir, func(info os.FileInfo) bool {
		// Skip test files unless configured to include them
		if i.config.SkipTests && strings.HasSuffix(info.Name(), "_test.go") {
			return false
		}
		return true
	}, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse package: %w", err)
	}

	// Process each package (main, tests, etc.)
	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			// Read file content for method body extraction
			src, err := os.ReadFile(filename)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read file %s: %w", filename, err)
			}
			i.src = src

			aFile, err := i.processFile(file, filename)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to process file %s: %w", filename, err)
			}
			files = append(files, aFile)
		}
	}

	// Process non-Go files as assets if AllFilesInFolder is enabled
	if !i.config.SkipAsset {
		assets, err = repository.ReadAssetsRecursively(packageDir, true, getImportPath, "go")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read assets: %w", err)
		}
	}
	return files, assets, nil
}

// processParameters processes function parameters and extracts parameter information
func (i *Inspector) processParameters(fields *ast.FieldList, importMap map[string]string) []graph.Parameter {
	var result []graph.Parameter

	for _, field := range fields.List {
		paramType := exprToString(field.Type, importMap)

		if len(field.Names) > 0 {
			for _, name := range field.Names {
				result = append(result, graph.Parameter{
					Name: name.Name,
					Type: &graph.Type{Name: paramType},
				})
			}
		} else {
			// Unnamed parameter
			result = append(result, graph.Parameter{
				Name: "",
				Type: &graph.Type{Name: paramType},
			})
		}
	}

	return result
}

func (i *Inspector) readAssetsRecursively(packageDir string, isRoot bool) ([]*graph.Asset, error) {
	var assets []*graph.Asset
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	var subFolders []string
	var hasGoFiles bool
	for _, entry := range entries {
		if entry.IsDir() {
			subFolders = append(subFolders, entry.Name())
			continue
		}

		// Skip Go files (already processed)
		if strings.HasSuffix(entry.Name(), ".go") {
			hasGoFiles = true

			continue
		}

		// Skip test files if configured
		if i.config.SkipTests && strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		// Process as asset
		filePath := filepath.Join(packageDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read asset %s: %w", filePath, err)
		}

		asset := &graph.Asset{
			Path:       filePath,
			ImportPath: getImportPath(packageDir),
			Content:    content,
		}
		assets = append(assets, asset)
	}

	if hasGoFiles && !isRoot {
		return []*graph.Asset{}, nil
	}

	for _, subFolder := range subFolders {
		subAssets, err := i.readAssetsRecursively(filepath.Join(packageDir, subFolder), false)
		if err != nil {
			return nil, fmt.Errorf("failed to read assets in subfolder %s: %w", subFolder, err)
		}
		assets = append(assets, subAssets...)
	}

	return assets, nil
}
