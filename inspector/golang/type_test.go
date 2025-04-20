package golang

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/inspector/graph"
	"reflect"
	"testing"
)

func TestTypeManipulation(t *testing.T) {
	// Create a test type with fields and methods
	testType := &graph.Type{
		Name:        "TestType",
		Kind:        reflect.Struct,
		Package:     "test",
		PackagePath: "github.com/test",
		IsExported:  true,
		Fields:      []*graph.Field{},
		Methods:     []*graph.Function{},
	}

	// Add fields to the test type
	field1 := &graph.Field{
		Name:       "Field1",
		IsExported: true,
		Type: &graph.Type{
			Name:       "string",
			Kind:       reflect.String,
			IsExported: true,
		},
	}
	field2 := &graph.Field{
		Name:       "Field2",
		IsExported: true,
		Type: &graph.Type{
			Name:       "int",
			Kind:       reflect.Int,
			IsExported: true,
		},
	}
	field3 := &graph.Field{
		Name:       "Field3",
		IsExported: true,
		Type: &graph.Type{
			Name:       "bool",
			Kind:       reflect.Bool,
			IsExported: true,
		},
	}

	testType.AddField(field1)
	testType.AddField(field2)
	testType.AddField(field3)

	// Add methods to the test type
	method1 := &graph.Function{
		Name:       "Method1",
		IsExported: true,
		Parameters: []*graph.Parameter{},
		Results:    []*graph.Parameter{},
	}
	method2 := &graph.Function{
		Name:       "Method2",
		IsExported: true,
		Parameters: []*graph.Parameter{},
		Results:    []*graph.Parameter{},
	}
	method3 := &graph.Function{
		Name:       "Method3",
		IsExported: true,
		Parameters: []*graph.Parameter{},
		Results:    []*graph.Parameter{},
	}

	testType.AddMethod(method1)
	testType.AddMethod(method2)
	testType.AddMethod(method3)

	// Test adding and removing fields
	t.Run("AddRemoveField", func(t *testing.T) {
		// Test adding a field
		field4 := &graph.Field{
			Name:       "Field4",
			IsExported: true,
			Type: &graph.Type{
				Name:       "float64",
				Kind:       reflect.Float64,
				IsExported: true,
			},
		}
		testType.AddField(field4)
		assert.Equal(t, 4, len(testType.Fields))
		assert.NotNil(t, testType.GetField("Field4"))

		// Test removing a field
		success := testType.RemoveField("Field4")
		assert.True(t, success)
		assert.Equal(t, 3, len(testType.Fields))
		assert.Nil(t, testType.GetField("Field4"))

		// Test removing a non-existent field
		success = testType.RemoveField("NonExistentField")
		assert.False(t, success)
	})

	// Test adding and removing methods
	t.Run("AddRemoveMethod", func(t *testing.T) {
		// Test adding a method
		method4 := &graph.Function{
			Name:       "Method4",
			IsExported: true,
			Parameters: []*graph.Parameter{},
			Results:    []*graph.Parameter{},
		}
		testType.AddMethod(method4)
		assert.Equal(t, 4, len(testType.Methods))
		assert.NotNil(t, testType.GetMethod("Method4"))

		// Test removing a method
		success := testType.RemoveMethod("Method4")
		assert.True(t, success)
		assert.Equal(t, 3, len(testType.Methods))
		assert.Nil(t, testType.GetMethod("Method4"))

		// Test removing a non-existent method
		success = testType.RemoveMethod("NonExistentMethod")
		assert.False(t, success)
	})

	// Test creating a new type from fields
	t.Run("CreateTypeFromFields", func(t *testing.T) {
		newType := graph.CreateTypeFromFields("NewType", testType, []string{"Field1", "Field3"})
		assert.Equal(t, "NewType", newType.Name)
		assert.Equal(t, 2, len(newType.Fields))
		assert.NotNil(t, newType.GetField("Field1"))
		assert.NotNil(t, newType.GetField("Field3"))
		assert.Nil(t, newType.GetField("Field2"))
	})

	// Test creating a new type from methods
	t.Run("CreateTypeFromMethods", func(t *testing.T) {
		newType := graph.CreateTypeFromMethods("NewType", testType, []string{"Method1", "Method3"})
		assert.Equal(t, "NewType", newType.Name)
		assert.Equal(t, 2, len(newType.Methods))
		assert.NotNil(t, newType.GetMethod("Method1"))
		assert.NotNil(t, newType.GetMethod("Method3"))
		assert.Nil(t, newType.GetMethod("Method2"))
	})

	// Test creating a composite type
	t.Run("CreateCompositeType", func(t *testing.T) {
		// Create another test type
		testType2 := &graph.Type{
			Name:        "TestType2",
			Kind:        reflect.Struct,
			Package:     "test",
			PackagePath: "github.com/test",
			IsExported:  true,
			Fields:      []*graph.Field{},
			Methods:     []*graph.Function{},
		}

		// Add fields to the second test type
		field5 := &graph.Field{
			Name:       "Field5",
			IsExported: true,
			Type: &graph.Type{
				Name:       "string",
				Kind:       reflect.String,
				IsExported: true,
			},
		}
		field6 := &graph.Field{
			Name:       "Field6",
			IsExported: true,
			Type: &graph.Type{
				Name:       "int",
				Kind:       reflect.Int,
				IsExported: true,
			},
		}

		testType2.AddField(field5)
		testType2.AddField(field6)

		// Add methods to the second test type
		method5 := &graph.Function{
			Name:       "Method5",
			IsExported: true,
			Parameters: []*graph.Parameter{},
			Results:    []*graph.Parameter{},
		}
		method6 := &graph.Function{
			Name:       "Method6",
			IsExported: true,
			Parameters: []*graph.Parameter{},
			Results:    []*graph.Parameter{},
		}

		testType2.AddMethod(method5)
		testType2.AddMethod(method6)

		// Create a composite type
		compositeType := graph.CreateCompositeType(
			"CompositeType",
			[]*graph.Type{testType, testType2},
			[][]string{{"Field1", "Field3"}, {"Field5"}},
			[][]string{{"Method1"}, {"Method6"}},
		)

		assert.Equal(t, "CompositeType", compositeType.Name)
		assert.Equal(t, 3, len(compositeType.Fields))
		assert.Equal(t, 2, len(compositeType.Methods))

		// Check fields
		assert.NotNil(t, compositeType.GetField("Field1"))
		assert.NotNil(t, compositeType.GetField("Field3"))
		assert.NotNil(t, compositeType.GetField("Field5"))
		assert.Nil(t, compositeType.GetField("Field2"))
		assert.Nil(t, compositeType.GetField("Field6"))

		// Check methods
		assert.NotNil(t, compositeType.GetMethod("Method1"))
		assert.NotNil(t, compositeType.GetMethod("Method6"))
		assert.Nil(t, compositeType.GetMethod("Method2"))
		assert.Nil(t, compositeType.GetMethod("Method5"))
	})
}
