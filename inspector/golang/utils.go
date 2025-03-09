package golang

import (
	"fmt"
	"github.com/viant/linager/inspector/info"
	"go/ast"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// formatFuncType formats a function type as a string
func formatFuncType(name string, fn *ast.FuncType, importMap map[string]string) string {
	var sb strings.Builder
	sb.WriteString("func " + name + "(")

	// Format parameters
	if fn.Params != nil {
		var params []string
		for _, param := range fn.Params.List {
			paramType := exprToString(param.Type, importMap)
			if len(param.Names) == 0 {
				params = append(params, paramType)
			} else {
				for _, name := range param.Names {
					params = append(params, name.Name+" "+paramType)
				}
			}
		}
		sb.WriteString(strings.Join(params, ", "))
	}

	sb.WriteString(")")

	// Format results
	if fn.Results != nil {
		if len(fn.Results.List) == 1 && len(fn.Results.List[0].Names) == 0 {
			// Single unnamed result
			sb.WriteString(" " + exprToString(fn.Results.List[0].Type, importMap))
		} else {
			sb.WriteString(" (")
			var results []string
			for _, result := range fn.Results.List {
				resultType := exprToString(result.Type, importMap)
				if len(result.Names) == 0 {
					results = append(results, resultType)
				} else {
					for _, name := range result.Names {
						results = append(results, name.Name+" "+resultType)
					}
				}
			}
			sb.WriteString(strings.Join(results, ", "))
			sb.WriteString(")")
		}
	}

	return sb.String()
}

// buildImportMap creates a map of package name -> package path
func buildImportMap(file *ast.File) map[string]string {
	importMap := make(map[string]string)
	for _, imp := range file.Imports {
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
		} else {
			// Extract the package name from the path (last segment)
			path := strings.Trim(imp.Path.Value, "\"")
			pkgName = path[strings.LastIndex(path, "/")+1:]
		}
		importMap[pkgName] = strings.Trim(imp.Path.Value, "\"")
	}
	return importMap
}

// determineTypeKind returns a string representation of the type kind
func determineTypeKind(ts *ast.TypeSpec) string {
	switch ts.Type.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	case *ast.ArrayType:
		return "slice"
	case *ast.MapType:
		return "map"
	case *ast.ChanType:
		return "chan"
	case *ast.FuncType:
		return "func"
	default:
		// If it's a type alias (ts.Assign != 0), or a general type
		if ts.Assign > 0 {
			return "alias"
		}
		return "other"
	}
}

// kindFromString converts a string to a reflect.Kind
func kindFromString(kind string) reflect.Kind {
	switch kind {
	case "struct":
		return reflect.Struct
	case "interface":
		return reflect.Interface
	case "slice":
		return reflect.Slice
	case "map":
		return reflect.Map
	case "chan":
		return reflect.Chan
	case "func":
		return reflect.Func
	case "alias":
		return reflect.Interface // As placeholder for alias
	default:
		return reflect.Invalid
	}
}

// extractBaseTypeName extracts the base type name from a type string
// For example, for "*pkg.MyStruct[T]", it returns "MyStruct"
func extractBaseTypeName(typStr string) string {
	// Remove pointer stars
	typStr = strings.TrimLeft(typStr, "*")

	// Remove generic parameters
	if idx := strings.IndexByte(typStr, '['); idx >= 0 {
		typStr = typStr[:idx]
	}

	// Extract the type name from qualified name
	if idx := strings.LastIndexByte(typStr, '.'); idx >= 0 {
		typStr = typStr[idx+1:]
	}

	// Validate that it's a valid identifier
	if len(typStr) == 0 || !isValidIdent(typStr) {
		return ""
	}

	return typStr
}

// isValidIdent checks if a string is a valid Go identifier
func isValidIdent(s string) bool {
	if len(s) == 0 {
		return false
	}

	// First character must be a letter or underscore
	if !isLetter(rune(s[0])) && s[0] != '_' {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(s); i++ {
		if !isLetter(rune(s[i])) && !isDigit(rune(s[i])) && s[i] != '_' {
			return false
		}
	}

	return true
}

// isLetter checks if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit checks if a rune is a digit
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// extractTypeParams extracts type parameters from an ast.FieldList
func extractTypeParams(params *ast.FieldList, importMap map[string]string) []*info.TypeParam {
	if params == nil {
		return nil
	}

	var result []*info.TypeParam
	for _, param := range params.List {
		for _, name := range param.Names {
			typeParam := &info.TypeParam{
				Name:       name.Name,
				Constraint: exprToString(param.Type, importMap),
			}
			result = append(result, typeParam)
		}
	}

	return result
}

// isExportedType checks if a type expression represents an exported type
func isExportedType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.IsExported()
	case *ast.StarExpr:
		return isExportedType(t.X)
	case *ast.SelectorExpr:
		return t.Sel.IsExported()
	default:
		return false
	}
}

