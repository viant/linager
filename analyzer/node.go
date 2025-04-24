package analyzer

import (
	"fmt"
	"github.com/viant/linager/analyzer/linage"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// -----------------------------------------------------------------------------
// AST traversal
// -----------------------------------------------------------------------------

func (a *Analyzer) walk(n *sitter.Node, src []byte, scope *linage.Scope, model *linage.PackageModel) {
	// plugin hooks before processing each AST node
	for _, plugin := range a.plugins {
		plugin.BeforeWalk(n, src, scope, model)
	}
	switch n.Type() {
	case "block":
		blk := &linage.Scope{ID: fmt.Sprintf("%s.block@%d", scope.ID, n.StartByte()), Kind: "block", Parent: scope, Symbols: map[string]*linage.Identifier{}, Start: int(n.StartByte()), End: int(n.EndByte())}
		model.Scopes = append(model.Scopes, blk)
		for i := 0; i < int(n.ChildCount()); i++ {
			a.walk(n.Child(i), src, blk, model)
		}
		return
	case "function_declaration":
		a.handleFunction(n, src, scope, model)
		return
	case "type_spec":
		a.handleTypeSpec(n, src, scope, model)
		return
	case "short_var_declaration", "assignment_statement":
		a.handleAssignment(n, src, scope, model)
		return
	case "call_expression":
		a.handleCall(n, src, scope, model)
		return
	case "import_spec":
		// capture import alias mapping
		a.handleImportSpec(n, src)
		return
	case "go_statement":
		// handle goroutine invocation
		a.handleGo(n, src, scope, model)
		return
	case "send_statement":
		// handle channel send
		a.handleSend(n, src, scope, model)
		return
	case "select_statement":
		// handle select on channels (concurrent cases)
		a.handleSelect(n, src, scope, model)
		return
	case "return_statement":
		// capture return flows: map returned identifiers into function summary
		a.handleReturn(n, src, scope, model)
		return
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		a.walk(n.Child(i), src, scope, model)
	}
}

// -------------------- Declarations -------------------------

func (a *Analyzer) handleFunction(n *sitter.Node, src []byte, current *linage.Scope, model *linage.PackageModel) {
	fnNameNode := n.ChildByFieldName("name")
	name := string(src[fnNameNode.StartByte():fnNameNode.EndByte()])
	fnID := fmt.Sprintf("%s.%s", current.ID, name)
	fnScope := &linage.Scope{ID: fnID, Kind: "function", Name: name, Parent: current, Symbols: map[string]*linage.Identifier{}, Start: int(n.StartByte()), End: int(n.EndByte())}
	// create function identifier with signature
	// signature: raw text from func start to body start, e.g. "func main(x int) error"
	var signature string
	if body := n.ChildByFieldName("body"); body != nil {
		signature = strings.TrimSpace(string(src[n.StartByte():body.StartByte()]))
	}
	ident := &linage.Identifier{
		ID:         fnID,
		Name:       name,
		Kind:       "func",
		Package:    model.Path,
		File:       current.ID,
		StartByte:  fnNameNode.StartByte(),
		Type:       signature,
		Node:       n,
		Annotation: a.extractAnnotations(n, src),
	}
	// register function identifier in current scope
	current.Symbols[name] = ident
	model.Scopes = append(model.Scopes, fnScope)
	// inter-procedural summary: capture formal parameters and return identifiers
	if a.interprocedural {
		summary := &FuncSummary{Params: make([]*linage.Identifier, 0), Returns: make([]*linage.Identifier, 0), Flows: make(map[int][]int)}
		// parameters
		if paramsNode := n.ChildByFieldName("parameters"); paramsNode != nil {
			for i := 0; i < int(paramsNode.NamedChildCount()); i++ {
				param := paramsNode.NamedChild(i)
				if param.Type() != "parameter" {
					continue
				}
				if nameNode := param.ChildByFieldName("name"); nameNode != nil {
					paramIdent := a.resolveIdent(nameNode, nil, src, fnScope, model)
					summary.Params = append(summary.Params, paramIdent)
				}
			}
		}
		// results (named returns)
		if resultNode := n.ChildByFieldName("result"); resultNode != nil {
			if resultNode.Type() == "parameter_list" {
				for i := 0; i < int(resultNode.NamedChildCount()); i++ {
					param := resultNode.NamedChild(i)
					if param.Type() != "parameter" {
						continue
					}
					if nameNode := param.ChildByFieldName("name"); nameNode != nil {
						retIdent := a.resolveIdent(nameNode, nil, src, fnScope, model)
						summary.Returns = append(summary.Returns, retIdent)
					}
				}
			} else {
				// anonymous return: use function identifier as return
				summary.Returns = append(summary.Returns, ident)
			}
		} else {
			// no explicit result: default to function identifier as return
			summary.Returns = append(summary.Returns, ident)
		}
		a.funcSummaries[ident] = summary
	}

	body := n.ChildByFieldName("body")
	for i := 0; i < int(body.ChildCount()); i++ {
		a.walk(body.Child(i), src, fnScope, model)
	}
}

// handle type specifications (e.g., type Foo struct {...})
func (a *Analyzer) handleTypeSpec(n *sitter.Node, src []byte, scope *linage.Scope, model *linage.PackageModel) {
	// In the Go grammar `type_spec` may expose the identifier as either a
	// child with the field name "name" **or** as a plain `type_identifier`
	// (depending on the exact grammar version). Try the field lookup first and
	// fall back to scanning for the first `type_identifier` child.
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		// fallback – walk children and grab the first `type_identifier`
		for i := 0; i < int(n.NamedChildCount()); i++ {
			ch := n.NamedChild(i)
			if ch.Type() == "type_identifier" {
				nameNode = ch
				break
			}
		}
	}
	if nameNode == nil {
		return // cannot resolve type name
	}
	// register type identifier
	id := a.resolveIdent(nameNode, nil, src, scope, model)
	id.Kind = "type"

	// If this is a struct type, capture its field definitions so we can later
	// infer field selector types (e.g. Foo.ID -> int).
	var typeNode *sitter.Node
	// try field lookup first (grammar v1)
	typeNode = n.ChildByFieldName("type")
	if typeNode == nil {
		// fallback – iterate children and pick the first struct_type / interface_type etc.
		for i := 0; i < int(n.NamedChildCount()); i++ {
			ch := n.NamedChild(i)
			if ch.Type() == "struct_type" {
				typeNode = ch
				break
			}
		}
	}

	if typeNode != nil && typeNode.Type() == "struct_type" {
		// Find the field declaration list (named "body" in older grammars or
		// "field_declaration_list" in newer ones).
		body := typeNode.ChildByFieldName("body")
		if body == nil {
			for i := 0; i < int(typeNode.NamedChildCount()); i++ {
				cand := typeNode.NamedChild(i)
				if cand.Type() == "field_declaration_list" {
					body = cand
					break
				}
			}
		}

		if body != nil {
			fields := map[string]string{}
			for i := 0; i < int(body.NamedChildCount()); i++ {
				fldDecl := body.NamedChild(i)
				if fldDecl.Type() != "field_declaration" {
					continue
				}
				// resolve field type (first child with type_identifier / qualified_type / etc.)
				var fieldType string
				typeChild := fldDecl.ChildByFieldName("type")
				if typeChild == nil {
					// fallback – pick last named child assuming it is the type
					if fldDecl.NamedChildCount() > 0 {
						last := fldDecl.NamedChild(int(fldDecl.NamedChildCount()) - 1)
						if last != nil && (strings.HasSuffix(last.Type(), "_identifier") || strings.HasSuffix(last.Type(), "_type") || last.Type() == "type_identifier") {
							typeChild = last
						}
					}
				}
				if typeChild != nil {
					fieldType = strings.TrimSpace(string(src[typeChild.StartByte():typeChild.EndByte()]))
				}
				// collect field identifiers
				for j := 0; j < int(fldDecl.NamedChildCount()); j++ {
					ch := fldDecl.NamedChild(j)
					if ch.Type() == "field_identifier" || ch.Type() == "identifier" {
						fieldName := string(src[ch.StartByte():ch.EndByte()])
						fields[fieldName] = fieldType
					}
				}
			}
			if len(fields) > 0 {
				a.structFields[id.Name] = fields
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Assignments & composite literals
// -----------------------------------------------------------------------------

func (a *Analyzer) handleAssignment(n *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	left := n.ChildByFieldName("left")
	right := n.ChildByFieldName("right")
	if left == nil || right == nil {
		return
	}
	lhs := a.extractIdentifiers(left, src, Scope, model)
	rhs := a.extractIdentifiers(right, src, Scope, model)

	// handle short variable declarations (:=) with go_basic.gox type inference
	if n.Type() == "short_var_declaration" {
		// collect right-hand expression nodes (skip commas)
		exprNodes := make([]*sitter.Node, 0)
		for i := 0; i < int(right.ChildCount()); i++ {
			ch := right.Child(i)
			if ch.Type() == "," {
				continue
			}
			exprNodes = append(exprNodes, ch)
		}
		// declare variables with writes, infer types, and record flows
		for idx, id := range lhs {
			id.Kind = "var"
			// infer simple types based on tree-sitter node kinds
			if idx < len(exprNodes) {
				expr := exprNodes[idx]
				switch expr.Type() {
				case "composite_literal":
					// infer type including generics from literal
					body := expr.ChildByFieldName("body")
					typeNode := expr.ChildByFieldName("type")
					if body != nil {
						id.Type = strings.TrimSpace(string(src[expr.StartByte():body.StartByte()]))
					} else if typeNode != nil {
						id.Type = strings.TrimSpace(string(src[typeNode.StartByte():typeNode.EndByte()]))
					}
					// record nested field flows for composite literal
					a.handleCompositeLiteral(id, expr, src, Scope, model)
				case "identifier":
					if idx < len(rhs) && rhs[idx].Type != "" {
						id.Type = rhs[idx].Type
					}
				default:
					raw := strings.TrimSpace(string(src[expr.StartByte():expr.EndByte()]))
					if raw != "" {
						switch raw[0] {
						case '"', '`':
							id.Type = "string"
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if strings.Contains(raw, ".") {
								id.Type = "float64"
							} else {
								id.Type = "int"
							}
						}
					}
				}
			}
			// write to variable: use literal as source for composite literals
			var srcIdent *linage.Identifier
			if idx < len(exprNodes) && exprNodes[idx].Type() == "composite_literal" {
				srcIdent = a.literalIdent(exprNodes[idx], src, Scope, model)
			} else {
				srcIdent = id
			}
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: srcIdent, Dst: id, Kind: linage.Write, Scope: Scope.ID})
		}
		// record reads and transfers for each expression
		for idx, expr := range exprNodes {
			// call-through: handle call expressions
			if expr.Type() == "call_expression" {
				if a.interprocedural {
					a.handleCallInAssignment(expr, src, Scope, model, lhs)
				} else {
					// legacy mapping: directly pass arguments to variables
					if argList := expr.ChildByFieldName("argument_list"); argList != nil {
						argIds := a.extractIdentifiers(argList, src, Scope, model)
						for _, v := range argIds {
							model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: Scope.ID})
							if idx < len(lhs) {
								dst := lhs[idx]
								model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: dst, Kind: linage.Xfer, Scope: Scope.ID})
							}
						}
					}
				}
				continue
			}
			vals := a.extractIdentifiers(expr, src, Scope, model)
			for _, v := range vals {
				// read from source
				model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: Scope.ID})
				// transfer from source to destination variable
				if idx < len(lhs) {
					dst := lhs[idx]
					model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: dst, Kind: linage.Xfer, Scope: Scope.ID})
				}
			}
		}
		return
	}

	// handle standard assignment (=)
	for _, id := range lhs {
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: id, Dst: id, Kind: linage.Write, Scope: Scope.ID})
	}
	for idx, srcID := range rhs {
		// read from source
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: srcID, Dst: srcID, Kind: linage.Read, Scope: Scope.ID})
		// transfer to destination
		if idx < len(lhs) {
			dst := lhs[idx]
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: srcID, Dst: dst, Kind: linage.Xfer, Scope: Scope.ID})
		}
	}
}

