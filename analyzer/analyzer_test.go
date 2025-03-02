package analyzer

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestAnalyzeSourceCode(t *testing.T) {
	tests := []testCase{
		{
			description: "Basic data linager tracking",
			path:        "example.go",
			code: `
package main

import "math/rand"

type Foo struct {
	Id   int
	Name string
}

type Bar struct {
	Id int
}

func main() {
	foo := Foo{Id: 1, Name: "test"}
	bar := Bar{}
	x := rand.Intn(100)
	id := 0
	if x > 10 {
		id = foo.Id
	}
	if x > 20 {
		bar.Id = id
	}
}
`,
			expectYaml: `- identity:
    ref: main:Foo:Id
    pkgPath: main
    package: main
    holderType: Foo
    name: Id
    kind: int
  definition:
    filePath: example.go
    lineNumber: 7
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 16
        columnStart: 9
        columnEnd: 33
      context:
        function: main
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 21
        columnStart: 8
        columnEnd: 14
      context:
        function: main
      conditionalExpression: x > 10
- identity:
    ref: main:Foo:Name
    pkgPath: main
    package: main
    holderType: Foo
    name: Name
    kind: string
  definition:
    filePath: example.go
    lineNumber: 8
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 16
        columnStart: 9
        columnEnd: 33
      context:
        function: main
- identity:
    ref: main:Bar:Id
    pkgPath: main
    package: main
    holderType: Bar
    name: Id
    kind: int
  definition:
    filePath: example.go
    lineNumber: 12
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 24
        columnStart: 3
        columnEnd: 9
      context:
        function: main
      conditionalExpression: x > 20
- identity:
    ref: foo
    name: foo
    kind: main.Foo
  definition:
    filePath: example.go
    lineNumber: 16
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 16
        columnStart: 2
        columnEnd: 5
      context:
        function: main
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 21
        columnStart: 8
        columnEnd: 11
      context:
        function: main
      conditionalExpression: x > 10
- identity:
    ref: bar
    name: bar
    kind: main.Bar
  definition:
    filePath: example.go
    lineNumber: 17
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 17
        columnStart: 2
        columnEnd: 5
      context:
        function: main
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 24
        columnStart: 3
        columnEnd: 6
      context:
        function: main
      conditionalExpression: x > 20
- identity:
    ref: x
    name: x
    kind: int
  definition:
    filePath: example.go
    lineNumber: 18
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 18
        columnStart: 2
        columnEnd: 3
      context:
        function: main
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 20
        columnStart: 5
        columnEnd: 6
      context:
        function: main
      conditionalExpression: x > 10
    - codeLocation:
        filePath: example.go
        lineNumber: 23
        columnStart: 5
        columnEnd: 6
      context:
        function: main
      conditionalExpression: x > 20
- identity:
    ref: math/rand.Intn
    pkgPath: math/rand
    package: math/rand
    holderType: math/rand
    name: Intn
    kind: func(n int) int
  definition:
    filePath: example.go
    lineNumber: 18
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 18
        columnStart: 7
        columnEnd: 16
      context:
        function: main
- identity:
    ref: id
    name: id
    kind: int
  definition:
    filePath: example.go
    lineNumber: 19
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 19
        columnStart: 2
        columnEnd: 4
      context:
        function: main
      dependencies:
        - main:Foo:Id
        - foo
    - codeLocation:
        filePath: example.go
        lineNumber: 21
        columnStart: 3
        columnEnd: 5
      context:
        function: main
      dependencies:
        - main:Foo:Id
        - foo
      conditionalExpression: x > 10
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 24
        columnStart: 12
        columnEnd: 14
      context:
        function: main
      conditionalExpression: x > 20`,
		},
		{
			description: "Struct tags analysis",
			path:        "struct_tags.go",
			code: `
package main

import "time"

type User struct {
	ID        int       ` + "`json:\"id\" db:\"user_id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	Email     string    ` + "`json:\"email\" validate:\"required,email\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
}

func main() {
	user := User{
		ID:   1,
		Name: "John Doe",
		Email: "john@example.com",
	}

	// Access a field
	id := user.ID
	_ = id // Use id to avoid "declared and not used" error

	// Modify a field
	user.Name = "Jane Doe"
}
`,
			expectYaml: `
- identity:
    ref: main:User:ID
    pkgPath: main
    package: main
    holderType: User
    name: ID
    kind: int
  definition:
    filePath: struct_tags.go
    lineNumber: 7
  metadata:
    tags: json:"id" db:"user_id"
  writes:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 14
        columnStart: 10
        columnEnd: 72
      context:
        function: main
  reads:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 21
        columnStart: 8
        columnEnd: 15
      context:
        function: main
- identity:
    ref: main:User:Name
    pkgPath: main
    package: main
    holderType: User
    name: Name
    kind: string
  definition:
    filePath: struct_tags.go
    lineNumber: 8
  metadata:
    tags: json:"name"
  writes:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 14
        columnStart: 10
        columnEnd: 72
      context:
        function: main
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 25
        columnStart: 2
        columnEnd: 11
      context:
        function: main
- identity:
    ref: main:User:Email
    pkgPath: main
    package: main
    holderType: User
    name: Email
    kind: string
  definition:
    filePath: struct_tags.go
    lineNumber: 9
  metadata:
    tags: json:"email" validate:"required,email"
  writes:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 14
        columnStart: 10
        columnEnd: 72
      context:
        function: main
- identity:
    ref: main:User:CreatedAt
    pkgPath: main
    package: main
    holderType: User
    name: CreatedAt
    kind: time.Time
  definition:
    filePath: struct_tags.go
    lineNumber: 10
  metadata:
    tags: json:"created_at" db:"created_at"
- identity:
    ref: user
    name: user
    kind: main.User
  definition:
    filePath: struct_tags.go
    lineNumber: 14
  writes:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 14
        columnStart: 2
        columnEnd: 6
      context:
        function: main
  reads:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 21
        columnStart: 8
        columnEnd: 12
      context:
        function: main
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 25
        columnStart: 2
        columnEnd: 6
      context:
        function: main
- identity:
    ref: id
    name: id
    kind: int
  definition:
    filePath: struct_tags.go
    lineNumber: 21
  writes:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 21
        columnStart: 2
        columnEnd: 4
      context:
        function: main
      dependencies:
        - main:User:ID
        - user
  reads:
    - codeLocation:
        filePath: struct_tags.go
        lineNumber: 22
        columnStart: 6
        columnEnd: 8
      context:
        function: main
`,
		},
		{
			description: "Generics analysis",
			path:        "generics.go",
			code: `
package main

// Generic container type
type Container[T any] struct {
	Value T
}

// Generic function
func Process[T comparable](c Container[T]) T {
	return c.Value
}

func main() {
	// Integer container
	intContainer := Container[int]{Value: 42}
	intValue := Process(intContainer)
	_ = intValue // Use intValue to avoid "declared and not used" error

	// String container
	strContainer := Container[string]{Value: "hello"}
	strValue := Process(strContainer)
	_ = strValue // Use strValue to avoid "declared and not used" error
}
`,
			expectYaml: `- identity:
    ref: main:Container[T]:Value
    pkgPath: main
    package: main
    holderType: Container
    name: Value
    kind: TypeParam(T)
  definition:
    filePath: generics.go
    lineNumber: 6
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 11
        columnStart: 9
        columnEnd: 16
      context:
        function: Process
- identity:
    ref: c
    name: c
    kind: Container[T]
  definition:
    filePath: generics.go
    lineNumber: 10
  metadata:
    parameter: true
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 10
        columnStart: 28
        columnEnd: 29
      context:
        function: Process
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 10
        columnStart: 28
        columnEnd: 29
      context:
        function: Process
    - codeLocation:
        filePath: generics.go
        lineNumber: 11
        columnStart: 9
        columnEnd: 10
      context:
        function: Process
- identity:
    ref: intContainer
    name: intContainer
    kind: main.Container[int]
  definition:
    filePath: generics.go
    lineNumber: 16
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 16
        columnStart: 2
        columnEnd: 14
      context:
        function: main
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 17
        columnStart: 22
        columnEnd: 34
      context:
        function: main
- identity:
    ref: main:Container[int]:Value
    holderType: Container[int]
    name: Value
    kind: int
  definition:
    filePath: generics.go
    lineNumber: 16
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 16
        columnStart: 18
        columnEnd: 43
      context:
        function: main
- identity:
    ref: intValue
    name: intValue
    kind: int
  definition:
    filePath: generics.go
    lineNumber: 17
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 17
        columnStart: 2
        columnEnd: 10
      context:
        function: main
      dependencies:
        - Process
        - intContainer
        - main:Container[T]:Value
        - c
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 18
        columnStart: 6
        columnEnd: 14
      context:
        function: main
- identity:
    ref: Process
    name: Process
    kind: func(c main.Container[int]) int
  definition:
    filePath: generics.go
    lineNumber: 17
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 17
        columnStart: 14
        columnEnd: 21
      context:
        function: main
    - codeLocation:
        filePath: generics.go
        lineNumber: 22
        columnStart: 14
        columnEnd: 21
      context:
        function: main
- identity:
    ref: strContainer
    name: strContainer
    kind: main.Container[string]
  definition:
    filePath: generics.go
    lineNumber: 21
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 21
        columnStart: 2
        columnEnd: 14
      context:
        function: main
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 22
        columnStart: 22
        columnEnd: 34
      context:
        function: main
- identity:
    ref: main:Container[string]:Value
    holderType: Container[string]
    name: Value
    kind: string
  definition:
    filePath: generics.go
    lineNumber: 21
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 21
        columnStart: 18
        columnEnd: 51
      context:
        function: main
- identity:
    ref: strValue
    name: strValue
    kind: string
  definition:
    filePath: generics.go
    lineNumber: 22
  writes:
    - codeLocation:
        filePath: generics.go
        lineNumber: 22
        columnStart: 2
        columnEnd: 10
      context:
        function: main
      dependencies:
        - Process
        - strContainer
        - main:Container[T]:Value
        - c
  reads:
    - codeLocation:
        filePath: generics.go
        lineNumber: 23
        columnStart: 6
        columnEnd: 14
      context:
        function: main`,
		},
		{
			description: "Complex data flow",
			path:        "complex_flow.go",
			code: `
package main

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street  string
	City    string
	ZipCode string
}

func main() {
	p := Person{
		Name: "John",
		Age:  30,
		Address: Address{
			Street:  "123 Main St",
			City:    "Anytown",
			ZipCode: "12345",
		},
	}

	// Direct field access
	name := p.Name

	// Nested field access
	city := p.Address.City

	// Conditional modification
	if p.Age > 18 {
		p.Address.ZipCode = "54321"
	}

	// Using fields in expressions
	adultYears := p.Age - 18
	if adultYears > 0 {
		p.Name = p.Name + " (Adult)"
	}

	// Use variables to avoid "declared and not used" errors
	_ = name
	_ = city
	_ = adultYears
}
`,
			expectYaml: `- identity:
    ref: main:Person:Name
    pkgPath: main
    package: main
    holderType: Person
    name: Name
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 5
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 17
        columnStart: 7
        columnEnd: 130
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 41
        columnStart: 3
        columnEnd: 9
      context:
        function: main
      conditionalExpression: adultYears > 0
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 28
        columnStart: 10
        columnEnd: 16
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 41
        columnStart: 12
        columnEnd: 18
      context:
        function: main
      conditionalExpression: adultYears > 0
- identity:
    ref: main:Person:Age
    pkgPath: main
    package: main
    holderType: Person
    name: Age
    kind: int
  definition:
    filePath: complex_flow.go
    lineNumber: 6
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 17
        columnStart: 7
        columnEnd: 130
      context:
        function: main
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 34
        columnStart: 5
        columnEnd: 10
      context:
        function: main
      conditionalExpression: p.Age > 18
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 39
        columnStart: 16
        columnEnd: 21
      context:
        function: main
- identity:
    ref: main:Person:Address
    pkgPath: main
    package: main
    holderType: Person
    name: Address
    kind: Address
  definition:
    filePath: complex_flow.go
    lineNumber: 7
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 17
        columnStart: 7
        columnEnd: 130
      context:
        function: main
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 31
        columnStart: 10
        columnEnd: 19
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 35
        columnStart: 3
        columnEnd: 12
      context:
        function: main
      conditionalExpression: p.Age > 18
- identity:
    ref: main:Address:Street
    pkgPath: main
    package: main
    holderType: Address
    name: Street
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 11
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 20
        columnStart: 12
        columnEnd: 85
      context:
        function: main
- identity:
    ref: main:Address:City
    pkgPath: main
    package: main
    holderType: Address
    name: City
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 12
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 20
        columnStart: 12
        columnEnd: 85
      context:
        function: main
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 31
        columnStart: 10
        columnEnd: 24
      context:
        function: main
- identity:
    ref: main:Address:ZipCode
    pkgPath: main
    package: main
    holderType: Address
    name: ZipCode
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 13
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 20
        columnStart: 12
        columnEnd: 85
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 35
        columnStart: 3
        columnEnd: 20
      context:
        function: main
      conditionalExpression: p.Age > 18
- identity:
    ref: p
    name: p
    kind: main.Person
  definition:
    filePath: complex_flow.go
    lineNumber: 17
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 17
        columnStart: 2
        columnEnd: 3
      context:
        function: main
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 28
        columnStart: 10
        columnEnd: 11
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 31
        columnStart: 10
        columnEnd: 11
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 34
        columnStart: 5
        columnEnd: 6
      context:
        function: main
      conditionalExpression: p.Age > 18
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 35
        columnStart: 3
        columnEnd: 4
      context:
        function: main
      conditionalExpression: p.Age > 18
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 39
        columnStart: 16
        columnEnd: 17
      context:
        function: main
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 41
        columnStart: 3
        columnEnd: 4
      context:
        function: main
      conditionalExpression: adultYears > 0
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 41
        columnStart: 12
        columnEnd: 13
      context:
        function: main
      conditionalExpression: adultYears > 0
- identity:
    ref: name
    name: name
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 28
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 28
        columnStart: 2
        columnEnd: 6
      context:
        function: main
      dependencies:
        - main:Person:Name
        - p
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 45
        columnStart: 6
        columnEnd: 10
      context:
        function: main
- identity:
    ref: city
    name: city
    kind: string
  definition:
    filePath: complex_flow.go
    lineNumber: 31
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 31
        columnStart: 2
        columnEnd: 6
      context:
        function: main
      dependencies:
        - main:Address:City
        - main:Person:Address
        - p
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 46
        columnStart: 6
        columnEnd: 10
      context:
        function: main
- identity:
    ref: adultYears
    name: adultYears
    kind: int
  definition:
    filePath: complex_flow.go
    lineNumber: 39
  writes:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 39
        columnStart: 2
        columnEnd: 12
      context:
        function: main
      dependencies:
        - main:Person:Age
        - p
  reads:
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 40
        columnStart: 5
        columnEnd: 15
      context:
        function: main
      conditionalExpression: adultYears > 0
    - codeLocation:
        filePath: complex_flow.go
        lineNumber: 47
        columnStart: 6
        columnEnd: 16
      context:
        function: main`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			dataPoints, err := AnalyzeSourceCode(tc.code, "", tc.path)
			assert.NoError(t, err)
			if !assert.True(t, tc.expectYaml != "") {
				return
			}

			if tc.expectYaml != "" {
				if err = yaml.Unmarshal([]byte(tc.expectYaml), &tc.expect); !assert.Nil(t, err) {
					return
				}
			}
			adjustNils(tc.expect)
			adjustNils(dataPoints)
			if !assert.EqualValues(t, tc.expect, dataPoints) {
				data, _ := yaml.Marshal(dataPoints)
				fmt.Println("ACTUAL:", string(data))
			}
		})
	}
}

