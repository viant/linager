package linage

import sitter "github.com/smacker/go-tree-sitter"

type Selector struct {
	Field  string    `json:"field,omitempty"`
	Parent *Selector `json:"parent,omitempty"`
}

type Annotations map[string]string

type Identifier struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Kind       string       `json:"kind,omitempty"`
	Package    string       `json:"package,omitempty"`
	File       string       `json:"file,omitempty"`
	StartByte  uint32       `json:"startByte,omitempty"`
	Type       string       `json:"type,omitempty"`
	Selector   *Selector    `json:"selector,omitempty"`
	Annotation Annotations  `json:"annotations,omitempty"`
	Node       *sitter.Node `json:"-"`
}

func (i *Identifier) String() string { return i.ID }
