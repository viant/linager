package jsx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/viant/linager/inspector/graph"
)

// Inspector provides functionality to inspect JSX code and extract type information
type Inspector struct {
	config    *graph.Config
	importMap map[string]string
	source    []byte
}

// NewInspector creates a new JSX Inspector with the provided configuration
func NewInspector(config *graph.Config) *Inspector {
	if config == nil {
		config = &graph.Config{
			IncludeUnexported: true,
			SkipTests:         false,
			RecursivePackages: false,
		}
	}
	return &Inspector{
		config:    config,
		importMap: make(map[string]string),
	}
}

// InspectSource parses JSX source code from a byte slice and extracts types
func (i *Inspector) InspectSource(src []byte) (*graph.File, error) {
	i.source = src

	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	rootNode := tree.RootNode()

	return i.processJSXFile(rootNode, src, "source.jsx")
}

// InspectFile parses a JSX source file and extracts types
func (i *Inspector) InspectFile(filename string) (*graph.File, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	i.source = src

	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	rootNode := tree.RootNode()

	return i.processJSXFile(rootNode, src, filename)
}

// InspectPackage inspects a JSX package directory and extracts all types
func (i *Inspector) InspectPackage(packagePath string) (*graph.Package, error) {
	// Get the absolute path of the package
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create a new Package to store all discovered types
	pkg := &graph.Package{
		FileSet:    []*graph.File{},
		Name:       filepath.Base(absPath),
		ImportPath: absPath,
	}

	// Walk through the package directory
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only .jsx and .tsx files
		ext := filepath.Ext(path)
		if ext != ".jsx" && ext != ".tsx" {
			return nil
		}

		// Skip test files unless configured to include them
		if i.config.SkipTests && strings.Contains(filepath.Base(path), ".test.") {
			return nil
		}

		file, err := i.InspectFile(path)
		if err != nil {
			return fmt.Errorf("error processing %s: %w", path, err)
		}

		// Add file to package
		pkg.AddFile(file)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking package directory: %w", err)
	}

	if len(pkg.FileSet) == 0 {
		return nil, fmt.Errorf("no JSX files found in package: %s", packagePath)
	}

	return pkg, nil
}

// processJSXFile extracts components, imports, and other elements from a JSX file
func (i *Inspector) processJSXFile(rootNode *sitter.Node, src []byte, filename string) (*graph.File, error) {
	aFile := &graph.File{
		Path:       filename,
		ImportPath: filepath.Dir(filename),
		Package:    filepath.Base(filepath.Dir(filename)),
		Types:      []*graph.Type{},
		Constants:  []*graph.Constant{},
		Variables:  []*graph.Variable{},
		Functions:  []*graph.Function{},
		Imports:    []graph.Import{},
	}

	// Process imports
	importNodes := findImportNodes(rootNode)
	for _, importNode := range importNodes {
		imports := parseImportDeclarations(importNode, src)
		for name, path := range imports {
			aFile.Imports = append(aFile.Imports, graph.Import{
				Name: name,
				Path: path,
			})
		}
	}

	// Process components (function and class declarations)
	componentTypes, err := i.processJSXComponents(rootNode, src)
	if err != nil {
		return nil, err
	}
	aFile.Types = append(aFile.Types, componentTypes...)

	// Process variables (including state variables)
	variables, err := i.processJSXVariables(rootNode, src)
	if err != nil {
		return nil, err
	}
	aFile.Variables = append(aFile.Variables, variables...)

	// Process functions (including hooks and event handlers)
	functions, err := i.processJSXFunctions(rootNode, src)
	if err != nil {
		return nil, err
	}
	aFile.Functions = append(aFile.Functions, functions...)

	return aFile, nil
}

// findImportNodes finds all import declaration nodes in the AST
func findImportNodes(rootNode *sitter.Node) []*sitter.Node {
	var importNodes []*sitter.Node

	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))
		if childNode.Type() == "import_statement" {
			importNodes = append(importNodes, childNode)
		}
	}

	return importNodes
}

