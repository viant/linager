package inspector_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/viant/linager/inspector"
	"github.com/viant/linager/inspector/graph"
)

func TestFactory_GetInspector(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantErr   bool
		inspector string
	}{
		{
			name:      "Go file",
			filename:  "test.go",
			wantErr:   false,
			inspector: "golang",
		},
		{
			name:      "Java file",
			filename:  "Test.java",
			wantErr:   false,
			inspector: "java",
		},
		{
			name:      "JS file",
			filename:  "test.js",
			wantErr:   false,
			inspector: "javascript",
		},
		{
			name:      "JSX file",
			filename:  "Component.jsx",
			wantErr:   false,
			inspector: "javascript",
		},
		{
			name:      "Unsupported file",
			filename:  "test.cpp",
			wantErr:   true,
			inspector: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := inspector.NewFactory(&graph.Config{
				IncludeUnexported: true,
			})

			insp, err := factory.GetInspector(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("Factory.GetInspector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if insp == nil {
					t.Errorf("Factory.GetInspector() returned nil inspector")
					return
				}

				// Check inspector type using the package path
				inspType := GetInspectorType(insp)
				if !strings.Contains(inspType, tt.inspector) {
					t.Errorf("Factory.GetInspector() returned %s inspector, want %s", inspType, tt.inspector)
				}
			}
		})
	}
}

// GetInspectorType returns the package path of the inspector implementation
func GetInspectorType(i interface{}) string {
	return reflect.TypeOf(i).String()
}

func TestFactory_InspectFile(t *testing.T) {
	// Skip the test that requires actual files
	t.Skip("Skipping test that requires actual files on disk")
}

func TestFactory_InspectPackage(t *testing.T) {
	// Skip the test that requires actual package on disk
	t.Skip("Skipping test that requires actual package directory on disk")
}
