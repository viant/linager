package info

import (
	"fmt"
	"strconv"
	"strings"
)

// IdentityRef is a unique reference to an identifier (variable, struct field, function, etc.)
type IdentityRef string

// Identity provides structured information about an identifier
type Identity struct {
	Ref        IdentityRef `yaml:"ref"`                  // Unique reference
	PkgPath    string      `yaml:"pkgPath,omitempty"`    // Full package path
	Package    string      `yaml:"package,omitempty"`    // Package name
	HolderType string      `yaml:"holderType,omitempty"` // Type containing the field (for struct fields)
	Function   string      `yaml:"function,omitempty"`   // Function containing the identifier
	Line       int         `yaml:"line,omitempty"`       // Line number within file
	Name       string      `yaml:"name"`                 // Identifier name
	Kind       string      `yaml:"kind"`                 // Type of the identifier
}

// MakeStructFieldIdentityRef creates a standard reference for struct fields
func MakeStructFieldIdentityRef(pkgPath, structType, fieldName string) IdentityRef {
	return IdentityRef(fmt.Sprintf("%s:%s:%s", pkgPath, structType, fieldName))
}

// MakeFunctionIdentityRef creates a reference for function
func MakeFunctionIdentityRef(pkgPath, funcName string) IdentityRef {
	return IdentityRef(fmt.Sprintf("%s:%s", pkgPath, funcName))
}

// MakeVarIdentityRef creates a reference for local variables
func MakeVarIdentityRef(pkgPath, filePath, function string, line int, varName string) IdentityRef {
	if function != "" {
		return IdentityRef(fmt.Sprintf("%s:%s:[%s]:%d:%s", pkgPath, filePath, function, line, varName))
	}
	return IdentityRef(fmt.Sprintf("%s:%s:%d:%s", pkgPath, filePath, line, varName))
}

// Identity parses the reference into a structured Identity
func (r IdentityRef) Identity() Identity {
	id := Identity{Ref: r}
	if r == "" {
		return id
	}

	parts := strings.Split(string(r), ":")
	if len(parts) < 2 {
		return id // Not enough parts for meaningful parsing
	}

	// Try to determine format based on parts count and content
	switch {
	case len(parts) == 3 && !strings.Contains(parts[1], ".go"):
		// Format: pkgPath:structType:fieldName
		id.PkgPath = parts[0]
		id.HolderType = parts[1]
		id.Name = parts[2]
		if lastDot := strings.LastIndex(id.PkgPath, "/"); lastDot >= 0 {
			id.Package = id.PkgPath[lastDot+1:]
		} else {
			id.Package = id.PkgPath
		}
		if id.HolderType != "" && id.Name != "" {
			id.Name = fmt.Sprintf("%s.%s", id.HolderType, id.Name)
		}

	case len(parts) == 5 && strings.HasPrefix(parts[2], "[") && strings.HasSuffix(parts[2], "]"):
		// Format: pkgPath:filePath:[function]:line:varName
		id.PkgPath = parts[0]
		id.Function = strings.TrimSuffix(strings.TrimPrefix(parts[2], "["), "]")
		if lineNum, err := strconv.Atoi(parts[3]); err == nil {
			id.Line = lineNum
		}
		id.Name = parts[4]
		if lastDot := strings.LastIndex(id.PkgPath, "."); lastDot >= 0 {
			id.Package = id.PkgPath[lastDot+1:]
		} else {
			id.Package = id.PkgPath
		}

	case len(parts) == 4 && strings.HasSuffix(parts[1], ".go"):
		// Format: pkgPath:filePath:line:varName
		id.PkgPath = parts[0]
		if lineNum, err := strconv.Atoi(parts[2]); err == nil {
			id.Line = lineNum
		}
		id.Name = parts[3]
		if lastDot := strings.LastIndex(id.PkgPath, "."); lastDot >= 0 {
			id.Package = id.PkgPath[lastDot+1:]
		} else {
			id.Package = id.PkgPath
		}
	}

	return id
}
