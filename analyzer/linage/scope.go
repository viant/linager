package linage

// Scope represents a scope in the code
type Scope struct {
	ID      string                 `json:"id"`
	Kind    string                 `json:"kind"`
	Name    string                 `json:"name,omitempty"`
	Start   int                    `json:"start"`
	End     int                    `json:"end"`
	Parent  *Scope                 `json:"-"`
	Symbols map[string]*Identifier `json:"symbols,omitempty"`
}

// Find searches for an identifier in the current scope and its parent scopes
func (s *Scope) Find(name string) *Identifier {
	for cur := s; cur != nil; cur = cur.Parent {
		if ident, ok := cur.Symbols[name]; ok {
			return ident
		}
	}
	return nil
}

func NewScope() *Scope {
	return &Scope{
		Symbols: make(map[string]*Identifier),
	}
}

type DataFlowEdge struct {
	Src   *Identifier `json:"src,omitempty"`
	Dst   *Identifier `json:"dst,omitempty"`
	Kind  AccessKind  `json:"kind,omitempty"`
	Scope string      `json:"scope,omitempty"`
	// Attributes holds optional metadata for this edge (e.g., annotation key/value, source location)
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type PackageModel struct {
	Path string `json:"path,omitempty"`
	// Language indicates the code language for this package (e.g., "go", "java")
	Language  string                 `json:"language,omitempty"`
	Files     []string               `json:"files,omitempty"`
	Scopes    []*Scope               `json:"scopes,omitempty"`
	Idents    map[string]*Identifier `json:"idents,omitempty"`
	DataFlows []*DataFlowEdge        `json:"dataflows,omitempty"`
}

func NewPackageModel() *PackageModel {
	return &PackageModel{
		Idents:    make(map[string]*Identifier),
		DataFlows: make([]*DataFlowEdge, 0),
	}
}
