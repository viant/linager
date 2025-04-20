package analyzer_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/analyzer"
	"github.com/viant/linager/analyzer/linage"
	"testing"
)

func TestTreeSitterAnalyzer_AnalyzeSourceCode(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		project  string
		path     string
		expected []*linage.DataPoint
		wantErr  bool
	}{
		{
			name: "function definition and usage",
			source: `package test

func calculateSum(a, b int) int {
	return a + b
}

func main() {
	x := 5
	y := 10
	result := calculateSum(x, y)
}`,
			project: "test",
			path:    "test.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:     "test:calculateSum",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "calculateSum",
						Kind:    "function",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "func(a int, b int) int",
					},
					Writes: []*linage.TouchPoint{},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 10,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:calculateSum:a",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "a",
						Kind:    "parameter",
						Scope:   "test.calculateSum",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 4,
							},
							Context: linage.TouchContext{
								Scope: "test.calculateSum",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:calculateSum:b",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "b",
						Kind:    "parameter",
						Scope:   "test.calculateSum",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 4,
							},
							Context: linage.TouchContext{
								Scope: "test.calculateSum",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:main:x",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "x",
						Kind:    "variable",
						Scope:   "test.main",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 8,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 8,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 10,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:main:y",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "y",
						Kind:    "variable",
						Scope:   "test.main",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 9,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 9,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 10,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:main:result",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "result",
						Kind:    "variable",
						Scope:   "test.main",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 10,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 10,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
		{
			name: "function call",
			source: `package test

func evaluate(f Foo) (bool, error) {
	return f.Score > 10, nil
}

type Foo struct {
	Name  string
	Score int
}

func main() {
	f := Foo{Name: "Test", Score: 20}
	b, err := evaluate(f)
}`,
			project: "test",
			path:    "test.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "test:Foo:Name",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "Foo",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 8,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "test:Foo:Score",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "Foo",
						Name:       "Score",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 9,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:     "test:main:f",
						Module:  "test",
						PkgPath: "test",
						Package: "test",
						Name:    "f",
						Kind:    "variable",
						Scope:   "test.main",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 13,
					},
					Metadata: map[string]interface{}{
						"type": "Foo",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 13,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 14,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "struct field write",
			source: `package test

type User struct {
	Name string
	Age  int
}

func main() {
	u := User{Name: "John", Age: 30}
	u.Name = "Jane"
	u.Age = 25
}`,
			project: "test",
			path:    "test.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "test:User:Name",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "User",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 10,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "test:User:Age",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "User",
						Name:       "Age",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 5,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 11,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
		{
			name: "simple variable declaration",
			source: `package test

func main() {
	x := 10
}`,
			project:  "test",
			path:     "test.go",
			expected: []*linage.DataPoint{},
			wantErr:  false,
		},
		{
			name: "variable assignment",
			source: `package test

func main() {
	x := 10
	x = 20
}`,
			project:  "test",
			path:     "test.go",
			expected: []*linage.DataPoint{},
			wantErr:  false,
		},
		{
			name: "variable read",
			source: `package test

func main() {
	x := 10
	y := x + 5
}`,
			project:  "test",
			path:     "test.go",
			expected: []*linage.DataPoint{},
			wantErr:  false,
		},
		{
			name: "struct field access",
			source: `package test

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{Name: "John", Age: 30}
	name := p.Name
}`,
			project: "test",
			path:    "test.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "test.string.Name",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "string",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "",
					},
				},
				{
					Identity: linage.Identity{
						Ref:        "test.int.Age",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "int",
						Name:       "Age",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 5,
					},
					Metadata: map[string]interface{}{
						"type": "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "complex case",
			source: `package test

			type Foo struct {
			Name  string
			Score int
			}

			type Bar struct {
			Name string
			}

			func main() {
			b := Bar{Name: "XXXX"}

			p := Foo{Name: "John", Score: 30}
			if p.Score > 18 {
			p.Name = "Adult: " + b.Name
			}
			}`,
			project: "test",
			path:    "test.go",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:        "test:Foo:Name",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "Foo",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 17,
							},
							Context: linage.TouchContext{
								Scope: "test.main.if_15_3",
							},
						},
					},
					Reads: []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "test:Foo:Score",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "Foo",
						Name:       "Score",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 5,
					},
					Metadata: map[string]interface{}{
						"type": "int",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 15,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 16,
							},
							Context: linage.TouchContext{
								Scope: "test.main.if_15_3",
							},
						},
					},
				},
				{
					Identity: linage.Identity{
						Ref:        "test:Bar:Name",
						Module:     "test",
						PkgPath:    "test",
						Package:    "test",
						ParentType: "Bar",
						Name:       "Name",
						Kind:       "field",
					},
					Definition: linage.CodeLocation{
						FilePath:   "test.go",
						LineNumber: 9,
					},
					Metadata: map[string]interface{}{
						"type": "string",
					},
					Writes: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 13,
							},
							Context: linage.TouchContext{
								Scope: "test.main",
							},
						},
					},
					Reads: []*linage.TouchPoint{
						{
							CodeLocation: linage.CodeLocation{
								FilePath:   "test.go",
								LineNumber: 17,
							},
							Context: linage.TouchContext{
								Scope: "test.main.if_15_3",
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
			a := analyzer.NewTreeSitterAnalyzer(tt.project)
			got, err := a.AnalyzeSourceCode(tt.source, tt.project, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Analyzer.AnalyzeSourceCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.expected, got)
			// Basic validation of the results

		})
	}
}
