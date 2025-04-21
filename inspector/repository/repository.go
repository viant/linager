package repository

import "golang.org/x/mod/modfile"

type Repository struct {
	Kind   string
	Root   string
	Origin string
	Info   *Project
}

// Project represents information about a detected project
type Project struct {
	RootPath     string // Absolute path to the project root directory
	Type         string // Type of project (go, java, js, python, etc.)
	Name         string // Name of the project (extracted from config files)
	RelativePath string // Path from project root to the specified file
	GoModule     *modfile.Module
}
