package graph

import (
	"context"
	"fmt"
	"strings"
)

const chunkSize = 8192 - 256

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
	KindAsset      DocumentKind = "Asset" // Package-level information
	KindCode       DocumentKind = "Code"  // Package-level information

)

// Document represents a code element with its metadata for vector embedding
type Document struct {
	ID        string       `json:"id"`        // Unique identifier for the document
	Kind      DocumentKind `json:"kind"`      // Kind of document
	Project   string       `json:"project"`   // Project name
	Path      string       `json:"path"`      // File path
	Package   string       `json:"package"`   // Package name
	Name      string       `json:"name"`      // Element name
	Type      string       `json:"type"`      // Type of the element (e.g., function signature)
	Hash      uint64       `json:"hash"`      // Hash of the content
	Signature string       `json:"signature"` //Signature
	Content   string       `json:"content"`   // Full content of the element including comments, annotations, etc.
	Part      int          `json:"part"`      // Part number for large documents
}

type Documents []*Document

func (d *Documents) Append(doc *Document) {
	if len(doc.Content) > chunkSize {
		// Split the document into chunks
		*d = append(*d, SplitDocument(doc)...)
		return
	}
	*d = append(*d, doc)
}

func (d Documents) Size() int {
	size := 0
	for _, doc := range d {
		if doc == nil {
			continue
		}
		size += doc.Size()
	}
	return size

}

// SplitDocument splits a large document into multiple chunks of 8k - 256 bytes.
func SplitDocument(doc *Document) Documents {
	content := doc.Content
	var docs Documents
	n := len(content)

	if n <= chunkSize {
		// No need to split, return as a single document
		doc.Part = 0
		docs.Append(doc)
		return docs
	}

	// Split into chunks
	for i, start := 0, 0; start < n; i++ {
		end := start + chunkSize
		if end > n {
			end = n
		}

		// Create a new document chunk
		chunk := &Document{
			Kind:      doc.Kind,
			Project:   doc.Project,
			Package:   doc.Package,
			Path:      doc.Path,
			Name:      doc.Name,
			Type:      doc.Type,
			Signature: doc.Signature,
			Content:   content[start:end],
			Part:      i + 1,
			Hash:      doc.HashContent(),
		}
		docs.Append(chunk)
		start = end
	}

	return docs
}

func (d *Document) Size() int {
	size := len(d.Content) + len(d.Type) + len(d.Signature) + len(d.Path)
	if d.Kind == KindType {
		size += len(d.Name)
	}
	return size + 20 //keys in meta
}

func (d Documents) FilterBySize(totalSize int) Documents {
	size := 0
	var result Documents
	for _, doc := range d {
		if doc == nil {
			continue
		}
		size += doc.Size()
		if size >= totalSize {
			break
		}
		result = append(result, doc)
	}
	return result
}

func (d Documents) GroupBy() Documents {
	// Group documents by path
	pathMap := make(map[string][]*Document)
	orderedPaths := make([]string, 0)

	// First pass: group documents by file path
	for _, doc := range d {
		if doc == nil {
			continue
		}
		if _, ok := pathMap[doc.Path]; !ok {
			orderedPaths = append(orderedPaths, doc.Path)
		}
		pathMap[doc.Path] = append(pathMap[doc.Path], doc)

	}

	var result Documents

	// Second pass: create code documents for each file
outer:
	for _, path := range orderedPaths {
		docs := pathMap[path]
		// Extract package name (should be the same for all docs in a file)
		pkgName := ""
		for _, doc := range docs {
			if doc.Kind == KindAsset {
				result = append(result, doc)
				continue outer
			}
			if doc.Package != "" {
				pkgName = doc.Package
				break
			}
			continue
		}

		if pkgName == "" {
			continue // Skip if we can't determine the package
		}

		// Start constructing the file content
		fileContent := fmt.Sprintf("package %s\n\n", pkgName)

		// Find imports (we would need to add proper import detection)
		// TODO: Properly extract imports from documents

		// Group documents by their kinds for proper ordering
		var types []*Document
		var consts []*Document
		var vars []*Document
		var funcs []*Document
		var methods []*Document

		for _, doc := range docs {
			switch doc.Kind {
			case KindType:
				types = append(types, doc)
			case KindConstant:
				consts = append(consts, doc)
			case KindVariable:
				vars = append(vars, doc)
			case KindFileFunc:
				funcs = append(funcs, doc)
			case KindTypeMethod:
				methods = append(methods, doc)
			}
		}

		// Add constants
		if len(consts) > 0 {
			for _, c := range consts {
				fileContent += c.Content + "\n\n"
			}
		}

		// Add variables
		if len(vars) > 0 {
			for _, v := range vars {
				fileContent += v.Content + "\n\n"
			}
		}

		// Add types
		if len(types) > 0 {
			for _, t := range types {
				fileContent += t.Content + "\n\n"
			}
		}

		// Add functions (without receivers)
		if len(funcs) > 0 {
			for _, f := range funcs {
				fileContent += f.Content + "\n\n"
			}
		}

		// Add methods - group by receiver type
		receiverMethods := make(map[string][]*Document)
		for _, m := range methods {
			if m.Type != "" { // Only process methods with a receiver
				receiverMethods[m.Type] = append(receiverMethods[m.Type], m)
			}
		}

		for _, typeMethods := range receiverMethods {
			for _, m := range typeMethods {
				fileContent += m.Content + "\n\n"
			}
		}

		// Create a document for the entire file
		codeDoc := &Document{
			Kind:    KindCode,
			Project: d[0].Project,
			Path:    path,
			Package: pkgName,
			Name:    strings.TrimSuffix(strings.TrimPrefix(path, pkgName+"/"), ".go"),
			Content: fileContent,
		}
		codeDoc.Hash = codeDoc.HashContent()
		result = append(result, codeDoc)
	}

	return result
}