// parseImportDeclarations extracts import information from an import node
func parseImportDeclarations(importNode *sitter.Node, src []byte) map[string]string {
	imports := make(map[string]string)

	// Extract the import path (string literal)
	var importPath string
	for j := uint32(0); j < importNode.NamedChildCount(); j++ {
		child := importNode.NamedChild(int(j))
		if child.Type() == "string" {
			// Remove quotes from the string
			pathStr := child.Content(src)
			importPath = strings.Trim(pathStr, "'\"")
			break
		}
	}

	if importPath == "" {
		return imports
	}

	// Extract the import name(s)
	var defaultImport string
	for j := uint32(0); j < importNode.NamedChildCount(); j++ {
		child := importNode.NamedChild(int(j))
		if child.Type() == "identifier" {
			defaultImport = child.Content(src)
			imports[defaultImport] = importPath
			break
		} else if child.Type() == "import_clause" {
			// Handle named imports
			for k := uint32(0); k < child.NamedChildCount(); k++ {
				namedImport := child.NamedChild(int(k))
				if namedImport.Type() == "identifier" {
					imports[namedImport.Content(src)] = importPath
				} else if namedImport.Type() == "named_imports" {
					// Handle { Component, useState } style imports
					for l := uint32(0); l < namedImport.NamedChildCount(); l++ {
						specifier := namedImport.NamedChild(int(l))
						if specifier.Type() == "import_specifier" {
							for m := uint32(0); m < specifier.NamedChildCount(); m++ {
								name := specifier.NamedChild(int(m))
								if name.Type() == "identifier" {
									imports[name.Content(src)] = importPath
								}
							}
						}
					}
				}
			}
		}
	}

	return imports
}

// processJSXComponents extracts component information from JSX code
func (i *Inspector) processJSXComponents(rootNode *sitter.Node, src []byte) ([]*graph.Type, error) {
	var components []*graph.Type

	// Find function and class declarations
	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))

		// Function components
		if childNode.Type() == "function_declaration" {
			component := processFunctionComponent(childNode, src)
			if component != nil {
				components = append(components, component)
			}
		} else if childNode.Type() == "class_declaration" {
			component := processClassComponent(childNode, src)
			if component != nil {
				components = append(components, component)
			}
		} else if childNode.Type() == "lexical_declaration" {
			// Arrow function components (const Component = () => {...})
			component := processArrowFunctionComponent(childNode, src)
			if component != nil {
				components = append(components, component)
			}
		}
	}

	return components, nil
}

// processFunctionComponent extracts information from a function component
func processFunctionComponent(node *sitter.Node, src []byte) *graph.Type {
	// Get the function name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(src)

	// Create a new Type for the component
	component := &graph.Type{
		Name:       name,
		Kind:       reflect.Struct, // Use Struct kind for components
		IsExported: true,           // Assume exported for now
		Methods:    []*graph.Function{},
		Fields:     []*graph.Field{},
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
			Raw:   string(src[node.StartByte():node.EndByte()]),
		},
	}

	// Extract props from parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		for k := uint32(0); k < paramsNode.NamedChildCount(); k++ {
			paramNode := paramsNode.NamedChild(int(k))
			if paramNode.Type() == "identifier" {
				propName := paramNode.Content(src)
				component.Fields = append(component.Fields, &graph.Field{
					Name:    propName,
					Type:    &graph.Type{Name: "any"}, // Default type
					Comment: "prop",
				})
			} else if paramNode.Type() == "object_pattern" {
				// Destructured props like { name, age }
				for l := uint32(0); l < paramNode.NamedChildCount(); l++ {
					propNode := paramNode.NamedChild(int(l))
					if propNode.Type() == "shorthand_property_identifier" || propNode.Type() == "identifier" {
						propName := propNode.Content(src)
						component.Fields = append(component.Fields, &graph.Field{
							Name:    propName,
							Type:    &graph.Type{Name: "any"}, // Default type
							Comment: "prop",
						})
					}
				}
			}
		}
	}

	return component
}

