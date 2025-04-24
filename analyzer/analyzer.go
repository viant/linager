package analyzer

// lineage_analyzer.go
// -----------------------------------------------------------------------------
// A *self‑contained* static‑analysis tool that walks a Go code‑base, builds a
// fine‑grained Scope graph, captures READ/WRITE/CALL/XFER edges between
// identifiers (including transitive flows), and merges comment annotations and
// struct‑field tags into `Identifier.Annotation`.
// -----------------------------------------------------------------------------
// Compile & run:
//     go get github.com/smacker/go-tree-sitter@v0.19.0
//     go run lineage_analyzer.go /path/to/module > lineage.json
// -----------------------------------------------------------------------------

import (
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/linager/analyzer/linage"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// FuncSummary captures a function's parameter and return identifiers and interprocedural flow mapping
type FuncSummary struct {
	// Params holds the formal parameter identifiers in order
	Params []*linage.Identifier
	// Returns holds the return identifiers in order (named returns or function ident for anonymous)
	Returns []*linage.Identifier
	// Flows maps a parameter index to a list of return indices indicating data flows
	Flows map[int][]int
}

// -----------------------------------------------------------------------------
// Analyzer
// -----------------------------------------------------------------------------

type Analyzer struct {
	parser *sitter.Parser
	match  MatcherFn
	fs     afs.Service
	// Language tag for this analyzer instance (e.g., "go", "java")
	Language string
	// optional service name for normalization across microservices
	serviceName string
	// optional exporter to send the IR graph to a graph store
	graphExporter GraphExporter

	// structFields keeps a mapping of struct type name -> map[fieldName]fieldType
	// It is populated while walking type specifications so the information can
	// later be used to infer the type of selector expressions (e.g. f.ID).
	structFields map[string]map[string]string
	// importAliases maps import alias to its full import path for the current file
	importAliases map[string]string
	// projectFiles lists manifest filenames that denote project roots (e.g. go.mod, pom.xml)
	projectFiles []string
	// annotationHooks holds callbacks to process annotations and add custom data-flow edges
	annotationHooks []AnnotationHook
	// plugins holds registered AnalyzerPlugin instances for custom analysis
	plugins []AnalyzerPlugin
	// interprocedural toggles inter-procedural call-return analysis
	interprocedural bool
	// funcSummaries holds parsed function signatures and flow summaries
	funcSummaries map[*linage.Identifier]*FuncSummary
}

// handleGo captures a goroutine invocation as a concurrent call
func (a *Analyzer) handleGo(n *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	// find the call_expression child
	var callNode *sitter.Node
	for i := 0; i < int(n.ChildCount()); i++ {
		ch := n.Child(i)
		if ch.Type() == "call_expression" {
			callNode = ch
			break
		}
	}
	if callNode == nil {
		return
	}
	// regular call handling
	a.handleCall(callNode, src, Scope, model)
	// mark concurrent call for the function identifier(s)
	fnNode := callNode.ChildByFieldName("function")
	if fnNode != nil {
		fns := a.extractIdentifiers(fnNode, src, Scope, model)
		for _, fn := range fns {
			model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: fn, Dst: fn, Kind: linage.Call, Scope: Scope.ID + "#go"})
		}
	}
}

// handleSend captures channel send operations (ch <- value)
func (a *Analyzer) handleSend(n *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	// send_statement: channel <- value
	chNode := n.ChildByFieldName("channel")
	valNode := n.ChildByFieldName("value")
	// fallback: use first and last child
	if chNode == nil && n.ChildCount() >= 3 {
		chNode = n.Child(0)
		valNode = n.Child(2)
	}
	if chNode == nil || valNode == nil {
		return
	}
	chIdent := a.resolveIdent(chNode, nil, src, Scope, model)
	vals := a.extractIdentifiers(valNode, src, Scope, model)
	for _, v := range vals {
		// read from value
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: v, Kind: linage.Read, Scope: Scope.ID})
		// transfer into channel
		model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: v, Dst: chIdent, Kind: linage.Xfer, Scope: Scope.ID})
	}
}

