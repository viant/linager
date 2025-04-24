package golang

import (
	"github.com/viant/linager/inspector/graph"
	"go/ast"
	"reflect"
	"strings"
)

// InspectStatement inspects an AST statement and returns type information
func (i *Inspector) InspectStatement(stmt ast.Stmt, importMap map[string]string) (*graph.Type, error) {
	if stmt == nil {
		return nil, nil
	}

	switch s := stmt.(type) {
	case *ast.DeclStmt:
		// Statement containing a declaration (like a var or type declaration)
		genDecl, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return nil, nil
		}

		// Only consider the first specification for simplicity
		if len(genDecl.Specs) == 0 {
			return nil, nil
		}

		switch spec := genDecl.Specs[0].(type) {
		case *ast.TypeSpec:
			// Type declaration
			kind := determineTypeKindReflect(spec)
			// Type assertion to convert interface{} to reflect.Kind
			return &graph.Type{
				Name:       spec.Name.Name,
				Kind:       kind.(reflect.Kind),
				IsExported: spec.Name.IsExported(),
			}, nil

		case *ast.ValueSpec:
			// Var or const declaration
			if len(spec.Values) == 0 {
				// No initial value
				if spec.Type != nil {
					// With explicit type
					typeStr := exprToString(spec.Type, importMap)
					kind := kindFromTypeName(typeStr)
					// Type assertion to convert interface{} to reflect.Kind
					return &graph.Type{
						Name: typeStr,
						Kind: kind.(reflect.Kind),
					}, nil
				}
				return nil, nil
			}

			// With initial value - infer type from first value
			// For simplicity, we only look at the first value's type
			expr := spec.Values[0]
			return i.InspectExpression(expr, importMap)
		}

		return nil, nil

	case *ast.AssignStmt:
		// Assignment statement: a = b or var a = b
		if len(s.Rhs) == 0 {
			return nil, nil
		}

		// For simplicity, we only infer from the first right-hand side expression
		return i.InspectExpression(s.Rhs[0], importMap)

	case *ast.ReturnStmt:
		// Return statement
		if len(s.Results) == 0 {
			return nil, nil
		}

		// For simplicity, we only infer from the first return value
		return i.InspectExpression(s.Results[0], importMap)

	case *ast.ExprStmt:
		// Expression statement like function calls
		return i.InspectExpression(s.X, importMap)

	case *ast.BlockStmt:
		// Block of statements - we'd need to analyze control flow
		// For simplicity, we don't infer types from blocks
		return nil, nil

	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		// Control flow statements - we'd need to analyze control flow
		// For simplicity, we don't infer types from these
		return nil, nil

	default:
		// Other statement types like defer, go, etc.
		return nil, nil
	}
}

// determineTypeKindReflect converts a type spec kind to a reflect.Kind
func determineTypeKindReflect(ts *ast.TypeSpec) interface{} {
	switch ts.Type.(type) {
	case *ast.StructType:
		return kindFromString("struct")
	case *ast.InterfaceType:
		return kindFromString("interface")
	case *ast.ArrayType:
		return kindFromString("slice")
	case *ast.MapType:
		return kindFromString("map")
	case *ast.ChanType:
		return kindFromString("chan")
	case *ast.FuncType:
		return kindFromString("func")
	default:
		// If it's a type alias (ts.Assign != 0), or a general type
		if ts.Assign > 0 {
			return kindFromString("alias")
		}
		return kindFromString("other")
	}
}

// kindFromTypeName tries to determine reflect.Kind from a type name
func kindFromTypeName(typeName string) interface{} {
	// Strip any package prefix
	parts := strings.Split(typeName, ".")
	baseName := parts[len(parts)-1]

	// Strip any generic parameters
	if idx := strings.IndexByte(baseName, '['); idx >= 0 {
		baseName = baseName[:idx]
	}

	// Check if it's a go_basic.gox type
	if kind := kindFromBasicType(baseName); kind != reflect.Invalid {
		return kind
	}

	// Can't determine the exact kind, so return Invalid
	return reflect.Invalid
}
