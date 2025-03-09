package info

type Constant struct {
	Name       string
	Comment    string
	Value      string
	Type       *Type     // Type of the constant if specified
	File       *File     // File where this constant is defined
	IsExported bool      // Whether the constant is exported (public) or not
	Location   *Location // Location of the constant in the source code
}
