package golang_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/inspector/golang"
	graph "github.com/viant/linager/inspector/graph"
	"go/parser"
	"go/token"
	"testing"
)

func TestInspectConstantsAndVariables(t *testing.T) {
	// Test source code
	src := `package test

// APIVersion represents the current API version
const APIVersion = "v1.0.0"

// These constants define error codes
const (
        // ErrNotFound indicates resource not found
        ErrNotFound = 404
        // ErrUnauthorized indicates missing or invalid authentication
        ErrUnauthorized = 401
)

// MaxRetries defines the maximum number of retry attempts
var MaxRetries = 3

// These variables are configuration defaults
var (
        // DefaultTimeout is the default operation timeout in seconds
        DefaultTimeout = 30

        // DefaultHeaders contains standard HTTP headers
        DefaultHeaders = map[string]string{
                "Content-Type": "application/json",
                "Accept":       "application/json",
        }
)`

	// Parse the source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Create inspector
	inspector := golang.NewInspector(&graph.Config{IncludeUnexported: true})

	// Test cases for constants
	t.Run("test_constants", func(t *testing.T) {
		// Define test cases
		tests := []struct {
			name     string
			expected []*graph.Constant
		}{
			{
				name: "inspect constants",
				expected: []*graph.Constant{
					{
						Name:    "APIVersion",
						Value:   "\"v1.0.0\"",
						Comment: "APIVersion represents the current API version",
					},
					{
						Name:    "ErrNotFound",
						Value:   "404",
						Comment: "ErrNotFound indicates resource not found",
					},
					{
						Name:    "ErrUnauthorized",
						Value:   "401",
						Comment: "ErrUnauthorized indicates missing or invalid authentication",
					},
				},
			},
		}

		// Run test cases
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Get actual constants
				actual, err := inspector.InspectConstants(file, map[string]string{})
				assert.NoError(t, err, "Failed to inspect constants")

				// Check number of constants
				assert.Equal(t, len(tt.expected), len(actual), "Expected %d constants, got %d", len(tt.expected), len(actual))

				// Create a map of actual constants by name for easier comparison
				actualMap := make(map[string]*graph.Constant)
				for _, c := range actual {
					actualMap[c.Name] = c
				}

				// Compare each expected constant with actual
				for _, expected := range tt.expected {
					actual, exists := actualMap[expected.Name]
					assert.True(t, exists, "Expected constant %s not found", expected.Name)
					if exists {
						assert.Equal(t, expected.Value, actual.Value, "Constant %s: expected value %q, got %q", expected.Name, expected.Value, actual.Value)
						assert.Contains(t, actual.Comment, expected.Comment, "Missing or incorrect comment for %s", expected.Name)
					}
				}
			})
		}
	})

	// Test cases for variables
	t.Run("test_variables", func(t *testing.T) {
		// Define test cases
		tests := []struct {
			name     string
			expected []*graph.Variable
		}{
			{
				name: "inspect variables",
				expected: []*graph.Variable{
					{
						Name:    "MaxRetries",
						Value:   "3",
						Comment: "MaxRetries defines the maximum number of retry attempts",
					},
					{
						Name:    "DefaultTimeout",
						Value:   "30",
						Comment: "DefaultTimeout is the default operation timeout in seconds",
					},
					{
						Name:    "DefaultHeaders",
						Value:   "<unhandled *ast.MapType>{\"Content-Type\": \"application/json\", \"Accept\": \"application/json\"}",
						Comment: "DefaultHeaders contains standard HTTP headers",
					},
				},
			},
		}

		// Run test cases
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Get actual variables
				actual, err := inspector.InspectVariables(file, map[string]string{})
				assert.NoError(t, err, "Failed to inspect variables")

				// Check number of variables
				assert.Equal(t, len(tt.expected), len(actual), "Expected %d variables, got %d", len(tt.expected), len(actual))

				// Create a map of actual variables by name for easier comparison
				actualMap := make(map[string]*graph.Variable)
				for _, v := range actual {
					actualMap[v.Name] = v
				}

				// Compare each expected variable with actual
				for _, expected := range tt.expected {
					actual, exists := actualMap[expected.Name]
					assert.True(t, exists, "Expected variable %s not found", expected.Name)
					if exists {
						assert.Equal(t, expected.Value, actual.Value, "Variable %s: expected value %q, got %q", expected.Name, expected.Value, actual.Value)
						assert.Contains(t, actual.Comment, expected.Comment, "Missing or incorrect comment for %s", expected.Name)

						// Check type for variables that should have type information
						if expected.Type != nil {
							assert.NotNil(t, actual.Type, "Variable %s should have type information", expected.Name)
							if actual.Type != nil {
								assert.Equal(t, expected.Type.Kind, actual.Type.Kind, "Variable %s: expected type kind %v, got %v", expected.Name, expected.Type.Kind, actual.Type.Kind)
							}
						}
					}
				}
			})
		}
	})
}
