package analyzer

import (
	"bytes"
	"fmt"
	"github.com/viant/linager/analyzer/info"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type visitor struct {
	fset           *token.FileSet
	pkg            *types.Package
	info           *types.Info
	project        string
	dataPoints     map[string]*info.DataPoint // Map from identity ref to DataPoint
	dataPointsKey  []string
	conditions     []string            // Stack of conditions
	conditionDeps  map[string][]string // Map from condition expressions to dependencies
	path           string              // File path
	functionStack  []string            // Stack of function calls
	methodContext  []methodInfo        // Stack of method context information
	callGraph      map[string][]string // Map of function calls to track transitive dependencies
	modInfo        *ModuleInfo         // go.mod information
	currentFunc    *types.Signature    // Current function being analyzed
	importedPkgs   map[string]*types.Package
	fullPkgPath    string            // Full package path for current file
	genericTypeMap map[string]string // Map from type parameter to concrete type
}

// methodInfo holds information about the current method context
type methodInfo struct {
	methodName string
	holderType string
}

// ModuleInfo holds information from go.mod
type ModuleInfo struct {
	Name         string
	Version      string
	Dependencies map[string]string // module -> version
}

// AnalyzeSourceCode analyzes the given Go source code and returns the data lineage information.
func AnalyzeSourceCode(source, project, path string) ([]*info.DataPoint, error) {
	fset := token.NewFileSet()

	// Parse the source code
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Create the type information
	conf := types.Config{Importer: importer.Default()}
	anInfo := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, err := conf.Check("main", fset, []*ast.File{file}, anInfo)
	if err != nil {
		// Don't fail completely on type errors, try to continue
		fmt.Printf("Type checking error (non-fatal): %v\n", err)
	}

	// Try to get module information from go.mod
	var modInfo *ModuleInfo
	if dir := filepath.Dir(path); dir != "" {
		modInfo, _ = LoadModuleInfo(dir)
	}

	// Now, traverse the AST and collect data lineage anInfo
	v := &visitor{
		fset:           fset,
		pkg:            pkg,
		project:        project,
		info:           anInfo,
		dataPoints:     make(map[string]*info.DataPoint),
		conditionDeps:  make(map[string][]string),
		path:           path,
		functionStack:  []string{},
		methodContext:  []methodInfo{},
		callGraph:      make(map[string][]string),
		modInfo:        modInfo,
		importedPkgs:   make(map[string]*types.Package),
		fullPkgPath:    pkg.Path(),
		genericTypeMap: make(map[string]string),
		dataPointsKey:  []string{},
	}

	// Collect imported packages
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if imported, err := importer.Default().Import(importPath); err == nil {
			v.importedPkgs[imported.Name()] = imported
		}
	}

	ast.Walk(v, file)

	// Process dependencies
	v.establishDependencies()

	// Build the return data points list in sorted order
	var points = make([]*info.DataPoint, 0)

	// First, add struct fields in expected order
	for _, key := range v.dataPointsKey {
		if dp, ok := v.dataPoints[key]; ok {
			points = append(points, dp)
		}
	}

	return points, nil
}

// LoadModuleInfo loads go.mod information from the given directory
func LoadModuleInfo(dir string) (*ModuleInfo, error) {
	modInfo := &ModuleInfo{
		Dependencies: make(map[string]string),
	}

	// Find go.mod in parent directories
	goModPath := findGoMod(dir)
	if goModPath == "" {
		return modInfo, nil
	}

	// Read go.mod content
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return modInfo, err
	}

	// Basic parsing of go.mod
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modInfo.Name = strings.TrimSpace(strings.TrimPrefix(line, "module "))
		} else if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				modInfo.Dependencies[parts[1]] = parts[2]
			}
		}
	}

	return modInfo, nil
}

// findGoMod searches up the directory tree for a go.mod file
func findGoMod(dir string) string {
	for {
		path := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Get the parent directory
		parent := filepath.Dir(dir)
		// Stop if we're at the root or the parent didn't change
		if parent == dir || parent == "." || parent == "/" {
			break
		}
		dir = parent
	}
	return ""
}