// getImportPath attempts to determine a package import path from file path
func getImportPath(filePath string) string {
	// This is a heuristic and might not work for all project layouts
	// Try to extract import path by looking for "src" or "go/src" in the path
	srcPattern := regexp.MustCompile(`(?:^|/)(?:go/)?src/(.+)$`)
	matches := srcPattern.FindStringSubmatch(filePath)
	if len(matches) > 1 {
		return matches[1]
	}
	return filePath
}

// extractValueAsString extracts a value from an expression as a string
func extractValueAsString(expr ast.Expr, fset *token.FileSet) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.CompositeLit:
		return "{...}" // Simplified representation
	case *ast.FuncLit:
		return "func(){...}" // Simplified representation
	case *ast.BinaryExpr:
		leftVal := extractValueAsString(e.X, fset)
		rightVal := extractValueAsString(e.Y, fset)
		return fmt.Sprintf("%s %s %s", leftVal, e.Op.String(), rightVal)
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", extractValueAsString(e.X, fset), e.Sel.Name)
	case *ast.CallExpr:
		return fmt.Sprintf("%s(...)", extractValueAsString(e.Fun, fset))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", e.Op.String(), extractValueAsString(e.X, fset))
	default:
		return "..." // Default for complex expressions
	}
}

// kindFromBasicType returns the reflect.Kind for Go basic types
func kindFromBasicType(typeName string) reflect.Kind {
	switch strings.ToLower(typeName) {
	case "bool":
		return reflect.Bool
	case "int":
		return reflect.Int
	case "int8":
		return reflect.Int8
	case "int16":
		return reflect.Int16
	case "int32", "rune":
		return reflect.Int32
	case "int64":
		return reflect.Int64
	case "uint":
		return reflect.Uint
	case "uint8", "byte":
		return reflect.Uint8
	case "uint16":
		return reflect.Uint16
	case "uint32":
		return reflect.Uint32
	case "uint64":
		return reflect.Uint64
	case "float32":
		return reflect.Float32
	case "float64":
		return reflect.Float64
	case "complex64":
		return reflect.Complex64
	case "complex128":
		return reflect.Complex128
	case "string":
		return reflect.String
	case "uintptr":
		return reflect.Uintptr
	case "error":
		return reflect.Interface
	}
	return reflect.Invalid
}

// exprToString converts an AST expression to a type string
func exprToString(expr ast.Expr, importMap map[string]string) string {
	if expr == nil {
		return ""
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name

	case *ast.SelectorExpr:
		return exprToString(t.X, importMap) + "." + t.Sel.Name

	case *ast.StarExpr:
		return "*" + exprToString(t.X, importMap)

	case *ast.ArrayType:
		if t.Len == nil {
			// Slice
			return "[]" + exprToString(t.Elt, importMap)
		}
		// Array
		return "[" + exprToString(t.Len, importMap) + "]" + exprToString(t.Elt, importMap)

	case *ast.MapType:
		return "map[" + exprToString(t.Key, importMap) + "]" + exprToString(t.Value, importMap)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.ChanType:
		var prefix string
		switch t.Dir {
		case ast.RECV:
			prefix = "<-chan "
		case ast.SEND:
			prefix = "chan<- "
		default:
			prefix = "chan "
		}
		return prefix + exprToString(t.Value, importMap)

	case *ast.FuncType:
		return formatFuncType("", t, importMap)

	case *ast.BasicLit:
		return t.Value

	case *ast.IndexExpr: // For simple generics like List[T]
		return exprToString(t.X, importMap) + "[" + exprToString(t.Index, importMap) + "]"

	case *ast.IndexListExpr: // For generics with multiple parameters like Map[K, V]
		params := make([]string, 0, len(t.Indices))
		for _, idx := range t.Indices {
			params = append(params, exprToString(idx, importMap))
		}
		return exprToString(t.X, importMap) + "[" + strings.Join(params, ", ") + "]"

	case *ast.Ellipsis: // For variadic parameters like ...T
		return "..." + exprToString(t.Elt, importMap)

	case *ast.StructType:
		return "struct{...}"

	case *ast.UnaryExpr:
		return t.Op.String() + exprToString(t.X, importMap)

	case *ast.BinaryExpr:
		return exprToString(t.X, importMap) + " " + t.Op.String() + " " + exprToString(t.Y, importMap)

	case *ast.CompositeLit:
		return exprToString(t.Type, importMap) + "{...}"

	case *ast.ParenExpr:
		return "(" + exprToString(t.X, importMap) + ")"

	default:
		return fmt.Sprintf("<%T>", expr)
	}
}

// hasGoFilesInDir checks if a directory contains Go files
func hasGoFilesInDir(dirPath string, skipTests bool) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			if skipTests && strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			return true, nil
		}
	}

	return false, nil
}