func (a *Analyzer) handleCompositeLiteral(dest *linage.Identifier, comp *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	body := comp.ChildByFieldName("body")
	if body == nil {
		return
	}
	// prepare struct field types if available
	var fieldTypes map[string]string
	if dest.Type != "" {
		fieldTypes = a.structFields[dest.Type]
	}
	for i := 0; i < int(body.ChildCount()); i++ {
		elem := body.Child(i)
		if elem.Type() != "keyed_element" {
			continue
		}
		keyNode := elem.ChildByFieldName("key")
		valNode := elem.ChildByFieldName("value")
		if keyNode == nil || valNode == nil {
			continue
		}
		fieldName := strings.TrimSpace(string(src[keyNode.StartByte():keyNode.EndByte()]))
		// build selector chain: dest -> field
		var parent *linage.Selector
		if dest.Selector != nil {
			parent = dest.Selector
		} else {
			parent = &linage.Selector{Field: dest.Name}
		}
		sel := &linage.Selector{Field: fieldName, Parent: parent}
		// key for field identifier uses keyNode position
		keyID := fmt.Sprintf("%s::%s::%d:%s", dest.Package, dest.File, keyNode.StartByte(), fieldName)
		fld := model.Idents[keyID]
		if fld == nil {
			fld = &linage.Identifier{
				ID:        keyID,
				Name:      fieldName,
				Kind:      "field",
				Package:   dest.Package,
				File:      dest.File,
				StartByte: keyNode.StartByte(),
				Selector:  sel,
			}
			// assign field type if known
			if fieldTypes != nil {
				if t, ok := fieldTypes[fieldName]; ok {
					fld.Type = t
				}
			}
			model.Idents[keyID] = fld
		}
		// record write to field
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: fld, Dst: fld, Kind: linage.Write, Scope: Scope.ID})
		// record value flows into field
		vals := a.extractIdentifiers(valNode, src, Scope, model)
		for _, v := range vals {
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: Scope.ID})
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: fld, Kind: linage.Xfer, Scope: Scope.ID})
		}
	}
}