// processClassComponent extracts information from a class component
func processClassComponent(node *sitter.Node, src []byte) *graph.Type {
	// Get the class name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(src)

	// Create a new Type for the component
	component := &graph.Type{
		Name:       name,
		Kind:       reflect.Struct,
		IsExported: true, // Assume exported for now
		Methods:    []*graph.Function{},
		Fields:     []*graph.Field{},
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
			Raw:   string(src[node.StartByte():node.EndByte()]),
		},
	}

	// Find the class body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Extract methods and fields
		for k := uint32(0); k < bodyNode.NamedChildCount(); k++ {
			memberNode := bodyNode.NamedChild(int(k))

			if memberNode.Type() == "method_definition" {
				methodName := ""
				methodNameNode := memberNode.ChildByFieldName("name")
				if methodNameNode != nil {
					methodName = methodNameNode.Content(src)
				}

				if methodName != "" {
					method := &graph.Function{
						Name: methodName,
						Location: &graph.Location{
							Start: int(memberNode.StartByte()),
							End:   int(memberNode.EndByte()),
							Raw:   string(src[memberNode.StartByte():memberNode.EndByte()]),
						},
					}
					component.Methods = append(component.Methods, method)
				}
			} else if memberNode.Type() == "public_field_definition" {
				fieldName := ""
				fieldNameNode := memberNode.ChildByFieldName("name")
				if fieldNameNode != nil {
					fieldName = fieldNameNode.Content(src)
				}

				if fieldName != "" {
					field := &graph.Field{
						Name: fieldName,
						Type: &graph.Type{Name: "any"}, // Default type
					}
					component.Fields = append(component.Fields, field)
				}
			}
		}
	}

	return component
}

// processArrowFunctionComponent extracts information from an arrow function component
func processArrowFunctionComponent(node *sitter.Node, src []byte) *graph.Type {
	// Check if this is a variable declaration with an arrow function
	if node.Type() != "lexical_declaration" {
		return nil
	}

	// Find the variable declarator
	var declaratorNode *sitter.Node
	for j := uint32(0); j < node.NamedChildCount(); j++ {
		child := node.NamedChild(int(j))
		if child.Type() == "variable_declarator" {
			declaratorNode = child
			break
		}
	}

	if declaratorNode == nil {
		return nil
	}

	// Get the variable name
	nameNode := declaratorNode.ChildByFieldName("name")
	if nameNode == nil || nameNode.Type() != "identifier" {
		return nil
	}

	name := nameNode.Content(src)

	// Check if the value is an arrow function
	valueNode := declaratorNode.ChildByFieldName("value")
	if valueNode == nil || valueNode.Type() != "arrow_function" {
		return nil
	}

	// Create a new Type for the component
	component := &graph.Type{
		Name:       name,
		Kind:       reflect.Struct,
		IsExported: true, // Assume exported for now
		Methods:    []*graph.Function{},
		Fields:     []*graph.Field{},
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
			Raw:   string(src[node.StartByte():node.EndByte()]),
		},
	}

	// Extract props from parameters
	paramsNode := valueNode.ChildByFieldName("parameters")
	if paramsNode != nil {
		for k := uint32(0); k < paramsNode.NamedChildCount(); k++ {
			paramNode := paramsNode.NamedChild(int(k))
			if paramNode.Type() == "identifier" {
				propName := paramNode.Content(src)
				component.Fields = append(component.Fields, &graph.Field{
					Name:    propName,
					Type:    &graph.Type{Name: "any"}, // Default type
					Comment: "prop",
				})
			} else if paramNode.Type() == "object_pattern" {
				// Destructured props like { name, age }
				for l := uint32(0); l < paramNode.NamedChildCount(); l++ {
					propNode := paramNode.NamedChild(int(l))
					if propNode.Type() == "shorthand_property_identifier" || propNode.Type() == "identifier" {
						propName := propNode.Content(src)
						component.Fields = append(component.Fields, &graph.Field{
							Name:    propName,
							Type:    &graph.Type{Name: "any"}, // Default type
							Comment: "prop",
						})
					}
				}
			}
		}
	}

	return component
}

