package golang

import (
	"fmt"
	"github.com/viant/linager/inspector/graph"
	"strings"
)

type Emitter struct{}

func (g *Emitter) Emit(file *graph.File) ([]byte, error) {
	// Start with package declaration and imports
	builder := &strings.Builder{}
	builder.WriteString(fmt.Sprintf("package %s\n\n", file.Package))

	// Add imports if any
	if len(file.Imports) > 0 {
		builder.WriteString("import (\n")
		for _, imp := range file.Imports {
			if imp.Name != "" {
				builder.WriteString(fmt.Sprintf("\t%s %q\n", imp.Name, imp.Path))
			} else {
				builder.WriteString(fmt.Sprintf("\t%q\n", imp.Path))
			}
		}
		builder.WriteString(")\n\n")
	}

	// Add constants if any
	for _, constant := range file.Constants {
		if constant.Location != nil && constant.Location.Raw != "" {
			builder.WriteString(constant.Location.Raw)
			builder.WriteString("\n\n")
		}
	}

	// Add variables if any
	for _, variable := range file.Variables {
		if variable.Location != nil && variable.Location.Raw != "" {
			builder.WriteString(variable.Location.Raw)
			builder.WriteString("\n\n")
		}
	}

	// Add types if any
	for _, typ := range file.Types {
		if typ.Location != nil && typ.Location.Raw != "" {
			// Use the Type.Content() method which includes fields
			builder.WriteString(typ.Content())
			builder.WriteString("\n\n")
		}
	}

	// Add functions if any
	for _, function := range file.Functions {
		if function.Location != nil && function.Location.Raw != "" {
			builder.WriteString(function.Location.Raw)
			builder.WriteString("\n\n")
		}
	}

	return []byte(builder.String()), nil
}
