package graph

type Variable struct {
	Name       string
	Comment    string
	Type       *Type
	Value      string
	File       *File     // File where this variable is defined
	IsExported bool      // Whether the variable is exported (public) or not
	Annotation string    // Annotation associated with the variable
	IsConst    bool      // Whether the variable is a constant
	Location   *Location // Location of the variable in the source code
}