func (v *visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch node := n.(type) {
	case *ast.AssignStmt:
		v.handleAssignment(node)

	case *ast.IfStmt:
		// Handle conditions
		condStr := v.exprToString(node.Cond)
		v.conditions = append(v.conditions, condStr)

		// Visit the condition expression to capture variable reads and their dependencies
		v.handleExpression(node.Cond, false)

		// Record all dependencies in this condition for later use
		v.conditionDeps[condStr] = v.getDataPointsFromExpr(node.Cond)

		// Visit inner statements
		ast.Walk(v, node.Body)
		// Handle Else branch if it exists
		if node.Else != nil {
			// For else blocks, use the negation of the condition
			v.conditions = v.conditions[:len(v.conditions)-1]
			negCondStr := fmt.Sprintf("!(%s)", condStr)
			v.conditions = append(v.conditions, negCondStr)
			v.conditionDeps[negCondStr] = v.conditionDeps[condStr] // Same dependencies for negated condition
			ast.Walk(v, node.Else)
		}
		// Pop condition
		v.conditions = v.conditions[:len(v.conditions)-1]
		// Skip further traversal as we've handled the body
		return nil

	case *ast.TypeSpec:
		v.handleTypeSpec(node)

	case *ast.FuncDecl:
		v.handleFuncDecl(node)
		return nil

	case *ast.ReturnStmt:
		v.handleReturnStmt(node)

	case *ast.GenDecl:
		// Process variable declarations
		for _, spec := range node.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for i, name := range valueSpec.Names {
					if name.Name == "_" {
						continue // Skip underscore identifiers
					}

					var valueExpr ast.Expr
					if i < len(valueSpec.Values) {
						valueExpr = valueSpec.Values[i]
					}

					dp := v.getDataPoint(name.Name, name)
					if valueSpec.Type != nil {
						dp.Identity.Kind = v.exprToString(valueSpec.Type)
					}

					// Mark as written
					v.updateTouchPoints(dp, true, name)

					// If there's an initial value, process it
					if valueExpr != nil {
						v.handleExpression(valueExpr, false)

						// Track dependency
						rhsDeps := v.getDataPointsFromExpr(valueExpr)
						if len(dp.Writes) > 0 {
							writePoint := dp.Writes[len(dp.Writes)-1]
							sort.Strings(rhsDeps)
							for _, dep := range rhsDeps {
								if dep != string(dp.Identity.Ref) { // Avoid self-references
									writePoint.Dependencies = append(writePoint.Dependencies, info.IdentityRef(dep))
								}
							}
						}
					}
				}
			}
		}
	}

	return v
}

// handleAssignment processes assignment statements for data lineage tracking
func (v *visitor) handleAssignment(node *ast.AssignStmt) {
	if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
		// Simple assignment: lhs = rhs
		lhsExpr := node.Lhs[0]
		rhsExpr := node.Rhs[0]

		// Handle the LHS (writer)
		lhsDP, lhsTP := v.handleExpression(lhsExpr, true)

		// Skip underscore identifier
		if id, isIdent := lhsExpr.(*ast.Ident); isIdent && id.Name == "_" {
			v.handleExpression(rhsExpr, false) // Still process the rhs for reads
			return
		}

		// Handle the RHS (reader)
		v.handleExpression(rhsExpr, false)

		// Track dependency: lhs depends on rhs
		if lhsDP != nil && lhsTP != nil {
			lhsRef := string(lhsDP.Identity.Ref)
			// Get all dependencies from the right-hand side expression
			rhsDeps := v.getDataPointsFromExpr(rhsExpr)
			sort.Strings(rhsDeps)
			for _, dep := range rhsDeps {
				if dep != lhsRef { // Avoid self-references
					if !containsIdentityRef(lhsTP.Dependencies, info.IdentityRef(dep)) {
						lhsTP.Dependencies = append(lhsTP.Dependencies, info.IdentityRef(dep))
					}
				}
			}
		}
	} else {
		// Multiple assignments: a, b = c, d
		for i, lhs := range node.Lhs {
			// Skip underscore identifier
			if id, isIdent := lhs.(*ast.Ident); isIdent && id.Name == "_" {
				continue
			}

			// Handle LHS
			lhsDP, lhsTP := v.handleExpression(lhs, true)
			if lhsDP == nil || lhsTP == nil {
				continue
			}
			lhsRef := string(lhsDP.Identity.Ref)

			// Handle RHS if available
			if i < len(node.Rhs) {
				rhs := node.Rhs[i]
				v.handleExpression(rhs, false)

				// Track dependency
				rhsDeps := v.getDataPointsFromExpr(rhs)
				sort.Strings(rhsDeps)
				for _, dep := range rhsDeps {
					if dep != lhsRef { // Avoid self-references
						if !containsIdentityRef(lhsTP.Dependencies, info.IdentityRef(dep)) {
							lhsTP.Dependencies = append(lhsTP.Dependencies, info.IdentityRef(dep))
						}
					}
				}
			}
		}
	}
}

