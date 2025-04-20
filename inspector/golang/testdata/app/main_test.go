package main

import (
	"fmt"
	"myapp/stack"
	"myapp/util"
	"reflect"
	"testing"
)

// TestDynamicTypeManipulation demonstrates the dynamic type manipulation functionality
func TestDynamicTypeManipulation(t *testing.T) {
	// Create a stack of strings
	s := stack.New[string]()
	s.Push("first")
	s.Push("second")

	// Use reflection to examine the stack type
	fmt.Println("Original stack:")
	util.Inspect(s)

	// Get the type of the stack
	stackType := reflect.TypeOf(s).Elem()
	fmt.Printf("\nStack type: %s\n", stackType.Name())

	// Get the fields of the stack type
	fmt.Println("\nFields:")
	for i := 0; i < stackType.NumField(); i++ {
		field := stackType.Field(i)
		fmt.Printf("  %s: %s\n", field.Name, field.Type)
	}

	// Get the methods of the stack type
	fmt.Println("\nMethods:")
	methodType := reflect.TypeOf(s)
	for i := 0; i < methodType.NumMethod(); i++ {
		method := methodType.Method(i)
		fmt.Printf("  %s\n", method.Name)
	}

	// Demonstrate dynamic type creation
	fmt.Println("\nDemonstrating dynamic type creation:")

	// In a real application, we would create a new type with only selected fields and methods
	// For demonstration, we'll just show how to access the fields and methods

	// Access the 'items' field using reflection
	itemsField, _ := stackType.FieldByName("items")
	fmt.Printf("Items field: %s (%s)\n", itemsField.Name, itemsField.Type)

	// Access the 'Push' method using reflection
	pushMethod, _ := methodType.MethodByName("Push")
	fmt.Printf("Push method: %s\n", pushMethod.Name)

	// Access the 'Pop' method using reflection
	popMethod, _ := methodType.MethodByName("Pop")
	fmt.Printf("Pop method: %s\n", popMethod.Name)

	// Access the 'String' method using reflection
	stringMethod, _ := methodType.MethodByName("String")
	fmt.Printf("String method: %s\n", stringMethod.Name)

	// Demonstrate how to use the methods
	fmt.Println("\nDemonstrating method usage:")

	// Use the Push method
	s.Push("third")
	fmt.Printf("After pushing 'third': %s\n", s)

	// Use the Pop method
	value, _ := s.Pop()
	fmt.Printf("Popped value: %s\n", value)
	fmt.Printf("After popping: %s\n", s)

	// Use the String method
	fmt.Printf("String representation: %s\n", s.String())

	fmt.Println("\nDynamic type manipulation demonstration complete")
}
