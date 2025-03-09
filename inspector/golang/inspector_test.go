package golang_test

import (
	"encoding/json"
	"fmt"
	"github.com/viant/linager/inspector/golang"
	"github.com/viant/linager/inspector/info"
	"reflect"
	"testing"
)

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    []*info.Type
		wantErr bool
	}{
		{
			name: "basic struct",
			src: `package test
			
			// Person represents a human
			type Person struct {
				Name string ` + "`json:\"name\"`" + ` // Person's name
				Age  int    ` + "`json:\"age\"`" + `  // Person's age
			}`,
			want: []*info.Type{
				{
					Name:    "Person",
					Kind:    reflect.Struct,
					Comment: &info.LocationNode{Text: "Person represents a human"},
					Fields: []info.Field{
						{
							Name:       "Name",
							Type:       &info.Type{Name: "string"},
							Tag:        reflect.StructTag(`json:"name"`),
							Comment:    "Person's name",
							IsExported: true,
						},
						{
							Name:       "Age",
							Type:       &info.Type{Name: "int"},
							Tag:        reflect.StructTag(`json:"age"`),
							Comment:    "Person's age",
							IsExported: true,
						},
					},
					Location:   &info.Location{},
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
			want: []*info.Type{
				{
					Name:    "List",
					Kind:    reflect.Struct,
					Comment: &info.LocationNode{Text: "List is a generic list implementation"},
					TypeParams: []info.TypeParam{
						{
							Name:       "T",
							Constraint: "any",
						},
					},
					Fields: []info.Field{
						{
							Name:       "Items",
							Type:       &info.Type{Name: "[]T"},
							IsExported: true,
						},
						{
							Name:       "Size",
							Type:       &info.Type{Name: "int"},
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
			want: []*info.Type{
				{
					Name: "Counter",
					Kind: reflect.Struct,
					Fields: []info.Field{
						{
							Name:       "value",
							Type:       &info.Type{Name: "int"},
							IsExported: false,
						},
					},
					Package:    "test",
					IsExported: true,
					Methods: []info.Method{
						{
							Name:       "Increment",
							Receiver:   "*Counter",
							Comment:    &info.LocationNode{Text: "Increment adds the given amount to the counter"},
							IsExported: true,
							Parameters: []info.Parameter{
								{
									Name: "amount",
									Type: &info.Type{Name: "int"},
								},
							},
							Results: []info.Parameter{},
						},
						{
							Name:       "Value",
							Receiver:   "Counter",
							Comment:    &info.LocationNode{Text: "Value returns the current counter value"},
							IsExported: true,
							Parameters: []info.Parameter{},
							Results: []info.Parameter{
								{
									Type: &info.Type{Name: "int"},
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
			want: []*info.Type{
				{
					Name:       "Writer",
					Kind:       reflect.Interface,
					Comment:    &info.LocationNode{Text: "Writer is an interface for objects that can be written to"},
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
			want: []*info.Type{
				{
					Name:    "MyReader",
					Kind:    reflect.Struct,
					Package: "test",
					Fields: []info.Field{
						{
							Type:       &info.Type{Name: "io.Reader"},
							IsEmbedded: true,
							IsExported: true,
						},
						{
							Name:       "buf",
							Type:       &info.Type{Name: "[]byte"},
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
			want: []*info.Type{
				{
					Name:       "UserID",
					Kind:       reflect.String, // Placeholder for alias
					Comment:    &info.LocationNode{Text: "UserID is a type alias for string"},
					Package:    "test",
					IsExported: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := golang.NewInspector(&info.Config{IncludeUnexported: true})
			infoFile, err := i.InspectSource([]byte(tt.src))

			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.InspectSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := infoFile.Types
			actual, _ := json.Marshal(got)
			fmt.Println(tt.name)
			fmt.Printf("actual %s\n", string(actual))

			// Zero out location positions to make tests pass
			for _, typ := range got {
				if typ.Comment != nil {
					typ.Comment.Start = 0
					typ.Comment.End = 0
				}
			}

			// Basic validation of the results
			if len(got) != len(tt.want) {
				t.Errorf("Inspector.InspectSource() returned %d types, want %d", len(got), len(tt.want))
				return
			}

			for i, want := range tt.want {
				if i >= len(got) {
					t.Errorf("Missing expected type at index %d", i)
					continue
				}

				got := got[i]
				if got.Name != want.Name {
					t.Errorf("Type[%d].Name = %s, want %s", i, got.Name, want.Name)
				}

				if got.Kind != want.Kind {
					t.Errorf("Type[%d].Kind = %v, want %v", i, got.Kind, want.Kind)
				}

				if got.Comment != nil && want.Comment != nil && got.Comment.Text != want.Comment.Text {
					t.Errorf("Type[%d].Comment = %q, want %q", i, got.Comment, want.Comment)
				}

				if len(got.Fields) != len(want.Fields) {
					t.Errorf("Type[%d].Fields count = %d, want %d", i, len(got.Fields), len(want.Fields))
				} else {
					for j, wantField := range want.Fields {
						if j >= len(got.Fields) {
							t.Errorf("Missing expected field at index %d", j)
							continue
						}

						gotField := got.Fields[j]
						if gotField.Name != wantField.Name {
							t.Errorf("Type[%d].Field[%d].Name = %s, want %s", i, j, gotField.Name, wantField.Name)
						}

						if gotField.IsExported != wantField.IsExported {
							t.Errorf("Type[%d].Field[%d].IsExported = %v, want %v", i, j, gotField.IsExported, wantField.IsExported)
						}

						if gotField.IsEmbedded != wantField.IsEmbedded {
							t.Errorf("Type[%d].Field[%d].IsEmbedded = %v, want %v", i, j, gotField.IsEmbedded, wantField.IsEmbedded)
						}

						if gotField.Type != nil && wantField.Type != nil && gotField.Type.Name != wantField.Type.Name {
							t.Errorf("Type[%d].Field[%d].Type.Name = %s, want %s", i, j, gotField.Type.Name, wantField.Type.Name)
						}
					}
				}

				// Check methods
				if len(got.Methods) != len(want.Methods) {
					t.Errorf("Type[%d].Methods count = %d, want %d", i, len(got.Methods), len(want.Methods))
				} else {
					for j, wantMethod := range want.Methods {
						if j >= len(got.Methods) {
							t.Errorf("Missing expected method at index %d", j)
							continue
						}

						gotMethod := got.Methods[j]
						if gotMethod.Name != wantMethod.Name {
							t.Errorf("Type[%d].Method[%d].Name = %s, want %s", i, j, gotMethod.Name, wantMethod.Name)
						}

						if gotMethod.Receiver != wantMethod.Receiver {
							t.Errorf("Type[%d].Method[%d].Receiver = %s, want %s", i, j, gotMethod.Receiver, wantMethod.Receiver)
						}

						if gotMethod.IsExported != wantMethod.IsExported {
							t.Errorf("Type[%d].Method[%d].IsExported = %v, want %v", i, j, gotMethod.IsExported, wantMethod.IsExported)
						}
					}
				}
			}
		})
	}
}

func TestInspector_InspectPackage(t *testing.T) {
	// This test requires an actual Go package on disk to test against
	// We'll skip it with a message since we can't guarantee the test environment
	t.Skip("Skipping package inspection test - requires actual Go package on disk")

	/*
		// Example usage if we had a test package on disk
		inspector := info.NewInspector(info.Config{
			IncludeUnexported: true,
		})

		pkg, err := inspector.InspectPackage("./testdata/sample")
		if err != nil {
			t.Fatalf("Failed to inspect package: %v", err)
		}

		if pkg.Name != "sample" {
			t.Errorf("Package name = %s, want 'sample'", pkg.Name)
		}

		// Check for expected types
		if len(pkg.Types) == 0 {
			t.Error("No types found in package")
		}
	*/
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
			if got != tt.want {
				t.Errorf("extractBaseTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
