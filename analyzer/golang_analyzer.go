package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"

	"github.com/viant/linager/analyzer/linage"
)

// GolangAnalyzer analyzes Go source code to extract data lineage information
type GolangAnalyzer struct {
	project    string
	fset       *token.FileSet
	scopes     map[string]*linage.Scope
	dataPoints map[string]*linage.DataPoint
}

// NewGolangAnalyzer creates a new GolangAnalyzer
func NewGolangAnalyzer(project string) *GolangAnalyzer {
	return &GolangAnalyzer{
		project:    project,
		fset:       token.NewFileSet(),
		scopes:     make(map[string]*linage.Scope),
		dataPoints: make(map[string]*linage.DataPoint),
	}
}

// AnalyzeSourceCode analyzes Go source code and extracts data lineage information
func (a *GolangAnalyzer) AnalyzeSourceCode(source string, project string, path string) ([]*linage.DataPoint, error) {
	// Reset state for new analysis
	a.scopes = make(map[string]*linage.Scope)
	a.dataPoints = make(map[string]*linage.DataPoint)
	a.project = project // Set the project name for use in data points
	a.fset = token.NewFileSet()

	// Parse the source code using Go's standard library
	file, err := parser.ParseFile(a.fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %w", err)
	}

	// Build scope hierarchy
	err = a.buildScopeHierarchy(file)
	if err != nil {
		return nil, fmt.Errorf("failed to build scope hierarchy: %w", err)
	}

	// Process declarations (packages, functions, types, variables)
	err = a.processDeclarations(file, path)
	if err != nil {
		return nil, fmt.Errorf("failed to process declarations: %w", err)
	}

	// Process expressions (function calls, field access, etc.)
	err = a.processExpressions(file, path)
	if err != nil {
		return nil, fmt.Errorf("failed to process expressions: %w", err)
	}

	// Convert dataPoints map to slice
	result := make([]*linage.DataPoint, 0, len(a.dataPoints))
	for _, dp := range a.dataPoints {
		result = append(result, dp)
	}

	// Sort the data points by their Ref field to ensure consistent order
	sort.Slice(result, func(i, j int) bool {
		return string(result[i].Identity.Ref) < string(result[j].Identity.Ref)
	})

	return result, nil
}

// AnalyzeFile analyzes a Go file and extracts data lineage information
func (a *GolangAnalyzer) AnalyzeFile(filePath string) ([]*linage.DataPoint, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Extract project name from file path if not already set
	project := a.project
	if project == "" {
		project = filepath.Base(filepath.Dir(filePath))
	}

	return a.AnalyzeSourceCode(string(source), project, filePath)
}

// AnalyzePackage analyzes all Go files in a package and extracts data lineage information
func (a *GolangAnalyzer) AnalyzePackage(packagePath string) ([]*linage.DataPoint, error) {
	// Get the absolute path of the package
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Extract project name from package path if not already set
	project := a.project
	if project == "" {
		project = filepath.Base(absPath)
	}

	var allDataPoints []*linage.DataPoint

	// Walk through the package directory
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only .go files
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// Analyze the file
		dataPoints, err := a.AnalyzeFile(path)
		if err != nil {
			return fmt.Errorf("error analyzing file %s: %w", path, err)
		}

		// Add data points to the result
		allDataPoints = append(allDataPoints, dataPoints...)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking package directory: %w", err)
	}

	return allDataPoints, nil
}

