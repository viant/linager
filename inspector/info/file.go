package info

// File represents a source code file with its types and symbols
type File struct {
	Name       string      // File name
	Path       string      // File path
	Package    string      // Package name
	ImportPath string      // Import path
	Types      []*Type     // Types declared in this file
	Constants  []*Constant // Constants declared in this file
	Variables  []*Variable // Variables declared in this file
	Imports    []Import    // Imports used in this file
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
	Asset      []*Asset // Assets associated with this package
}

type Asset struct {
	Path       string
	ImportPath string
	Content    []byte
}

// AddFile adds a file to the package
func (p *Package) AddFile(file *File) {
	p.FileSet = append(p.FileSet, file)
}
