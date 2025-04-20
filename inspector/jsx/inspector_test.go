package jsx_test

import (
	"testing"

	"github.com/viant/linager/inspector/graph"
	"github.com/viant/linager/inspector/jsx"
)

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantPath string
		wantErr  bool
	}{
		{
			name: "Function Component",
			source: `import React from 'react';

function Greeting(props) {
  return <h1>Hello, {props.name}!</h1>;
}

export default Greeting;`,
			wantPath: "source.jsx",
			wantErr:  false,
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
			wantPath: "source.jsx",
			wantErr:  false,
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
			wantPath: "source.jsx",
			wantErr:  false,
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
			wantPath: "source.jsx",
			wantErr:  false,
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

			if file.Path != tt.wantPath {
				t.Errorf("Inspector.InspectSource() file path = %s, want %s", file.Path, tt.wantPath)
			}

			// Basic validation to ensure we got something back
			// Note: Since the JSX inspector is currently a placeholder implementation,
			// we're not checking for specific types, functions, etc.
			// Once the implementation is complete, these tests should be expanded.
		})
	}
}

func TestInspector_InspectFile(t *testing.T) {
	// This test requires actual JSX files on disk, so we'll skip it
	t.Skip("Skipping file-based tests - requires JSX files on disk")
}

func TestInspector_InspectPackage(t *testing.T) {
	// This test requires actual JSX packages on disk, so we'll skip it
	t.Skip("Skipping package-based tests - requires JSX packages on disk")
}

func TestInspector_processJSXComponents(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantType string
		wantErr  bool
	}{
		{
			name: "Function Component",
			source: `function Greeting(props) {
  return <h1>Hello, {props.name}!</h1>;
}`,
			wantType: "Greeting",
			wantErr:  false,
		},
		{
			name: "Arrow Function Component",
			source: `const Welcome = (props) => {
  return <h1>Welcome, {props.name}!</h1>;
};`,
			wantType: "Welcome",
			wantErr:  false,
		},
		{
			name: "Class Component",
			source: `class Counter extends React.Component {
  render() {
    return <div>Count: {this.state.count}</div>;
  }
}`,
			wantType: "Counter",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := jsx.NewInspector(&graph.Config{IncludeUnexported: true})

			// Since processJSXComponents is not exported, we'll test it indirectly
			// through InspectSource
			file, err := inspector.InspectSource([]byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.processJSXComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Basic validation to ensure we got something back
			if file == nil {
				if !tt.wantErr {
					t.Errorf("Inspector.InspectSource() returned nil file, expected non-nil")
				}
				return
			}

			// Note: Since the JSX inspector is currently a placeholder implementation,
			// we're not checking for specific types.
			// Once the implementation is complete, these tests should check for the component type:
			// - For function components: check if a Type with the correct name exists
			// - For class components: check if a Type with methods (including render) exists
			// - For components with props: check if the Type has the correct fields
		})
	}
}