type testCase struct {
	description string
	code        string
	path        string
	expectYaml  string
	expect      []*linager.DataPoint
}

func TestIdentityRef_MakeStructFieldIdentityRef(t *testing.T) {
	tests := []struct {
		description string
		project     string
		pkgPath     string
		structType  string
		fieldName   string
		expected    linager.IdentityRef
	}{
		{
			description: "Simple struct field reference",
			project:     "",
			pkgPath:     "github.com/foo/bar",
			structType:  "MyStruct",
			fieldName:   "FieldX",
			expected:    "github.com/foo/bar:MyStruct:FieldX",
		},

		{
			description: "Main package struct field",
			project:     "",
			pkgPath:     "main",
			structType:  "User",
			fieldName:   "ID",
			expected:    "main:User:ID",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := linager.MakeStructFieldIdentityRef(tc.pkgPath, tc.structType, tc.fieldName)
			assert.EqualValues(t, tc.expected, result)
		})
	}
}

func adjustNils(expect []*linager.DataPoint) {
	for _, item := range expect {
		if len(item.Metadata) == 0 {
			item.Metadata = nil
		}
		if len(item.Writes) == 0 {
			item.Writes = nil
		}
		if len(item.Reads) == 0 {
			item.Reads = nil
		}
	}
}
