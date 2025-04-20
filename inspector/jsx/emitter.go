package jsx

import (
	"fmt"
	"github.com/viant/linager/inspector/graph"
	"strings"
)

// Emitter is responsible for converting graph representation back to JSX source code
type Emitter struct{}

// Emit converts a graph.File to JSX source code
func (e *Emitter) Emit(file *graph.File) ([]byte, error) {
	// Start with imports
	builder := &strings.Builder{}

	// Add imports if any
	if len(file.Imports) > 0 {
		for _, imp := range file.Imports {
			if imp.Name != "" {
				builder.WriteString(fmt.Sprintf("import %s from '%s';\n", imp.Name, imp.Path))
			} else {
				builder.WriteString(fmt.Sprintf("import '%s';\n", imp.Path))
			}
		}
		builder.WriteString("\n")
	}

	// Add constants if any
	for _, constant := range file.Constants {
		if constant.Location != nil && constant.Location.Raw != "" {
			builder.WriteString(constant.Location.Raw)
			builder.WriteString("\n\n")
		} else {
			// Fallback if raw content is not available
			builder.WriteString(fmt.Sprintf("const %s = %s;\n\n", constant.Name, constant.Value))
		}
	}

	// Add variables if any
	for _, variable := range file.Variables {
		if variable.Location != nil && variable.Location.Raw != "" {
			builder.WriteString(variable.Location.Raw)
			builder.WriteString("\n\n")
		} else {
			// Fallback if raw content is not available
			varType := ""
			if variable.Type != nil {
				varType = variable.Type.Name
			}
			builder.WriteString(fmt.Sprintf("let %s%s;\n\n", variable.Name, varType))
		}
	}

	// Add types (components) if any
	for _, typ := range file.Types {
		if typ.Location != nil && typ.Location.Raw != "" {
			// Use the Type.Content() method which includes fields
			builder.WriteString(typ.Content())
			builder.WriteString("\n\n")
		} else {
			// Fallback if raw content is not available
			builder.WriteString(fmt.Sprintf("// Component: %s\n", typ.Name))

			// Determine if it's a class or function component
			if strings.Contains(typ.Name, "Component") || len(typ.Methods) > 0 {
				// Class component
				builder.WriteString(fmt.Sprintf("class %s extends React.Component {\n", typ.Name))

				// Add render method
				builder.WriteString("  render() {\n    return (\n      <div>\n        {/* JSX content */}\n      </div>\n    );\n  }\n")

				builder.WriteString("}\n\n")
			} else {
				// Function component
				builder.WriteString(fmt.Sprintf("function %s(props) {\n", typ.Name))
				builder.WriteString("  return (\n    <div>\n      {/* JSX content */}\n    </div>\n  );\n")
				builder.WriteString("}\n\n")
			}
		}
	}

	// Add functions if any
	for _, function := range file.Functions {
		if function.Location != nil && function.Location.Raw != "" {
			builder.WriteString(function.Location.Raw)
			builder.WriteString("\n\n")
		} else {
			// Fallback if raw content is not available
			builder.WriteString(fmt.Sprintf("function %s() {\n  // Function implementation\n}\n\n", function.Name))
		}
	}

	// Add export statement if needed
	if len(file.Types) > 0 {
		// Export the last type as default
		builder.WriteString(fmt.Sprintf("export default %s;\n", file.Types[len(file.Types)-1].Name))
	}

	return []byte(builder.String()), nil
}
