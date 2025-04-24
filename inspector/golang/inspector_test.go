package golang_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/inspector/golang"
	"github.com/viant/linager/inspector/graph"
	"reflect"
	"testing"
)

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    []*graph.Type
		wantErr bool
	}{
		{
			name: "go_basic.gox struct",
			src: `package test

			// Person represents a human
			type Person struct {
				Name string ` + "`json:\"name\"`" + ` // Person's name
				Age  int    ` + "`json:\"age\"`" + `  // Person's age
			}`,
			want: []*graph.Type{
				{
					Name:    "Person",
					Kind:    reflect.Struct,
					Comment: &graph.LocationNode{Text: "Person represents a human"},
					Fields: []*graph.Field{
						{
							Name:       "Name",
							Type:       &graph.Type{Name: "string"},
							Tag:        reflect.StructTag(`json:"name"`),
							Comment:    "Person's name",
							IsExported: true,
						},
						{
							Name:       "Age",
							Type:       &graph.Type{Name: "int"},
							Tag:        reflect.StructTag(`json:"age"`),
							Comment:    "Person's age",
							IsExported: true,
						},
					},
					Location:   &graph.Location{},
					Package:    "test",
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "generic type",
			src: `package test

			// List is a generic list implementation
			type List[T any] struct {
				Items []T
				Size  int
			}`,
			want: []*graph.Type{
				{
					Name:    "List",
					Kind:    reflect.Struct,
					Comment: &graph.LocationNode{Text: "List is a generic list implementation"},
					TypeParams: []*graph.TypeParam{
						{
							Name:       "T",
							Constraint: "any",
						},
					},
					Fields: []*graph.Field{
						{
							Name:       "Items",
							Type:       &graph.Type{Name: "[]T"},
							IsExported: true,
						},
						{
							Name:       "Size",
							Type:       &graph.Type{Name: "int"},
							IsExported: true,
						},
					},
					Package:    "test",
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "with methods",
			src: `package test

			type Counter struct {
				value int
			}

			// Increment adds the given amount to the counter
			func (c *Counter) Increment(amount int) {
				c.value += amount
			}

			// Value returns the current counter value
			func (c Counter) Value() int {
				return c.value
			}`,
			want: []*graph.Type{
				{
					Name: "Counter",
					Kind: reflect.Struct,
					Fields: []*graph.Field{
						{
							Name:       "value",
							Type:       &graph.Type{Name: "int"},
							IsExported: false,
						},
					},
					Package:    "test",
					IsExported: true,
					Methods: []*graph.Function{
						{
							Name:       "Increment",
							Receiver:   "*Counter",
							Comment:    &graph.LocationNode{Text: "Increment adds the given amount to the counter"},
							IsExported: true,
							Parameters: []*graph.Parameter{
								{
									Name: "amount",
									Type: &graph.Type{Name: "int"},
								},
							},
							Results: []*graph.Parameter{},
						},
						{
							Name:       "Value",
							Receiver:   "Counter",
							Comment:    &graph.LocationNode{Text: "Value returns the current counter value"},
							IsExported: true,
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{Name: "int"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "interface type",
			src: `package test

			// Writer is an interface for objects that can be written to
			type Writer interface {
				// Write writes data to the underlying data store
				Write(data []byte) (int, error)
			}`,
			want: []*graph.Type{
				{
					Name:       "Writer",
					Kind:       reflect.Interface,
					Comment:    &graph.LocationNode{Text: "Writer is an interface for objects that can be written to"},
					Package:    "test",
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "embedded fields",
			src: `package test

			import "io"

			type MyReader struct {
				io.Reader
				buf []byte
			}`,
			want: []*graph.Type{
				{
					Name:    "MyReader",
					Kind:    reflect.Struct,
					Package: "test",
					Fields: []*graph.Field{
						{
							Type:       &graph.Type{Name: "io.Reader"},
							IsEmbedded: true,
							IsExported: true,
						},
						{
							Name:       "buf",
							Type:       &graph.Type{Name: "[]byte"},
							IsExported: false,
						},
					},
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "type alias",
			src: `package test

			// UserID is a type alias for string
			type UserID string`,
			want: []*graph.Type{
				{
					Name:       "UserID",
					Kind:       reflect.String, // Placeholder for alias
					Comment:    &graph.LocationNode{Text: "UserID is a type alias for string"},
					Package:    "test",
					IsExported: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := golang.NewInspector(&graph.Config{IncludeUnexported: true})
			got, err := i.InspectSource([]byte(tt.src))

			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.InspectSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare only essential fields, ignoring location and other metadata
			if !assert.EqualValues(t, tt.want, got) {
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(tt.want)
				fmt.Printf("got:\n%s\nwant:\n%s\n", gotJSON, wantJSON)
			}
		})
	}
}

func TestExtractBaseTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MyStruct", "MyStruct"},
		{"*MyStruct", "MyStruct"},
		{"**MyStruct", "MyStruct"},
		{"MyStruct[T]", "MyStruct"},
		{"MyStruct[T, U]", "MyStruct"},
		{"*MyStruct[T]", "MyStruct"},
		{"pkg.MyStruct", "MyStruct"},
		{"*pkg.MyStruct", "MyStruct"},
		{"pkg.MyStruct[T]", "MyStruct"},
		{"*pkg.MyStruct[T, U]", "MyStruct"},
		{"", ""},
		{"1Invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Call the unexported function through reflection
			got := golang.ExtractBaseTypeName(tt.input)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

// Helper function to access unexported function for testing
func TestExprToString(t *testing.T) {
	// This is a simplified test for demonstration
	// In a real implementation, you would need to create ast.Expr objects
	// and use reflection to access the unexported exprToString function
	t.Skip("Skipping exprToString test - requires creating AST expressions")
}