func (d *Document) GetID() string {
	if d.ID != "" {
		return d.ID
	}
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
	d.ID = builder.String()
	return d.ID
}

// HashContent generates content hash
func (d *Document) HashContent() uint64 {
	hash, _ := Hash([]byte(d.Content))
	return hash
}

// CreateDocuments creates Document instances for embedding from a project
func (p *Project) CreateDocuments(ctx context.Context, pkgPath string) (Documents, error) {
	var documents Documents

	for _, pkg := range p.Packages {

		if len(pkg.Assets) > 0 {
			candidatePath := pkg.Assets[0].Path
			if pkgPath != "" && !strings.HasPrefix(candidatePath, pkgPath) {
				continue // Skip packages that don't match the specified package path
			}
			for _, asset := range pkg.Assets {
				if len(asset.Content) > 16*1024 { //for not skipping, needs to split
					continue
				}
				methodDoc := &Document{
					Kind:    KindAsset,
					Project: p.Name,
					Package: pkg.Name,
					Name:    asset.Name,
					Path:    asset.Path,
					Content: string(asset.Content),
				}
				methodDoc.Hash = methodDoc.HashContent()
				documents.Append(methodDoc)

			}
		}

		if len(pkg.FileSet) == 0 {
			continue
		}

		candidatePath := pkg.FileSet[0].Path
		if pkgPath != "" && !strings.HasPrefix(candidatePath, pkgPath) {
			continue // Skip packages that don't match the specified package path
		}

		var typeFields = map[string]int{}
		for _, file := range pkg.FileSet {
			// Process constants
			for _, constant := range file.Constants {
				content := ""
				if constant.Location != nil {
					content = constant.Location.Raw
				}

				doc := &Document{
					Kind:    KindConstant,
					Project: p.Name,
					Package: pkg.Name,
					Name:    constant.Name,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents.Append(doc)
			}

			// Process variables
			for _, variable := range file.Variables {
				content := variable.Value
				if variable.Location != nil {
					content = variable.Location.Raw
				}

				typeName := ""
				if variable.Type != nil {
					typeName = variable.Type.Name
				}

				doc := &Document{
					Kind:    KindVariable,
					Project: p.Name,
					Package: pkg.Name,
					Name:    variable.Name,
					Type:    typeName,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents.Append(doc)
			}

			// Process file functions (without receiver)
			for _, function := range file.Functions {
				if function.Receiver == "" {
					doc := &Document{
						Kind:      KindFileFunc,
						Project:   p.Name,
						Package:   pkg.Name,
						Path:      file.Path,
						Signature: function.Signature,
						Name:      function.Name,
						Content:   function.Content(),
					}
					doc.Hash = doc.HashContent()
					documents.Append(doc)
				}
			}

			// Process types
			for _, aType := range file.Types {
				typeFields[aType.Name] = +len(aType.Fields)

				if len(aType.Fields) > 0 {
					// Pure type (type declaration)
					content := aType.Content()
					doc := &Document{
						Kind:    KindType,
						Project: p.Name,
						Package: pkg.Name,
						Path:    file.Path,
						Name:    aType.Name,
						Content: content,
					}
					doc.Hash = doc.HashContent()
					documents.Append(doc)
				}
				// Type fields
				if len(aType.Fields) > 0 {
					for _, field := range aType.Fields {
						if field.Location != nil {
							fieldContent := field.Content()

							// Individual field
							fieldDoc := &Document{
								Kind:    KindTypeField,
								Project: p.Name,
								Package: pkg.Name,
								Name:    field.Name,
								Path:    file.Path,
								Type:    aType.Name,
								Content: fieldContent,
							}
							fieldDoc.Hash = fieldDoc.HashContent()
							documents.Append(fieldDoc)
						}
					}
				}

				// Type methods
				for _, method := range aType.Methods {
					methodDoc := &Document{
						Kind:      KindTypeMethod,
						Project:   p.Name,
						Package:   pkg.Name,
						Path:      file.Path,
						Type:      aType.Name,
						Signature: method.Signature,
						Content:   method.Content(),
					}
					methodDoc.Hash = methodDoc.HashContent()
					documents.Append(methodDoc)
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
				content := aType.Content()
				doc := &Document{
					Kind:    KindType,
					Project: p.Name,
					Package: pkg.Name,
					Name:    aType.Name,
					Path:    file.Path,
					Content: content,
				}
				doc.Hash = doc.HashContent()
				documents.Append(doc)
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
