package info

import (
	"reflect"
)

// Type represents a parsed Go type with rich metadata
type Type struct {
	Name          string            // Type name
	Kind          reflect.Kind      // Go reflect kind
	Tag           reflect.StructTag // Tags for struct fields
	Package       string            // Package name
	PackagePath   string            // Full package import path
	ComponentType string            // For container types (slices, maps)
	KeyType       string            // For map types
	Comment       *LocationNode     // Type documentation

	Annotation *LocationNode // Annotations for the type
	IsExported bool          // Whether the type is exported
	Fields     []Field       // Struct fields (if applicable)
	Methods    []Method      // Type methods
	TypeParams []TypeParam   // Generic type parameters
	Implements []string      // Interfaces this type implements
	IsPointer  bool          // Whether the type is a pointer
	Location   *Location     // Location of the type in the source code
	Extends    []string
}

// Content returns the content of the method including its receiver, parameters, and results
func (m *Type) Content(source []byte) string {
	if m.Location == nil {
		return ""
	}
	start := m.Location.Start
	if m.Comment != nil && m.Comment.Location.Start > 0 {
		start = min(start, m.Comment.Location.Start)
	}
	if m.Annotation != nil && m.Annotation.Location.Start > 0 {
		start = min(start, m.Annotation.Location.Start)
	}
	return string(source[start:m.Location.End])
}

type LocationNode struct {
	Text string
	Location
}

func NewNodeLocation(text string) *LocationNode {
	return &LocationNode{
		Text: text,
	}
}

// Field represents a struct field
type Field struct {
	Name       string
	Type       *Type
	Tag        reflect.StructTag
	Location   *Location
	Comment    string
	Annotation string
	IsExported bool
	IsEmbedded bool
	IsStatic   bool
	IsConstant bool
}

// Method represents a type method
type Method struct {
	Name          string
	Comment       *LocationNode
	Annotation    *LocationNode
	Receiver      string
	TypeParams    []TypeParam
	Parameters    []Parameter
	Results       []Parameter
	Body          *LocationNode
	IsExported    bool
	Location      *Location // Location of the method in the source code
	IsStatic      bool      // Whether the method is static (class method)
	IsConstructor bool
	Signature     string
}

// Content returns the content of the method including its receiver, parameters, and results
func (m *Method) Content(source []byte) string {
	if m.Location == nil {
		return ""
	}
	start := m.Location.Start
	if m.Comment != nil && m.Comment.Location.Start > 0 {
		start = min(start, m.Comment.Location.Start)
	}
	if m.Annotation != nil && m.Annotation.Location.Start > 0 {
		start = min(start, m.Annotation.Location.Start)
	}
	return string(source[start:m.Location.End])
}

// TypeParam represents a generic type parameter
type TypeParam struct {
	Name       string
	Constraint string
}

// Parameter represents a function parameter or result
type Parameter struct {
	Name string
	Type *Type
}
