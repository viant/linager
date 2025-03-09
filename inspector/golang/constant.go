package golang

import (
	"github.com/viant/linager/inspector/info"
	"go/ast"
	"go/token"
	"strings"
)

// InspectConstants extracts constants from an AST file
func (i *Inspector) InspectConstants(file *ast.File, importMap map[string]string) ([]*info.Constant, error) {
	var constants []*info.Constant

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		// Get doc comment for the const block
		var blockDoc string
		if genDecl.Doc != nil {
			blockDoc = genDecl.Doc.Text()
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Use spec doc or fall back to block doc
			docText := blockDoc
			if valueSpec.Doc != nil {
				docText = valueSpec.Doc.Text()
			}

			// Extract and create constants
			for j, name := range valueSpec.Names {
				if !i.config.IncludeUnexported && !name.IsExported() {
					continue
				}

				// Create constant with location info
				constant := &info.Constant{
					Name:    name.Name,
					Value:   "", // Will be populated below
					Comment: strings.TrimSpace(docText),
					Location: &info.Location{
						Start: i.fset.Position(name.Pos()).Offset,
						End:   i.fset.Position(name.End()).Offset,
					},
				}

				// Extract value if available
				if j < len(valueSpec.Values) {
					constant.Value = extractValueAsString(valueSpec.Values[j], i.fset)
				}

				// Extract type if available
				if valueSpec.Type != nil {

					var constType *info.Type
					if valueSpec.Type != nil {
						typeName := exprToString(valueSpec.Type, importMap)
						constType = &info.Type{
							Name: typeName,
							Kind: kindFromBasicType(typeName),
						}
					}
					constant.Type = constType
				}

				constants = append(constants, constant)
			}
		}
	}

	return constants, nil
}
