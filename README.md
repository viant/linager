# Linager - Go Code Lineage Analysis Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/viant/linager)](https://goreportcard.com/report/github.com/viant/linager)
[![GoDoc](https://godoc.org/github.com/viant/linager?status.svg)](https://godoc.org/github.com/viant/linager)

Linager is a sophisticated static analysis tool for Go that tracks data lineage, dependencies, and usage patterns within your codebase. It provides detailed insights about how data flows through your application, making it easier to understand, refactor, and maintain complex code.

## Features

- **Data Flow Analysis**: Track where variables and fields are defined, read, and written
- **Dependency Tracking**: Identify data dependencies between variables and functions
- **Conditional Execution**: Capture conditions under which data is accessed or modified
- **Struct Field Analysis**: Detailed tracking of struct fields and their usage
- **Generic Types Support**: Full analysis of generic type parameters and instantiations
- **Call Graph Analysis**: Understand transitive dependencies through function calls
- **Rich Metadata**: Capture additional context like struct tags and parameter info

## Installation

```bash
go get github.com/viant/linager
```

## Usage

### Basic Example

```go
package main

import (
	"fmt"
	"github.com/viant/linager/analyzer"
	"gopkg.in/yaml.v3"
)

func main() {
	code := `
package main

type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

func main() {
	user := User{ID: 1, Name: "John"}
	id := user.ID
	fmt.Println(id)
}
`

	dataPoints, err := analyzer.AnalyzeSourceCode(code, "", "example.go")
	if err != nil {
		fmt.Printf("Analysis error: %v\n", err)
		return
	}

	data, _ := yaml.Marshal(dataPoints)
	fmt.Println(string(data))
}
```

### Analyzing Project Files

```go
package main

import (
	"fmt"
	"github.com/viant/linager/analyzer"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	// Load and analyze an entire project
	pkgs, err := analyzer.LoadProject("path/to/project")
	if err != nil {
		fmt.Printf("Error loading project: %v\n", err)
		return
	}

	dataPoints, err := analyzer.AnalyzePackages(pkgs)
	if err != nil {
		fmt.Printf("Analysis error: %v\n", err)
		return
	}

	data, _ := yaml.Marshal(dataPoints)
	os.WriteFile("analysis.yaml", data, 0644)
}
```

## Data Model

Linager uses a structured data model to represent the results of its analysis:

### DataPoint

A `DataPoint` represents information about an identifier (variable, struct field, function):

```go
type DataPoint struct {
	Identity   Identity               // Identity information
	Definition CodeLocation           // Where the identifier is defined
	Metadata   map[string]interface{} // Additional metadata
	Writes     []*TouchPoint          // Where the identifier is written
	Reads      []*TouchPoint          // Where the identifier is read
}
```

### TouchPoint

A `TouchPoint` represents a location where data is accessed or modified:

```go
type TouchPoint struct {
	CodeLocation          CodeLocation  // Location in code
	Context               TouchContext  // Context information
	Dependencies          []IdentityRef // Dependencies for this touch point
	ConditionalExpression string        // Condition under which this happens
}
```

## Example Output

For the following code:

```go
package main

type Person struct {
	Name string
	Age  int
}

func main() {
	p := Person{Name: "John", Age: 30}
	if p.Age > 18 {
		p.Name = "Adult: " + p.Name
	}
}
```

Linager produces detailed analysis like:

```yaml
- identity:
    ref: main:Person:Name
    pkgPath: main
    package: main
    holderType: Person
    name: Name
    kind: string
  definition:
    filePath: example.go
    lineNumber: 4
  writes:
    - codeLocation:
        filePath: example.go
        lineNumber: 9
        columnStart: 6
        columnEnd: 28
      context:
        function: main
    - codeLocation:
        filePath: example.go
        lineNumber: 11
        columnStart: 3
        columnEnd: 9
      context:
        function: main
      conditionalExpression: p.Age > 18
  reads:
    - codeLocation:
        filePath: example.go
        lineNumber: 11
        columnStart: 22
        columnEnd: 28
      context:
        function: main
      conditionalExpression: p.Age > 18
```

## Advanced Features

### Generic Type Analysis

Linager provides detailed analysis of generic types:

```go
type Container[T any] struct {
    Value T
}

func Process[T comparable](c Container[T]) T {
    return c.Value
}

func main() {
    intContainer := Container[int]{Value: 42}
    intValue := Process(intContainer)
}
```

### Struct Tag Analysis

Linager extracts and analyzes struct tags:

```go
type User struct {
    ID        int       `json:"id" db:"user_id"`
    Email     string    `json:"email" validate:"required,email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

## Contributing

Contributions to Linager are welcome! Please feel free to submit a Pull Request.

## License


The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

## Credits

Developed and maintained by [Viant](https://github.com/viant).

