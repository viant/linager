package analyzer

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/analyzer/linage"
	"testing"
)

func TestJavaAnalyzer_AnalyzeSourceCode(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		project  string
		path     string
		expected []*linage.DataPoint
		wantErr  bool
	}{
		{
			name: "simple class with field and method",
			source: `package com.example;

public class Person {
    private String name;

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }
}`,
			project: "example",
			path:    "Person.java",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "com.example:Person.getName",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Person",
						Name:       "getName",
						Kind:       "method",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Person.java",
						LineNumber: 6,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Person.setName",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Person",
						Name:       "setName",
						Kind:       "method",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Person.java",
						LineNumber: 10,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Person.setName:name",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Person",
						Name:       "name",
						Kind:       "parameter",
						Scope:      "com.example.Person.setName",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Person.java",
						LineNumber: 10,
					},
					Metadata: map[string]interface{}{
						"type": "String",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Person:name",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Person",
						Name:       "name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Person.java",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "String",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "Person.java",
								LineNumber: 11,
							},
							Context: linage.TouchContext{
								Scope: "com.example.Person.setName",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "Person.java",
								LineNumber: 11,
							},
							Context: linage.TouchContext{
								Scope: "com.example.Person.setName",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "class with conditional logic",
			source: `package com.example;

public class Calculator {
    private int value;

    public int calculate(int a, int b, String operation) {
        if (operation.equals("add")) {
            value = a + b;
        } else if (operation.equals("subtract")) {
            value = a - b;
        } else {
            value = 0;
        }
        return value;
    }
}`,
			project: "example",
			path:    "Calculator.java",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "com.example:Calculator.calculate",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Calculator",
						Name:       "calculate",
						Kind:       "method",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Calculator.java",
						LineNumber: 6,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Calculator.calculate:a",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Calculator",
						Name:       "a",
						Kind:       "parameter",
						Scope:      "com.example.Calculator.calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Calculator.java",
						LineNumber: 6,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Calculator.calculate:b",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Calculator",
						Name:       "b",
						Kind:       "parameter",
						Scope:      "com.example.Calculator.calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Calculator.java",
						LineNumber: 6,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Calculator.calculate:operation",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Calculator",
						Name:       "operation",
						Kind:       "parameter",
						Scope:      "com.example.Calculator.calculate",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Calculator.java",
						LineNumber: 6,
					},
					Metadata: map[string]interface{}{
						"type": "String",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "com.example:Calculator:value",
						Module:     "example",
						PkgPath:    "com.example",
						Package:    "com.example",
						ParentType: "Calculator",
						Name:       "value",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Calculator.java",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "int",
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
			a := NewJavaAnalyzer(tt.project)
			got, err := a.AnalyzeSourceCode(tt.source, tt.project, tt.path)

			// Debug output
			t.Logf("[DEBUG_LOG] Got %d data points", len(got))
			t.Logf("[DEBUG_LOG] Test case: %s", tt.name)
			for _, dp := range got {
				t.Logf("[DEBUG_LOG] Data point: %s", dp.Identity.Ref)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("JavaAnalyzer.AnalyzeSourceCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.EqualValues(t, tt.expected, got)
			}
		})
	}
}