// buildScopeHierarchy builds a hierarchy of scopes from the Go AST
func (a *GolangAnalyzer) buildScopeHierarchy(file *ast.File) error {
	// Create root scope for the package
	packageName := file.Name.Name

	// Get the position of the package declaration
	packagePos := a.fset.Position(file.Package)

	// Find the end of the file
	var fileEnd token.Pos
	if len(file.Decls) > 0 {
		lastDecl := file.Decls[len(file.Decls)-1]
		fileEnd = lastDecl.End()
	} else {
		fileEnd = file.End()
	}
	fileEndPos := a.fset.Position(fileEnd)

	rootScope := &linage.Scope{
		ID:       packageName,
		Kind:     "package",
		ParentID: "",
		Start:    packagePos.Line,
		End:      fileEndPos.Line,
	}
	a.scopes[packageName] = rootScope

	// Process function declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// Get function name
			funcName := node.Name.Name

			// Get function position
			funcPos := a.fset.Position(node.Pos())
			funcEndPos := a.fset.Position(node.End())

			// Create scope ID based on whether it's a method or a function
			var scopeID string
			if node.Recv != nil && len(node.Recv.List) > 0 {
				// It's a method, get the receiver type
				receiverType := ""
				switch recvType := node.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					// Pointer receiver
					if ident, ok := recvType.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					// Non-pointer receiver
					receiverType = recvType.Name
				}

				if receiverType != "" {
					scopeID = fmt.Sprintf("%s.%s.%s", packageName, receiverType, funcName)
				} else {
					scopeID = fmt.Sprintf("%s.%s", packageName, funcName)
				}
			} else {
				// It's a function
				scopeID = fmt.Sprintf("%s.%s", packageName, funcName)
			}

			// Create function/method scope
			funcScope := &linage.Scope{
				ID:       scopeID,
				Kind:     "function",
				ParentID: packageName,
				Start:    funcPos.Line,
				End:      funcEndPos.Line,
			}
			a.scopes[scopeID] = funcScope

			// Process blocks within the function
			if node.Body != nil {
				a.processBlockScopes(node.Body, scopeID)
			}
		}
		return true
	})

	return nil
}

// processBlockScopes processes block scopes (if, for, switch, etc.) within a function or method
func (a *GolangAnalyzer) processBlockScopes(block *ast.BlockStmt, parentID string) {
	// Keep track of block indices for each type
	blockIndices := make(map[string]int)

	ast.Inspect(block, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			// Get if statement position
			ifPos := a.fset.Position(node.Pos())
			ifEndPos := a.fset.Position(node.End())

			// Create a unique ID for this if block
			blockType := "if"
			blockIndices[blockType]++
			blockID := fmt.Sprintf("%s.%s_%d", parentID, blockType, blockIndices[blockType])

			// Create if scope
			ifScope := &linage.Scope{
				ID:       blockID,
				Kind:     blockType,
				ParentID: parentID,
				Start:    ifPos.Line,
				End:      ifEndPos.Line,
			}
			a.scopes[blockID] = ifScope

			// Process the if body
			if node.Body != nil {
				a.processBlockScopes(node.Body, blockID)
			}

			// Process the else body if it exists
			if node.Else != nil {
				// Get else statement position
				elsePos := a.fset.Position(node.Else.Pos())
				elseEndPos := a.fset.Position(node.Else.End())

				// Create a unique ID for this else block
				blockType = "else"
				blockIndices[blockType]++
				blockID = fmt.Sprintf("%s.%s_%d", parentID, blockType, blockIndices[blockType])

				// Create else scope
				elseScope := &linage.Scope{
					ID:       blockID,
					Kind:     blockType,
					ParentID: parentID,
					Start:    elsePos.Line,
					End:      elseEndPos.Line,
				}
				a.scopes[blockID] = elseScope

				// Process the else body
				switch elseNode := node.Else.(type) {
				case *ast.BlockStmt:
					a.processBlockScopes(elseNode, blockID)
				case *ast.IfStmt:
					// This is an else if, so we'll process it as a separate if statement
					// The parent is still the original parent
					return true
				}
			}

		case *ast.ForStmt:
			// Get for statement position
			forPos := a.fset.Position(node.Pos())
			forEndPos := a.fset.Position(node.End())

			// Create a unique ID for this for block
			blockType := "for"
			blockIndices[blockType]++
			blockID := fmt.Sprintf("%s.%s_%d", parentID, blockType, blockIndices[blockType])

			// Create for scope
			forScope := &linage.Scope{
				ID:       blockID,
				Kind:     blockType,
				ParentID: parentID,
				Start:    forPos.Line,
				End:      forEndPos.Line,
			}
			a.scopes[blockID] = forScope

			// Process the for body
			if node.Body != nil {
				a.processBlockScopes(node.Body, blockID)
			}

		case *ast.RangeStmt:
			// Get range statement position
			rangePos := a.fset.Position(node.Pos())
			rangeEndPos := a.fset.Position(node.End())

			// Create a unique ID for this range block
			blockType := "range"
			blockIndices[blockType]++
			blockID := fmt.Sprintf("%s.%s_%d", parentID, blockType, blockIndices[blockType])

			// Create range scope
			rangeScope := &linage.Scope{
				ID:       blockID,
				Kind:     blockType,
				ParentID: parentID,
				Start:    rangePos.Line,
				End:      rangeEndPos.Line,
			}
			a.scopes[blockID] = rangeScope

			// Process the range body
			if node.Body != nil {
				a.processBlockScopes(node.Body, blockID)
			}

		case *ast.SwitchStmt:
			// Get switch statement position
			switchPos := a.fset.Position(node.Pos())
			switchEndPos := a.fset.Position(node.End())

			// Create a unique ID for this switch block
			blockType := "switch"
			blockIndices[blockType]++
			blockID := fmt.Sprintf("%s.%s_%d", parentID, blockType, blockIndices[blockType])

			// Create switch scope
			switchScope := &linage.Scope{
				ID:       blockID,
				Kind:     blockType,
				ParentID: parentID,
				Start:    switchPos.Line,
				End:      switchEndPos.Line,
			}
			a.scopes[blockID] = switchScope

			// Process the switch body
			if node.Body != nil {
				// Process each case
				for i, caseClause := range node.Body.List {
					if cc, ok := caseClause.(*ast.CaseClause); ok {
						// Get case position
						casePos := a.fset.Position(cc.Pos())
						caseEndPos := a.fset.Position(cc.End())

						// Create a unique ID for this case block
						caseType := "case"
						caseID := fmt.Sprintf("%s.%s_%d", blockID, caseType, i+1)

						// Create case scope
						caseScope := &linage.Scope{
							ID:       caseID,
							Kind:     caseType,
							ParentID: blockID,
							Start:    casePos.Line,
							End:      caseEndPos.Line,
						}
						a.scopes[caseID] = caseScope
					}
				}
			}
		}
		return true
	})
}

