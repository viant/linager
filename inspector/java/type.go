package java

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/inspector/graph"
	"reflect"
	"strings"
)

// JavaTypeToGoType maps Java primitive types to Go equivalents
var JavaTypeToGoType = map[string]struct {
	GoName string
	Kind   reflect.Kind
}{
	"boolean": {"bool", reflect.Bool},
	"char":    {"rune", reflect.Int32},
	"byte":    {"byte", reflect.Uint8},
	"short":   {"int16", reflect.Int16},
	"int":     {"int32", reflect.Int32},
	"long":    {"int64", reflect.Int64},
	"float":   {"float32", reflect.Float32},
	"double":  {"float64", reflect.Float64},
	"String":  {"string", reflect.String},
	"void":    {"void", reflect.Invalid},
}

// resolveJavaType resolves a Java type name to a Go equivalent and reflect.Kind
func resolveJavaType(typeName string) (string, reflect.Kind) {
	// Check for array types
	if strings.HasSuffix(typeName, "[]") {
		baseTypeName := strings.TrimSuffix(typeName, "[]")
		goName, _ := resolveJavaType(baseTypeName)
		return "[]" + goName, reflect.Slice
	}

	// Check primitive types map
	if javaType, ok := JavaTypeToGoType[typeName]; ok {
		return javaType.GoName, javaType.Kind
	}

	// For non-primitive types, treat as interfaces/pointers
	return typeName, reflect.Ptr
}

// extractTypeParameters extracts generic type parameters from a node
func extractTypeParameters(node *sitter.Node, source []byte) []*graph.TypeParam {
	// Look for type_parameters node
	var typeParamNode *sitter.Node

	for i := uint32(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(int(i))
		if child.Type() == "type_parameters" {
			typeParamNode = child
			break
		}
	}

	if typeParamNode == nil {
		return nil
	}

	var params []*graph.TypeParam

	// Process each type parameter
	for i := uint32(0); i < typeParamNode.NamedChildCount(); i++ {
		paramNode := typeParamNode.NamedChild(int(i))

		if paramNode.Type() == "type_parameter" {
			var name, constraint string

			// Get name
			if paramNode.NamedChildCount() > 0 {
				name = paramNode.NamedChild(0).Content(source)
			}

			// Get bounds (constraints)
			for j := uint32(1); j < paramNode.NamedChildCount(); j++ {
				boundNode := paramNode.NamedChild(int(j))
				if boundNode.Type() == "type_bound" {
					if constraint == "" {
						constraint = boundNode.Content(source)
					} else {
						constraint += " & " + boundNode.Content(source)
					}
				}
			}

			// Default constraint to "any" if not specified
			if constraint == "" {
				constraint = "any"
			}

			params = append(params, &graph.TypeParam{
				Name:       name,
				Constraint: constraint,
			})
		}
	}

	return params
}

// extractEnumValues gets the enum constants from an enum declaration
func extractEnumValues(node *sitter.Node, source []byte) []string {
	if node.Type() != "enum_declaration" {
		return nil
	}

	var values []string

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		for i := uint32(0); i < bodyNode.NamedChildCount(); i++ {
			child := bodyNode.NamedChild(int(i))
			if child.Type() == "enum_constant" {
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					values = append(values, nameNode.Content(source))
				}
			}
		}
	}

	return values
}

// extractMethodBody extracts the body of a method as a string
func extractMethodBody(node *sitter.Node, source []byte) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		return bodyNode.Content(source)
	}
	return ""
}

