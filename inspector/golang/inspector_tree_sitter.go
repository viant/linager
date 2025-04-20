package golang

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/viant/linager/inspector/graph"
)

// TreeSitterInspector provides functionality to inspect Go code using tree-sitter
type TreeSitterInspector struct {
	config *graph.Config
	src    []byte // Store source for method body extraction
}

// NewTreeSitterInspector creates a new TreeSitterInspector with the provided configuration
func NewTreeSitterInspector(config *graph.Config) *TreeSitterInspector {
	return &TreeSitterInspector{
		config: config,
	}
}

// InspectSource parses Go source code from a byte slice and extracts types
func (i *TreeSitterInspector) InspectSource(src []byte) (*graph.File, error) {
	filename := defaultFilename
	i.src = src // Store source for method body extraction

	// Create a new tree-sitter parser
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	// Parse the source code
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// Process the tree
	return i.processFile(tree.RootNode(), src, filename)
}

// InspectFile parses a Go source file and extracts types
func (i *TreeSitterInspector) InspectFile(filename string) (*graph.File, error) {
	// Read file content
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	i.src = src // Store source for method body extraction

	// Create a new tree-sitter parser
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	// Parse the source code
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	// Process the tree
	return i.processFile(tree.RootNode(), src, filename)
}

// InspectPackage parses all Go files in a package and extracts types
func (i *TreeSitterInspector) InspectPackage(packagePath string) (*graph.Package, error) {
	pkg := &graph.Package{
		Name:       "",
		ImportPath: packagePath,
		FileSet:    []*graph.File{},
	}

	// Walk through the package directory
	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files if configured to do so
		if i.config.SkipTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Inspect the file
		file, err := i.InspectFile(path)
		if err != nil {
			return fmt.Errorf("failed to inspect file %s: %w", path, err)
		}

		// Add the file to the package
		pkg.FileSet = append(pkg.FileSet, file)
		pkg.Name = file.Package // Use the package name from the first file

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk package directory: %w", err)
	}

	if len(pkg.FileSet) == 0 {
		return nil, fmt.Errorf("no Go files found in package: %s", packagePath)
	}

	return pkg, nil
}

