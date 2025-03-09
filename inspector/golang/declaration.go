package golang

import (
	"bytes"
	"github.com/viant/linager/inspector/info"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

// InspectDeclaration inspects a declaration and returns type information
func (i *Inspector) InspectDeclaration(decl ast.Decl, importMap map[string]string) ([]*info.Type, error) {
	switch d := decl.(type) {
	case *ast.GenDecl:
		return i.inspectGenDecl(d, importMap)
	case *ast.FuncDecl:
		return i.inspectFuncDecl(d)
	default:
		return nil, nil
	}
}

// inspectGenDecl handles type declarations, vars, consts, and imports
func (i *Inspector) inspectGenDecl(decl *ast.GenDecl, importMap map[string]string) ([]*info.Type, error) {
	var types []*info.Type

	switch decl.Tok {
	case token.TYPE: // Changed from ast.TYPE to token.TYPE
		// Type declarations
		for _, spec := range decl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if !i.config.IncludeUnexported && !ts.Name.IsExported() {
				continue
			}

			// Pass both the DocGroup from TypeSpec and from GenDecl
			// This ensures we capture comments properly in all cases
			typeInfo := i.processTypeSpec(ts, ts.Doc, decl.Doc, importMap)
			types = append(types, typeInfo)
		}
	}

	return types, nil
}

// inspectFuncDecl handles function and method declarations
func (i *Inspector) inspectFuncDecl(decl *ast.FuncDecl) ([]*info.Type, error) {
	// We don't create types for standalone functions
	// Methods are handled separately when processing types
	return nil, nil
}

// processTypeSpec converts an ast.TypeSpec to our Type
func (i *Inspector) processTypeSpec(ts *ast.TypeSpec, typeSpecDoc, genDeclDoc *ast.CommentGroup, importMap map[string]string) *info.Type {
	// Extract comments - prioritize TypeSpec doc comment over GenDecl doc comment
	comment := ""
	var commentLocation info.Location
	if typeSpecDoc != nil {
		comment = typeSpecDoc.Text()
		commentLocation = info.Location{
			Start: i.fset.Position(typeSpecDoc.Pos()).Offset,
			End:   i.fset.Position(typeSpecDoc.End()).Offset,
		}
	} else if genDeclDoc != nil {
		comment = genDeclDoc.Text()
		commentLocation = info.Location{
			Start: i.fset.Position(genDeclDoc.Pos()).Offset,
			End:   i.fset.Position(genDeclDoc.End()).Offset,
		}
	}

	typeKind := determineTypeKind(ts)

	// Create location information for the type
	typeLocation := &info.Location{
		Start: i.fset.Position(ts.Pos()).Offset,
		End:   i.fset.Position(ts.End()).Offset,
	}

	t := &info.Type{
		Name:       ts.Name.Name,
		Kind:       kindFromString(typeKind),
		Comment:    &info.LocationNode{Text: strings.TrimSpace(comment), Location: commentLocation},
		IsExported: ts.Name.IsExported(),
		TypeParams: extractTypeParams(ts.TypeParams, importMap),
		Location:   typeLocation,
	}

	// Process fields if it's a struct
	if typeKind == "struct" {
		st, ok := ts.Type.(*ast.StructType)
		if ok && st.Fields != nil {
			t.Fields = i.processFields(st.Fields, importMap)
		}
	} else if typeKind == "interface" {
		// Process interface methods
		iface, ok := ts.Type.(*ast.InterfaceType)
		if ok && iface.Methods != nil {
			// Process interface methods - can be extended if needed
		}
	} else if typeKind == "alias" {
		// For type aliases, generate a comment if none exists
		if t.Comment != nil && t.Comment.Text == "" {
			baseType := ""
			switch typeExpr := ts.Type.(type) {
			case *ast.Ident:
				baseType = typeExpr.Name
			case *ast.SelectorExpr:
				baseType = exprToString(typeExpr, importMap)
			default:
				baseType = exprToString(ts.Type, importMap)
			}
			t.Comment.Text = t.Name + " is a type alias for " + baseType
		}
	}

	// Additional type-specific processing
	i.processTypeDetails(ts, t, importMap)

	return t
}

