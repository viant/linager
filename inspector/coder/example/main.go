package main

import (
	"context"
	"fmt"
	"github.com/viant/linager/inspector/coder"
	"github.com/viant/linager/inspector/graph"
	"os"
	"path/filepath"
	"reflect"
)

func main() {
	// Create a new project
	project := &graph.Project{
		Name:     "example",
		Type:     "go",
		RootPath: "/path/to/project",
	}

	// Create a new coder
	c := coder.NewCoder(project)

	// Create a new package
	pkg := c.CreatePackage("main", "github.com/example/main")
	fmt.Printf("Created package: %s\n", pkg.Name)

	// Create a new file
	file, err := c.CreateFile("main", "main.go", "main.go")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	fmt.Printf("Created file: %s\n", file.Name)

	// Create a new type
	userType, err := c.CreateType("main", "main.go", "User", reflect.Struct)
	if err != nil {
		fmt.Printf("Error creating type: %v\n", err)
		return
	}
	fmt.Printf("Created type: %s\n", userType.Name)

	// Create fields for the User type
	idField, err := c.CreateField("main", "main.go", "User", "ID", nil, `json:"id"`)
	if err != nil {
		fmt.Printf("Error creating field: %v\n", err)
		return
	}
	fmt.Printf("Created field: %s\n", idField.Name)

	nameField, err := c.CreateField("main", "main.go", "User", "Name", nil, `json:"name"`)
	if err != nil {
		fmt.Printf("Error creating field: %v\n", err)
		return
	}
	fmt.Printf("Created field: %s\n", nameField.Name)

	// Create a method for the User type
	getNameMethod, err := c.CreateMethod("main", "main.go", "User", "GetName", nil, nil, "return u.Name")
	if err != nil {
		fmt.Printf("Error creating method: %v\n", err)
		return
	}
	fmt.Printf("Created method: %s\n", getNameMethod.Name)

	// Create a function
	mainFunc, err := c.CreateFunction("main", "main.go", "main", nil, nil, "fmt.Println(\"Hello, World!\")")
	if err != nil {
		fmt.Printf("Error creating function: %v\n", err)
		return
	}
	fmt.Printf("Created function: %s\n", mainFunc.Name)

	// Create a new type with selected fields from User
	userInfoType, err := c.CreateTypeFromFields("main", "main.go", "UserInfo", "User", []string{"Name"})
	if err != nil {
		fmt.Printf("Error creating type from fields: %v\n", err)
		return
	}
	fmt.Printf("Created type from fields: %s\n", userInfoType.Name)

	// Create a new type with selected methods from User
	userMethodsType, err := c.CreateTypeFromMethods("main", "main.go", "UserMethods", "User", []string{"GetName"})
	if err != nil {
		fmt.Printf("Error creating type from methods: %v\n", err)
		return
	}
	fmt.Printf("Created type from methods: %s\n", userMethodsType.Name)

	// Create a composite type
	compositeType, err := c.CreateCompositeType("main", "main.go", "CompositeUser", []string{"User", "UserInfo"}, [][]string{{"ID"}, {"Name"}}, [][]string{{"GetName"}, {}})
	if err != nil {
		fmt.Printf("Error creating composite type: %v\n", err)
		return
	}
	fmt.Printf("Created composite type: %s\n", compositeType.Name)

	// Apply a patch diff to the main function
	err = c.ApplyPatchDiff("main", "main.go", "", "main", "function", "fmt.Println(\"Hello, Modified World!\")")
	if err != nil {
		fmt.Printf("Error applying patch diff: %v\n", err)
		return
	}
	fmt.Println("Applied patch diff to main function")

	// Print the modified function body
	fmt.Printf("Modified function body: %s\n", mainFunc.Body.Text)

	// Remove a field
	removed := c.RemoveField("main", "main.go", "User", "ID")
	fmt.Printf("Removed field ID: %v\n", removed)

	// Remove a method
	removed = c.RemoveMethod("main", "main.go", "User", "GetName")
	fmt.Printf("Removed method GetName: %v\n", removed)

	// Remove a function
	removed = c.RemoveFunction("main", "main.go", "main")
	fmt.Printf("Removed function main: %v\n", removed)

	// Remove a type
	removed = c.RemoveType("main", "main.go", "User")
	fmt.Printf("Removed type User: %v\n", removed)

	// Remove a file
	removed = c.RemoveFile("main", "main.go")
	fmt.Printf("Removed file main.go: %v\n", removed)

	// Remove a package
	removed = c.RemovePackage("main")
	fmt.Printf("Removed package main: %v\n", removed)

	// Example of loading and storing a project
	fmt.Println("\n--- LoadProject and StoreProject Example ---")

	// Create a new coder
	projectCoder := coder.NewCoder(nil)

	// Get the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Path to the project to load (adjust this to a valid Go project path)
	projectPath := filepath.Join(currentDir, "..", "..", "..")
	fmt.Printf("Loading project from: %s\n", projectPath)

	// Load the project
	ctx := context.Background()
	err = projectCoder.LoadProject(ctx, projectPath)
	if err != nil {
		fmt.Printf("Error loading project: %v\n", err)
		return
	}

	fmt.Printf("Loaded project: %s (type: %s)\n", projectCoder.Project.Name, projectCoder.Project.Type)
	fmt.Printf("Number of packages: %d\n", len(projectCoder.Project.Packages))

	// Create a temporary directory to store the project
	tempDir, err := os.MkdirTemp("", "project-store-example")
	if err != nil {
		fmt.Printf("Error creating temporary directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	fmt.Printf("Storing project to: %s\n", tempDir)

	// Store the project
	err = projectCoder.StoreProject(ctx, tempDir)
	if err != nil {
		fmt.Printf("Error storing project: %v\n", err)
		return
	}

	fmt.Println("Project stored successfully!")

	// Count the number of files stored
	var fileCount int
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error counting files: %v\n", err)
		return
	}

	fmt.Printf("Number of files stored: %d\n", fileCount)
}