// handleSelect captures select statements with channel send/receive cases
func (a *Analyzer) handleSelect(n *sitter.Node, src []byte, Scope *linage.Scope, model *linage.PackageModel) {
	// traverse select subtree to find send and receive operations
	var stack []*sitter.Node
	stack = append(stack, n)
	for len(stack) > 0 {
		node := stack[0]
		stack = stack[1:]
		switch node.Type() {
		case "send_statement":
			a.handleSend(node, src, Scope, model)
		case "unary_expression":
			// possible channel receive: '<-ch'
			if node.ChildCount() >= 2 {
				op := node.Child(0)
				if string(src[op.StartByte():op.EndByte()]) == "<-" {
					// operand is channel
					chNode := node.Child(1)
					chIds := a.extractIdentifiers(chNode, src, Scope, model)
					for _, chId := range chIds {
						// record channel receive (read)
						model.DataFlows = append(model.DataFlows, &linage.DataFlowEdge{Src: chId, Dst: chId, Kind: linage.Read, Scope: Scope.ID})
					}
				}
			}
		default:
			for i := int(node.ChildCount()) - 1; i >= 0; i-- {
				stack = append(stack, node.Child(i))
			}
		}
	}
}

// literalIdent returns or creates a synthetic Identifier for a composite literal
// so that the literal value can serve as a data source (avoiding self-loops).
func (a *Analyzer) literalIdent(expr *sitter.Node, src []byte, scope *linage.Scope, model *linage.PackageModel) *linage.Identifier {
	// determine composite literal's type text (including generics) and start position
	typeNode := expr.ChildByFieldName("type")
	body := expr.ChildByFieldName("body")
	// derive literal name: between expr start and body start (captures generics), or fallback
	var (
		name  string
		start = expr.StartByte()
	)
	if body != nil {
		name = strings.TrimSpace(string(src[expr.StartByte():body.StartByte()]))
	} else if typeNode != nil {
		name = strings.TrimSpace(string(src[typeNode.StartByte():typeNode.EndByte()]))
		start = typeNode.StartByte()
	} else {
		name = strings.TrimSpace(string(src[expr.StartByte():expr.EndByte()]))
	}
	// build key for synthetic literal identifier
	fileScope := topFileScope(scope)
	file := strings.TrimPrefix(fileScope.ID, model.Path+":")
	key := fmt.Sprintf("%s::%s::%d", model.Path, file, start)
	if existing := model.Idents[key]; existing != nil {
		return existing
	}
	lit := &linage.Identifier{
		ID:        key,
		Name:      name,
		Package:   model.Path,
		File:      file,
		StartByte: start,
	}
	model.Idents[key] = lit
	return lit
}

func NewAnalyzer(options ...Option) *Analyzer {
	p := sitter.NewParser()
	ret := &Analyzer{
		parser:        p,
		fs:            afs.New(),
		structFields:  map[string]map[string]string{},
		importAliases: map[string]string{},
		// prepare function summaries mapping for interprocedural analysis
		funcSummaries: make(map[*linage.Identifier]*FuncSummary),
	}
	for _, opt := range options {
		if opt != nil {
			opt(ret)
		}
	}
	return ret
}

// -----------------------------------------------------------------------------
// Transitive closure across XFER edges (BFS per source)
// -----------------------------------------------------------------------------

// computeTransitiveClosure adds summary XFER edges per-call-site, preserving original scope context.
func (a *Analyzer) computeTransitiveClosure(model *linage.PackageModel) {
	// adjacency of direct XFERs: srcID -> list of dst identifiers
	adj := map[string][]*linage.Identifier{}
	for _, e := range model.DataFlows {
		if e.Kind == linage.Xfer {
			adj[e.Src.ID] = append(adj[e.Src.ID], e.Dst)
		}
	}
	var additional []*linage.DataFlowEdge
	// For each direct XFER, propagate its context through further XFER chains
	for _, e := range model.DataFlows {
		if e.Kind != linage.Xfer {
			continue
		}
		baseSrc := e.Src
		baseScope := e.Scope
		// start from the first hop (e.Dst)
		visited := map[string]bool{e.Dst.ID: true}
		queue := []*linage.Identifier{e.Dst}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, next := range adj[cur.ID] {
				if visited[next.ID] {
					continue
				}
				visited[next.ID] = true
				// add a context-sensitive summary edge
				additional = append(additional, &linage.DataFlowEdge{
					Src:   baseSrc,
					Dst:   next,
					Kind:  linage.Xfer,
					Scope: baseScope,
				})
				queue = append(queue, next)
			}
		}
	}
	model.DataFlows = append(model.DataFlows, additional...)
}

// -----------------------------------------------------------------------------
// Helpers & main
// -----------------------------------------------------------------------------

func topFileScope(s *linage.Scope) *linage.Scope {
	for cur := s; cur != nil; cur = cur.Parent {
		if cur.Kind == "file" {
			return cur
		}
	}
	return s
}

// AnalyzeAll runs analysis over all detected project roots under the given directory
// and merges their PackageModels into a single global model.
// Implementation moved to package.go