// extractFieldDocumentation gets documentation for a struct field
func extractFieldDocumentation(field *ast.Field) string {
	if field.Doc != nil {
		return field.Doc.Text()
	}
	if field.Comment != nil {
		return field.Comment.Text()
	}
	return ""
}

// processMethod converts an ast.FuncDecl to our Method
func (i *Inspector) processMethod(funcDecl *ast.FuncDecl, importMap map[string]string) info.Method {
	comment := ""
	var commentLocation info.Location
	if funcDecl.Doc != nil {
		comment = funcDecl.Doc.Text()
		commentLocation = info.Location{
			Start: i.fset.Position(funcDecl.Doc.Pos()).Offset,
			End:   i.fset.Position(funcDecl.Doc.End()).Offset,
		}
	}

	recvField := funcDecl.Recv.List[0]
	recvTypeStr := exprToString(recvField.Type, importMap)

	// Create method location information
	methodLocation := &info.Location{
		Start: i.fset.Position(funcDecl.Pos()).Offset,
		End:   i.fset.Position(funcDecl.End()).Offset,
	}

	method := info.Method{
		Name:       funcDecl.Name.Name,
		Comment:    &info.LocationNode{Text: strings.TrimSpace(comment), Location: commentLocation},
		Receiver:   recvTypeStr,
		TypeParams: extractTypeParams(funcDecl.Type.TypeParams, importMap),
		IsExported: funcDecl.Name.IsExported(),
		Location:   methodLocation,
	}
	method.Signature = formatFuncType(funcDecl.Name.Name, funcDecl.Type, importMap)

	// Process parameters
	if funcDecl.Type.Params != nil {
		for _, p := range funcDecl.Type.Params.List {
			paramType := &info.Type{
				Name: exprToString(p.Type, importMap),
			}

			if len(p.Names) == 0 {
				method.Parameters = append(method.Parameters, info.Parameter{
					Type: paramType,
				})
			} else {
				for _, name := range p.Names {
					method.Parameters = append(method.Parameters, info.Parameter{
						Name: name.Name,
						Type: paramType,
					})
				}
			}
		}
	}

	// Process results
	if funcDecl.Type.Results != nil {
		for _, r := range funcDecl.Type.Results.List {
			resultType := &info.Type{
				Name: exprToString(r.Type, importMap),
			}

			if len(r.Names) == 0 {
				method.Results = append(method.Results, info.Parameter{
					Type: resultType,
				})
			} else {
				for _, name := range r.Names {
					method.Results = append(method.Results, info.Parameter{
						Name: name.Name,
						Type: resultType,
					})
				}
			}
		}
	}

	// Extract function body if available
	if funcDecl.Body != nil {
		// Use the printer package to extract the function body as a string
		var buf bytes.Buffer
		err := printer.Fprint(&buf, i.fset, funcDecl.Body)

		bodyLocation := info.Location{
			Start: i.fset.Position(funcDecl.Body.Pos()).Offset,
			End:   i.fset.Position(funcDecl.Body.End()).Offset,
		}

		if err == nil {
			method.Body = &info.LocationNode{
				Text:     buf.String(),
				Location: bodyLocation,
			}
		} else {
			// Fallback method to read from file if printer fails
			bodyStart := i.fset.Position(funcDecl.Body.Lbrace)
			bodyEnd := i.fset.Position(funcDecl.Body.Rbrace)

			if bodyStart.Filename != "" {
				fileBytes, err := os.ReadFile(bodyStart.Filename)
				if err == nil {
					bodyStartOffset := bodyStart.Offset
					bodyEndOffset := bodyEnd.Offset + 1 // Include the closing brace

					if bodyEndOffset <= len(fileBytes) && bodyStartOffset < bodyEndOffset {
						method.Body = &info.LocationNode{
							Text: string(fileBytes[bodyStartOffset:bodyEndOffset]),
							Location: info.Location{
								Start: bodyStartOffset,
								End:   bodyEndOffset,
							},
						}
					}
				}
			}
		}
	}

	return method
}
