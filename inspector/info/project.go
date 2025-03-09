package info

type Project struct {
	Name          string
	RootDir       string
	ImportPath    string
	RepositoryURL string
	Branch        string
	Packages      []*Package
	Type          string
	RootPath      string
}
