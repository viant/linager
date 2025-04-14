package golang

import (
	"github.com/viant/linager/inspector/graph"
	"go/ast"
	"reflect"
	"strings"
)

// processFields processes struct fields
func (i *Inspector) processFields(fields *ast.FieldList, importMap map[string]string) []*graph.Field {
	var result []*graph.Field

	for _, field := range fields.List {
		// Skip if this is an embedded field with a receiver and we're not including unexported
		if len(field.Names) == 0 && !i.config.IncludeUnexported {
			if ident, ok := field.Type.(*ast.Ident); ok && !ident.IsExported() {
				continue
			}
			if star, ok := field.Type.(*ast.StarExpr); ok {
				if ident, ok := star.X.(*ast.Ident); ok && !ident.IsExported() {
					continue
				}
			}
		}

		// Extract field documentation
		comment := strings.TrimSpace(extractFieldDocumentation(field))

		// Process field tag
		var tag reflect.StructTag
		if field.Tag != nil {
			tagValue := field.Tag.Value
			// Strip the surrounding backticks
			if len(tagValue) > 1 {
				tagValue = tagValue[1 : len(tagValue)-1]
			}
			tag = reflect.StructTag(tagValue)
		}

		// Create a field for each name, or a single field for embedded types
		if len(field.Names) == 0 {
			// Embedded field
			fieldType := &graph.Type{
				Name: exprToString(field.Type, importMap),
			}

			if expr, ok := field.Type.(*ast.StarExpr); ok {
				fieldType.IsPointer = true
				fieldType.Name = exprToString(expr.X, importMap)
			}

			_, annotation := parseCommentsAndAnnotations(comment)
			result = append(result, &graph.Field{
				Type:       fieldType,
				Tag:        tag,
				Comment:    comment,
				Annotation: annotation,
				IsExported: isExportedType(field.Type),
				IsEmbedded: true,
			})
		} else {
			// Named fields
			for _, name := range field.Names {
				if !i.config.IncludeUnexported && !name.IsExported() {
					continue
				}

				fieldType := &graph.Type{
					Name: exprToString(field.Type, importMap),
				}

				result = append(result, &graph.Field{
					Name:       name.Name,
					Type:       fieldType,
					Tag:        tag,
					Comment:    comment,
					IsExported: name.IsExported(),
				})
			}
		}
	}

	return result
}

// extractFieldTag extracts the tag from a field
func extractFieldTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	return field.Tag.Value
}

// isFieldExported checks if an embedded field is exported
func isFieldExported(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return field.Names[0].IsExported()
	}

	// For embedded fields, check the type name
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.IsExported()
	case *ast.SelectorExpr:
		return t.Sel.IsExported()
	case *ast.StarExpr:
		// For pointer types, check the base type
		switch st := t.X.(type) {
		case *ast.Ident:
			return st.IsExported()
		case *ast.SelectorExpr:
			return st.Sel.IsExported()
		}
	}
	return false
}

// parseCommentsAndAnnotations separates annotations from regular comments
func parseCommentsAndAnnotations(comment string) (string, string) {
	var annotations []string
	var comments []string

	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@") {
			annotations = append(annotations, line)
		} else if line != "" {
			comments = append(comments, line)
		}
	}

	return strings.Join(comments, "\n"), strings.Join(annotations, "\n")
}
