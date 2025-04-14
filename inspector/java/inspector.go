package java

import (
	"context"
	"fmt"
	"os"
	path "path"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/viant/linager/inspector/graph"
)

// Inspector provides functionality to inspect Java code and extract type information
type Inspector struct {
	config    *graph.Config
	importMap map[string]string
	source    []byte
}

// NewInspector creates a new Java Inspector with the provided configuration
func NewInspector(config *graph.Config) *Inspector {
	if config == nil {
		config = &graph.Config{
			IncludeUnexported: true,
			SkipTests:         false,
			RecursivePackages: false,
		}
	}
	return &Inspector{
		config: config,
	}
}

// InspectSource parses Java source code from a byte slice and extracts types
func (i *Inspector) InspectSource(src []byte) (*graph.File, error) {
	i.source = src

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	rootNode := tree.RootNode()

	return i.processJavaFile(rootNode, src, "source.java")
}

// InspectFile parses a Java source file and extracts types
func (i *Inspector) InspectFile(filename string) (*graph.File, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	i.source = src

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	rootNode := tree.RootNode()

	aFile, err := i.processJavaFile(rootNode, src, filename)
	if err != nil {
		return nil, err
	}

	// Extract imports
	for _, importNode := range findImportNodes(rootNode) {
		imports := parseImportDeclarations(importNode, src)
		for name, path := range imports {
			aFile.Imports = append(aFile.Imports, graph.Import{
				Name: name,
				Path: path,
			})
		}
	}

	return aFile, nil
}

// findImportNodes finds all import declaration nodes in the AST
func findImportNodes(rootNode *sitter.Node) []*sitter.Node {
	var importNodes []*sitter.Node

	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))
		if childNode.Type() == "import_declaration" {
			importNodes = append(importNodes, childNode)
		}
	}

	return importNodes
}

// InspectPackage inspects a Java package directory and extracts all types
func (i *Inspector) InspectPackage(packagePath string) (*graph.Package, error) {
	// Get the absolute path of the package
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	_, pkgName := path.Split(absPath)
	// Create a new Package to store all discovered types
	pkg := &graph.Package{
		FileSet: []*graph.File{},
		Name:    pkgName,
	}

	entries, err := os.ReadDir(absPath)

	for _, entry := range entries {

		filePath := path.Join(packagePath, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, err
		}

		// Skip directories
		if fileInfo.IsDir() {
			continue
		}

		// Process only .java files
		if filepath.Ext(filePath) != ".java" {
			continue
		}
		// Skip test files unless configured to include them
		if i.config.SkipTests && (filepath.Base(filePath) == "Test.java" ||
			filepath.Base(filePath) == "Tests.java" ||
			filepath.Base(filePath) == "IT.java" ||
			filepath.Base(filePath) == "ITCase.java") {
			continue
		}

		file, err := i.InspectFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error processing %s: %w", filePath, err)
		}

		// Add file to the package
		if file.ImportPath != "" {
			pkg.ImportPath = file.ImportPath
		}

		// Add file to package
		pkg.AddFile(file)

	}

	if err != nil {
		return nil, fmt.Errorf("error walking package directory: %w", err)
	}

	if len(pkg.FileSet) == 0 {
		return nil, fmt.Errorf("no Java files found in package: %s", packagePath)
	}

	return pkg, nil
}

