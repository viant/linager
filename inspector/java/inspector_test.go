package java_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/java"
	"reflect"
	"testing"
)

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    []*graph.Type
		wantErr bool
	}{
		{
			name: "Simple class",
			source: `package com.example;
@SuppressWarnings("unused")
public class Person {
    private String name;
    private int age;

    public Person(String name, int age) {
        this.name = name;
        this.age = age;
    }

    public String getName() {
        return name;
    }

    public int getAge() {
        return age;
    }
}`,
			want: []*graph.Type{
				{
					Name:       "Person",
					Kind:       reflect.Struct,
					Annotation: &graph.LocationNode{Text: "@SuppressWarnings(\"unused\")"},
					Fields: []*graph.Field{
						{
							Name: "name",
							Type: &graph.Type{
								Name:        "string",
								Kind:        reflect.String,
								PackagePath: "java.lang",
							},
							IsExported: false,
						},
						{
							Name: "age",
							Type: &graph.Type{
								Name: "int",
							},
							IsExported: false,
						},
					},
					Methods: []*graph.Function{
						{
							Name:          "Person",
							IsConstructor: true,
							Parameters: []*graph.Parameter{
								{
									Name: "name",
									Type: &graph.Type{
										Name:        "string",
										Kind:        reflect.String,
										PackagePath: "java.lang",
									},
								},
								{
									Name: "age",
									Type: &graph.Type{
										Name: "int",
									},
								},
							},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "Person",
									},
								},
							},
						},
						{
							Name:       "getName",
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name:        "string",
										Kind:        reflect.String,
										PackagePath: "java.lang",
									},
								},
							},
						},
						{
							Name:       "getAge",
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "int",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Interface",
			source: `package com.example.interfaces;

public interface UserService {
    User findById(Long id);
    List<User> findAll();
    User save(User user);
    void delete(Long id);
}`,
			want: []*graph.Type{
				{
					Name: "UserService",
					Kind: reflect.Interface,
					Methods: []*graph.Function{
						{
							Name: "findById",
							Parameters: []*graph.Parameter{
								{
									Name: "id",
									Type: &graph.Type{
										Name: "Long",
									},
								},
							},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "User",
									},
								},
							},
						},
						{
							Name:       "findAll",
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "List",
									},
								},
							},
						},
						{
							Name: "save",
							Parameters: []*graph.Parameter{
								{
									Name: "user",
									Type: &graph.Type{
										Name: "User",
									},
								},
							},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "User",
									},
								},
							},
						},
						{
							Name: "delete",
							Parameters: []*graph.Parameter{
								{
									Name: "id",
									Type: &graph.Type{
										Name: "Long",
									},
								},
							},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "void",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Enum",
			source: `package com.example.enums;

public enum Day {
    MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY, SATURDAY, SUNDAY
}`,
			want: []*graph.Type{
				{
					Name: "Day",
					Kind: reflect.Int,
				},
			},
			wantErr: false,
		},
		{
			name: "Complex class with generics",
			source: `package com.example.collections;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;

public class CustomList<E> implements Collection<E> {
    private ArrayList<E> internal = new ArrayList<>();

    @Override
    public int size() {
        return internal.size();
    }

    @Override
    public boolean add(E element) {
        return internal.add(element);
    }

    @Override
    public Iterator<E> iterator() {
        return internal.iterator();
    }
}`,
			want: []*graph.Type{
				{
					Name: "CustomList",
					Kind: reflect.Struct,
					TypeParams: []*graph.TypeParam{
						{
							Name: "E",
						},
					},
					Implements: []string{"Collection<E>"},
					Fields: []*graph.Field{
						{
							Name: "internal",
							Type: &graph.Type{
								Name: "ArrayList",
							},
							IsExported: false,
						},
					},
					Methods: []*graph.Function{
						{
							Name:       "size",
							Annotation: &graph.LocationNode{Text: "@Override"},
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "int",
									},
								},
							},
						},
						{
							Name:       "add",
							Annotation: &graph.LocationNode{Text: "@Override"},
							Parameters: []*graph.Parameter{
								{
									Name: "element",
									Type: &graph.Type{
										Name: "E",
									},
								},
							},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "bool",
									},
								},
							},
						},
						{
							Name:       "iterator",
							Annotation: &graph.LocationNode{Text: "@Override"},
							Parameters: []*graph.Parameter{},
							Results: []*graph.Parameter{
								{
									Type: &graph.Type{
										Name: "Iterator",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := java.NewInspector(&graph.Config{IncludeUnexported: true})
			file, err := inspector.InspectSource([]byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.InspectSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if file == nil {
				if !tt.wantErr {
					t.Errorf("Inspector.InspectSource() returned nil file, expected non-nil")
				}
				return
			}

			// Compare only essential fields, ignoring location and other metadata
			// For each type in the expected output, verify it exists in the actual output
			for _, wantType := range tt.want {
				found := false
				for _, gotType := range file.Types {
					if wantType.Name == gotType.Name && wantType.Kind == gotType.Kind {
						found = true

						// Verify fields
						if len(wantType.Fields) > 0 {
							for _, wantField := range wantType.Fields {
								fieldFound := false
								for _, gotField := range gotType.Fields {
									if wantField.Name == gotField.Name {
										fieldFound = true
										assert.Equal(t, wantField.IsExported, gotField.IsExported, "Field %s.IsExported", wantField.Name)
										assert.Equal(t, wantField.IsConstant, gotField.IsConstant, "Field %s.IsConstant", wantField.Name)
										assert.Equal(t, wantField.Type.Name, gotField.Type.Name, "Field %s.Type.Name", wantField.Name)
										break
									}
								}
								assert.True(t, fieldFound, "Field %s not found", wantField.Name)
							}
						}

						// Verify methods
						if len(wantType.Methods) > 0 {
							for _, wantMethod := range wantType.Methods {
								methodFound := false
								for _, gotMethod := range gotType.Methods {
									if wantMethod.Name == gotMethod.Name {
										methodFound = true
										assert.Equal(t, wantMethod.IsConstructor, gotMethod.IsConstructor, "Method %s.IsConstructor", wantMethod.Name)

										// Verify parameters
										if len(wantMethod.Parameters) > 0 {
											assert.Equal(t, len(wantMethod.Parameters), len(gotMethod.Parameters), "Method %s parameters count", wantMethod.Name)
											for i, wantParam := range wantMethod.Parameters {
												if i < len(gotMethod.Parameters) {
													assert.Equal(t, wantParam.Name, gotMethod.Parameters[i].Name, "Method %s parameter %d name", wantMethod.Name, i)
													assert.Equal(t, wantParam.Type.Name, gotMethod.Parameters[i].Type.Name, "Method %s parameter %d type", wantMethod.Name, i)
												}
											}
										}

										// Verify results
										if len(wantMethod.Results) > 0 {
											assert.Equal(t, len(wantMethod.Results), len(gotMethod.Results), "Method %s results count", wantMethod.Name)
											for i, wantResult := range wantMethod.Results {
												if i < len(gotMethod.Results) {
													assert.Equal(t, wantResult.Type.Name, gotMethod.Results[i].Type.Name, "Method %s result %d type", wantMethod.Name, i)
												}
											}
										}

										break
									}
								}
								assert.True(t, methodFound, "Method %s not found", wantMethod.Name)
							}
						}

						// Verify type parameters
						if len(wantType.TypeParams) > 0 {
							assert.Equal(t, len(wantType.TypeParams), len(gotType.TypeParams), "TypeParams count")
							for i, wantParam := range wantType.TypeParams {
								if i < len(gotType.TypeParams) {
									assert.Equal(t, wantParam.Name, gotType.TypeParams[i].Name, "TypeParam %d name", i)
								}
							}
						}

						// Verify implements
						if len(wantType.Implements) > 0 {
							assert.Equal(t, len(wantType.Implements), len(gotType.Implements), "Implements count")
							for i, wantImpl := range wantType.Implements {
								if i < len(gotType.Implements) {
									assert.Equal(t, wantImpl, gotType.Implements[i], "Implements %d", i)
								}
							}
						}

						break
					}
				}
				assert.True(t, found, "Type %s not found", wantType.Name)
			}
		})
	}
}

func TestInspector_InspectFile(t *testing.T) {
	// This test requires actual Java files on disk, so we'll skip it
	t.Skip("Skipping file-based tests - requires Java files on disk")
}

func TestInspector_InspectPackage(t *testing.T) {
	// This test requires actual Java packages on disk, so we'll skip it
	t.Skip("Skipping package-based tests - requires Java packages on disk")
}