// handleTypeSpec processes type declarations including structs and interfaces
func (v *visitor) handleTypeSpec(node *ast.TypeSpec) {
	structType, isStruct := node.Type.(*ast.StructType)
	if !isStruct {
		// Not a struct, maybe interface or other type
		return
	}

	// Record struct information
	structName := node.Name.Name
	packagePath := v.fullPkgPath

	// Check if this is a generic type
	typeParams := ""
	var typeParamsList []string
	if node.TypeParams != nil && len(node.TypeParams.List) > 0 {
		var params []string
		for _, p := range node.TypeParams.List {
			for _, n := range p.Names {
				params = append(params, n.Name)
				typeParamsList = append(typeParamsList, n.Name)
			}
		}
		typeParams = "[" + strings.Join(params, ", ") + "]"
	}

	// Process struct fields
	for _, field := range structType.Fields.List {
		// Handle embedded fields which might not have names
		if len(field.Names) == 0 {
			// This is an embedded field
			fieldType := v.exprToString(field.Type)

			// Remove pointer if present
			embeddedType := strings.TrimPrefix(fieldType, "*")

			// Create an identity for the embedded field
			ref := info.MakeStructFieldIdentityRef(packagePath, structName+typeParams, embeddedType)

			dp := &info.DataPoint{
				Identity: info.Identity{
					Ref:        ref,
					PkgPath:    packagePath,
					Package:    packagePath,
					HolderType: structName,
					Name:       embeddedType,
					Kind:       fieldType,
				},
				Definition: info.CodeLocation{
					FilePath:   v.path,
					LineNumber: v.fset.Position(field.Pos()).Line,
				},
				Metadata: map[string]interface{}{
					"embedded": true,
				},
			}

			// Store struct tags if present
			if field.Tag != nil {
				tagText := field.Tag.Value
				tagText = tagText[1 : len(tagText)-1] // Remove outer quotes
				dp.Metadata["tags"] = tagText
			}

			v.dataPoints[string(ref)] = dp
			v.dataPointsKey = append(v.dataPointsKey, string(ref))
			continue
		}

		// Regular named fields
		for _, name := range field.Names {
			fieldName := name.Name
			ref := info.MakeStructFieldIdentityRef(packagePath, structName+typeParams, fieldName)

			fieldType := v.exprToString(field.Type)

			// For generic type parameters, mark them
			if typeParamsList != nil {
				for _, param := range typeParamsList {
					if fieldType == param {
						fieldType = fmt.Sprintf("TypeParam(%s)", param)
						break
					}
				}
			}

			var tagText string
			if field.Tag != nil {
				tagText = field.Tag.Value
				// Remove the outer quotes
				tagText = tagText[1 : len(tagText)-1]
			}

			dp := &info.DataPoint{
				Identity: info.Identity{
					Ref:        ref,
					PkgPath:    packagePath,
					Package:    packagePath,
					HolderType: structName,
					Name:       fieldName,
					Kind:       fieldType,
				},
				Definition: info.CodeLocation{
					FilePath:   v.path,
					LineNumber: v.fset.Position(name.Pos()).Line,
				},
				Metadata: map[string]interface{}{},
			}

			// Store struct tags if present
			if tagText != "" {
				dp.Metadata["tags"] = tagText
			}

			v.dataPoints[string(ref)] = dp
			v.dataPointsKey = append(v.dataPointsKey, string(ref))
		}
	}
}