// processFile extracts package, types, constants, and variables from a Go file
func (i *TreeSitterInspector) processFile(rootNode *sitter.Node, src []byte, filename string) (*graph.File, error) {
	aFile := &graph.File{
		Path: filename,
		Name: filepath.Base(filename),
	}

	// Find package declaration
	var packageNode *sitter.Node
	var importNodes []*sitter.Node
	var typeNodes []*sitter.Node
	var funcNodes []*sitter.Node
	var constNodes []*sitter.Node
	var varNodes []*sitter.Node

	// Query for package declaration
	packageQuery := sitter.NewQuery([]byte("(package_clause (package_identifier) @package)"), golang.GetLanguage())
	packageCursor := sitter.NewQueryCursor()
	packageCursor.Exec(packageQuery, rootNode)

	for {
		match, ok := packageCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			if capture.Node.Type() == "package_identifier" {
				packageNode = capture.Node
				aFile.Package = packageNode.Content(src)
				break
			}
		}
	}

	// Query for import declarations
	importQuery := sitter.NewQuery([]byte("(import_declaration) @import"), golang.GetLanguage())
	importCursor := sitter.NewQueryCursor()
	importCursor.Exec(importQuery, rootNode)

	for {
		match, ok := importCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			importNodes = append(importNodes, capture.Node)
		}
	}

	// Process imports
	aFile.Imports = parseImports(importNodes, src)

	// Query for type declarations
	typeQuery := sitter.NewQuery([]byte("(type_declaration) @type"), golang.GetLanguage())
	typeCursor := sitter.NewQueryCursor()
	typeCursor.Exec(typeQuery, rootNode)

	for {
		match, ok := typeCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			typeNodes = append(typeNodes, capture.Node)
		}
	}

	// Process types
	for _, typeNode := range typeNodes {
		types := parseTypeDeclaration(typeNode, src)
		for _, t := range types {
			if !i.config.IncludeUnexported && !t.IsExported {
				continue
			}
			aFile.Types = append(aFile.Types, t)
		}
	}

	// Query for function declarations
	funcQuery := sitter.NewQuery([]byte("(function_declaration) @func"), golang.GetLanguage())
	funcCursor := sitter.NewQueryCursor()
	funcCursor.Exec(funcQuery, rootNode)

	for {
		match, ok := funcCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			funcNodes = append(funcNodes, capture.Node)
		}
	}

	// Query for method declarations
	methodQuery := sitter.NewQuery([]byte("(method_declaration) @method"), golang.GetLanguage())
	methodCursor := sitter.NewQueryCursor()
	methodCursor.Exec(methodQuery, rootNode)

	// Process functions and methods
	for _, funcNode := range funcNodes {
		if funcNode.Type() == "function_declaration" {
			function := parseFunctionDeclaration(funcNode, src)
			if !i.config.IncludeUnexported && !function.IsExported {
				continue
			}
			aFile.Functions = append(aFile.Functions, function)
		} else if funcNode.Type() == "method_declaration" {
			method := parseMethodDeclaration(funcNode, src)
			if !i.config.IncludeUnexported && !method.IsExported {
				continue
			}

			// Find the receiver type and add the method to it
			receiverType := method.Receiver
			if receiverType != "" {
				// Remove pointer symbol if present
				if strings.HasPrefix(receiverType, "*") {
					receiverType = receiverType[1:]
				}

				// Find the type in the file's types
				var targetType *graph.Type
				for _, t := range aFile.Types {
					if t.Name == receiverType {
						targetType = t
						break
					}
				}

				// If type not found, create a new one
				if targetType == nil {
					targetType = &graph.Type{
						Name:       receiverType,
						IsExported: isExported(receiverType),
						Methods:    []*graph.Function{},
						Kind:       reflect.Struct, // Assume struct for now
					}
					aFile.Types = append(aFile.Types, targetType)
				}

				// Add the method to the type
				targetType.Methods = append(targetType.Methods, method)
			}
		}
	}

	// Query for constant declarations
	constQuery := sitter.NewQuery([]byte("(const_declaration) @const"), golang.GetLanguage())
	constCursor := sitter.NewQueryCursor()
	constCursor.Exec(constQuery, rootNode)

	for {
		match, ok := constCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			constNodes = append(constNodes, capture.Node)
		}
	}

	// Process constants
	for _, constNode := range constNodes {
		constants := parseConstDeclaration(constNode, src)
		for _, c := range constants {
			if !i.config.IncludeUnexported && !isExported(c.Name) {
				continue
			}
			aFile.Constants = append(aFile.Constants, c)
		}
	}

	// Query for variable declarations
	varQuery := sitter.NewQuery([]byte("(var_declaration) @var"), golang.GetLanguage())
	varCursor := sitter.NewQueryCursor()
	varCursor.Exec(varQuery, rootNode)

	for {
		match, ok := varCursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			varNodes = append(varNodes, capture.Node)
		}
	}

	// Process variables
	for _, varNode := range varNodes {
		variables := parseVarDeclaration(varNode, src)
		for _, v := range variables {
			if !i.config.IncludeUnexported && !isExported(v.Name) {
				continue
			}
			aFile.Variables = append(aFile.Variables, v)
		}
	}

	return aFile, nil
}

// Helper functions

