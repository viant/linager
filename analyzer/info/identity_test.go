package info

import (
	"testing"
)

// TestIdentityRef_Identity tests multiple IdentityRef formats in a data-driven way.
func TestIdentityRef_Identity(t *testing.T) {
	tests := []struct {
		description string
		input       IdentityRef
		expected    Identity
	}{
		{
			description: "Struct field reference",
			input:       "github.com/foo/bar:MyStruct:FieldX",
			expected: Identity{
				Ref:        "github.com/foo/bar:MyStruct:FieldX",
				PkgPath:    "github.com/foo/bar",
				Package:    "bar",
				HolderType: "MyStruct",
				Name:       "FieldX",
			},
		},
		{
			description: "Variable reference with function",
			input:       "main:test.go:[main]:5:foo",
			expected: Identity{
				Ref:      "main:test.go:[main]:5:foo",
				PkgPath:  "main",
				Package:  "main",
				Function: "main",
				Line:     5,
				Name:     "foo",
			},
		},
		{
			description: "Variable reference without function",
			input:       "main:test.go:10:tempVar",
			expected: Identity{
				Ref:      "main:test.go:10:tempVar",
				PkgPath:  "main",
				Package:  "main",
				Function: "",
				Line:     10,
				Name:     "tempVar",
			},
		},
		{
			description: "Edge case: Empty input",
			input:       "",
			expected: Identity{
				Ref: "",
			},
		},
		{
			description: "Malformed reference (too many parts)",
			input:       "main:test.go:extra:[main]:5:foo",
			expected: Identity{
				Ref: "main:test.go:extra:[main]:5:foo",
			},
		},
		{
			description: "Malformed reference (not enough parts)",
			input:       "main:test.go",
			expected: Identity{
				Ref: "main:test.go",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := tc.input.Identity()

			if result.Ref != tc.expected.Ref ||
				result.PkgPath != tc.expected.PkgPath ||
				result.Package != tc.expected.Package ||
				result.HolderType != tc.expected.HolderType ||
				result.Function != tc.expected.Function ||
				result.Line != tc.expected.Line ||
				result.Name != tc.expected.Name {
				t.Errorf("Identity() failed for %s\nGot:  %+v\nWant: %+v",
					tc.input, result, tc.expected)
			}
		})
	}
}