// -----------------------------------------------------------------------------
// Calls
// -----------------------------------------------------------------------------

func (a *Analyzer) handleCall(n *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	fnNode := n.ChildByFieldName("function")
	if fnNode == nil {
		return
	}
	fns := a.extractIdentifiers(fnNode, src, Scope, model)
	for _, fn := range fns {
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: fn, Dst: fn, Kind: linage.Call, Scope: Scope.ID})
	}
	// Concurrency: track sync.WaitGroup Done/Wait as synthetic channel flows
	if n.Type() == "call_expression" && fnNode.Type() == "selector_expression" {
		// resolve base WaitGroup identifier
		baseNode := fnNode.ChildByFieldName("operand")
		if baseNode != nil {
			baseIdent := a.resolveIdent(baseNode, nil, src, Scope, model)
			// simple type check for WaitGroup
			if strings.HasSuffix(baseIdent.Type, "WaitGroup") {
				// synthetic channel key for this WaitGroup
				wgKey := fmt.Sprintf("%s::wg::%d", baseIdent.ID, baseNode.StartByte())
				wgChan, ok := model.Idents[wgKey]
				if !ok {
					wgChan = &linage.Identifier{ID: wgKey, Name: baseIdent.Name + ".wgChan", Package: baseIdent.Package, File: baseIdent.File, StartByte: baseNode.StartByte()}
					model.Idents[wgKey] = wgChan
				}
				// map Done to write, Wait to read on synthetic channel
				for _, fn := range fns {
					if fn.Selector != nil {
						switch fn.Selector.Field {
						case "Done":
							model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: wgChan, Dst: wgChan, Kind: linage.Write, Scope: Scope.ID})
						case "Wait":
							model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: wgChan, Dst: wgChan, Kind: linage.Read, Scope: Scope.ID})
						}
					}
				}
			}
		}
	}
	args := n.ChildByFieldName("argument_list")
	if args != nil {
		ids := a.extractIdentifiers(args, src, Scope, model)
		for _, id := range ids {
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: id, Dst: id, Kind: linage.Read, Scope: Scope.ID})
		}
	}
}

