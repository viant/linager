package analyzer

import (
	"github.com/viant/linager/analyzer/info"
	"go/ast"
	"go/token"
	"strings"
)

// establishDependencies builds the dependency graph from collected data links
func (v *visitor) establishDependencies() {
	// Apply transitive dependencies from call graph
	v.applyTransitiveDependencies()

	// Remove duplicate dependencies
	v.removeDuplicateDependencies()
}

// updateTouchPoints adds read/write information to a DataPoint
func (v *visitor) updateTouchPoints(dp *info.DataPoint, isWrite bool, expr interface{}) *info.TouchPoint {
	condition := v.currentCondition()

	// Get position information from the AST node
	var pos token.Position
	var exprStr string
	switch e := expr.(type) {
	case ast.Node:
		pos = v.fset.Position(e.Pos())
		if astExpr, ok := e.(ast.Expr); ok {
			exprStr = v.exprToString(astExpr)
		} else {
			// For non-expression nodes, just use a reasonable default
			exprStr = "expr"
		}
	default:
		// For other types, we cannot get position information
		return nil
	}

	lineNumber := pos.Line
	columnEnd := pos.Column + len(exprStr)

	// Create the TouchContext from the current function/method context
	context := info.TouchContext{}
	if len(v.functionStack) > 0 {
		context.Function = v.functionStack[len(v.functionStack)-1]
	}

	if len(v.methodContext) > 0 {
		currentCtx := v.methodContext[len(v.methodContext)-1]
		if currentCtx.holderType != "" {
			// This is a method
			context.Method = currentCtx.methodName
			context.HolderType = currentCtx.holderType
		} else if context.Function == "" {
			// This is a function (if not already set from functionStack)
			context.Function = currentCtx.methodName
		}
	}

	touchPoint := &info.TouchPoint{
		CodeLocation: info.CodeLocation{
			FilePath:    v.path,
			LineNumber:  lineNumber,
			ColumnStart: pos.Column,
			ColumnEnd:   columnEnd,
		},
		Context: context,
	}
	if condition != "" {
		touchPoint.ConditionalExpression = condition
	}

	if isWrite {
		dp.Writes = append(dp.Writes, touchPoint)
	} else {
		dp.Reads = append(dp.Reads, touchPoint)
	}

	return touchPoint
}

// applyTransitiveDependencies follows the call graph to find all transitive dependencies
func (v *visitor) applyTransitiveDependencies() {
	// Keep track of processed functions to avoid cycles
	processed := make(map[string]bool)

	// Helper function to recursively find all dependencies
	var findAllDependencies func(funcName string) []string
	findAllDependencies = func(funcName string) []string {
		if processed[funcName] {
			return nil // Already processed this function, avoid cycles
		}
		processed[funcName] = true

		result := make([]string, 0)
		// Get direct dependencies
		directDeps := v.callGraph[funcName]
		result = append(result, directDeps...)

		// Get transitive dependencies
		for _, dep := range directDeps {
			// If the dependency is a function, get its dependencies
			if _, ok := v.callGraph[dep]; ok {
				transitiveDeps := findAllDependencies(dep)
				result = append(result, transitiveDeps...)
			}
		}

		return result
	}

	// For each function, find all its transitive dependencies
	for funcName := range v.callGraph {
		processed = make(map[string]bool) // Reset for each function
		allDeps := findAllDependencies(funcName)

		// Update the call graph with all dependencies
		v.callGraph[funcName] = allDeps
	}
}

// removeDuplicateDependencies removes duplicate dependencies from touch points
func (v *visitor) removeDuplicateDependencies() {
	for _, dp := range v.dataPoints {
		// Deduplicate dependencies in write points
		for _, writePoint := range dp.Writes {
			if len(writePoint.Dependencies) > 0 {
				uniqueDeps := make([]info.IdentityRef, 0, len(writePoint.Dependencies))
				seen := make(map[info.IdentityRef]bool)

				for _, dep := range writePoint.Dependencies {
					if !seen[dep] {
						seen[dep] = true
						uniqueDeps = append(uniqueDeps, dep)
					}
				}
				writePoint.Dependencies = uniqueDeps
			}
		}

		// Deduplicate dependencies in read points
		for _, readPoint := range dp.Reads {
			if len(readPoint.Dependencies) > 0 {
				uniqueDeps := make([]info.IdentityRef, 0, len(readPoint.Dependencies))
				seen := make(map[info.IdentityRef]bool)

				for _, dep := range readPoint.Dependencies {
					if !seen[dep] {
						seen[dep] = true
						uniqueDeps = append(uniqueDeps, dep)
					}
				}
				readPoint.Dependencies = uniqueDeps
			}
		}
	}
}

// Helper function to check if an identity ref is in a slice
func containsIdentityRef(slice []info.IdentityRef, ref info.IdentityRef) bool {
	for _, r := range slice {
		if r == ref {
			return true
		}
	}
	return false
}

// currentCondition returns the current condition stack as a string
func (v *visitor) currentCondition() string {
	if len(v.conditions) == 0 {
		return ""
	}
	return strings.Join(v.conditions, " && ")
}
