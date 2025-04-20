package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/viant/linager/analyzer/linage"
)

// JSXAnalyzer analyzes JSX source code to extract data lineage information
type JSXAnalyzer struct {
	project    string
	scopes     map[string]*linage.Scope
	dataPoints map[string]*linage.DataPoint
}

// NewJSXAnalyzer creates a new JSXAnalyzer
func NewJSXAnalyzer(project string) *JSXAnalyzer {
	return &JSXAnalyzer{
		project:    project,
		scopes:     make(map[string]*linage.Scope),
		dataPoints: make(map[string]*linage.DataPoint),
	}
}

// AnalyzeSourceCode analyzes JSX source code and extracts data lineage information
func (a *JSXAnalyzer) AnalyzeSourceCode(source string, project string, path string) ([]*linage.DataPoint, error) {
	// Reset state for new analysis
	a.scopes = make(map[string]*linage.Scope)
	a.dataPoints = make(map[string]*linage.DataPoint)
	a.project = project // Set the project name for use in data points

	// TODO: Implement JSX parsing using tree-sitter
	// For now, we'll use a simple implementation that extracts basic information

	// Build a simple scope hierarchy
	a.buildSimpleScopeHierarchy(source, path)

	// Process component declarations
	a.processComponentDeclarations(source, path)

	// Process hooks and state
	a.processHooksAndState(source, path)

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

// AnalyzeFile analyzes a JSX file and extracts data lineage information
func (a *JSXAnalyzer) AnalyzeFile(filePath string) ([]*linage.DataPoint, error) {
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

// AnalyzePackage analyzes all JSX files in a package and extracts data lineage information
func (a *JSXAnalyzer) AnalyzePackage(packagePath string) ([]*linage.DataPoint, error) {
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

		// Process only .jsx and .tsx files
		ext := filepath.Ext(path)
		if ext != ".jsx" && ext != ".tsx" {
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

// buildSimpleScopeHierarchy builds a simple scope hierarchy for JSX code
func (a *JSXAnalyzer) buildSimpleScopeHierarchy(source string, path string) {
	// Create a file scope
	fileName := filepath.Base(path)
	fileScope := &linage.Scope{
		ID:       fileName,
		Kind:     "file",
		ParentID: "",
		Start:    1,
		End:      len(strings.Split(source, "\n")),
	}
	a.scopes[fileName] = fileScope

	// Simple component detection (this is a placeholder implementation)
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		// Look for component declarations (function or class)
		if strings.Contains(line, "function") && strings.Contains(line, "(") {
			// Extract component name
			parts := strings.Split(line, "function ")
			if len(parts) > 1 {
				nameParts := strings.Split(parts[1], "(")
				if len(nameParts) > 0 {
					componentName := strings.TrimSpace(nameParts[0])
					if componentName != "" {
						// Create a component scope
						componentScope := &linage.Scope{
							ID:       fileName + "." + componentName,
							Kind:     "component",
							ParentID: fileName,
							Start:    i + 1,
							End:      i + 1, // This is a placeholder, we'd need to find the actual end
						}
						a.scopes[componentScope.ID] = componentScope
					}
				}
			}
		} else if strings.Contains(line, "class") && strings.Contains(line, "extends") {
			// Extract component name
			parts := strings.Split(line, "class ")
			if len(parts) > 1 {
				nameParts := strings.Split(parts[1], " ")
				if len(nameParts) > 0 {
					componentName := strings.TrimSpace(nameParts[0])
					if componentName != "" {
						// Create a component scope
						componentScope := &linage.Scope{
							ID:       fileName + "." + componentName,
							Kind:     "component",
							ParentID: fileName,
							Start:    i + 1,
							End:      i + 1, // This is a placeholder, we'd need to find the actual end
						}
						a.scopes[componentScope.ID] = componentScope
					}
				}
			}
		}
	}
}

// processComponentDeclarations processes component declarations in JSX code
func (a *JSXAnalyzer) processComponentDeclarations(source string, path string) {
	// Simple component detection (this is a placeholder implementation)
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		// Look for component declarations (function or class)
		if strings.Contains(line, "function") && strings.Contains(line, "(") {
			// Extract component name
			parts := strings.Split(line, "function ")
			if len(parts) > 1 {
				nameParts := strings.Split(parts[1], "(")
				if len(nameParts) > 0 {
					componentName := strings.TrimSpace(nameParts[0])
					if componentName != "" {
						// Create a data point for the component
						componentRef := linage.IdentityRef(fmt.Sprintf("%s:%s", a.project, componentName))
						dataPoint := &linage.DataPoint{
							Identity: linage.Identity{
								Ref:     componentRef,
								Module:  a.project,
								PkgPath: filepath.Dir(path),
								Package: filepath.Base(filepath.Dir(path)),
								Name:    componentName,
								Kind:    "component",
							},
							Definition: linage.CodeLocation{
								FilePath:   path,
								LineNumber: i + 1,
							},
							Metadata: map[string]interface{}{
								"type": "function",
							},
							Writes: []*linage.TouchPoint{},
							Reads:  []*linage.TouchPoint{},
						}
						a.dataPoints[string(componentRef)] = dataPoint
					}
				}
			}
		} else if strings.Contains(line, "class") && strings.Contains(line, "extends") {
			// Extract component name
			parts := strings.Split(line, "class ")
			if len(parts) > 1 {
				nameParts := strings.Split(parts[1], " ")
				if len(nameParts) > 0 {
					componentName := strings.TrimSpace(nameParts[0])
					if componentName != "" {
						// Create a data point for the component
						componentRef := linage.IdentityRef(fmt.Sprintf("%s:%s", a.project, componentName))
						dataPoint := &linage.DataPoint{
							Identity: linage.Identity{
								Ref:     componentRef,
								Module:  a.project,
								PkgPath: filepath.Dir(path),
								Package: filepath.Base(filepath.Dir(path)),
								Name:    componentName,
								Kind:    "component",
							},
							Definition: linage.CodeLocation{
								FilePath:   path,
								LineNumber: i + 1,
							},
							Metadata: map[string]interface{}{
								"type": "class",
							},
							Writes: []*linage.TouchPoint{},
							Reads:  []*linage.TouchPoint{},
						}
						a.dataPoints[string(componentRef)] = dataPoint
					}
				}
			}
		}
	}
}

// processHooksAndState processes React hooks and state in JSX code
func (a *JSXAnalyzer) processHooksAndState(source string, path string) {
	// Simple hook detection (this is a placeholder implementation)
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		// Look for useState hooks
		if strings.Contains(line, "useState") {
			// Extract state variable name
			parts := strings.Split(line, "useState")
			if len(parts) > 1 && strings.Contains(parts[1], "[") && strings.Contains(parts[1], "]") {
				stateDecl := parts[1]
				startIdx := strings.Index(stateDecl, "[")
				endIdx := strings.Index(stateDecl, "]")
				if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
					stateParts := strings.Split(stateDecl[startIdx+1:endIdx], ",")
					if len(stateParts) >= 1 {
						stateName := strings.TrimSpace(stateParts[0])
						if stateName != "" {
							// Find the component scope for this line
							var componentScope *linage.Scope
							for _, scope := range a.scopes {
								if scope.Kind == "component" && i+1 >= scope.Start && i+1 <= scope.End {
									componentScope = scope
									break
								}
							}

							if componentScope != nil {
								// Create a data point for the state
								stateRef := linage.IdentityRef(fmt.Sprintf("%s:%s.%s", a.project, componentScope.ID, stateName))
								dataPoint := &linage.DataPoint{
									Identity: linage.Identity{
										Ref:        stateRef,
										Module:     a.project,
										PkgPath:    filepath.Dir(path),
										Package:    filepath.Base(filepath.Dir(path)),
										ParentType: componentScope.ID,
										Name:       stateName,
										Kind:       "state",
										Scope:      componentScope.ID,
									},
									Definition: linage.CodeLocation{
										FilePath:   path,
										LineNumber: i + 1,
									},
									Metadata: map[string]interface{}{
										"hook": "useState",
									},
									Writes: []*linage.TouchPoint{},
									Reads:  []*linage.TouchPoint{},
								}
								a.dataPoints[string(stateRef)] = dataPoint
							}
						}
					}
				}
			}
		}
	}
}
