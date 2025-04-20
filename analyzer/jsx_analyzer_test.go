package analyzer

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/analyzer/linage"
	"testing"
)

func TestJSXAnalyzer_AnalyzeSourceCode(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		project  string
		path     string
		expected []*linage.DataPoint
		wantErr  bool
	}{
		{
			name: "simple function component",
			source: `import React from 'react';

function Greeting(props) {
  return (
    <div>
      <h1>Hello, {props.name}!</h1>
    </div>
  );
}

export default Greeting;`,
			project: "example",
			path:    "Greeting.jsx",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:     "example:Greeting",
						Module:  "example",
						PkgPath: ".",
						Package: ".",
						Name:    "Greeting",
						Kind:    "component",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Greeting.jsx",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "function",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
		{
			name: "class component with state",
			source: `import React from 'react';

class Counter extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      count: 0
    };
  }

  render() {
    return (
      <div>
        <p>Count: {this.state.count}</p>
        <button onClick={() => this.setState({ count: this.state.count + 1 })}>
          Increment
        </button>
      </div>
    );
  }
}

export default Counter;`,
			project: "example",
			path:    "Counter.jsx",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:     "example:Counter",
						Module:  "example",
						PkgPath: ".",
						Package: ".",
						Name:    "Counter",
						Kind:    "component",
					},
					Definition: linage.CodeLocation{
						FilePath:   "Counter.jsx",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "class",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
		{
			name: "function component with useState hook",
			source: `import React, { useState } from 'react';

function Counter() {
  const [count, setCount] = useState(0);

  return (
    <div>
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>
        Increment
      </button>
    </div>
  );
}

export default Counter;`,
			project: "example",
			path:    "HookCounter.jsx",
			expected: []*linage.DataPoint{
				{
					Identity: linage.Identity{
						Ref:     "example:Counter",
						Module:  "example",
						PkgPath: ".",
						Package: ".",
						Name:    "Counter",
						Kind:    "component",
					},
					Definition: linage.CodeLocation{
						FilePath:   "HookCounter.jsx",
						LineNumber: 3,
					},
					Metadata: map[string]interface{}{
						"type": "function",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
				{
					Identity: linage.Identity{
						Ref:        "example:HookCounter.jsx.Counter.count",
						Module:     "example",
						PkgPath:    ".",
						Package:    ".",
						ParentType: "HookCounter.jsx.Counter",
						Name:       "count",
						Kind:       "state",
						Scope:      "HookCounter.jsx.Counter",
					},
					Definition: linage.CodeLocation{
						FilePath:   "HookCounter.jsx",
						LineNumber: 4,
					},
					Metadata: map[string]interface{}{
						"hook": "useState",
					},
					Writes: []*linage.TouchPoint{},
					Reads:  []*linage.TouchPoint{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewJSXAnalyzer(tt.project)
			got, err := analyzer.AnalyzeSourceCode(tt.source, tt.project, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSXAnalyzer.AnalyzeSourceCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check that we have the expected number of data points
			assert.Equal(t, len(tt.expected), len(got), "Expected %d data points, got %d", len(tt.expected), len(got))

			// For each expected data point, find a matching one in the result
			for _, expected := range tt.expected {
				found := false
				for _, actual := range got {
					if string(expected.Identity.Ref) == string(actual.Identity.Ref) {
						// Check identity fields
						assert.Equal(t, expected.Identity.Module, actual.Identity.Module)
						assert.Equal(t, expected.Identity.Name, actual.Identity.Name)
						assert.Equal(t, expected.Identity.Kind, actual.Identity.Kind)

						// Check definition
						assert.Equal(t, expected.Definition.FilePath, actual.Definition.FilePath)
						assert.Equal(t, expected.Definition.LineNumber, actual.Definition.LineNumber)

						// Check metadata
						for k, v := range expected.Metadata {
							assert.Equal(t, v, actual.Metadata[k])
						}

						found = true
						break
					}
				}
				assert.True(t, found, "Expected data point with ref %s not found", expected.Identity.Ref)
			}
		})
	}
}
