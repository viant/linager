package golang

import (
	"fmt"
	"github.com/viant/linager/inspector/info"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Inspector provides functionality to inspect Go code and extract type information
type Inspector struct {
	fset   *token.FileSet
	config *info.Config
	src    []byte // Store source for method body extraction
}

// Config holds configuration options for the Inspector

// NewInspector creates a new Inspector with the provided configuration
func NewInspector(config *info.Config) *Inspector {
	return &Inspector{
		fset:   token.NewFileSet(),
		config: config,
	}
}

const defaultFilename = "source.go"

// InspectSource parses Go source code from a byte slice and extracts types
func (i *Inspector) InspectSource(src []byte) (*info.File, error) {
	filename := defaultFilename
	i.src = src // Store source for method body extraction
	file, err := parser.ParseFile(i.fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	return i.processFile(file, filename)
}

// InspectFile parses a Go source file and extracts types
func (i *Inspector) InspectFile(filename string) (*info.File, error) {
	// Read file content
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	i.src = src // Store source for method body extraction
	file, err := parser.ParseFile(i.fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	return i.processFile(file, filename)
}

// processFile extracts type information from an AST file
func (i *Inspector) processFile(file *ast.File, filename string) (*info.File, error) {
	// Reset and rebuild import map for this file
	importMap := buildImportMap(file)

	// Get imports from the file
	imports := ParseImports(file)

	// Create the File structure with import information
	infoFile := &info.File{
		Name:    filepath.Base(filename),
		Path:    filename,
		Package: file.Name.Name,
		Imports: make([]info.Import, len(imports)),
	}

	// Convert inspector's import specs to info.Import objects
	for i, imp := range imports {
		infoFile.Imports[i] = info.Import{
			Name: imp.Name,
			Path: imp.Path,
		}
	}

	// Extract the import path from the file path
	infoFile.ImportPath = getImportPathFromFilePath(filename)

	// Add constants and variables to the file
	constants, err := i.InspectConstants(file, importMap)
	if err == nil {
		infoFile.Constants = constants
		// Set file reference in constants
		for _, c := range infoFile.Constants {
			c.File = infoFile
		}
	}

	if infoFile.Variables, err = i.InspectVariables(file, importMap); err != nil {
		return nil, err
	}

	if infoFile.Constants, err = i.InspectConstants(file, importMap); err != nil {
		return nil, err
	}
	// Collect type declarations
	typeMap := make(map[string]*ast.TypeSpec)
	var typeOrder []string
	docMap := make(map[string]*ast.CommentGroup) // Map to store doc comments for type specs
	locMap := make(map[string]*info.Location)    // Map to store locations for type specs

	// First pass: collect all type specs and their associated comments
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// Store the GenDecl's doc comment, which might apply to multiple type specs
		var declDoc *ast.CommentGroup
		if genDecl.Doc != nil {
			declDoc = genDecl.Doc
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if !i.config.IncludeUnexported && !ts.Name.IsExported() {
				continue
			}

			typeMap[ts.Name.Name] = ts
			typeOrder = append(typeOrder, ts.Name.Name)

			// Store the associated doc comment, prioritizing the TypeSpec's own doc if available
			if ts.Doc != nil {
				docMap[ts.Name.Name] = ts.Doc
			} else if declDoc != nil {
				// If TypeSpec doesn't have its own doc, use the GenDecl's doc
				docMap[ts.Name.Name] = declDoc
			}

			// Store location information
			pos := i.fset.Position(ts.Pos())
			end := i.fset.Position(ts.End())
			locMap[ts.Name.Name] = &info.Location{
				Start: pos.Offset,
				End:   end.Offset,
			}
		}
	}

	// Create type info objects
	var types []*info.Type

	// Second pass: process each type
	for _, typeName := range typeOrder {
		ts := typeMap[typeName]

		// Create a new Type without relying on processTypeSpec
		t := &info.Type{
			Name:       ts.Name.Name,
			IsExported: ts.Name.IsExported(),
			Location:   locMap[typeName], // Add location information
		}

		// Set comment from docMap if available
		if doc, ok := docMap[typeName]; ok {
			commentText := strings.TrimSpace(doc.Text())
			t.Comment = &info.LocationNode{
				Text: commentText,
				Location: info.Location{
					Start: i.fset.Position(doc.Pos()).Offset,
					End:   i.fset.Position(doc.End()).Offset,
				},
			}
		}

		// Process type details based on the specific type kind
		switch typeExpr := ts.Type.(type) {
		case *ast.StructType:
			t.Kind = reflect.Struct
			if typeExpr.Fields != nil {
				t.Fields = i.processFields(typeExpr.Fields, importMap)
			}
		case *ast.InterfaceType:
			t.Kind = reflect.Interface
		case *ast.ArrayType:
			t.Kind = reflect.Slice
			t.ComponentType = exprToString(typeExpr.Elt, importMap)
		case *ast.MapType:
			t.Kind = reflect.Map
			t.KeyType = exprToString(typeExpr.Key, importMap)
			t.ComponentType = exprToString(typeExpr.Value, importMap)
		case *ast.StarExpr:
			t.IsPointer = true
			t.ComponentType = exprToString(typeExpr.X, importMap)
		case *ast.Ident:
			// Type alias (ts.Assign != 0) or basic type
			if ts.Assign != 0 {
				// Type alias
				t.Kind = reflect.String // Default to String for type alias

				// If no comment provided, generate one
				if t.Comment != nil && t.Comment.Text == "" {
					t.Comment.Text = t.Name + " is a type alias for " + typeExpr.Name
				}
			} else {
				// Basic type
				t.Kind = kindFromBasicType(typeExpr.Name)
			}
		}

		// Extract type parameters
		t.TypeParams = extractTypeParams(ts.TypeParams, importMap)

		types = append(types, t)
	}

	infoFile.Types = types

	// Collect receiver types from methods
	receiverTypes := make(map[string]*info.Type)

	// Third pass: collect methods and create types for receivers if they don't exist
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		if !i.config.IncludeUnexported && !funcDecl.Name.IsExported() {
			continue
		}

		recvField := funcDecl.Recv.List[0]
		recvTypeStr := exprToString(recvField.Type, importMap)
		baseTypeName := extractBaseTypeName(recvTypeStr)

		// Find existing type or create a new one if not found
		var targetType *info.Type

		// Look through existing types first
		for _, t := range types {
			if t.Name == baseTypeName {
				targetType = t
				break
			}
		}

		// If type not found in existing types, check if we've already created it from a receiver
		if targetType == nil {
			if t, exists := receiverTypes[baseTypeName]; exists {
				targetType = t
			} else {
				// Create a new type based on the receiver
				targetType = &info.Type{
					Name:       baseTypeName,
					IsExported: true, // Assume exported since we have an exported method on it
					Methods:    []info.Method{},
					// We don't know the kind yet, but we can assume it's a struct most of the time
					Kind: reflect.Struct,
				}

				// Store the new type
				receiverTypes[baseTypeName] = targetType
				infoFile.Types = append(infoFile.Types, targetType)
			}
		}

		method := i.processMethod(funcDecl, importMap)
		targetType.Methods = append(targetType.Methods, method)
	}

	return infoFile, nil
}

// processTypeDetails extracts additional type information based on the specific type
func (i *Inspector) processTypeDetails(ts *ast.TypeSpec, t *info.Type, importMap map[string]string) {
	switch typeExpr := ts.Type.(type) {
	case *ast.StructType:
		t.Kind = reflect.Struct
	case *ast.InterfaceType:
		t.Kind = reflect.Interface
	case *ast.ArrayType:
		t.Kind = reflect.Slice
		t.ComponentType = exprToString(typeExpr.Elt, importMap)
	case *ast.MapType:
		t.Kind = reflect.Map
		t.KeyType = exprToString(typeExpr.Key, importMap)
		t.ComponentType = exprToString(typeExpr.Value, importMap)
	case *ast.StarExpr:
		t.IsPointer = true
		t.ComponentType = exprToString(typeExpr.X, importMap)
	case *ast.Ident:
		// Basic type or type alias
		if ts.Assign != 0 {
			// Type alias
			t.Kind = reflect.String // Using String as a default for type alias
		} else {
			// Basic type or reference to another type
			t.Kind = kindFromBasicType(typeExpr.Name)
		}
	case *ast.SelectorExpr:
		// Type from another package
		pkgName := ""
		if ident, ok := typeExpr.X.(*ast.Ident); ok {
			pkgName = ident.Name
		}

		if pkgPath, ok := importMap[pkgName]; ok {
			t.Package = pkgName
			t.PackagePath = pkgPath
		}
	}
}

// ExtractBaseTypeName tries to parse a type string like "*MyStruct[T]" and returns just "MyStruct"
// Exported for testing
func ExtractBaseTypeName(typStr string) string {
	return extractBaseTypeName(typStr)
}