// parseImports extracts import declarations from import nodes
func parseImports(importNodes []*sitter.Node, src []byte) []graph.Import {
	var imports []graph.Import

	for _, importNode := range importNodes {
		// Handle both single imports and import blocks
		if importNode.NamedChildCount() == 0 {
			continue
		}

		// For import blocks with multiple import specs
		for i := uint32(0); i < importNode.NamedChildCount(); i++ {
			child := importNode.NamedChild(int(i))

			if child.Type() == "import_spec" {
				var importName string
				var importPath string

				// Check for named import
				if child.NamedChildCount() > 1 {
					// First child is the import name
					nameNode := child.NamedChild(0)
					if nameNode.Type() == "package_identifier" {
						importName = nameNode.Content(src)
					}

					// Second child is the import path
					pathNode := child.NamedChild(1)
					if pathNode.Type() == "interpreted_string_literal" {
						// Remove quotes from string literal
						importPath = strings.Trim(pathNode.Content(src), "\"")
					}
				} else if child.NamedChildCount() == 1 {
					// Only import path, no name
					pathNode := child.NamedChild(0)
					if pathNode.Type() == "interpreted_string_literal" {
						// Remove quotes from string literal
						importPath = strings.Trim(pathNode.Content(src), "\"")

						// Extract the last part of the path as the default name
						parts := strings.Split(importPath, "/")
						importName = parts[len(parts)-1]
					}
				}

				if importPath != "" {
					imports = append(imports, graph.Import{
						Name: importName,
						Path: importPath,
					})
				}
			}
		}
	}

	return imports
}

// parseTypeDeclaration extracts type information from a type declaration node
func parseTypeDeclaration(typeNode *sitter.Node, src []byte) []*graph.Type {
	var types []*graph.Type

	// Process each type spec in the type declaration
	for i := uint32(0); i < typeNode.NamedChildCount(); i++ {
		child := typeNode.NamedChild(int(i))

		if child.Type() == "type_spec" {
			// Get the type name
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			typeName := nameNode.Content(src)
			isExported := isExported(typeName)

			// Create a new type
			t := &graph.Type{
				Name:       typeName,
				IsExported: isExported,
				Location: &graph.Location{
					Start: int(child.StartByte()),
					End:   int(child.EndByte()),
					Raw:   child.Content(src),
				},
			}

			// Get the type value
			typeValue := child.ChildByFieldName("type")
			if typeValue == nil {
				continue
			}

			// Process the type based on its kind
			switch typeValue.Type() {
			case "struct_type":
				t.Kind = reflect.Struct
				t.Fields = parseStructFields(typeValue, src)
			case "interface_type":
				t.Kind = reflect.Interface
				// TODO: Parse interface methods if needed
			case "array_type":
				t.Kind = reflect.Array
				elemType := typeValue.ChildByFieldName("element")
				if elemType != nil {
					t.ComponentType = elemType.Content(src)
				}
			case "slice_type":
				t.Kind = reflect.Slice
				elemType := typeValue.ChildByFieldName("element")
				if elemType != nil {
					t.ComponentType = elemType.Content(src)
				}
			case "map_type":
				t.Kind = reflect.Map
				keyType := typeValue.ChildByFieldName("key")
				if keyType != nil {
					t.KeyType = keyType.Content(src)
				}
				valueType := typeValue.ChildByFieldName("value")
				if valueType != nil {
					t.ComponentType = valueType.Content(src)
				}
			case "pointer_type":
				t.IsPointer = true
				baseType := typeValue.ChildByFieldName("type")
				if baseType != nil {
					t.ComponentType = baseType.Content(src)
				}
			case "type_identifier":
				// Basic type or reference to another type
				t.Kind = kindFromBasicType(typeValue.Content(src))
			}

			types = append(types, t)
		}
	}

	return types
}

