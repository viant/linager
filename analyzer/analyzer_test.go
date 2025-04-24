package analyzer

import (
	_ "embed"
	"encoding/json"
	"fmt"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/analyzer/linage"
	"testing"
)

//go:embed testdata/go_basic_expect.json
var expectGoBasic string

//go:embed testdata/go_basic.gox
var goBasic string

//go:embed testdata/go_flow.gox
var goFlow string

//go:embed testdata/go_flow_expected.json
var goExpectedFlow string

//go:embed testdata/go_channel_source.gox
var channelSource string

//go:embed testdata/go_channel_flows.json
var channelFlows string

//go:embed testdata/go_context_source.gox
var contextSource string

//go:embed testdata/go_context_flows.json
var contextFlows string

// TestAnalyzer_AnalyzeSourceCode drives data-flow tests for Go snippets
func TestAnalyzer_AnalyzeSourceCode(t *testing.T) {
	scenarios := []struct {
		name       string
		source     string
		expectJSON string
		dir        string
		file       string
	}{
		{name: "basic", source: goBasic, expectJSON: expectGoBasic, dir: "/test/dir", file: "test.go"},
		{name: "cross package", source: goFlow, expectJSON: goExpectedFlow, dir: "/app/dao", file: "customer_dao.go"},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithLanguage(golang.GetLanguage()),
				WithMacher(GolangFiles),
			)
			pkgScope := linage.NewScope()
			model := linage.NewPackageModel()
			// analyze
			assert.NoError(t, analyzer.AnalyzeSourceCode(sc.dir, []byte(sc.source), sc.file, pkgScope, model), sc.name)
			// prepare expected
			expect := linage.NewPackageModel()
			assert.NoError(t, json.Unmarshal([]byte(sc.expectJSON), expect), sc.name+" unmarshal expected")
			// round-trip actual via JSON to strip AST nodes
			raw, err := json.Marshal(model)
			assert.NoError(t, err, sc.name+" marshal actual")
			got := linage.NewPackageModel()
			assert.NoError(t, json.Unmarshal(raw, got), sc.name+" unmarshal actual")
			// compare
			if !assert.EqualValues(t, expect, got, sc.name) {
				gotJSON, _ := json.Marshal(got)
				expectJSON, _ := json.Marshal(expect)
				fmt.Printf("gotJSON: %s\n", string(gotJSON))
				fmt.Printf("expectJSON: %s\n", string(expectJSON))
			}
		})
	}
}

// DataFlowEdge represents a simplified data flow edge for testing
type DataFlowEdge struct {
	Src   string            `json:"src"`
	Dst   string            `json:"dst"`
	Scope string            `json:"scope"`
	Kind  linage.AccessKind `json:"kind"`
}

// TestDataFlows tests data flow edges for various scenarios
func TestDataFlows(t *testing.T) {
	scenarios := []struct {
		name       string
		source     string
		expectJSON string
	}{
		{name: "channel flows", source: channelSource, expectJSON: channelFlows},
		{name: "context sensitivity", source: contextSource, expectJSON: contextFlows},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Setup analyzer and analyze source code
			analyzer := NewAnalyzer(
				WithLanguage(golang.GetLanguage()),
				WithMacher(GolangFiles),
			)
			pkgScope := linage.NewScope()
			model := linage.NewPackageModel()
			err := analyzer.AnalyzeSourceCode("", []byte(sc.source), "test.go", pkgScope, model)
			assert.NoError(t, err)

			// Load expected data flow edges
			var expect []DataFlowEdge
			err = json.Unmarshal([]byte(sc.expectJSON), &expect)
			assert.NoError(t, err)

			// Extract actual XFER edges
			var actual []DataFlowEdge
			for _, e := range model.DataFlows {
				if e.Kind != linage.Xfer {
					continue
				}
				actual = append(actual, DataFlowEdge{
					Src:   e.Src.Name,
					Dst:   e.Dst.Name,
					Scope: e.Scope,
					Kind:  e.Kind,
				})
			}

			// Compare expected and actual using ElementsMatch
			if !assert.ElementsMatch(t, expect, actual) {
				gotJSON, _ := json.Marshal(actual)
				expectJSON, _ := json.Marshal(expect)
				fmt.Printf("expectedJSON: %s\n", string(expectJSON))
				fmt.Printf("actualJSON: %s\n", string(gotJSON))
			}
		})
	}
}