// processDeclarations processes declarations (packages, functions, types, variables) in the Go AST
func (a *GolangAnalyzer) processDeclarations(file *ast.File, path string) error {
	packageName := file.Name.Name

	// Process function declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Get function name
			funcName := d.Name.Name

			// Get function position
			funcPos := a.fset.Position(d.Pos())

			// Determine if it's a method or a function
			var dataPoint *linage.DataPoint
			if d.Recv != nil && len(d.Recv.List) > 0 {
				// It's a method, get the receiver type
				receiverType := ""
				switch recvType := d.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					// Pointer receiver
					if ident, ok := recvType.X.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				case *ast.Ident:
					// Non-pointer receiver
					receiverType = recvType.Name
				}

				if receiverType != "" {
					// Create a data point for the method
					methodRef := linage.IdentityRef(fmt.Sprintf("%s:%s.%s", packageName, receiverType, funcName))
					dataPoint = &linage.DataPoint{
						Identity: linage.Identity{
							Ref:        methodRef,
							Module:     a.project,
							PkgPath:    packageName,
							Package:    packageName,
							ParentType: receiverType,
							Name:       funcName,
							Kind:       "method",
						},
						Definition: linage.CodeLocation{
							FilePath:   path,
							LineNumber: funcPos.Line,
						},
						Metadata: map[string]interface{}{},
						Writes:   []*linage.TouchPoint{},
						Reads:    []*linage.TouchPoint{},
					}
				}
			} else {
				// It's a function
				funcRef := linage.IdentityRef(fmt.Sprintf("%s:%s", packageName, funcName))
				dataPoint = &linage.DataPoint{
					Identity: linage.Identity{
						Ref:     funcRef,
						Module:  a.project,
						PkgPath: packageName,
						Package: packageName,
						Name:    funcName,
						Kind:    "function",
					},
					Definition: linage.CodeLocation{
						FilePath:   path,
						LineNumber: funcPos.Line,
					},
					Metadata: map[string]interface{}{},
					Writes:   []*linage.TouchPoint{},
					Reads:    []*linage.TouchPoint{},
				}
			}

			if dataPoint != nil {
				a.dataPoints[string(dataPoint.Identity.Ref)] = dataPoint

				// Process function parameters
				if d.Type.Params != nil {
					for _, param := range d.Type.Params.List {
						for _, paramName := range param.Names {
							// Get parameter type as string
							paramType := ""
							if param.Type != nil {
								paramType = typeToString(param.Type)
							}

							// Get parameter position
							paramPos := a.fset.Position(paramName.Pos())

							// Create a data point for the parameter
							var paramRef linage.IdentityRef
							if dataPoint.Identity.ParentType != "" {
								// Method parameter
								paramRef = linage.IdentityRef(fmt.Sprintf("%s:%s.%s:%s", packageName, dataPoint.Identity.ParentType, funcName, paramName.Name))
							} else {
								// Function parameter
								paramRef = linage.IdentityRef(fmt.Sprintf("%s:%s:%s", packageName, funcName, paramName.Name))
							}

							paramDataPoint := &linage.DataPoint{
								Identity: linage.Identity{
									Ref:        paramRef,
									Module:     a.project,
									PkgPath:    packageName,
									Package:    packageName,
									ParentType: dataPoint.Identity.ParentType,
									Name:       paramName.Name,
									Kind:       "parameter",
									Scope:      string(dataPoint.Identity.Ref),
								},
								Definition: linage.CodeLocation{
									FilePath:   path,
									LineNumber: paramPos.Line,
								},
								Metadata: map[string]interface{}{
									"type": paramType,
								},
								Writes: []*linage.TouchPoint{},
								Reads:  []*linage.TouchPoint{},
							}

							a.dataPoints[string(paramRef)] = paramDataPoint
						}
					}
				}
			}

		case *ast.GenDecl:
			// Process type declarations
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					// Get type name
					typeName := s.Name.Name

					// Check if it's a struct type
					if structType, ok := s.Type.(*ast.StructType); ok {
						// Process struct fields
						if structType.Fields != nil {
							for _, field := range structType.Fields.List {
								for _, fieldName := range field.Names {
									// Get field type as string
									fieldType := ""
									if field.Type != nil {
										fieldType = typeToString(field.Type)
									}

									// Get field position
									fieldPos := a.fset.Position(fieldName.Pos())

									// Create a data point for the field
									fieldRef := linage.MakeStructFieldIdentityRef(packageName, typeName, fieldName.Name)
									fieldDataPoint := &linage.DataPoint{
										Identity: linage.Identity{
											Ref:        fieldRef,
											Module:     a.project,
											PkgPath:    packageName,
											Package:    packageName,
											ParentType: typeName,
											Name:       fieldName.Name,
											Kind:       "field",
										},
										Definition: linage.CodeLocation{
											FilePath:   path,
											LineNumber: fieldPos.Line,
										},
										Metadata: map[string]interface{}{
											"type": fieldType,
										},
										Writes: []*linage.TouchPoint{},
										Reads:  []*linage.TouchPoint{},
									}

									a.dataPoints[string(fieldRef)] = fieldDataPoint
								}
							}
						}
					}

				case *ast.ValueSpec:
					// Process variable declarations
					for _, varName := range s.Names {
						// Get variable type as string
						varType := ""
						if s.Type != nil {
							varType = typeToString(s.Type)
						}

						// Get variable position
						varPos := a.fset.Position(varName.Pos())

						// Find the scope for this variable
						scope := a.findScopeForLine(varPos.Line)

						// Create a data point for the variable
						varRef := linage.IdentityRef(fmt.Sprintf("%s:%s", packageName, varName.Name))
						varDataPoint := &linage.DataPoint{
							Identity: linage.Identity{
								Ref:     varRef,
								Module:  a.project,
								PkgPath: packageName,
								Package: packageName,
								Name:    varName.Name,
								Kind:    "variable",
								Scope:   scope,
							},
							Definition: linage.CodeLocation{
								FilePath:   path,
								LineNumber: varPos.Line,
							},
							Metadata: map[string]interface{}{
								"type": varType,
							},
							Writes: []*linage.TouchPoint{},
							Reads:  []*linage.TouchPoint{},
						}

						a.dataPoints[string(varRef)] = varDataPoint
					}
				}
			}
		}
	}

	return nil
}