// handleFuncDecl processes function declarations
func (v *visitor) handleFuncDecl(node *ast.FuncDecl) {
	funcName := node.Name.Name

	// For type checking
	obj := v.info.ObjectOf(node.Name)
	if fnObj, ok := obj.(*types.Func); ok {
		v.currentFunc = fnObj.Type().(*types.Signature)
	}

	// Push function name onto stack for call graph tracking
	v.functionStack = append(v.functionStack, funcName)

	// Process receiver if present (methods)
	if node.Recv != nil && len(node.Recv.List) > 0 {
		recvField := node.Recv.List[0]
		// Extract receiver type for method calls tracking
		recvType := v.exprToString(recvField.Type)
		// Handle pointer receivers
		baseType := recvType
		if strings.HasPrefix(recvType, "*") {
			baseType = recvType[1:]
		}

		// Push method context
		methodCtx := methodInfo{
			methodName: funcName,
			holderType: baseType,
		}
		v.methodContext = append(v.methodContext, methodCtx)

		// Record this method for call graph
		methodName := fmt.Sprintf("%s.%s", baseType, funcName)
		v.callGraph[methodName] = make([]string, 0)

		// Also process receiver as a parameter
		if len(recvField.Names) > 0 {
			recvName := recvField.Names[0].Name
			recvDP := v.getDataPoint(recvName, recvField)
			recvDP.Identity.Kind = recvType

			// The receiver is both read and written
			v.updateTouchPoints(recvDP, true, recvField)
			v.updateTouchPoints(recvDP, false, recvField)
		}
	} else {
		// Push function context
		v.methodContext = append(v.methodContext, methodInfo{
			methodName: funcName,
			holderType: "",
		})
	}

	// Process function parameters
	if node.Type.Params != nil {
		for _, paramField := range node.Type.Params.List {
			// For each parameter name
			for _, name := range paramField.Names {
				paramName := name.Name
				paramType := v.exprToString(paramField.Type)

				// Create DataPoint for parameter
				dp := &info.DataPoint{
					Identity: info.Identity{
						Ref:  info.IdentityRef(paramName),
						Name: paramName,
						Kind: paramType,
					},
					Definition: info.CodeLocation{
						FilePath:   v.path,
						LineNumber: v.fset.Position(name.Pos()).Line,
					},
					Metadata: map[string]interface{}{
						"parameter": true,
					},
				}

				// Parameters are considered written (when function is called)
				// and read (within the function)
				v.updateTouchPoints(dp, true, name)
				v.updateTouchPoints(dp, false, name)

				v.dataPoints[string(dp.Identity.Ref)] = dp
				v.dataPointsKey = append(v.dataPointsKey, string(dp.Identity.Ref))
			}
		}
	}

	// Process function return values if named
	if node.Type.Results != nil {
		for _, resultField := range node.Type.Results.List {
			// For each named return value
			for _, name := range resultField.Names {
				returnName := name.Name
				returnType := v.exprToString(resultField.Type)

				// Create DataPoint for named return value
				dp := &info.DataPoint{
					Identity: info.Identity{
						Ref:  info.IdentityRef(returnName),
						Name: returnName,
						Kind: returnType,
					},
					Definition: info.CodeLocation{
						FilePath:   v.path,
						LineNumber: v.fset.Position(name.Pos()).Line,
					},
					Metadata: map[string]interface{}{
						"returnValue": true,
					},
				}

				// Named return values are considered written within the function
				v.updateTouchPoints(dp, true, name)

				v.dataPoints[string(dp.Identity.Ref)] = dp
				v.dataPointsKey = append(v.dataPointsKey, string(dp.Identity.Ref))
			}
		}
	}

	if node.Body != nil {
		ast.Walk(v, node.Body)
	}

	// Pop function name from stack
	if len(v.functionStack) > 0 {
		v.functionStack = v.functionStack[:len(v.functionStack)-1]
	}

	// Pop method context
	if len(v.methodContext) > 0 {
		v.methodContext = v.methodContext[:len(v.methodContext)-1]
	}

	// Reset current function
	v.currentFunc = nil
}

// handleReturnStmt processes return statements for data flow tracking
func (v *visitor) handleReturnStmt(node *ast.ReturnStmt) {
	if len(v.functionStack) == 0 {
		return
	}

	currentFunc := v.functionStack[len(v.functionStack)-1]

	// For each return value, track its dependencies
	for idx, expr := range node.Results {
		v.handleExpression(expr, false)

		// Get all variables/fields referenced in the return expression
		deps := v.getDataPointsFromExpr(expr)

		// Add each dependency to the call graph for this function
		for _, dep := range deps {
			v.callGraph[currentFunc] = append(v.callGraph[currentFunc], dep)
		}

		// If we have function signature information, try to associate return values
		// with named returns
		if v.currentFunc != nil && v.currentFunc.Results() != nil {
			resultCount := v.currentFunc.Results().Len()
			if idx < resultCount {
				// If return value has a name, associate the dependencies
				resultVar := v.currentFunc.Results().At(idx)
				if resultVar.Name() != "" {
					dp := v.getDataPoint(resultVar.Name(), expr)
					if len(dp.Writes) > 0 {
						writePoint := dp.Writes[len(dp.Writes)-1]
						sort.Strings(deps)
						for _, dep := range deps {
							if dep != string(dp.Identity.Ref) { // Avoid self-references
								writePoint.Dependencies = append(writePoint.Dependencies, info.IdentityRef(dep))
							}
						}
					}
				}
			}
		}
	}
}

