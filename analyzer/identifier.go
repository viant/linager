package analyzer

import (
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/analyzer/linage"
	"strings"
)

// isIdentitySignature returns true if sig is a func with one parameter and a single return of the same type
func isIdentitySignature(sig string) bool {
	// sig example: "func identity(x int) int"
	open := strings.Index(sig, "(")
	close := strings.Index(sig, ")")
	if open < 0 || close < 0 || close <= open+1 {
		return false
	}
	params := strings.TrimSpace(sig[open+1 : close])
	ret := strings.TrimSpace(sig[close+1:])
	// strip parentheses around return
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		ret = strings.TrimSpace(ret[1 : len(ret)-1])
	}
	if params == "" || ret == "" {
		return false
	}
	// exactly one param and one return, no commas
	if strings.Contains(params, ",") || strings.Contains(ret, ",") {
		return false
	}
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return false
	}
	// parameter type is last field
	paramType := parts[len(parts)-1]
	// return type should match param type
	return paramType == ret
}

// -----------------------------------------------------------------------------
// Identifier extraction & resolution
// -----------------------------------------------------------------------------

func (a *Analyzer) extractIdentifiers(root *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) []*linage.Identifier {
	var ids []*linage.Identifier
	// Handle pointer/unary expressions: &x or *p yield the underlying identifier
	switch root.Type() {
	case "unary_expression":
		// &x or *p yields the underlying identifier
		if operand := root.ChildByFieldName("operand"); operand != nil {
			return a.extractIdentifiers(operand, src, Scope, model)
		}
	case "index_expression":
		// m[k] or arr[i] yields a synthetic element identifier
		obj := root.ChildByFieldName("object")
		idx := root.ChildByFieldName("index")
		if obj != nil && idx != nil {
			base := a.resolveIdent(obj, nil, src, Scope, model)
			keyTxt := strings.TrimSpace(string(src[idx.StartByte():idx.EndByte()]))
			elemKey := fmt.Sprintf("%s[%s]@%d", base.ID, keyTxt, root.StartByte())
			if elem := model.Idents[elemKey]; elem != nil {
				return []*linage.Identifier{elem}
			}
			elem := &linage.Identifier{
				ID:        elemKey,
				Name:      base.Name + "[" + keyTxt + "]",
				Package:   base.Package,
				File:      base.File,
				StartByte: root.StartByte(),
			}
			model.Idents[elemKey] = elem
			return []*linage.Identifier{elem}
		}
	}
	// general recursive extraction
	stack := []*sitter.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch n.Type() {
		case "identifier":
			ids = append(ids, a.resolveIdent(n, nil, src, Scope, model))
		case "selector_expression":
			op := n.ChildByFieldName("operand")
			fld := n.ChildByFieldName("field")
			base := a.resolveIdent(op, nil, src, Scope, model)
			field := string(src[fld.StartByte():fld.EndByte()])
			// build selector with operand as parent if no nested selector
			var parent *linage.Selector
			if base.Selector != nil {
				parent = base.Selector
			} else {
				parent = &linage.Selector{Field: base.Name}
			}
			sel := &linage.Selector{Field: field, Parent: parent}
			id := a.resolveIdent(fld, sel, src, Scope, model)

			// Attempt to infer kind/type based on the operand (base) identifier.
			if base != nil {
				// 1. Struct field access: if operand has a concrete type that we have
				//    a field mapping for, propagate the field type.
				if base.Type != "" {
					if fieldMap, ok := a.structFields[base.Type]; ok {
						if t, ok2 := fieldMap[field]; ok2 {
							id.Type = t
							if id.Kind == "" {
								id.Kind = "field"
							}
						}
					}
				} else {
					// 2. Package selector (e.g. fmt.Printf). Treat the selected
					//    identifier as a function if it is later invoked, but as a
					//    heuristic we mark it as func now so it has at least a kind
					//    and pseudo type.
					if id.Kind == "" {
						id.Kind = "func"
					}
					if id.Type == "" {
						// Not an exact signature, but provides useful metadata.
						id.Type = "func"
					}
				}
			}

			ids = append(ids, id)
			continue
		default:
			for i := int(n.ChildCount()) - 1; i >= 0; i-- {
				stack = append(stack, n.Child(i))
			}
		}
	}
	return ids
}

func (a *Analyzer) resolveIdent(n *sitter.Node, sel *linage.Selector, src []byte, Scope *linage.Scope, model *linage.PackageModel) *linage.Identifier {
	name := string(src[n.StartByte():n.EndByte()])
	// reuse existing identifiers (vars, types, funcs) in scope
	if sel == nil {
		if existing := Scope.Find(name); existing != nil {
			return existing
		}
	}
	fileScope := topFileScope(Scope)
	file := strings.TrimPrefix(fileScope.ID, model.Path+":")
	key := fmt.Sprintf("%s::%s::%d", model.Path, file, n.StartByte())

	if id, ok := model.Idents[key]; ok {
		if id.Selector == nil && sel != nil {
			id.Selector = sel
		}
		return id
	}

	// determine package for identifier, override for import aliases if present
	pkg := model.Path
	if imp, ok := a.importAliases[name]; ok {
		pkg = imp
	}
	// create new identifier with extracted annotations
	id := &linage.Identifier{
		ID:         key,
		Name:       name,
		Package:    pkg,
		File:       file,
		StartByte:  n.StartByte(),
		Selector:   sel,
		Node:       n,
		Annotation: a.extractAnnotations(n, src),
	}
	model.Idents[key] = id
	// invoke annotation hooks to allow custom edge creation based on metadata
	for _, hook := range a.annotationHooks {
		hook(id, id.Annotation, Scope, model)
	}
	// default: create metadata edges for each annotation key/value
	if len(id.Annotation) > 0 {
		for key, val := range id.Annotation {
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{
				Src:        id,
				Dst:        id,
				Kind:       linage.Metadata,
				Scope:      Scope.ID,
				Attributes: map[string]interface{}{"annotationKey": key, "annotationValue": val},
			})
		}
	}
	// plugin hooks after identifier resolution
	for _, plugin := range a.plugins {
		plugin.AfterResolveIdent(n, id, Scope, model)
	}

	if sel == nil { // only plain identifiers go into symbol tables
		Scope.Symbols[name] = id
	}
	return id
}