// processJSXVariables extracts variable information from JSX code
func (i *Inspector) processJSXVariables(rootNode *sitter.Node, src []byte) ([]*graph.Variable, error) {
	var variables []*graph.Variable

	// Find variable declarations
	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))

		if childNode.Type() == "lexical_declaration" || childNode.Type() == "variable_declaration" {
			// Process each declarator
			for k := uint32(0); k < childNode.NamedChildCount(); k++ {
				declaratorNode := childNode.NamedChild(int(k))
				if declaratorNode.Type() == "variable_declarator" {
					nameNode := declaratorNode.ChildByFieldName("name")
					if nameNode != nil && nameNode.Type() == "identifier" {
						name := nameNode.Content(src)

						// Create a variable
						variable := &graph.Variable{
							Name: name,
							Location: &graph.Location{
								Start: int(declaratorNode.StartByte()),
								End:   int(declaratorNode.EndByte()),
								Raw:   string(src[declaratorNode.StartByte():declaratorNode.EndByte()]),
							},
						}

						// Check if it's a state variable (useState hook)
						valueNode := declaratorNode.ChildByFieldName("value")
						if valueNode != nil && valueNode.Type() == "call_expression" {
							functionNode := valueNode.ChildByFieldName("function")
							if functionNode != nil && functionNode.Type() == "identifier" {
								functionName := functionNode.Content(src)
								if functionName == "useState" {
									variable.Comment = "state variable"
								}
							}
						}

						variables = append(variables, variable)
					}
				}
			}
		}
	}

	return variables, nil
}

// processJSXFunctions extracts function information from JSX code
func (i *Inspector) processJSXFunctions(rootNode *sitter.Node, src []byte) ([]*graph.Function, error) {
	var functions []*graph.Function

	// Find function declarations and expressions
	for j := uint32(0); j < rootNode.NamedChildCount(); j++ {
		childNode := rootNode.NamedChild(int(j))

		if childNode.Type() == "function_declaration" {
			nameNode := childNode.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Content(src)

				// Skip if this is a component (already processed)
				if isComponent(childNode, src) {
					continue
				}

				function := &graph.Function{
					Name: name,
					Location: &graph.Location{
						Start: int(childNode.StartByte()),
						End:   int(childNode.EndByte()),
						Raw:   string(src[childNode.StartByte():childNode.EndByte()]),
					},
				}

				functions = append(functions, function)
			}
		} else if childNode.Type() == "lexical_declaration" || childNode.Type() == "variable_declaration" {
			// Look for arrow functions
			for k := uint32(0); k < childNode.NamedChildCount(); k++ {
				declaratorNode := childNode.NamedChild(int(k))
				if declaratorNode.Type() == "variable_declarator" {
					nameNode := declaratorNode.ChildByFieldName("name")
					valueNode := declaratorNode.ChildByFieldName("value")

					if nameNode != nil && nameNode.Type() == "identifier" &&
						valueNode != nil && (valueNode.Type() == "arrow_function" || valueNode.Type() == "function") {

						name := nameNode.Content(src)

						// Skip if this is a component (already processed)
						if isArrowFunctionComponent(declaratorNode, src) {
							continue
						}

						function := &graph.Function{
							Name: name,
							Location: &graph.Location{
								Start: int(declaratorNode.StartByte()),
								End:   int(declaratorNode.EndByte()),
								Raw:   string(src[declaratorNode.StartByte():declaratorNode.EndByte()]),
							},
						}

						functions = append(functions, function)
					}
				}
			}
		}
	}

	return functions, nil
}

// isComponent checks if a function declaration is a React component
func isComponent(node *sitter.Node, src []byte) bool {
	// Check if the function returns JSX
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return false
	}

	// Look for return statements with JSX
	return containsJSX(bodyNode, src)
}

// isArrowFunctionComponent checks if an arrow function is a React component
func isArrowFunctionComponent(node *sitter.Node, src []byte) bool {
	valueNode := node.ChildByFieldName("value")
	if valueNode == nil || valueNode.Type() != "arrow_function" {
		return false
	}

	bodyNode := valueNode.ChildByFieldName("body")
	if bodyNode == nil {
		return false
	}

	// Look for JSX in the body
	return containsJSX(bodyNode, src)
}

// containsJSX checks if a node contains JSX elements
func containsJSX(node *sitter.Node, src []byte) bool {
	// This is a simplified check - in a real implementation, we would traverse
	// the AST more thoroughly to find JSX elements
	nodeStr := string(src[node.StartByte():node.EndByte()])
	return strings.Contains(nodeStr, "<") && strings.Contains(nodeStr, "/>") ||
		strings.Contains(nodeStr, "<") && strings.Contains(nodeStr, ">") && strings.Contains(nodeStr, "</")
}
