package coder

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/linager/inspector"
	"github.com/viant/linager/inspector/golang"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/java"
	"github.com/viant/linager/inspector/repository"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Coder provides functionality for creating, removing, and recomposing packages, files, types, and their components.
// It enables runtime reassembly of types from selected fields/methods and static variables for further processing or transformation.
// It also enables applying patch diffs at the function, struct, and field levels.
type Coder struct {
	Project *graph.Project // The project being manipulated
	fs      afs.Service
}

// NewCoder creates a new Coder instance for the given project
func NewCoder(project *graph.Project) *Coder {
	return &Coder{
		Project: project,
	}
}

// CreatePackage creates a new package in the project
func (c *Coder) CreatePackage(name, importPath string) *graph.Package {
	pkg := &graph.Package{
		Name:       name,
		ImportPath: importPath,
		FileSet:    []*graph.File{},
		Assets:     []*graph.Asset{},
	}

	// Add the package to the project
	c.Project.Packages = append(c.Project.Packages, pkg)

	return pkg
}

// RemovePackage removes a package from the project by name
func (c *Coder) RemovePackage(name string) bool {
	// Find the package index
	var idx int
	var found bool
	for i, pkg := range c.Project.Packages {
		if pkg.Name == name {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return false
	}

	// Remove the package from the packages slice
	c.Project.Packages = append(c.Project.Packages[:idx], c.Project.Packages[idx+1:]...)

	return true
}

// CreateFile creates a new file in the specified package
func (c *Coder) CreateFile(packageName, fileName, filePath string) (*graph.File, error) {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	file := &graph.File{
		Name:       fileName,
		Path:       filePath,
		Package:    packageName,
		ImportPath: pkg.ImportPath,
		Types:      []*graph.Type{},
		Constants:  []*graph.Constant{},
		Variables:  []*graph.Variable{},
		Functions:  []*graph.Function{},
		Imports:    []graph.Import{},
	}

	// Add the file to the package
	pkg.AddFile(file)

	return file, nil
}

// RemoveFile removes a file from the specified package by name
func (c *Coder) RemoveFile(packageName, fileName string) bool {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return false
	}

	// Find the file index
	var idx int
	var found bool
	for i, file := range pkg.FileSet {
		if file.Name == fileName {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return false
	}

	// Remove the file from the files slice
	pkg.FileSet = append(pkg.FileSet[:idx], pkg.FileSet[idx+1:]...)

	return true
}

// CreateType creates a new type in the specified file
func (c *Coder) CreateType(packageName, fileName, typeName string, kind reflect.Kind) (*graph.Type, error) {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("file %s not found in package %s", fileName, packageName)
	}

	// Create a new type
	newType := &graph.Type{
		Name:        typeName,
		Kind:        kind,
		Package:     packageName,
		PackagePath: pkg.ImportPath,
		IsExported:  strings.ToUpper(typeName[:1]) == typeName[:1],
		Fields:      []*graph.Field{},
		Methods:     []*graph.Function{},
	}

	// Add the type to the file
	file.Types = append(file.Types, newType)
	file.IndexTypes()

	return newType, nil
}

// RemoveType removes a type from the specified file by name
func (c *Coder) RemoveType(packageName, fileName, typeName string) bool {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return false
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return false
	}

	// Find the type index
	var idx int
	var found bool
	for i, typ := range file.Types {
		if typ.Name == typeName {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return false
	}

	// Remove the type from the types slice
	file.Types = append(file.Types[:idx], file.Types[idx+1:]...)
	file.IndexTypes()

	return true
}

// CreateField creates a new field in the specified type
func (c *Coder) CreateField(packageName, fileName, typeName, fieldName string, fieldType *graph.Type, tag reflect.StructTag) (*graph.Field, error) {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("file %s not found in package %s", fileName, packageName)
	}

	var typ *graph.Type
	for _, t := range file.Types {
		if t.Name == typeName {
			typ = t
			break
		}
	}

	if typ == nil {
		return nil, fmt.Errorf("type %s not found in file %s", typeName, fileName)
	}

	// Create a new field
	field := &graph.Field{
		Name:       fieldName,
		Type:       fieldType,
		Tag:        tag,
		IsExported: strings.ToUpper(fieldName[:1]) == fieldName[:1],
	}

	// Add the field to the type
	typ.AddField(field)

	return field, nil
}

// RemoveField removes a field from the specified type by name
func (c *Coder) RemoveField(packageName, fileName, typeName, fieldName string) bool {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return false
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return false
	}

	var typ *graph.Type
	for _, t := range file.Types {
		if t.Name == typeName {
			typ = t
			break
		}
	}

	if typ == nil {
		return false
	}

	return typ.RemoveField(fieldName)
}