// handleCallInAssignment applies inter-procedural call-return flows for call expressions in assignments
func (a *Analyzer) handleCallInAssignment(expr *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel, lhs []*linage.Identifier) {
	fnNode := expr.ChildByFieldName("function")
	if fnNode == nil {
		return
	}
	fns := a.extractIdentifiers(fnNode, src, Scope, model)
	// collect argument expression nodes (skip commas)
	var argExprs []*sitter.Node
	if argList := expr.ChildByFieldName("argument_list"); argList != nil {
		for i := 0; i < int(argList.ChildCount()); i++ {
			ch := argList.Child(i)
			if ch.Type() == "," {
				continue
			}
			argExprs = append(argExprs, ch)
		}
	}
	// for each referenced function, apply its summary or fallback mapping
	for _, fn := range fns {
		if summary, ok := a.funcSummaries[fn]; ok {
			// prepare synthetic return identifiers for the call site
			fileScope := topFileScope(Scope)
			file := strings.TrimPrefix(fileScope.ID, model.Path+":")
			callStart := expr.StartByte()
			callRets := make([]*linage.Identifier, len(summary.Returns))
			for retIdx := range summary.Returns {
				callKey := fmt.Sprintf("%s::%s::%d#ret%d", model.Path, file, callStart, retIdx)
				if id := model.Idents[callKey]; id != nil {
					callRets[retIdx] = id
				} else {
					id := &linage.Identifier{
						ID:        callKey,
						Name:      summary.Returns[retIdx].Name,
						Package:   model.Path,
						File:      file,
						StartByte: callStart,
						Kind:      "call",
					}
					model.Idents[callKey] = id
					callRets[retIdx] = id
				}
			}
			// map actual arguments to synthetic call returns based on summary
			for pIdx := range summary.Params {
				if pIdx < len(argExprs) {
					actuals := a.extractIdentifiers(argExprs[pIdx], src, Scope, model)
					// determine which return indices flow from this parameter
					rets := summary.Flows[pIdx]
					if len(rets) == 0 {
						// fallback: map to all returns
						rets = make([]int, len(callRets))
						for j := range callRets {
							rets[j] = j
						}
					}
					for _, actual := range actuals {
						// read from argument
						model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: actual, Dst: actual, Kind: linage.Read, Scope: Scope.ID})
						// transfer to each mapped return
						for _, retIdx := range rets {
							if retIdx < len(callRets) {
								model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: actual, Dst: callRets[retIdx], Kind: linage.Xfer, Scope: Scope.ID})
							}
						}
					}
				}
			}
			// finally map synthetic call returns to LHS variables
			for retIdx, dst := range lhs {
				if retIdx < len(callRets) {
					model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: callRets[retIdx], Dst: dst, Kind: linage.Xfer, Scope: Scope.ID})
				}
			}
		} else {
			// fallback: conservative mapping actual args to LHS
			for idx, argExpr := range argExprs {
				actuals := a.extractIdentifiers(argExpr, src, Scope, model)
				for _, actual := range actuals {
					model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: actual, Dst: actual, Kind: linage.Read, Scope: Scope.ID})
					if idx < len(lhs) {
						model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: actual, Dst: lhs[idx], Kind: linage.Xfer, Scope: Scope.ID})
					}
				}
			}
		}
	}
}