// processExpressions processes expressions (function calls, field access, etc.) in the Go AST
func (a *GolangAnalyzer) processExpressions(file *ast.File, path string) error {
	// Keep track of selector expressions that are part of assignment statements
	writeSelectors := make(map[ast.Expr]bool)

	// First pass: identify selector expressions that are part of assignment statements
	ast.Inspect(file, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			for _, lhs := range assignStmt.Lhs {
				if selExpr, ok := lhs.(*ast.SelectorExpr); ok {
					writeSelectors[selExpr] = true
				}
			}
		}
		return true
	})

	// Process expressions
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// This could be a field access or a method call
			if ident, ok := node.X.(*ast.Ident); ok {
				// Skip if this selector expression is part of an assignment statement (write operation)
				if !writeSelectors[node] {
					// Get the field/method name and the object being accessed
					fieldName := node.Sel.Name
					objectName := ident.Name

					// Get the position
					pos := a.fset.Position(node.Pos())

					// Find the scope for this expression
					scope := a.findScopeForLine(pos.Line)

					// Try to find the field in our data points
					for ref, dataPoint := range a.dataPoints {
						if dataPoint.Identity.Kind == "field" && dataPoint.Identity.Name == fieldName {
							// Add a read touch point for this field
							dataPoint.Reads = append(dataPoint.Reads, &linage.TouchPoint{
								CodeLocation: linage.CodeLocation{
									FilePath:   path,
									LineNumber: pos.Line,
								},
								Context: linage.TouchContext{
									Scope: scope,
								},
								Dependencies:          []linage.IdentityRef{},
								ConditionalExpression: fmt.Sprintf("%s.%s", objectName, fieldName),
							})

							a.dataPoints[ref] = dataPoint
						}
					}
				}
			}

		case *ast.AssignStmt:
			// Process assignments
			for _, lhs := range node.Lhs {
				if selExpr, ok := lhs.(*ast.SelectorExpr); ok {
					// This is a field assignment
					fieldName := selExpr.Sel.Name

					// Get the position
					pos := a.fset.Position(node.Pos())

					// Find the scope for this expression
					scope := a.findScopeForLine(pos.Line)

					// Try to find the field in our data points
					for ref, dataPoint := range a.dataPoints {
						if dataPoint.Identity.Kind == "field" && dataPoint.Identity.Name == fieldName {
							// Add a write touch point for this field
							dataPoint.Writes = append(dataPoint.Writes, &linage.TouchPoint{
								CodeLocation: linage.CodeLocation{
									FilePath:   path,
									LineNumber: pos.Line,
								},
								Context: linage.TouchContext{
									Scope: scope,
								},
								Dependencies:          []linage.IdentityRef{},
								ConditionalExpression: fmt.Sprintf("%s.%s", selExpr.X.(*ast.Ident).Name, fieldName),
							})

							a.dataPoints[ref] = dataPoint
						}
					}
				}
			}

		case *ast.CallExpr:
			// Process function/method calls
			var funcName string
			switch fun := node.Fun.(type) {
			case *ast.Ident:
				// Direct function call
				funcName = fun.Name
			case *ast.SelectorExpr:
				// Method call or function from another package
				funcName = fun.Sel.Name
			}

			if funcName != "" {
				// Get the position
				pos := a.fset.Position(node.Pos())

				// Find the scope for this expression
				scope := a.findScopeForLine(pos.Line)

				// Try to find the function or method in our data points
				for ref, dataPoint := range a.dataPoints {
					if (dataPoint.Identity.Kind == "function" || dataPoint.Identity.Kind == "method") && dataPoint.Identity.Name == funcName {
						// Add a read touch point for this function or method
						dataPoint.Reads = append(dataPoint.Reads, &linage.TouchPoint{
							CodeLocation: linage.CodeLocation{
								FilePath:   path,
								LineNumber: pos.Line,
							},
							Context: linage.TouchContext{
								Scope: scope,
							},
						})

						a.dataPoints[ref] = dataPoint
					}
				}
			}
		}
		return true
	})

	return nil
}

// findScopeForLine finds the most specific scope for a given line number
func (a *GolangAnalyzer) findScopeForLine(line int) string {
	// Find the most specific scope that contains the given line number
	var bestScope *linage.Scope
	var bestScopeDepth int

	for _, scope := range a.scopes {
		// Check if the scope contains the line
		if line >= scope.Start && line <= scope.End {
			// Calculate the depth of this scope in the hierarchy
			depth := 0
			parentID := scope.ParentID
			for parentID != "" {
				depth++
				if parent, ok := a.scopes[parentID]; ok {
					parentID = parent.ParentID
				} else {
					break
				}
			}

			// If this scope is more specific than the current best, update the best
			if bestScope == nil || depth > bestScopeDepth {
				bestScope = scope
				bestScopeDepth = depth
			}
		}
	}

	if bestScope != nil {
		return bestScope.ID
	}

	return "unknown"
}

// typeToString converts an AST type expression to a string representation
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return "[" + typeToString(t.Len) + "]" + typeToString(t.Elt)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.FuncType:
		return "func()"
	case *ast.ChanType:
		return "chan " + typeToString(t.Value)
	case *ast.BasicLit:
		return t.Value
	default:
		return "unknown"
	}
}