// handleExpression processes an expression and returns the associated DataPoint and TouchPoint (if any)
func (v *visitor) handleExpression(expr ast.Expr, isWrite bool) (*info.DataPoint, *info.TouchPoint) {
	if expr == nil {
		return nil, nil
	}

	switch e := expr.(type) {
	case *ast.Ident:
		varName := e.Name
		if varName == "_" {
			return nil, nil // Skip underscore identifiers
		}
		dp := v.getDataPoint(varName, e)
		tp := v.updateTouchPoints(dp, isWrite, expr)
		return dp, tp

	case *ast.SelectorExpr:
		// Handle struct field access or package member access

		// Try to resolve the package path and struct name for the field
		id := v.resolveStructFieldID(e)
		if id == "" {
			// Fall back to basic format if we can't resolve ID
			id = v.exprToString(e)
		}

		dp := v.getDataPoint(id, e)
		tp := v.updateTouchPoints(dp, isWrite, expr)

		// For field selections, also process the container expression (the struct/receiver)
		v.handleExpression(e.X, false)
		return dp, tp

	case *ast.CallExpr:
		// Functions call, handle function and arguments
		funcDP, _ := v.handleExpression(e.Fun, false)

		// Track call graph information
		funcName := v.exprToString(e.Fun)
		callExprStr := v.exprToString(e)

		// Handle arguments
		for _, arg := range e.Args {
			v.handleExpression(arg, false)

			// Track argument dependencies
			argDeps := v.getDataPointsFromExpr(arg)
			for _, argDep := range argDeps {
				v.callGraph[funcName] = append(v.callGraph[funcName], argDep)
			}
		}

		// If we have function call graph data, use it for dependencies
		if deps, ok := v.callGraph[funcName]; ok {
			v.callGraph[callExprStr] = append(v.callGraph[callExprStr], deps...)
		}

		return funcDP, nil

	case *ast.BinaryExpr:
		// Handle both sides of binary expressions
		v.handleExpression(e.X, false)
		v.handleExpression(e.Y, false)

		// For expressions like a + b, track that result depends on both operands
		exprStr := v.exprToString(e)

		// Track dependencies from both sides
		xDeps := v.getDataPointsFromExpr(e.X)
		yDeps := v.getDataPointsFromExpr(e.Y)

		v.callGraph[exprStr] = append(v.callGraph[exprStr], xDeps...)
		v.callGraph[exprStr] = append(v.callGraph[exprStr], yDeps...)

		return nil, nil // Binary expressions themselves don't have a DataPoint

	case *ast.UnaryExpr:
		return v.handleExpression(e.X, false)

	case *ast.IndexExpr:
		v.handleExpression(e.X, false)
		v.handleExpression(e.Index, false)

		// Track data flow for indexed access
		exprStr := v.exprToString(e)
		arrayDeps := v.getDataPointsFromExpr(e.X)
		v.callGraph[exprStr] = append(v.callGraph[exprStr], arrayDeps...)

	case *ast.ParenExpr:
		return v.handleExpression(e.X, isWrite)

	case *ast.TypeAssertExpr:
		return v.handleExpression(e.X, false)

	case *ast.CompositeLit:
		// Handle composite literals (e.g., structs)
		var structType string
		var packagePath string

		// Extract type information
		switch typExpr := e.Type.(type) {
		case *ast.Ident:
			structType = typExpr.Name
			structObj := v.info.Uses[typExpr]
			if structObj != nil && structObj.Pkg() != nil {
				packagePath = structObj.Pkg().Path()
			} else {
				packagePath = v.fullPkgPath
			}
		case *ast.SelectorExpr:
			// Handle package-qualified types: pkg.Type
			structType = typExpr.Sel.Name
			if x, ok := typExpr.X.(*ast.Ident); ok {
				packageName := x.Name
				// Try to resolve the full package path
				if pkg, exists := v.importedPkgs[packageName]; exists {
					packagePath = pkg.Path()
				} else if obj := v.info.Uses[x]; obj != nil && obj.Pkg() != nil {
					packagePath = obj.Pkg().Path()
				} else {
					packagePath = packageName
				}
			}
		case *ast.IndexExpr:
			// Generic type instantiation: Container[T]
			if baseIdent, ok := typExpr.X.(*ast.Ident); ok {
				structType = baseIdent.Name
				structObj := v.info.Uses[baseIdent]
				if structObj != nil && structObj.Pkg() != nil {
					packagePath = structObj.Pkg().Path()
				} else {
					packagePath = v.fullPkgPath
				}

				// Track type parameter
				typeArgStr := v.exprToString(typExpr.Index)
				structType = fmt.Sprintf("%s[%s]", structType, typeArgStr)
			}
		}

		// Process fields in composite literal
		for _, elt := range e.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				// Named field initialization: Field: value
				if keyIdent, ok := kv.Key.(*ast.Ident); ok {
					fieldName := keyIdent.Name
					ref := string(info.MakeStructFieldIdentityRef(packagePath, structType, fieldName))

					dp := v.getDataPoint(ref, kv.Value)
					tp := v.updateTouchPoints(dp, true, kv) // Writing to field

					// Process the value expression
					v.handleExpression(kv.Value, false)

					// Track dependency
					rhsDeps := v.getDataPointsFromExpr(kv.Value)
					if tp != nil {
						sort.Strings(rhsDeps)
						for _, dep := range rhsDeps {
							if dep != ref { // Avoid self-references
								if !containsIdentityRef(tp.Dependencies, info.IdentityRef(dep)) {
									tp.Dependencies = append(tp.Dependencies, info.IdentityRef(dep))
								}
							}
						}
					}
				}
			} else {
				// Positional initialization or other elements
				v.handleExpression(elt, false)
			}
		}

	case *ast.KeyValueExpr:
		// Key-value pair in composite literals
		v.handleExpression(e.Key, false)
		v.handleExpression(e.Value, false)

	case *ast.StarExpr:
		// Pointer dereference
		return v.handleExpression(e.X, isWrite)

	case *ast.SliceExpr:
		// Slice expression: arr[low:high:max]
		v.handleExpression(e.X, false)
		if e.Low != nil {
			v.handleExpression(e.Low, false)
		}
		if e.High != nil {
			v.handleExpression(e.High, false)
		}
		if e.Max != nil {
			v.handleExpression(e.Max, false)
		}

		// Track dependency
		exprStr := v.exprToString(e)
		sliceDeps := v.getDataPointsFromExpr(e.X)
		v.callGraph[exprStr] = append(v.callGraph[exprStr], sliceDeps...)

	case *ast.BasicLit:
		// Literals have no dependencies
		return nil, nil
	}

	return nil, nil
}

