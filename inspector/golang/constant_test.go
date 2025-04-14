package golang_test

import (
	"github.com/viant/linager/inspector/golang"
	graph "github.com/viant/linager/inspector/graph"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestInspectConstantsAndVariables(t *testing.T) {
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

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	inspector := golang.NewInspector(&graph.Config{IncludeUnexported: true})

	// Test constants
	t.Run("test_constants", func(t *testing.T) {
		constants, err := inspector.InspectConstants(file, map[string]string{})
		if err != nil {
			t.Fatalf("Failed to inspect constants: %v", err)
		}

		if len(constants) != 3 {
			t.Errorf("Expected 3 constants, got %d", len(constants))
		}

		expectedConstants := map[string]string{
			"APIVersion":      "\"v1.0.0\"",
			"ErrNotFound":     "404",
			"ErrUnauthorized": "401",
		}

		for _, c := range constants {
			expectedValue, exists := expectedConstants[c.Name]
			if !exists {
				t.Errorf("Unexpected constant found: %s", c.Name)
				continue
			}

			if c.Value != expectedValue {
				t.Errorf("Constant %s: expected value %q, got %q", c.Name, expectedValue, c.Value)
			}

			// Check comments
			if c.Name == "APIVersion" && !contains(c.Comment, "represents the current API") {
				t.Errorf("Missing or incorrect comment for APIVersion: %q", c.Comment)
			}

			if c.Name == "ErrNotFound" && !contains(c.Comment, "indicates resource not found") {
				t.Errorf("Missing or incorrect comment for ErrNotFound: %q", c.Comment)
			}
		}
	})

	// Test variables
	t.Run("test_variables", func(t *testing.T) {
		variables, err := inspector.InspectVariables(file, map[string]string{})
		if err != nil {
			t.Fatalf("Failed to inspect variables: %v", err)
		}

		if len(variables) != 3 {
			t.Errorf("Expected 3 variables, got %d", len(variables))
		}

		for _, v := range variables {
			switch v.Name {
			case "MaxRetries":
				if v.Value != "3" {
					t.Errorf("Variable MaxRetries: expected value '3', got %q", v.Value)
				}
				if !contains(v.Comment, "maximum number of retry") {
					t.Errorf("Missing or incorrect comment for MaxRetries: %q", v.Comment)
				}

			case "DefaultTimeout":
				if v.Value != "30" {
					t.Errorf("Variable DefaultTimeout: expected value '30', got %q", v.Value)
				}
				if !contains(v.Comment, "default operation timeout") {
					t.Errorf("Missing or incorrect comment for DefaultTimeout: %q", v.Comment)
				}

			case "DefaultHeaders":
				if !contains(v.Value, "map[string]string") {
					t.Errorf("Variable DefaultHeaders: expected map type, got %q", v.Value)
				}
				if !contains(v.Comment, "standard HTTP headers") {
					t.Errorf("Missing or incorrect comment for DefaultHeaders: %q", v.Comment)
				}
				if v.Type == nil {
					t.Error("DefaultHeaders should have type information")
				} else if v.Type.Kind != reflect.Map {
					t.Errorf("DefaultHeaders should be a map, got %v", v.Type.Kind)
				}

			default:
				t.Errorf("Unexpected variable found: %s", v.Name)
			}
		}
	})
}

// contains checks if a substring is present in a string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
