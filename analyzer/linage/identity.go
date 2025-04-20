package linage

// IdentityRef is a unique reference to an identifier (variable, struct field, function, etc.)
type IdentityRef string

// Identity provides structured information about an identifier
type Identity struct {
	Ref         IdentityRef `yaml:"ref"`                   // Unique reference
	Module      string      `yaml:"module"`                //	Module
	PkgPath     string      `yaml:"pkgPath,omitempty"`     // Full package path
	Package     string      `yaml:"package,omitempty"`     // Package name
	ParentType  string      `yaml:"parentType,omitempty"`  //
	Name        string      `yaml:"name"`                  // Identifier name
	Kind        string      `yaml:"kind"`                  // Type of the identifier
	Scope       string      `yaml:"scope,omitempty"`       // Enclosing function/block
	ParentScope string      `yaml:"parentScope,omitempty"` // Parent scope i.r function name of function name with block lines
}

// MakeStructFieldIdentityRef creates an identity reference for a struct field
func MakeStructFieldIdentityRef(pkgPath, structType, fieldName string) IdentityRef {
	return IdentityRef(pkgPath + ":" + structType + ":" + fieldName)
}
