package graph

import (
	"reflect"
	"strings"
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
	Fields     []*Field      // Struct fields (if applicable)
	Methods    []*Function   // Type methods
	TypeParams []*TypeParam  // Generic type parameters
	Implements []string      // Interfaces this type implements
	IsPointer  bool          // Whether the type is a pointer
	Location   *Location     // Location of the type in the source code
	Extends    []string

	fieldMap  map[string]int // Map of fields for quick lookup
	methodMap map[string]int // Map of methods for quick lookup

}

// GetField retrieves a constant by name from the file
func (f *Type) GetField(name string) *Field {
	if f.Fields == nil {
		return nil
	}
	if idx, ok := f.fieldMap[name]; ok && idx < len(f.Fields) {
		return f.Fields[idx]
	}
	return nil
}

// GetMethod retrieves a constant by name from the file
func (f *Type) GetMethod(name string) *Function {
	if f.Methods == nil {
		return nil
	}
	if idx, ok := f.methodMap[name]; ok && idx < len(f.Methods) {
		return f.Methods[idx]
	}
	return nil
}

// Content returns the content of the method including its receiver, parameters, and results
func (m *Type) Content() string {
	builder := &strings.Builder{}
	if m.Location == nil {
		return ""
	}

	builder.WriteString(m.Location.Raw)

	for _, field := range m.Fields {
		if field.Location != nil {
			builder.WriteString("\n")
			builder.WriteString(field.Content())
		}
	}
	builder.WriteString("\n}\n")
	return builder.String()
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

func (f *Field) Content() string {
	if f.Location == nil {
		return ""
	}
	return f.Location.Raw
}

// Function represents a type method
type Function struct {
	Name          string
	Comment       *LocationNode
	Annotation    *LocationNode
	Receiver      string
	TypeParams    []*TypeParam
	Parameters    []*Parameter
	Results       []*Parameter
	Body          *LocationNode
	IsExported    bool
	Location      *Location // Location of the method in the source code
	IsStatic      bool      // Whether the method is static (class method)
	IsConstructor bool
	Signature     string
	Hash          int32
}

// Content returns the content of the method including its receiver, parameters, and results
func (m *Function) Content() string {
	if m.Location == nil {
		return ""
	}
	return m.Location.Raw
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
