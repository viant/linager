package info

// DataPoint represents an identifier and its data lineage information
type DataPoint struct {
	Identity   Identity               `yaml:"identity"`           // Identity information
	Definition CodeLocation           `yaml:"definition"`         // Where the identifier is defined
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"` // Additional metadata
	Writes     []*TouchPoint          `yaml:"writes,omitempty"`   // Where the identifier is written
	Reads      []*TouchPoint          `yaml:"reads,omitempty"`    // Where the identifier is read
}

// TouchPoint represents a single point where data is read or written
type TouchPoint struct {
	CodeLocation          CodeLocation  `yaml:"codeLocation"`                    // Location in code
	Context               TouchContext  `yaml:"context,omitempty"`               // Context information
	Dependencies          []IdentityRef `yaml:"dependencies,omitempty"`          // Dependencies for this touch point
	ConditionalExpression string        `yaml:"conditionalExpression,omitempty"` // Condition under which this happens
}

// CodeLocation represents a location in the code
type CodeLocation struct {
	FilePath    string `yaml:"filePath"`              // File path
	LineNumber  int    `yaml:"lineNumber"`            // Line number
	ColumnStart int    `yaml:"columnStart,omitempty"` // Starting column
	ColumnEnd   int    `yaml:"columnEnd,omitempty"`   // Ending column
}

// TouchContext provides context about where data is accessed
type TouchContext struct {
	Function     string `yaml:"function,omitempty"`     // Function name
	Method       string `yaml:"method,omitempty"`       // Method name
	HolderType   string `yaml:"holderType,omitempty"`   // Type on which method is called
	IsTransitive bool   `yaml:"isTransitive,omitempty"` // Whether this is a transitive access
}
