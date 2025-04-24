package linage

// DataPoint represents an identifier and its data lineage information
type DataPoint struct {
	Identifier `yaml:"identity"`      // Identity information
	Definition CodeLocation           `yaml:"definition"`         // Where the identifier is defined
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"` // Additional metadata
	Writes     []*DataFlowEdge        `yaml:"writes,omitempty"`   // Where the identifier is written
	Reads      []*DataFlowEdge        `yaml:"reads,omitempty"`    // Where the identifier is read
	Calls      []*DataFlowEdge        `yaml:"calls,omitempty"`    // Where the identifier is called
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
	Scope string `yaml:"scope"`
}
