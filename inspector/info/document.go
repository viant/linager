package info

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/viant/afs"
	"path"
	"strings"
)

// DocumentKind indicates the type of code element that the document represents
type DocumentKind string

const (
	// Document kinds
	KindConstant   DocumentKind = "Constant"
	KindVariable   DocumentKind = "Variable"
	KindFileFunc   DocumentKind = "Function" // Function without a receiver
	KindType       DocumentKind = "Type"     // Type declaration only
	KindTypeMethod DocumentKind = "Method"
	KindTypeField  DocumentKind = "Field"
)

// Document represents a code element with its metadata for vector embedding
type Document struct {
	Kind      DocumentKind `json:"kind"`      // Kind of document
	Path      string       `json:"path"`      // File path
	Package   string       `json:"package"`   // Package name
	Name      string       `json:"name"`      // Element name
	Type      string       `json:"type"`      // Type of the element (e.g., function signature)
	Hash      string       `json:"hash"`      // Hash of the content
	Signature string       `json:"signature"` //Signature
	Content   string       `json:"content"`   // Full content of the element including comments, annotations, etc.
}

func (d *Document) ID() string {
	builder := strings.Builder{}
	builder.WriteString(string(d.Kind))
	builder.WriteString(":")
	builder.WriteString(d.Path)
	builder.WriteString(":")
	if d.Signature != "" {
		builder.WriteString(d.Signature)
	} else {
		builder.WriteString(d.Name)
	}
	builder.WriteString(":")

	return builder.String()
}

// HashContent generates a SHA-256 hash of the document content
func (d *Document) HashContent() string {
	hasher := sha256.New()
	hasher.Write([]byte(d.Content))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateDocuments creates Document instances for embedding from a project
func (p *Project) CreateDocuments(ctx context.Context) ([]*Document, error) {
	var documents []*Document
	fs := afs.New()
	for _, pkg := range p.Packages {
		var typeFields = map[string]int{}
		for _, file := range pkg.FileSet {
			location := path.Join(p.RootPath, file.Path)
			source, err := fs.DownloadWithURL(ctx, location)
			if err != nil {
				return nil, err
			}
			// Process constants
			for _, constant := range file.Constants {
				content := ""
				if constant.Location != nil {
					content = string(source[constant.Location.Start:constant.Location.End])
				}

				doc := &Document{
					Kind:    KindConstant,
					Package: pkg.Name,
					Name:    constant.Name,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents = append(documents, doc)
			}

			// Process variables
			for _, variable := range file.Variables {
				content := ""
				if variable.Location != nil {
					content = string(source[variable.Location.Start:variable.Location.End])
				}

				typeName := ""
				if variable.Type != nil {
					typeName = variable.Type.Name
				}

				doc := &Document{
					Kind:    KindVariable,
					Package: pkg.Name,
					Name:    variable.Name,
					Type:    typeName,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents = append(documents, doc)
			}

			// Process file functions (without receiver)
			for _, function := range file.Functions {
				if function.Receiver == "" {
					doc := &Document{
						Kind:      KindFileFunc,
						Package:   pkg.Name,
						Path:      file.Path,
						Signature: function.Signature,
						Name:      function.Name,
						Content:   function.Content(source),
					}
					doc.Hash = doc.HashContent()
					documents = append(documents, doc)
				}
			}

			// Process types
			for _, aType := range file.Types {
				typeFields[aType.Name] = +len(aType.Fields)

				if len(aType.Fields) > 0 {
					// Pure type (type declaration)
					content := aType.Content(source)
					doc := &Document{
						Kind:    KindType,
						Package: pkg.Name,
						Path:    file.Path,
						Name:    aType.Name,
						Content: content,
					}
					doc.Hash = doc.HashContent()
					documents = append(documents, doc)
				}
				// Type fields
				if len(aType.Fields) > 0 {
					for _, field := range aType.Fields {
						if field.Location != nil {
							fieldContent := field.Content(source)

							// Individual field
							fieldDoc := &Document{
								Kind:    KindTypeField,
								Package: pkg.Name,
								Name:    field.Name,
								Path:    file.Path,
								Type:    aType.Name,
								Content: fieldContent,
							}
							fieldDoc.Hash = fieldDoc.HashContent()
							documents = append(documents, fieldDoc)
						}
					}
				}

				// Type methods
				for _, method := range aType.Methods {
					methodDoc := &Document{
						Kind:      KindTypeMethod,
						Package:   pkg.Name,
						Name:      method.Name,
						Path:      file.Path,
						Type:      aType.Name,
						Signature: method.Signature,
						Content:   method.Content(source),
					}
					methodDoc.Hash = methodDoc.HashContent()
					documents = append(documents, methodDoc)
				}
			}
			for typeName, count := range typeFields {
				if count > 0 {
					continue
				}
				aType := file.LookupType(typeName)
				if aType == nil {
					continue
				}
				// Pure type (type declaration)
				content := aType.Content(source)
				doc := &Document{
					Kind:    KindType,
					Package: pkg.Name,
					Name:    aType.Name,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents = append(documents, doc)
			}
		}
	}
	return documents, nil
}

// AddFunctionToFile adds a function to a file if it doesn't already exist
func (p *Project) AddFunctionToFile(packageName, fileName, functionName, functionContent string) error {
	pkg := p.GetPackage(packageName)
	if pkg == nil {
		return fmt.Errorf("package %s not found", packageName)
	}

	var targetFile *File
	for _, file := range pkg.FileSet {
		if strings.HasSuffix(file.Path, fileName) || file.Name == fileName {
			targetFile = file
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("file %s not found in package %s", fileName, packageName)
	}

	// Check if function already exists
	if targetFile.HasFunction(functionName) {
		return nil // Function already exists, nothing to do
	}

	// Create a new function
	newFunction := &Function{
		Name:       functionName,
		IsExported: strings.ToUpper(functionName[:1]) == functionName[:1],
		Location:   &Location{}, // Actual location will be filled in by parser
		Body:       &LocationNode{Text: functionContent},
	}

	// Add function to file
	targetFile.Functions = append(targetFile.Functions, newFunction)

	// Update function map
	if targetFile.functionMap == nil {
		targetFile.functionMap = make(map[string]int)
	}
	targetFile.functionMap[functionName] = len(targetFile.Functions) - 1

	return nil
}
