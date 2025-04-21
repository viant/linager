package analyzer

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/analyzer/linage"
	"testing"
)

func TestGolangAnalyzer_AnalyzeSourceCode(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		project  string
		path     string
		expected []*linage.DataPoint
		wantErr  bool
	}{
		{
			name: "simple struct with field and method",
			source: `package example

type Person struct {
	Name string
}

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) SetName(name string) {
	p.Name = name
}`,
			project: "example",
			path:    "person.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "example:Person.GetName",
						Module:     "example",
						PkgPath:    "example",
						Package:    "example",
						ParentType: "Person",
						Name:       "GetName",
						Kind:       "method",
					},
					Definition: linage.CodeLocation{
						FilePath:   "person.go",
						LineNumber: 7,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "example:Person.SetName",
						Module:     "example",
						PkgPath:    "example",
						Package:    "example",
						ParentType: "Person",
						Name:       "SetName",
						Kind:       "method",
					},
					Definition: linage.CodeLocation{
						FilePath:   "person.go",
						LineNumber: 11,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "example:Person.SetName:name",
						Module:     "example",
						PkgPath:    "example",
						Package:    "example",
						ParentType: "Person",
						Name:       "name",
						Kind:       "parameter",
						Scope:      "example:Person.SetName",
					},
					Definition: linage.CodeLocation{
						FilePath:   "person.go",
						LineNumber: 11,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "example:Person:Name",
						Module:     "example",
						PkgPath:    "example",
						Package:    "example",
						ParentType: "Person",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "person.go",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "person.go",
								LineNumber: 12,
							},
							Context: linage.TouchContext{
								Scope: "example.Person.SetName",
							},
							Dependencies:          []linage.IdentityRef{},
							ConditionalExpression: "p.Name",
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "person.go",
								LineNumber: 8,
							},
							Context: linage.TouchContext{
								Scope: "example.Person.GetName",
							},
							Dependencies:          []linage.IdentityRef{},
							ConditionalExpression: "p.Name",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function with conditional logic",
			source: `package example

func Calculate(a, b int, operation string) int {
	var result int
	if operation == "add" {
		result = a + b
	} else if operation == "subtract" {
		result = a - b
	} else {
		result = 0
	}
	return result
}`,
			project: "example",
			path:    "calculator.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:     "example:Calculate",
						Module:  "example",
						PkgPath: "example",
						Package: "example",
						Name:    "Calculate",
						Kind:    "function",
					},
					Definition: linage.CodeLocation{
						FilePath:   "calculator.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:     "example:Calculate:a",
						Module:  "example",
						PkgPath: "example",
						Package: "example",
						Name:    "a",
						Kind:    "parameter",
						Scope:   "example:Calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "calculator.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:     "example:Calculate:b",
						Module:  "example",
						PkgPath: "example",
						Package: "example",
						Name:    "b",
						Kind:    "parameter",
						Scope:   "example:Calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "calculator.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:     "example:Calculate:operation",
						Module:  "example",
						PkgPath: "example",
						Package: "example",
						Name:    "operation",
						Kind:    "parameter",
						Scope:   "example:Calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "calculator.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewGolangAnalyzer(tt.project)
			got, err := a.AnalyzeSourceCode(tt.source, tt.project, tt.path)

			// Debug output
			t.Logf("[DEBUG_LOG] Got %d data points", len(got))
			t.Logf("[DEBUG_LOG] Test case: %s", tt.name)
			for _, dp := range got {
				t.Logf("[DEBUG_LOG] Data point: %s", dp.Identity.Ref)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("GolangAnalyzer.AnalyzeSourceCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.EqualValues(t, tt.expected, got)
			}
		})
	}
}
