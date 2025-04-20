package linage

// Scope represents
type Scope struct {
	ID       string `yaml:"id"`                 // Unique scope ID (can be hierarchical like "pkg.FuncName.block1")
	Kind     string `yaml:"kind"`               // e.g., "function", "block", "loop", "if", "switch"
	Name     string `yaml:"name,omitempty"`     // e.g., "Init" for a function, or empty for anonymous block
	ParentID string `yaml:"parentId,omitempty"` // ID of the parent scope
	Start    int    `yaml:"start"`              // Start line or character offset
	End      int    `yaml:"end"`                // End line or character offset

}

/*
pkg: github.com/example/app
Func Init [lines 10–30]
  └── Block1 (if stmt) [12–18]
       └── Block2 (for loop) [13–17]
*/
