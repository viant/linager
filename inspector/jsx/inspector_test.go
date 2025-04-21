package jsx_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/jsx"
	"reflect"
	"testing"
)

// stripLocations creates a deep copy of the types with all Location fields set to nil
func stripLocations(types []*graph.Type) []*graph.Type {
	result := make([]*graph.Type, len(types))
	for i, t := range types {
		// Create a copy of the type
		typeCopy := &graph.Type{
			Name:          t.Name,
			Kind:          t.Kind,
			Tag:           t.Tag,
			Package:       t.Package,
			PackagePath:   t.PackagePath,
			ComponentType: t.ComponentType,
			KeyType:       t.KeyType,
			Comment:       t.Comment,
			Annotation:    t.Annotation,
			IsExported:    t.IsExported,
			IsPointer:     t.IsPointer,
			Implements:    t.Implements,
			Extends:       t.Extends,
			Location:      nil, // Set Location to nil
		}

		// Copy fields without Location
		if t.Fields != nil {
			typeCopy.Fields = make([]*graph.Field, len(t.Fields))
			for j, f := range t.Fields {
				fieldCopy := &graph.Field{
					Name:       f.Name,
					Type:       f.Type, // Assuming Type is already properly set
					Tag:        f.Tag,
					Comment:    f.Comment,
					Annotation: f.Annotation,
					IsExported: f.IsExported,
					IsEmbedded: f.IsEmbedded,
					IsStatic:   f.IsStatic,
					IsConstant: f.IsConstant,
					Location:   nil, // Set Location to nil
				}
				typeCopy.Fields[j] = fieldCopy
			}
		}

		// Copy methods without Location
		if t.Methods != nil {
			typeCopy.Methods = make([]*graph.Function, len(t.Methods))
			for j, m := range t.Methods {
				methodCopy := &graph.Function{
					Name:          m.Name,
					Comment:       m.Comment,
					Annotation:    m.Annotation,
					Receiver:      m.Receiver,
					TypeParams:    m.TypeParams,
					Parameters:    m.Parameters,
					Results:       m.Results,
					Body:          m.Body,
					IsExported:    m.IsExported,
					IsStatic:      m.IsStatic,
					IsConstructor: m.IsConstructor,
					Signature:     m.Signature,
					Hash:          m.Hash,
					Location:      nil, // Set Location to nil
				}
				typeCopy.Methods[j] = methodCopy
			}
		}

		result[i] = typeCopy
	}
	return result
}

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    []*graph.Type
		wantErr bool
	}{
		{
			name: "Function Component",
			source: `import React from 'react';

function Greeting(props) {
  return <h1>Hello, {props.name}!</h1>;
}

export default Greeting;`,
			want: []*graph.Type{
				{
					Name: "Greeting",
					Kind: reflect.Struct,
					Fields: []*graph.Field{
						{
							Name: "props",
							Type: &graph.Type{
								Name: "any",
							},
							Comment: "prop",
						},
					},
					Methods:    []*graph.Function{},
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Class Component",
			source: `import React, { Component } from 'react';

class Counter extends Component {
  constructor(props) {
    super(props);
    this.state = {
      count: 0
    };
  }

  increment = () => {
    this.setState({ count: this.state.count + 1 });
  }

  render() {
    return (
      <div>
        <p>Count: {this.state.count}</p>
        <button onClick={this.increment}>Increment</button>
      </div>
    );
  }
}

export default Counter;`,
			want: []*graph.Type{
				{
					Name: "Counter",
					Kind: reflect.Struct,
					Methods: []*graph.Function{
						{
							Name: "render",
						},
						{
							Name: "increment",
						},
					},
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Component with Props",
			source: `import React from 'react';
import PropTypes from 'prop-types';

function Button({ text, onClick, disabled }) {
  return (
    <button onClick={onClick} disabled={disabled}>
      {text}
    </button>
  );
}

Button.propTypes = {
  text: PropTypes.string.isRequired,
  onClick: PropTypes.func.isRequired,
  disabled: PropTypes.bool
};

Button.defaultProps = {
  disabled: false
};

export default Button;`,
			want: []*graph.Type{
				{
					Name: "Button",
					Kind: reflect.Struct,
					Fields: []*graph.Field{
						{
							Name: "text",
							Type: &graph.Type{
								Name: "any",
							},
							Comment: "prop",
						},
						{
							Name: "onClick",
							Type: &graph.Type{
								Name: "any",
							},
							Comment: "prop",
						},
						{
							Name: "disabled",
							Type: &graph.Type{
								Name: "any",
							},
							Comment: "prop",
						},
					},
					Methods:    []*graph.Function{},
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Component with Hooks",
			source: `import React, { useState, useEffect } from 'react';

function Timer() {
  const [seconds, setSeconds] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setSeconds(seconds => seconds + 1);
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return <div>Seconds: {seconds}</div>;
}

export default Timer;`,
			want: []*graph.Type{
				{
					Name:       "Timer",
					Kind:       reflect.Struct,
					Fields:     []*graph.Field{},
					Methods:    []*graph.Function{},
					IsExported: true,
				},
			},
			wantErr: false,
		},
		{
			name: "Arrow Function Component",
			source: `import React from 'react';

const Welcome = (props) => {
  return <h1>Welcome, {props.name}!</h1>;
};

export default Welcome;`,
			want: []*graph.Type{
				{
					Name: "Welcome",
					Kind: reflect.Struct,
					Fields: []*graph.Field{
						{
							Name: "props",
							Type: &graph.Type{
								Name: "any",
							},
							Comment: "prop",
						},
					},
					Methods:    []*graph.Function{},
					IsExported: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := jsx.NewInspector(&graph.Config{IncludeUnexported: true})
			file, err := inspector.InspectSource([]byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.InspectSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if file == nil {
				if !tt.wantErr {
					t.Errorf("Inspector.InspectSource() returned nil file, expected non-nil")
				}
				return
			}

			// Verify the file path
			assert.Equal(t, "source.jsx", file.Path, "File path should be source.jsx")

			// Strip location information before comparing
			strippedActual := stripLocations(file.Types)
			strippedExpected := stripLocations(tt.want)

			// Compare only essential fields, ignoring location and other metadata
			if !assert.EqualValues(t, strippedExpected, strippedActual) {
				gotJSON, _ := json.Marshal(strippedActual)
				wantJSON, _ := json.Marshal(strippedExpected)
				fmt.Printf("got:\n%s\nwant:\n%s\n", gotJSON, wantJSON)
			}
		})
	}
}