// processJavaFile extracts package, types, constants, and variables from a Java file
func (i *Inspector) processJavaFile(rootNode *sitter.Node, src []byte, filename string) (*graph.File, error) {
	aFile := &graph.File{Path: filename}

	// Find package declaration
	var packageNode *sitter.Node
	var importNodes []*sitter.Node
	var typeNodes []*sitter.Node

	// Collect nodes by type
	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))
		switch childNode.Type() {
		case "package_declaration":
			packageNode = childNode
		case "import_declaration":
			importNodes = append(importNodes, childNode)
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration":
			typeNodes = append(typeNodes, childNode)
		}
	}

	// Process package declaration
	if packageNode != nil {
		aFile.ImportPath = parsePackageDeclaration(packageNode, src) // In Java, the package name is the import path
	}

	// Process imports
	importMap := make(map[string]string)
	for _, importNode := range importNodes {
		for k, v := range parseImportDeclarations(importNode, src) {
			importMap[k] = v
		}
	}

	// Process types
	for _, typeNode := range typeNodes {
		switch typeNode.Type() {
		case "class_declaration":
			classType := parseClassDeclaration(typeNode, src, importMap)
			if classType != nil {
				if !i.config.IncludeUnexported && !classType.IsExported {
					continue
				}
				aFile.Types = append(aFile.Types, classType)
			}
		case "interface_declaration":
			interfaceType := parseInterfaceDeclaration(typeNode, src, importMap)
			if interfaceType != nil {
				if !i.config.IncludeUnexported && !interfaceType.IsExported {
					continue
				}
				aFile.Types = append(aFile.Types, interfaceType)
			}
		case "enum_declaration":
			enumType := parseEnumDeclaration(typeNode, src)
			if enumType != nil {
				if !i.config.IncludeUnexported && !enumType.IsExported {
					continue
				}
				aFile.Types = append(aFile.Types, enumType)

				// Add enum constants
				aFile.Constants = append(aFile.Constants, extractEnumConstants(typeNode, src, enumType.Name)...)
			}
		case "annotation_type_declaration":
			annotationType := parseAnnotationTypeDeclaration(typeNode, src)
			if annotationType != nil {
				if !i.config.IncludeUnexported && !annotationType.IsExported {
					continue
				}
				aFile.Types = append(aFile.Types, annotationType)
			}
		}
	}

	// Extract constants and variables
	aFile.Constants = append(aFile.Constants, extractConstantsFromTypes(aFile.Types)...)
	aFile.Variables = append(aFile.Variables, extractVariablesFromTypes(aFile.Types)...)

	return aFile, nil
}

// extractEnumConstants extracts enum constant values from an enum declaration
func extractEnumConstants(node *sitter.Node, source []byte, enumName string) []*graph.Constant {
	if node.Type() != "enum_declaration" {
		return nil
	}

	var constants []*graph.Constant

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		for i := uint32(0); i < bodyNode.NamedChildCount(); i++ {
			child := bodyNode.NamedChild(int(i))
			if child.Type() == "enum_constant" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					constantName := nameNode.Content(source)
					constants = append(constants, &graph.Constant{
						Name:  constantName,
						Value: enumName + "." + constantName,
					})
				}
			}
		}
	}

	return constants
}

// extractConstantsFromTypes extracts constants from class fields with final modifier
func extractConstantsFromTypes(types []*graph.Type) []*graph.Constant {
	var constants []*graph.Constant

	// In Java, constants are typically final static fields
	for _, t := range types {
		for _, field := range t.Fields {
			// Check if the field comment indicates it's final static
			// This is a simplistic approach - a proper parser would check the actual modifiers
			if field.Comment != "" && (strings.Contains(strings.ToLower(field.Comment), "final") &&
				strings.Contains(strings.ToLower(field.Comment), "static")) {
				constants = append(constants, &graph.Constant{
					Name:    field.Name,
					Comment: field.Comment,
					Value:   t.Name + "." + field.Name,
				})
			}
		}
	}

	return constants
}

// extractVariablesFromTypes extracts variables (non-constant fields) from classes
func extractVariablesFromTypes(types []*graph.Type) []*graph.Variable {
	var variables []*graph.Variable

	for _, t := range types {
		for _, field := range t.Fields {
			// Skip constants (already extracted)
			if field.Comment != "" && (strings.Contains(strings.ToLower(field.Comment), "final") &&
				strings.Contains(strings.ToLower(field.Comment), "static")) {
				continue
			}

			// Extract as variable
			variables = append(variables, &graph.Variable{
				Name:     field.Name,
				Comment:  field.Comment,
				Type:     field.Type,
				Location: field.Location,
			})
		}
	}

	return variables
}
