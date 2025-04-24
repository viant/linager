package analyzer

import (
	"fmt"
	"github.com/viant/linager/analyzer/linage"
)

// IRNode represents a node in the intermediate representation graph.
type IRNode struct {
	ID         string                 // normalized identifier across services/languages
	Type       string                 // node type (e.g., identifier kind)
	Properties map[string]interface{} // additional properties (name, package, file, etc.)
}

// IREdge represents an edge in the intermediate representation graph.
type IREdge struct {
	Source     string                 // source node ID
	Target     string                 // target node ID
	Type       string                 // edge type (e.g., Read, Write, Call)
	Properties map[string]interface{} // additional attributes (scope, annotations, etc.)
}

// IRGraph holds the nodes and edges for the intermediate representation.
type IRGraph struct {
	Nodes []IRNode
	Edges []IREdge
}

// GraphExporter defines an interface to export an IRGraph to a storage backend (e.g., Neo4j).
type GraphExporter interface {
	Export(graph *IRGraph) error
}

// WithGraphExporter registers a GraphExporter to send the IRGraph after analysis.
func WithGraphExporter(exporter GraphExporter) Option {
	return func(a *Analyzer) {
		a.graphExporter = exporter
	}
}

// WithServiceName sets a service name for normalization across microservices.
func WithServiceName(name string) Option {
	return func(a *Analyzer) {
		a.serviceName = name
	}
}

// normalizeID builds a unique ID combining language, service name, and original identifier ID.
func normalizeID(a *Analyzer, id *linage.Identifier) string {
	return fmt.Sprintf("%s:%s:%s", a.Language, a.serviceName, id.ID)
}

// buildIRGraph constructs an IRGraph from a PackageModel.
func buildIRGraph(a *Analyzer, model *linage.PackageModel) *IRGraph {
	graph := &IRGraph{}
	// create nodes for each identifier
	for _, id := range model.Idents {
		node := IRNode{
			ID:   normalizeID(a, id),
			Type: id.Kind,
			Properties: map[string]interface{}{
				"name":      id.Name,
				"package":   id.Package,
				"file":      id.File,
				"startByte": id.StartByte,
				"language":  model.Language,
				"service":   a.serviceName,
			},
		}
		graph.Nodes = append(graph.Nodes, node)
	}
	// create edges for each data flow
	for _, df := range model.DataFlows {
		if df.Src == nil || df.Dst == nil {
			continue
		}
		edge := IREdge{
			Source: normalizeID(a, df.Src),
			Target: normalizeID(a, df.Dst),
			Type:   string(df.Kind),
			Properties: map[string]interface{}{
				"scope": df.Scope,
			},
		}
		// copy any additional attributes
		if df.Attributes != nil {
			for k, v := range df.Attributes {
				edge.Properties[k] = v
			}
		}
		graph.Edges = append(graph.Edges, edge)
	}
	return graph
}