// parseJavaType converts a Java type node to an info.Type
func parseJavaType(node *sitter.Node, source []byte, importMap map[string]string) *graph.Type {
	// Create a basic type with location information
	typeInfo := &graph.Type{
		Name: node.Content(source),
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	switch node.Type() {
	case "integral_type":
		if node.NamedChildCount() == 0 {
			return typeInfo
		}
		switch node.NamedChild(0).Type() {
		case "int":
			typeInfo.Name = "int32"
			typeInfo.Kind = reflect.Int32
		case "short":
			typeInfo.Name = "int16"
			typeInfo.Kind = reflect.Int16
		case "long":
			typeInfo.Name = "int64"
			typeInfo.Kind = reflect.Int64
		case "char":
			typeInfo.Name = "rune"
			typeInfo.Kind = reflect.Int32
		case "byte":
			typeInfo.Name = "byte"
			typeInfo.Kind = reflect.Uint8
		}

	case "floating_point_type":
		if node.NamedChildCount() == 0 {
			return typeInfo
		}
		switch node.NamedChild(0).Type() {
		case "float":
			typeInfo.Name = "float32"
			typeInfo.Kind = reflect.Float32
		case "double":
			typeInfo.Name = "float64"
			typeInfo.Kind = reflect.Float64
		}

	case "boolean_type":
		typeInfo.Name = "bool"
		typeInfo.Kind = reflect.Bool

	case "void_type":
		typeInfo.Name = "void"

	case "array_type":
		elemType := parseJavaType(node.NamedChild(0), source, importMap)
		if elemType != nil {
			typeInfo.Name = "[]" + elemType.Name
			typeInfo.Kind = reflect.Array
			typeInfo.ComponentType = elemType.Name

			// Transfer package information from element type
			typeInfo.PackagePath = elemType.PackagePath
		}

	case "type_identifier":
		typeName := node.Content(source)
		if typeName == "String" {
			typeInfo.Name = "string"
			typeInfo.Kind = reflect.String
			typeInfo.PackagePath = "java.lang"
		} else {
			typeInfo.Kind = reflect.Ptr

			// Look for import information to add full package path
			if packagePath, ok := importMap[typeName]; ok {
				typeInfo.PackagePath = packagePath
				// Create fully qualified name if we have package info
				typeInfo.Name = typeName // Store unqualified name
			}
		}

	case "scoped_type_identifier":
		typeInfo.Kind = reflect.Interface

		// Extract package from scoped identifier
		scopedName := node.Content(source)
		if lastDotIndex := strings.LastIndex(scopedName, "."); lastDotIndex != -1 {
			packagePath := scopedName[:lastDotIndex]
			typeInfo.PackagePath = packagePath
			typeInfo.Name = scopedName // Store fully qualified name
		}

	case "generic_type":
		if node.NamedChildCount() > 0 {
			baseTypeNode := node.NamedChild(0)
			baseTypeName := baseTypeNode.Content(source)
			typeInfo.Kind = reflect.Ptr

			// Try to find package info for the base type
			if packagePath, ok := importMap[baseTypeName]; ok {
				typeInfo.PackagePath = packagePath
			}

			var typeParams []*graph.TypeParam

			// Process type parameters
			typeArgsNode := node.ChildByFieldName("type_arguments")
			if typeArgsNode != nil {
				var paramTypes []string
				for i := uint32(0); i < typeArgsNode.NamedChildCount(); i++ {
					paramNode := typeArgsNode.NamedChild(int(i))
					paramType := parseJavaType(paramNode, source, importMap)
					if paramType != nil {
						paramName := paramType.Name
						// Try to fully qualify parameter type
						if paramType.PackagePath != "" && !strings.Contains(paramName, ".") {
							paramName = paramType.PackagePath + "." + paramName
						}
						typeParams = append(typeParams, &graph.TypeParam{
							Name:       paramName,
							Constraint: "any",
						})
						paramTypes = append(paramTypes, paramName)
					}
				}

				// Add type parameters to name
				if len(paramTypes) > 0 {
					typeInfo.Name = baseTypeName + "<" + strings.Join(paramTypes, ", ") + ">"
				} else {
					typeInfo.Name = baseTypeName
				}
			} else {
				typeInfo.Name = baseTypeName
			}

			typeInfo.TypeParams = typeParams
		}
	}

	return typeInfo
}

// extractSimpleTypeName extracts the simple name from a possibly qualified name
// e.g., "java.util.List" -> "List"
func extractSimpleTypeName(qualifiedName string) string {
	lastDotIndex := strings.LastIndex(qualifiedName, ".")
	if lastDotIndex != -1 {
		return qualifiedName[lastDotIndex+1:]
	}
	return qualifiedName
}
