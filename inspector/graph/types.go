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

// AddField adds a field to the type
func (t *Type) AddField(field *Field) {
	// Initialize fieldMap if it doesn't exist
	if t.fieldMap == nil {
		t.fieldMap = make(map[string]int)
	}

	// Add field to the fields slice
	t.Fields = append(t.Fields, field)

	// Update the field map
	t.fieldMap[field.Name] = len(t.Fields) - 1
}

// RemoveField removes a field from the type by name
func (t *Type) RemoveField(fieldName string) bool {
	if t.fieldMap == nil {
		return false
	}

	idx, ok := t.fieldMap[fieldName]
	if !ok {
		return false
	}

	// Remove the field from the fields slice
	t.Fields = append(t.Fields[:idx], t.Fields[idx+1:]...)

	// Rebuild the field map
	delete(t.fieldMap, fieldName)
	for i := idx; i < len(t.Fields); i++ {
		t.fieldMap[t.Fields[i].Name] = i
	}

	return true
}

// AddMethod adds a method to the type
func (t *Type) AddMethod(method *Function) {
	// Initialize methodMap if it doesn't exist
	if t.methodMap == nil {
		t.methodMap = make(map[string]int)
	}

	// Add method to the methods slice
	t.Methods = append(t.Methods, method)

	// Update the method map
	t.methodMap[method.Name] = len(t.Methods) - 1
}

// RemoveMethod removes a method from the type by name
func (t *Type) RemoveMethod(methodName string) bool {
	if t.methodMap == nil {
		return false
	}

	idx, ok := t.methodMap[methodName]
	if !ok {
		return false
	}

	// Remove the method from the methods slice
	t.Methods = append(t.Methods[:idx], t.Methods[idx+1:]...)

	// Rebuild the method map
	delete(t.methodMap, methodName)
	for i := idx; i < len(t.Methods); i++ {
		t.methodMap[t.Methods[i].Name] = i
	}

	return true
}

// Clone creates a deep copy of the type
func (t *Type) Clone() *Type {
	newType := &Type{
		Name:          t.Name,
		Kind:          t.Kind,
		Tag:           t.Tag,
		Package:       t.Package,
		PackagePath:   t.PackagePath,
		ComponentType: t.ComponentType,
		KeyType:       t.KeyType,
		IsExported:    t.IsExported,
		IsPointer:     t.IsPointer,
		Implements:    make([]string, len(t.Implements)),
		Extends:       make([]string, len(t.Extends)),
		TypeParams:    make([]*TypeParam, len(t.TypeParams)),
	}

	// Copy comment and annotation if they exist
	if t.Comment != nil {
		newType.Comment = &LocationNode{
			Text:     t.Comment.Text,
			Location: t.Comment.Location,
		}
	}

	if t.Annotation != nil {
		newType.Annotation = &LocationNode{
			Text:     t.Annotation.Text,
			Location: t.Annotation.Location,
		}
	}

	// Copy location if it exists
	if t.Location != nil {
		newType.Location = &Location{
			Raw:   t.Location.Raw,
			Start: t.Location.Start,
			End:   t.Location.End,
		}
	}

	// Copy implements and extends
	copy(newType.Implements, t.Implements)
	copy(newType.Extends, t.Extends)

	// Copy type parameters
	for i, param := range t.TypeParams {
		newType.TypeParams[i] = &TypeParam{
			Name:       param.Name,
			Constraint: param.Constraint,
		}
	}

	// Initialize maps
	newType.fieldMap = make(map[string]int)
	newType.methodMap = make(map[string]int)

	return newType
}

// CreateTypeFromFields creates a new type with the given name and selected fields
func CreateTypeFromFields(name string, sourceType *Type, fieldNames []string) *Type {
	newType := sourceType.Clone()
	newType.Name = name
	newType.Fields = []*Field{}
	newType.fieldMap = make(map[string]int)

	// Add selected fields
	for _, fieldName := range fieldNames {
		field := sourceType.GetField(fieldName)
		if field != nil {
			newType.AddField(field)
		}
	}

	return newType
}

// CreateTypeFromMethods creates a new type with the given name and selected methods
func CreateTypeFromMethods(name string, sourceType *Type, methodNames []string) *Type {
	newType := sourceType.Clone()
	newType.Name = name
	newType.Methods = []*Function{}
	newType.methodMap = make(map[string]int)

	// Add selected methods
	for _, methodName := range methodNames {
		method := sourceType.GetMethod(methodName)
		if method != nil {
			newType.AddMethod(method)
		}
	}

	return newType
}

// CreateCompositeType creates a new type by combining fields and methods from multiple types
func CreateCompositeType(name string, sourceTypes []*Type, fieldNames [][]string, methodNames [][]string) *Type {
	if len(sourceTypes) == 0 {
		return nil
	}

	newType := &Type{
		Name:       name,
		Kind:       sourceTypes[0].Kind,
		Package:    sourceTypes[0].Package,
		PackagePath: sourceTypes[0].PackagePath,
		IsExported: true,
		Fields:     []*Field{},
		Methods:    []*Function{},
		fieldMap:   make(map[string]int),
		methodMap:  make(map[string]int),
	}

	// Add fields from each source type
	for i, sourceType := range sourceTypes {
		if i < len(fieldNames) {
			for _, fieldName := range fieldNames[i] {
				field := sourceType.GetField(fieldName)
				if field != nil {
					newType.AddField(field)
				}
			}
		}
	}

	// Add methods from each source type
	for i, sourceType := range sourceTypes {
		if i < len(methodNames) {
			for _, methodName := range methodNames[i] {
				method := sourceType.GetMethod(methodName)
				if method != nil {
					newType.AddMethod(method)
				}
			}
		}
	}

	return newType
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