// parseStructFields extracts field information from a struct type node
func parseStructFields(structNode *sitter.Node, src []byte) []*graph.Field {
	var fields []*graph.Field

	// Find the field_declaration_list node
	var fieldListNode *sitter.Node
	for i := uint32(0); i < structNode.NamedChildCount(); i++ {
		child := structNode.NamedChild(int(i))
		if child.Type() == "field_declaration_list" {
			fieldListNode = child
			break
		}
	}

	if fieldListNode == nil {
		return fields
	}

	// Process each field declaration
	for i := uint32(0); i < fieldListNode.NamedChildCount(); i++ {
		fieldNode := fieldListNode.NamedChild(int(i))

		if fieldNode.Type() == "field_declaration" {
			// Get the field name(s)
			var fieldNames []string
			var fieldTypeStr string
			var fieldTag string

			// Find name and type nodes
			nameNode := fieldNode.ChildByFieldName("name")
			typeNode := fieldNode.ChildByFieldName("type")
			tagNode := fieldNode.ChildByFieldName("tag")

			if nameNode != nil {
				fieldNames = append(fieldNames, nameNode.Content(src))
			}

			if typeNode != nil {
				fieldTypeStr = typeNode.Content(src)
			}

			if tagNode != nil {
				// Remove backticks from tag
				fieldTag = strings.Trim(tagNode.Content(src), "`")
			}

			// Create a type for the field
			fieldType := &graph.Type{
				Name: fieldTypeStr,
				Kind: kindFromBasicType(fieldTypeStr),
			}

			// Create a field for each name
			for _, name := range fieldNames {
				field := &graph.Field{
					Name:       name,
					Type:       fieldType,
					IsExported: isExported(name),
					Tag:        reflect.StructTag(fieldTag),
					Location: &graph.Location{
						Start: int(fieldNode.StartByte()),
						End:   int(fieldNode.EndByte()),
						Raw:   fieldNode.Content(src),
					},
				}

				fields = append(fields, field)
			}
		}
	}

	return fields
}

// parseFunctionDeclaration extracts function information from a function declaration node
func parseFunctionDeclaration(funcNode *sitter.Node, src []byte) *graph.Function {
	// Get the function name
	nameNode := funcNode.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	funcName := nameNode.Content(src)

	// Create a new function
	function := &graph.Function{
		Name:       funcName,
		IsExported: isExported(funcName),
		Location: &graph.Location{
			Start: int(funcNode.StartByte()),
			End:   int(funcNode.EndByte()),
			Raw:   funcNode.Content(src),
		},
	}

	// Get the parameter list
	paramListNode := funcNode.ChildByFieldName("parameters")
	if paramListNode != nil {
		function.Parameters = parseParameters(paramListNode, src)
	}

	// Get the return type
	resultNode := funcNode.ChildByFieldName("result")
	if resultNode != nil {
		// Create a comment node with the return type information
		function.Comment = &graph.LocationNode{
			Text: "Returns: " + resultNode.Content(src),
		}
	}

	return function
}

// parseMethodDeclaration extracts method information from a method declaration node
func parseMethodDeclaration(methodNode *sitter.Node, src []byte) *graph.Function {
	// Get the method name
	nameNode := methodNode.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	methodName := nameNode.Content(src)

	// Create a new function (method)
	method := &graph.Function{
		Name:       methodName,
		IsExported: isExported(methodName),
		Location: &graph.Location{
			Start: int(methodNode.StartByte()),
			End:   int(methodNode.EndByte()),
			Raw:   methodNode.Content(src),
		},
	}

	// Get the receiver
	receiverNode := methodNode.ChildByFieldName("receiver")
	if receiverNode != nil {
		// Extract receiver type
		receiverTypeNode := receiverNode.ChildByFieldName("type")
		if receiverTypeNode != nil {
			// Store receiver information in the Receiver field
			method.Receiver = receiverTypeNode.Content(src)
		}
	}

	// Get the parameter list
	paramListNode := methodNode.ChildByFieldName("parameters")
	if paramListNode != nil {
		method.Parameters = parseParameters(paramListNode, src)
	}

	// Get the return type
	resultNode := methodNode.ChildByFieldName("result")
	if resultNode != nil {
		// Create a comment node with the return type information
		method.Comment = &graph.LocationNode{
			Text: "Returns: " + resultNode.Content(src),
		}
	}

	return method
}

