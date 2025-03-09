package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/viant/linager/inspector/info"
)

// Inspector provides functionality to inspect Java code and extract type information
type Inspector struct {
	config    *info.Config
	importMap map[string]string
	source    []byte
}

// NewInspector creates a new Java Inspector with the provided configuration
func NewInspector(config *info.Config) *Inspector {
	if config == nil {
		config = &info.Config{
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
func (i *Inspector) InspectSource(src []byte) (*info.File, error) {
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
func (i *Inspector) InspectFile(filename string) (*info.File, error) {
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
			aFile.Imports = append(aFile.Imports, info.Import{
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
func (i *Inspector) InspectPackage(packagePath string) (*info.Package, error) {
	// Get the absolute path of the package
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create a new Package to store all discovered types
	pkg := &info.Package{
		FileSet: []*info.File{},
	}

	// Walk the directory to find Java files
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip subdirectories if not recursive
			if path != absPath && !i.config.RecursivePackages {
				return filepath.SkipDir
			}
			return nil
		}

		// Process only .java files
		if filepath.Ext(path) == ".java" {
			// Skip test files unless configured to include them
			if i.config.SkipTests && (filepath.Base(path) == "Test.java" ||
				filepath.Base(path) == "Tests.java" ||
				filepath.Base(path) == "IT.java" ||
				filepath.Base(path) == "ITCase.java") {
				return nil
			}

			file, err := i.InspectFile(path)
			if err != nil {
				return fmt.Errorf("error processing %s: %w", path, err)
			}

			// Add file to the package
			if pkg.Name == "" && file.Package != "" {
				pkg.Name = file.Package
				pkg.ImportPath = file.ImportPath
			}

			// Add file to package
			pkg.AddFile(file)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking package directory: %w", err)
	}

	if len(pkg.FileSet) == 0 {
		return nil, fmt.Errorf("no Java files found in package: %s", packagePath)
	}

	return pkg, nil
}

// processJavaFile extracts package, types, constants, and variables from a Java file
func (i *Inspector) processJavaFile(rootNode *sitter.Node, src []byte, filename string) (*info.File, error) {
	aFile := &info.File{Name: filename}

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
		aFile.Name = parsePackageDeclaration(packageNode, src)
		aFile.ImportPath = aFile.Name // In Java, the package name is the import path
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
func extractEnumConstants(node *sitter.Node, source []byte, enumName string) []*info.Constant {
	if node.Type() != "enum_declaration" {
		return nil
	}

	var constants []*info.Constant

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		for i := uint32(0); i < bodyNode.NamedChildCount(); i++ {
			child := bodyNode.NamedChild(int(i))
			if child.Type() == "enum_constant" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					constantName := nameNode.Content(source)
					constants = append(constants, &info.Constant{
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
func extractConstantsFromTypes(types []*info.Type) []*info.Constant {
	var constants []*info.Constant

	// In Java, constants are typically final static fields
	for _, t := range types {
		for _, field := range t.Fields {
			// Check if the field comment indicates it's final static
			// This is a simplistic approach - a proper parser would check the actual modifiers
			if field.Comment != "" && (strings.Contains(strings.ToLower(field.Comment), "final") &&
				strings.Contains(strings.ToLower(field.Comment), "static")) {
				constants = append(constants, &info.Constant{
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
func extractVariablesFromTypes(types []*info.Type) []*info.Variable {
	var variables []*info.Variable

	for _, t := range types {
		for _, field := range t.Fields {
			// Skip constants (already extracted)
			if field.Comment != "" && (strings.Contains(strings.ToLower(field.Comment), "final") &&
				strings.Contains(strings.ToLower(field.Comment), "static")) {
				continue
			}

			// Extract as variable
			variables = append(variables, &info.Variable{
				Name:     field.Name,
				Comment:  field.Comment,
				Type:     field.Type,
				Location: field.Location,
			})
		}
	}

	return variables
}
