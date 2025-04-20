package graph

// ContentGenerator defines an interface for generating content from a file
type ContentGenerator interface {
	// Generate generates content from a file
	Emit(file *File) ([]byte, error)
}

// File represents a source code file with its types and symbols
type File struct {
	Name       string      // File name
	Path       string      // File path
	Package    string      // Package name
	ImportPath string      // Import path
	Types      []*Type     // Types declared in this file
	Constants  []*Constant // Constants declared in this file
	Variables  []*Variable // Variables declared in this file
	Functions  []*Function // Functions declared in this file
	Imports    []Import    // Imports used in this file

	functionMap map[string]int // Map of functions for quick lookup
	variableMap map[string]int // Map of variables for quick lookup
	constantMap map[string]int // Map of constants for quick lookup
	typeMap     map[string]int // Map of types for quick lookup
}

// Import represents an imported package
type Import struct {
	Name string // Local name (may be empty for default)
	Path string // Import path
}

// Package represents a Go package with its files and types
type Package struct {
	Name       string
	ImportPath string
	FileSet    []*File  // Files that are part of this package
	Assets     []*Asset // Assets associated with this package

	assetMap map[string]int // Map of assets for quick lookup
	fileMap  map[string]int // Map of files for quick lookup
	typeMap  map[string][]int
}

func (p *Package) LookupMethod(typeName, methodName string) *Function {
	if len(p.typeMap) == 0 {
		p.IndexTypes()
		return nil
	}

	files, ok := p.typeMap[typeName]
	if !ok {
		return nil
	}
	for _, idx := range files {
		file := p.FileSet[idx]
		if file != nil {
			if function := file.LookupFunction(methodName); function != nil {
				return function
			}
		}
	}
	return nil
}

func (p *Package) AddFile(file *File) {
	p.FileSet = append(p.FileSet, file)
}

func (p *Package) IndexTypes() {
	p.typeMap = make(map[string][]int)
	for _, file := range p.FileSet {
		if file == nil {
			continue
		}
		for _, typ := range file.Types {
			if typ == nil {
				continue
			}
			if _, ok := p.typeMap[typ.Name]; !ok {
				p.typeMap[typ.Name] = make([]int, 0)
			}
			p.typeMap[typ.Name] = append(p.typeMap[typ.Name], len(p.FileSet)-1)
		}
	}
}

// LookupFunction retrieves a function by name from the file
func (f *File) LookupFunction(name string) *Function {
	if len(f.functionMap) == 0 {
		f.IndexFunctions()
	}
	if idx, ok := f.functionMap[name]; ok && idx < len(f.Functions) {
		return f.Functions[idx]
	}
	return nil
}

// HasFunction checks if a function with the given name exists in the file
func (f *File) HasFunction(name string) bool {
	if f.functionMap == nil {
		return false
	}
	_, ok := f.functionMap[name]
	return ok
}

// LookupType retrieves a type by name from the file
func (f *File) LookupType(name string) *Type {
	if len(f.typeMap) == 0 {
		f.IndexTypes()
	}

	if idx, ok := f.typeMap[name]; ok && idx < len(f.Types) {
		return f.Types[idx]
	}

	return nil
}

// LookupVariable retrieves a variable by name from the file
func (f *File) LookupVariable(name string) *Variable {
	if f.variableMap == nil {
		return nil
	}

	if idx, ok := f.variableMap[name]; ok && idx < len(f.Variables) {
		return f.Variables[idx]
	}

	return nil
}

// GetConstant retrieves a constant by name from the file
func (f *File) GetConstant(name string) *Constant {
	if f.constantMap == nil {
		return nil
	}

	if idx, ok := f.constantMap[name]; ok && idx < len(f.Constants) {
		return f.Constants[idx]
	}

	return nil
}

func (f *File) IndexFunctions() {
	f.functionMap = make(map[string]int)
	for _, function := range f.Functions {
		if function == nil {
			continue
		}
		if _, ok := f.functionMap[function.Name]; !ok {
			f.functionMap[function.Name] = len(f.Functions) - 1
		}
	}

}

func (f *File) IndexTypes() {
	f.typeMap = make(map[string]int)
	for _, typ := range f.Types {
		if typ == nil {
			continue
		}
		if _, ok := f.typeMap[typ.Name]; !ok {
			f.typeMap[typ.Name] = len(f.Types) - 1
		}
	}

}

type Asset struct {
	Name       string
	Path       string
	ImportPath string
	Content    []byte
}

// Content reconstructs the content of a file from its components
func (f *File) Content(generator Emitter) ([]byte, error) {
	return generator.Emit(f)
}
