package golang

import (
	"github.com/viant/linager/inspector/info"
	"go/ast"
	"go/token"
	"strings"
)

// InspectVariables inspects an AST file to extract variables
func (i *Inspector) InspectVariables(file *ast.File, importMap map[string]string) ([]*info.Variable, error) {
	var variables []*info.Variable

	// Iterate through all declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		// Extract documentation from the declaration
		var groupComment string
		if genDecl.Doc != nil {
			groupComment = genDecl.Doc.Text()
		}

		// Process each variable specification
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Get specific comment for this variable
			varComment := groupComment
			if valueSpec.Doc != nil {
				varComment = valueSpec.Doc.Text()
			} else if valueSpec.Comment != nil {
				varComment = valueSpec.Comment.Text()
			}
			varComment = strings.TrimSpace(varComment)

			// Create a variable for each name
			for idx, name := range valueSpec.Names {
				// Skip unexported if configured to do so
				if !i.config.IncludeUnexported && !name.IsExported() {
					continue
				}

				var varType *info.Type
				if valueSpec.Type != nil {
					typeName := exprToString(valueSpec.Type, importMap)
					kind := kindFromBasicType(typeName)
					varType = &info.Type{
						Name: typeName,
						Kind: kind,
					}
				}

				// Extract value as string
				var value string
				if idx < len(valueSpec.Values) {
					value = extractValueAsString(valueSpec.Values[idx], i.fset)
				}

				variables = append(variables, &info.Variable{
					Name:    name.Name,
					Comment: varComment,
					Value:   value,
					Type:    varType,
				})
			}
		}
	}

	return variables, nil
}
