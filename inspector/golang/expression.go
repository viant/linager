package golang

import (
	"github.com/viant/linager/inspector/graph"
	"go/ast"
	"go/token"
	"reflect"
)

// InspectExpression inspects an AST expression and returns type information
func (i *Inspector) InspectExpression(expr ast.Expr, importMap map[string]string) (*graph.Type, error) {
	if expr == nil {
		return nil, nil
	}

	switch e := expr.(type) {
	case *ast.Ident:
		// Basic identifier
		if e.Name == "nil" {
			return &graph.Type{Name: "nil", Kind: reflect.Invalid}, nil
		}

		// Check if it's a built-in type
		kind := kindFromBasicType(e.Name)
		if kind != reflect.Invalid {
			return &graph.Type{Name: e.Name, Kind: kind}, nil
		}

		// Might be a reference to another type or variable
		return &graph.Type{
			Name:       e.Name,
			IsExported: e.IsExported(),
		}, nil

	case *ast.SelectorExpr:
		// Type from another package (e.g., fmt.Println)
		pkgName, ok := e.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}

		pkgPath, ok := importMap[pkgName.Name]
		if ok {
			return &graph.Type{
				Name:        e.Sel.Name,
				Package:     pkgName.Name,
				PackagePath: pkgPath,
				IsExported:  e.Sel.IsExported(),
			}, nil
		}

		return &graph.Type{
			Name:       e.Sel.Name,
			IsExported: e.Sel.IsExported(),
		}, nil

	case *ast.StarExpr:
		// Pointer type (e.g., *T)
		baseType, err := i.InspectExpression(e.X, importMap)
		if err != nil {
			return nil, err
		}

		if baseType == nil {
			return nil, nil
		}

		return &graph.Type{
			Name:          "*" + baseType.Name,
			Kind:          reflect.Ptr,
			IsPointer:     true,
			ComponentType: baseType.Name,
		}, nil

	case *ast.ArrayType:
		// Array or slice type (e.g., []T, [5]T)
		elemType, err := i.InspectExpression(e.Elt, importMap)
		if err != nil {
			return nil, err
		}

		if elemType == nil {
			return nil, nil
		}

		if e.Len == nil {
			// Slice
			return &graph.Type{
				Name:          "[]" + elemType.Name,
				Kind:          reflect.Slice,
				ComponentType: elemType.Name,
			}, nil
		} else {
			// Array
			return &graph.Type{
				Name:          "[N]" + elemType.Name,
				Kind:          reflect.Array,
				ComponentType: elemType.Name,
			}, nil
		}

	case *ast.MapType:
		// Map type (e.g., map[K]V)
		keyType, err := i.InspectExpression(e.Key, importMap)
		if err != nil {
			return nil, err
		}

		valType, err := i.InspectExpression(e.Value, importMap)
		if err != nil {
			return nil, err
		}

		if keyType == nil || valType == nil {
			return nil, nil
		}

		return &graph.Type{
			Name:          "map[" + keyType.Name + "]" + valType.Name,
			Kind:          reflect.Map,
			KeyType:       keyType.Name,
			ComponentType: valType.Name,
		}, nil

	case *ast.ChanType:
		// Channel type
		valType, err := i.InspectExpression(e.Value, importMap)
		if err != nil {
			return nil, err
		}

		if valType == nil {
			return nil, nil
		}

		var dirPrefix string
		switch e.Dir {
		case ast.SEND:
			dirPrefix = "chan<- "
		case ast.RECV:
			dirPrefix = "<-chan "
		default:
			dirPrefix = "chan "
		}

		return &graph.Type{
			Name:          dirPrefix + valType.Name,
			Kind:          reflect.Chan,
			ComponentType: valType.Name,
		}, nil

	case *ast.InterfaceType:
		// Interface type
		return &graph.Type{
			Name: "interface{}",
			Kind: reflect.Interface,
		}, nil

	case *ast.StructType:
		// Anonymous struct
		return &graph.Type{
			Name: "struct{}",
			Kind: reflect.Struct,
		}, nil

	case *ast.FuncType:
		// Functions type
		return &graph.Type{
			Name: "func()",
			Kind: reflect.Func,
		}, nil

	case *ast.BasicLit:
		// Literal values
		switch e.Kind {
		case token.INT:
			return &graph.Type{Name: "int", Kind: reflect.Int}, nil
		case token.FLOAT:
			return &graph.Type{Name: "float64", Kind: reflect.Float64}, nil
		case token.IMAG:
			return &graph.Type{Name: "complex128", Kind: reflect.Complex128}, nil
		case token.CHAR:
			return &graph.Type{Name: "rune", Kind: reflect.Int32}, nil
		case token.STRING:
			return &graph.Type{Name: "string", Kind: reflect.String}, nil
		}

	case *ast.CallExpr:
		// Functions call - try to determine return type if possible
		if ident, ok := e.Fun.(*ast.Ident); ok {
			// Type conversions for go_basic.gox types
			kind := kindFromBasicType(ident.Name)
			if kind != reflect.Invalid {
				return &graph.Type{Name: ident.Name, Kind: kind}, nil
			}
		}

		// For other function calls, it's hard to determine the return type
		// without more semantic analysis
		return &graph.Type{Name: "any", Kind: reflect.Interface}, nil
	}

	// For other expressions, we can't determine the type easily
	return &graph.Type{Name: "unknown", Kind: reflect.Invalid}, nil
}
