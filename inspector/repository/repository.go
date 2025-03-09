package repository

type Repository struct {
	Kind   string
	Root   string
	Origin string
	Info   *Info
}

// Info represents information about a detected project
type Info struct {
	RootPath     string // Absolute path to the project root directory
	Type         string // Type of project (go, java, js, python, etc.)
	Name         string // Name of the project (extracted from config files)
	RelativePath string // Path from project root to the specified file
}
