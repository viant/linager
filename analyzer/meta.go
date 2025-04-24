package analyzer

import (
	"bytes"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/analyzer/linage"
	"reflect"
	"regexp"
	"strings"
)

// -----------------------------------------------------------------------------
// Annotation & tag parsing
// -----------------------------------------------------------------------------

var annRe = regexp.MustCompile(`@([\w:.-]+)(?:[=:]([^\s]+))?`)

func mergeAnnotations(dst *linage.Annotations, src map[string]string) {
	if len(src) == 0 {
		return
	}
	if *dst == nil {
		*dst = linage.Annotations{}
	}
	for k, v := range src {
		(*dst)[k] = v
	}
}

func (a *Analyzer) extractAnnotations(n *sitter.Node, src []byte) linage.Annotations {
	var anns linage.Annotations

	// 1) preceding line comments // @key=value
	start := int(n.StartByte())
	if start > 0 {
		i := start - 1
		// move to previous newline
		for i >= 0 && src[i] != '\n' {
			i--
		}
		for i >= 0 {
			lineEnd := bytes.IndexByte(src[i+1:], '\n')
			if lineEnd == -1 {
				lineEnd = len(src) - (i + 1)
			}
			line := bytes.TrimSpace(src[i+1 : i+1+lineEnd])
			if !bytes.HasPrefix(line, []byte("//")) {
				break
			}
			for _, m := range annRe.FindAllSubmatch(line, -1) {
				key := string(m[1])
				val := ""
				if len(m) > 2 {
					val = string(m[2])
				}
				mergeAnnotations(&anns, map[string]string{key: val})
			}
			i -= lineEnd + 1
			for i >= 0 && src[i] != '\n' {
				i--
			}
		}
	}

	// 2) struct tags on same field
	if p := n.Parent(); p != nil && p.Type() == "field_declaration" {
		for i := 0; i < int(p.ChildCount()); i++ {
			tagN := p.Child(i)
			if tagN.Type() == "raw_string_literal" || tagN.Type() == "interpreted_string_literal" {
				tag := strings.Trim(string(src[tagN.StartByte():tagN.EndByte()]), "`\"")
				st := reflect.StructTag(tag)
				for tag != "" {
					kv := strings.SplitN(tag, "\"", 3)
					if len(kv) < 2 {
						break
					}
					k := strings.TrimSpace(kv[0])
					v := kv[1]
					mergeAnnotations(&anns, map[string]string{k: v})
					idx := strings.Index(tag, " ")
					if idx == -1 {
						break
					}
					tag = strings.TrimSpace(tag[idx+1:])
				}

				for _, k := range []string{"json", "db", "yaml"} {

					if v := st.Get(k); v != "" {
						mergeAnnotations(&anns, map[string]string{k: v})
					}
				}
			}
		}
	}

	// 3) Java annotation AST on declarations (marker_annotation, normal_annotation, annotation)
	for cur := n.Parent(); cur != nil; cur = cur.Parent() {
		t := cur.Type()
		if t == "class_declaration" || t == "method_declaration" || t == "field_declaration" || t == "variable_declarator" {
			for i := 0; i < int(cur.ChildCount()); i++ {
				ch := cur.Child(i)
				if ch.Type() == "marker_annotation" || ch.Type() == "normal_annotation" || ch.Type() == "annotation" {
					nameNode := ch.ChildByFieldName("name")
					if nameNode != nil {
						annName := string(src[nameNode.StartByte():nameNode.EndByte()])
						mergeAnnotations(&anns, map[string]string{annName: ""})
						// parse key=value pairs in normal annotations
						for j := 0; j < int(ch.NamedChildCount()); j++ {
							pair := ch.NamedChild(j)
							if pair.Type() == "element_value_pair" {
								keyNode := pair.ChildByFieldName("name")
								valNode := pair.ChildByFieldName("value")
								if keyNode != nil && valNode != nil {
									key := string(src[keyNode.StartByte():keyNode.EndByte()])
									val := string(src[valNode.StartByte():valNode.EndByte()])
									mergeAnnotations(&anns, map[string]string{annName + "." + key: val})
								}
							}
						}
					}
				}
			}
			break
		}
	}

	if len(anns) == 0 {
		return nil
	}
	return anns
}