// CreateMethod creates a new method for the specified type
func (c *Coder) CreateMethod(packageName, fileName, typeName, methodName string, parameters []*graph.Parameter, results []*graph.Parameter, body string) (*graph.Function, error) {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("file %s not found in package %s", fileName, packageName)
	}

	var typ *graph.Type
	for _, t := range file.Types {
		if t.Name == typeName {
			typ = t
			break
		}
	}

	if typ == nil {
		return nil, fmt.Errorf("type %s not found in file %s", typeName, fileName)
	}

	// Create a new method
	method := &graph.Function{
		Name:       methodName,
		Receiver:   typeName,
		Parameters: parameters,
		Results:    results,
		Body:       &graph.LocationNode{Text: body},
		IsExported: strings.ToUpper(methodName[:1]) == methodName[:1],
	}

	// Add the method to the type
	typ.AddMethod(method)

	return method, nil
}

// RemoveMethod removes a method from the specified type by name
func (c *Coder) RemoveMethod(packageName, fileName, typeName, methodName string) bool {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return false
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return false
	}

	var typ *graph.Type
	for _, t := range file.Types {
		if t.Name == typeName {
			typ = t
			break
		}
	}

	if typ == nil {
		return false
	}

	return typ.RemoveMethod(methodName)
}

// CreateFunction creates a new function in the specified file
func (c *Coder) CreateFunction(packageName, fileName, functionName string, parameters []*graph.Parameter, results []*graph.Parameter, body string) (*graph.Function, error) {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("file %s not found in package %s", fileName, packageName)
	}

	// Create a new function
	function := &graph.Function{
		Name:       functionName,
		Parameters: parameters,
		Results:    results,
		Body:       &graph.LocationNode{Text: body},
		IsExported: strings.ToUpper(functionName[:1]) == functionName[:1],
	}

	// Add the function to the file
	file.Functions = append(file.Functions, function)
	file.IndexFunctions()

	return function, nil
}

// RemoveFunction removes a function from the specified file by name
func (c *Coder) RemoveFunction(packageName, fileName, functionName string) bool {
	pkg := c.Project.GetPackage(packageName)
	if pkg == nil {
		return false
	}

	var file *graph.File
	for _, f := range pkg.FileSet {
		if f.Name == fileName {
			file = f
			break
		}
	}

	if file == nil {
		return false
	}

	// Find the function index
	var idx int
	var found bool
	for i, function := range file.Functions {
		if function.Name == functionName {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return false
	}

	// Remove the function from the functions slice
	file.Functions = append(file.Functions[:idx], file.Functions[idx+1:]...)
	file.IndexFunctions()

	return true
}

// LoadProject loads a project from the specified location using an inspector
func (c *Coder) LoadProject(ctx context.Context, location string) error {
	// Create a repository project
	repoProject := &repository.Project{
		RootPath: location,
	}

	// Detect project type
	detector := repository.New()
	detectedProject, err := detector.DetectProject(location)
	if err != nil {
		return fmt.Errorf("failed to detect project: %w", err)
	}

	// Update repository project with detected information
	repoProject.Type = detectedProject.Type
	repoProject.Name = detectedProject.Name

	// Create an inspector factory
	factory := inspector.NewFactory(nil)

	// Inspect the project
	project, err := factory.InspectProject(repoProject)
	if err != nil {
		return fmt.Errorf("failed to inspect project: %w", err)
	}

	// Set the project
	c.Project = project

	return nil
}

// StoreProject stores the project to the specified URL
func (c *Coder) StoreProject(ctx context.Context, url string) error {
	if c.Project == nil {
		return fmt.Errorf("no project to store")
	}

	// Iterate through all packages in the project
	for _, pkg := range c.Project.Packages {
		// Iterate through all files in the package
		for _, file := range pkg.FileSet {

			contentGenerator := lookupEmitter(file)
			if contentGenerator == nil {
				continue
			}
			// Reconstruct the file content
			content, err := file.Content(contentGenerator)
			if err != nil {
				return fmt.Errorf("failed to reconstruct file content for %s: %w", file.Path, err)
			}

			// Construct the full path to the file
			filePath := filepath.Join(url, file.Path)

			// Ensure the directory exists
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			// Store the file
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				return fmt.Errorf("failed to store file %s: %w", filePath, err)
			}
		}

		// Store any assets associated with the package
		for _, asset := range pkg.Assets {
			if asset.Content == nil || len(asset.Content) == 0 {
				continue
			}

			// Construct the full path to the asset
			assetPath := filepath.Join(url, asset.Path)

			// Ensure the directory exists
			dir := filepath.Dir(assetPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			// Store the asset
			if err := os.WriteFile(assetPath, asset.Content, 0644); err != nil {
				return fmt.Errorf("failed to store asset %s: %w", assetPath, err)
			}
		}
	}

	return nil
}

func lookupEmitter(file *graph.File) graph.Emitter {
	ext := filepath.Ext(file.Path)
	switch ext {
	case ".go":
		return &golang.Emitter{}
	case ".java":
		return &java.Emitter{}
	}
	return nil
}