// getDataPointsFromExpr extracts all identity refs from an expression
func (v *visitor) getDataPointsFromExpr(expr ast.Expr) []string {
	var results []string

	switch e := expr.(type) {
	case *ast.Ident:
		// Simple variable
		if e.Name != "_" { // Skip underscore identifiers
			dp := v.getDataPoint(e.Name, e)
			if dp != nil {
				results = append(results, string(dp.Identity.Ref))
			}
		}
	case *ast.SelectorExpr:
		// Struct field access: x.y or package.member
		dp, _ := v.handleExpression(e, false)
		if dp != nil {
			results = append(results, string(dp.Identity.Ref))
		}

		// Also include dependencies from X
		xDeps := v.getDataPointsFromExpr(e.X)
		results = append(results, xDeps...)
	case *ast.BinaryExpr:
		// Both sides of binary expressions
		results = append(results, v.getDataPointsFromExpr(e.X)...)
		results = append(results, v.getDataPointsFromExpr(e.Y)...)
	case *ast.CallExpr:
		// Functions calls
		funcName := v.exprToString(e.Fun)
		results = append(results, funcName)

		// Include arguments as dependencies
		for _, arg := range e.Args {
			results = append(results, v.getDataPointsFromExpr(arg)...)
		}

		// Include function call graph dependencies
		if deps, ok := v.callGraph[funcName]; ok {
			results = append(results, deps...)
		}
	case *ast.ParenExpr:
		// Parenthesized expressions
		results = append(results, v.getDataPointsFromExpr(e.X)...)
	case *ast.UnaryExpr:
		// Unary expressions
		results = append(results, v.getDataPointsFromExpr(e.X)...)
	case *ast.IndexExpr:
		// Array indexing: arr[i]
		results = append(results, v.getDataPointsFromExpr(e.X)...)
		results = append(results, v.getDataPointsFromExpr(e.Index)...)
	case *ast.CompositeLit:
		// Composite literals
		for _, elt := range e.Elts {
			results = append(results, v.getDataPointsFromExpr(elt)...)
		}
	case *ast.KeyValueExpr:
		// Key-value pairs
		results = append(results, v.getDataPointsFromExpr(e.Key)...)
		results = append(results, v.getDataPointsFromExpr(e.Value)...)
	case *ast.StarExpr:
		// Pointer dereference
		results = append(results, v.getDataPointsFromExpr(e.X)...)
	case *ast.SliceExpr:
		// Slice expressions
		results = append(results, v.getDataPointsFromExpr(e.X)...)
		if e.Low != nil {
			results = append(results, v.getDataPointsFromExpr(e.Low)...)
		}
		if e.High != nil {
			results = append(results, v.getDataPointsFromExpr(e.High)...)
		}
		if e.Max != nil {
			results = append(results, v.getDataPointsFromExpr(e.Max)...)
		}
	case *ast.BasicLit:
		// Literals have no dependencies
		// Do nothing
	}

	// Remove duplicates
	uniqueResults := make(map[string]struct{})
	for _, r := range results {
		uniqueResults[r] = struct{}{}
	}
	results = make([]string, 0, len(uniqueResults))
	for r := range uniqueResults {
		results = append(results, r)
	}

	return results
}