// parseParameters extracts parameter information from a parameter list node
func parseParameters(paramListNode *sitter.Node, src []byte) []*graph.Parameter {
	var parameters []*graph.Parameter

	// Process each parameter
	for i := uint32(0); i < paramListNode.NamedChildCount(); i++ {
		paramNode := paramListNode.NamedChild(int(i))

		if paramNode.Type() == "parameter_declaration" {
			// Get parameter name and type
			nameNode := paramNode.ChildByFieldName("name")
			typeNode := paramNode.ChildByFieldName("type")

			if nameNode != nil && typeNode != nil {
				// Create a type for the parameter
				typeStr := typeNode.Content(src)
				paramType := &graph.Type{
					Name: typeStr,
					Kind: kindFromBasicType(typeStr),
				}

				param := &graph.Parameter{
					Name: nameNode.Content(src),
					Type: paramType,
				}

				parameters = append(parameters, param)
			}
		}
	}

	return parameters
}

// parseConstDeclaration extracts constant information from a const declaration node
func parseConstDeclaration(constNode *sitter.Node, src []byte) []*graph.Constant {
	var constants []*graph.Constant

	// Process each const spec
	for i := uint32(0); i < constNode.NamedChildCount(); i++ {
		child := constNode.NamedChild(int(i))

		if child.Type() == "const_spec" {
			// Get the constant name
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			constName := nameNode.Content(src)

			// Get the constant value if present
			valueNode := child.ChildByFieldName("value")
			var constValue string
			if valueNode != nil {
				constValue = valueNode.Content(src)
			}

			// Create a new constant
			constant := &graph.Constant{
				Name:  constName,
				Value: constValue,
				Location: &graph.Location{
					Start: int(child.StartByte()),
					End:   int(child.EndByte()),
					Raw:   child.Content(src),
				},
			}

			constants = append(constants, constant)
		}
	}

	return constants
}

// parseVarDeclaration extracts variable information from a var declaration node
func parseVarDeclaration(varNode *sitter.Node, src []byte) []*graph.Variable {
	var variables []*graph.Variable

	// Process each var spec
	for i := uint32(0); i < varNode.NamedChildCount(); i++ {
		child := varNode.NamedChild(int(i))

		if child.Type() == "var_spec" {
			// Get the variable name
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}

			varName := nameNode.Content(src)

			// Get the variable type if present
			typeNode := child.ChildByFieldName("type")
			var typeStr string
			if typeNode != nil {
				typeStr = typeNode.Content(src)
			}

			// Get the variable value if present
			valueNode := child.ChildByFieldName("value")
			var varValue string
			if valueNode != nil {
				varValue = valueNode.Content(src)
			}

			// Create a type for the variable
			varType := &graph.Type{
				Name: typeStr,
				Kind: kindFromBasicType(typeStr),
			}

			// Create a new variable
			variable := &graph.Variable{
				Name:  varName,
				Type:  varType,
				Value: varValue,
				Location: &graph.Location{
					Start: int(child.StartByte()),
					End:   int(child.EndByte()),
					Raw:   child.Content(src),
				},
			}

			variables = append(variables, variable)
		}
	}

	return variables
}

// isExported returns true if the identifier is exported (starts with an uppercase letter)
func isExported(name string) bool {
	if name == "" {
		return false
	}
	return strings.ToUpper(name[:1]) == name[:1]
}

// kindFromBasicType returns the reflect.Kind for a basic Go type
func kindFromBasicType(typeName string) reflect.Kind {
	switch typeName {
	case "bool":
		return reflect.Bool
	case "int", "int8", "int16", "int32", "int64":
		return reflect.Int
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte":
		return reflect.Uint
	case "float32", "float64":
		return reflect.Float64
	case "complex64", "complex128":
		return reflect.Complex128
	case "string":
		return reflect.String
	case "error":
		return reflect.Interface
	default:
		return reflect.Struct // Default to struct for custom types
	}
}