// handleImportSpec records import alias mapping for the current file
func (a *Analyzer) handleImportSpec(n *sitter.Node, src []byte) {
	var alias, path string
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "identifier":
			alias = string(src[child.StartByte():child.EndByte()])
		case "interpreted_string_literal", "raw_string_literal":
			lit := string(src[child.StartByte():child.EndByte()])
			path = strings.Trim(lit, "`\"")
			// strip vendor prefix in import paths
			if vIdx := strings.Index(path, "/vendor/"); vIdx != -1 {
				path = path[vIdx+len("/vendor/"):]
			}
		}
	}
	if alias == "" {
		if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
			alias = path[idx+1:]
		} else {
			alias = path
		}
	}
	if alias != "_" && alias != "." {
		if a.importAliases == nil {
			a.importAliases = map[string]string{}
		}
		a.importAliases[alias] = path
	}
}

// handleReturn captures data-flow from return-expression identifiers into the function summary or identity function
func (a *Analyzer) handleReturn(n *sitter.Node, src []byte, scope *linage.Scope, model *linage.PackageModel) {
	if scope.Kind != "function" {
		return
	}
	// locate the function identifier in the parent (package) scope
	funcIdent := scope.Parent.Symbols[scope.Name]
	if funcIdent == nil {
		return
	}
	// inter-procedural return mapping
	if a.interprocedural {
		// ensure summary exists
		summary, ok := a.funcSummaries[funcIdent]
		if !ok {
			summary = &FuncSummary{Params: nil, Returns: []*linage.Identifier{funcIdent}, Flows: make(map[int][]int)}
			a.funcSummaries[funcIdent] = summary
		}
		// collect return expression nodes (skip 'return' keyword and commas)
		var exprNodes []*sitter.Node
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(i)
			if child.Type() == "return" || child.Type() == "," {
				continue
			}
			exprNodes = append(exprNodes, child)
		}
		// map each returned identifier into its summary return and record inter-procedural flows
		for idx, expr := range exprNodes {
			vals := a.extractIdentifiers(expr, src, scope, model)
			for _, v := range vals {
				// read from returned value
				model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: scope.ID})
				// determine target return identifier
				var retIdent *linage.Identifier
				if idx < len(summary.Returns) {
					retIdent = summary.Returns[idx]
				} else {
					retIdent = funcIdent
				}
				// transfer into function return
				model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: retIdent, Kind: linage.Xfer, Scope: scope.ID})
				// if returned value matches a formal parameter, record summary mapping
				for pIdx, pIdent := range summary.Params {
					if pIdent == v {
						summary.Flows[pIdx] = append(summary.Flows[pIdx], idx)
						break
					}
				}
			}
		}
		return
	}
	// legacy: only identity functions get intra-procedural return→parameter flows
	if !isIdentitySignature(funcIdent.Type) {
		return
	}
	// map returned identifiers into the function identifier for identity functions
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "return" || child.Type() == "," {
			continue
		}
		vals := a.extractIdentifiers(child, src, scope, model)
		for _, v := range vals {
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: scope.ID})
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: funcIdent, Kind: linage.Xfer, Scope: scope.ID})
		}
	}
}