// resolveStructFieldID attempts to get the proper ID for a struct field
func (v *visitor) resolveStructFieldID(expr *ast.SelectorExpr) string {
	if selection, ok := v.info.Selections[expr]; ok {
		if selection.Obj() != nil {
			var packagePath string
			if selection.Obj().Pkg() != nil {
				packagePath = selection.Obj().Pkg().Path()
			} else {
				packagePath = v.fullPkgPath
			}

			// Get the struct type
			receiverType := selection.Recv().String()
			// Extract just the struct name
			structName := receiverType
			if strings.HasPrefix(receiverType, "*") {
				structName = receiverType[1:]
			}
			if idx := strings.LastIndex(structName, "."); idx >= 0 {
				structName = structName[idx+1:]
			}

			fieldName := expr.Sel.Name
			return string(info.MakeStructFieldIdentityRef(packagePath, structName, fieldName))
		}
	}

	// If selection info isn't available, try a basic approach
	if ident, ok := expr.X.(*ast.Ident); ok {
		// Look up the type information for the identifier
		if obj := v.info.ObjectOf(ident); obj != nil {
			if typeObj, ok := obj.Type().(*types.Named); ok {
				// Get package path and type name
				packagePath := ""
				if typeObj.Obj().Pkg() != nil {
					packagePath = typeObj.Obj().Pkg().Path()
				} else {
					packagePath = v.fullPkgPath
				}
				structName := typeObj.Obj().Name()
				fieldName := expr.Sel.Name
				return string(info.MakeStructFieldIdentityRef(packagePath, structName, fieldName))
			}
		}
	}

	return ""
}

// getDataPoint creates or retrieves a DataPoint for the given identity reference
func (v *visitor) getDataPoint(varName string, expr ast.Node) *info.DataPoint {
	// Resolve identity ref from varName and expr
	var identityRef info.IdentityRef
	// For struct fields, varName would be the identity ref
	if strings.Contains(varName, ":") {
		identityRef = info.IdentityRef(varName)
	} else {
		// For variables, use the variable name as identity ref
		identityRef = info.IdentityRef(varName)
	}

	// Check if this datapoint already exists
	dp, exists := v.dataPoints[string(identityRef)]
	if !exists {
		// Create new DataPoint
		kindStr := v.getTypeOf(expr)

		// Parse the field name from the identity reference if it's a struct field
		var name, holderType, pkgPath string

		// Check if this is a struct field reference
		identityParts := strings.Split(string(identityRef), ":")
		if len(identityParts) == 3 && identityParts[0] != "" { // format like "pkgpath:Struct:Field"
			// Extract the field name and package from the ref
			pkgPath = identityParts[0]
			holderType = identityParts[1]
			name = identityParts[2]
		} else {
			// For regular variables or other identities
			name = varName
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				holderType = name[:idx]
				name = name[idx+1:]
			}
		}

		dp = &info.DataPoint{
			Identity: info.Identity{
				Ref:        identityRef,
				PkgPath:    pkgPath,
				Package:    pkgPath,
				Name:       name,
				HolderType: holderType,
				Kind:       kindStr,
			},
			Definition: info.CodeLocation{
				FilePath:   v.path,
				LineNumber: v.fset.Position(expr.Pos()).Line,
			},
			Metadata: map[string]interface{}{},
			Writes:   []*info.TouchPoint{},
			Reads:    []*info.TouchPoint{},
		}

		v.dataPoints[string(identityRef)] = dp
		v.dataPointsKey = append(v.dataPointsKey, string(identityRef))
	}

	return dp
}

