package java

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/inspector/info"
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
func extractTypeParameters(node *sitter.Node, source []byte) []info.TypeParam {
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

	var params []info.TypeParam

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

			params = append(params, info.TypeParam{
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