// exprToString converts an AST expression to its string representation
func (v *visitor) exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	var buf bytes.Buffer
	err := printer.Fprint(&buf, v.fset, expr)
	if err != nil {
		return ""
	}
	return buf.String()
}

// getTypeOf attempts to determine the type of an expression
func (v *visitor) getTypeOf(node ast.Node) string {
	if expr, ok := node.(ast.Expr); ok {
		tv, ok := v.info.Types[expr]
		if ok {
			return tv.Type.String()
		}

		// Try to get type info from the info.Selections map for selector expressions
		if se, ok := expr.(*ast.SelectorExpr); ok {
			if selection, ok := v.info.Selections[se]; ok {
				return selection.Type().String()
			}
		}

		// For identifiers, try to get the object and its type
		if id, ok := expr.(*ast.Ident); ok {
			if obj := v.info.ObjectOf(id); obj != nil {
				return obj.Type().String()
			}
		}

		return v.inferTypeFromContext(expr)
	}

	return ""
}

// inferTypeFromContext tries to infer the type from context when type info is missing
func (v *visitor) inferTypeFromContext(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float64"
		case token.STRING:
			return "string"
		case token.CHAR:
			return "rune"
		case token.IMAG:
			return "complex128"
		}
	case *ast.Ident:
		if e.Obj != nil {
			switch decl := e.Obj.Decl.(type) {
			case *ast.ValueSpec:
				if len(decl.Values) > 0 {
					// This is a const or var declaration
					return v.getTypeOf(decl.Values[0])
				}
				if decl.Type != nil {
					return v.exprToString(decl.Type)
				}
			case *ast.Field:
				if decl.Type != nil {
					return v.exprToString(decl.Type)
				}
			case *ast.AssignStmt:
				// Find the position of this identifier in the LHS
				for i, lhs := range decl.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Name == e.Name {
						if i < len(decl.Rhs) {
							return v.getTypeOf(decl.Rhs[i])
						}
					}
				}
			}
		}
	case *ast.CompositeLit:
		if e.Type != nil {
			return v.exprToString(e.Type)
		}
	case *ast.CallExpr:
		// Try to infer type from function return type
		funcName := v.exprToString(e.Fun)
		if strings.HasPrefix(funcName, "func(") {
			parts := strings.Split(funcName, ") ")
			if len(parts) > 1 {
				return strings.TrimPrefix(parts[1], "(")
			}
		}
	}
	return ""
}

// LoadProject loads a Go project using go/packages
func LoadProject(dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	return pkgs, nil
}

// AnalyzePackages analyzes all packages from a loaded project
func AnalyzePackages(pkgs []*packages.Package) ([]*info.DataPoint, error) {
	allDataPoints := make([]*info.DataPoint, 0)

	for _, pkg := range pkgs {
		// Process each file in the package
		for i, file := range pkg.Syntax {
			if i >= len(pkg.GoFiles) {
				continue // Safety check
			}

			fset := pkg.Fset
			fileName := pkg.GoFiles[i]

			// Create visitor for each file
			v := &visitor{
				fset:           fset,
				pkg:            pkg.Types,
				info:           pkg.TypesInfo,
				dataPoints:     make(map[string]*info.DataPoint),
				path:           fileName,
				callGraph:      make(map[string][]string),
				functionStack:  []string{},
				methodContext:  []methodInfo{},
				importedPkgs:   make(map[string]*types.Package),
				fullPkgPath:    pkg.PkgPath,
				genericTypeMap: make(map[string]string),
				dataPointsKey:  []string{},
			}

			// Collect imported packages
			for _, imp := range pkg.Imports {
				v.importedPkgs[imp.Name] = imp.Types
			}

			// Walk the AST
			ast.Walk(v, file)

			// Establish dependencies
			v.establishDependencies()

			// Collect data points
			for _, dp := range v.dataPoints {
				allDataPoints = append(allDataPoints, dp)
			}
		}
	}

	return allDataPoints, nil
}
