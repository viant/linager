package java

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/inspector/graph"
	"reflect"
	"strings"
)

// parsePackageDeclaration extracts the package name from a Java source file
func parsePackageDeclaration(node *sitter.Node, source []byte) string {
	if node.Type() != "package_declaration" {
		return ""
	}

	nameNode := node.NamedChild(0)
	if nameNode == nil {
		return ""
	}

	return nameNode.Content(source)
}

// parseImportDeclarations extracts import declarations from a Java source file
func parseImportDeclarations(node *sitter.Node, source []byte) map[string]string {
	imports := make(map[string]string)

	if node.Type() != "import_declaration" {
		return imports
	}

	importNode := node.NamedChild(0)
	if importNode == nil {
		return imports
	}

	// Handle static imports
	if importNode.Type() == "static_import" {
		// Extract the fully qualified name
		scopeNode := importNode.ChildByFieldName("scope")
		if scopeNode != nil {
			packagePath := scopeNode.Content(source)
			// Extract the class name and static member
			lastDotIndex := strings.LastIndex(packagePath, ".")
			if lastDotIndex != -1 {
				className := packagePath[lastDotIndex+1:]
				packagePrefix := packagePath[:lastDotIndex]
				imports[className] = packagePrefix
			}
		}
		return imports
	}

	// Handle regular import
	scopeNode := importNode.ChildByFieldName("scope")
	nameNode := importNode.ChildByFieldName("name")

	if scopeNode != nil && nameNode != nil {
		packagePath := scopeNode.Content(source)
		className := nameNode.Content(source)
		imports[className] = packagePath
	} else if scopeNode != nil {
		// Handle wildcard imports (import java.util.*)
		packagePath := scopeNode.Content(source)
		// For wildcard imports, we use the package name as the key
		lastDot := strings.LastIndex(packagePath, ".")
		if lastDot != -1 {
			packageName := packagePath[lastDot+1:]
			imports[packageName+".*"] = packagePath
		}
	}

	return imports
}

// parseClassDeclaration extracts class information from a Java source file
func parseClassDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *graph.Type {
	if node.Type() != "class_declaration" {
		return nil
	}

	// Extract class name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	className := nameNode.Content(source)

	// Create class type with location information
	classType := &graph.Type{
		Name:       className,
		Kind:       reflect.Struct,
		IsExported: isNodePublic(node, source),
		Fields:     []*graph.Field{},
		Methods:    []*graph.Function{},
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract documentation and annotation using the helper function
	classType.Comment, classType.Annotation = extractDocumentation(node, source)

	// Extract type parameters (generics)
	classType.TypeParams = extractTypeParameters(node, source)

	// Extract superclass and interfaces
	superclassNode := node.ChildByFieldName("superclass")
	if superclassNode != nil {
		superclassName := superclassNode.Content(source)
		classType.Extends = append(classType.Extends, superclassName)

		// Try to resolve the package path for the superclass
		if packagePath, ok := importMap[extractSimpleTypeName(superclassName)]; ok {
			classType.Extends[0] = packagePath + "." + extractSimpleTypeName(superclassName)
		}
	}

	// Extract implemented interfaces
	interfacesNode := node.ChildByFieldName("interfaces")
	if interfacesNode != nil {
		for i := uint32(0); i < interfacesNode.NamedChildCount(); i++ {
			interfaceNode := interfacesNode.NamedChild(int(i))
			interfaceName := interfaceNode.Content(source)

			// Try to fully qualify the interface name
			if packagePath, ok := importMap[extractSimpleTypeName(interfaceName)]; ok {
				interfaceName = packagePath + "." + extractSimpleTypeName(interfaceName)
			}

			classType.Implements = append(classType.Implements, interfaceName)
		}
	}

	// Extract class body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Parse fields and methods
		for i := uint32(0); i < bodyNode.NamedChildCount(); i++ {
			child := bodyNode.NamedChild(int(i))

			switch child.Type() {
			case "field_declaration":
				field := parseFieldDeclaration(child, source, importMap)
				if field != nil {
					classType.Fields = append(classType.Fields, field)
				}

			case "method_declaration":
				method := parseMethodDeclaration(child, source, importMap)
				if method != nil {
					classType.Methods = append(classType.Methods, method)
				}
			case "constructor_declaration":
				constructor := parseConstructorDeclaration(child, source, className, importMap)
				if constructor != nil {
					classType.Methods = append(classType.Methods, constructor)
				}
			}
		}
	}

	return classType
}

// parseInterfaceDeclaration extracts interface information from a Java source file
func parseInterfaceDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *graph.Type {
	if node.Type() != "interface_declaration" {
		return nil
	}

	// Extract interface name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	interfaceName := nameNode.Content(source)

	// Create interface type with location information
	interfaceType := &graph.Type{
		Name:       interfaceName,
		Kind:       reflect.Interface,
		IsExported: isNodePublic(node, source),
		Methods:    []*graph.Function{},
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract comment/documentation and annotation using the helper function
	interfaceType.Comment, interfaceType.Annotation = extractDocumentation(node, source)

	// Extract type parameters (generics)
	interfaceType.TypeParams = extractTypeParameters(node, source)

	// Extract extended interfaces
	extendsNode := node.ChildByFieldName("interfaces")
	if extendsNode != nil {
		for i := uint32(0); i < extendsNode.NamedChildCount(); i++ {
			extendedNode := extendsNode.NamedChild(int(i))
			extendedName := extendedNode.Content(source)

			// Try to fully qualify the extended interface name
			if packagePath, ok := importMap[extractSimpleTypeName(extendedName)]; ok {
				extendedName = packagePath + "." + extractSimpleTypeName(extendedName)
			}

			// Add to extends list
			interfaceType.Extends = append(interfaceType.Extends, extendedName)
		}
	}

	// Extract interface body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Parse methods
		for i := uint32(0); i < bodyNode.NamedChildCount(); i++ {
			child := bodyNode.NamedChild(int(i))

			if child.Type() == "method_declaration" {
				method := parseMethodDeclaration(child, source, importMap)
				if method != nil {
					interfaceType.Methods = append(interfaceType.Methods, method)
				}
			}
		}
	}

	return interfaceType
}

// parseEnumDeclaration extracts enum information from a Java source file
func parseEnumDeclaration(node *sitter.Node, source []byte) *graph.Type {
	if node.Type() != "enum_declaration" {
		return nil
	}

	// Extract enum name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	enumName := nameNode.Content(source)

	// Create enum type with location information
	enumType := &graph.Type{
		Name:       enumName,
		Kind:       reflect.Int,
		IsExported: isNodePublic(node, source),
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract comment/documentation and annotation using the helper function
	enumType.Comment, enumType.Annotation = extractDocumentation(node, source)

	// Extract enum body and constants
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		// Process enum values handled separately
	}

	return enumType
}

// parseAnnotationTypeDeclaration extracts annotation information from a Java source file
func parseAnnotationTypeDeclaration(node *sitter.Node, source []byte) *graph.Type {
	if node.Type() != "annotation_type_declaration" {
		return nil
	}

	// Extract annotation name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	annotationName := nameNode.Content(source)

	// Create annotation type with location information
	annotationType := &graph.Type{
		Name:       annotationName,
		Kind:       reflect.Interface, // Annotations are interfaces in Java
		IsExported: isNodePublic(node, source),
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract comment/documentation and annotation using the helper function
	annotationType.Comment, annotationType.Annotation = extractDocumentation(node, source)

	return annotationType
}

// parseFieldDeclaration extracts field information from a class
func parseFieldDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *graph.Field {
	if node.Type() != "field_declaration" {
		return nil
	}

	// Get type
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return nil
	}
	fieldType := parseJavaType(typeNode, source, importMap)

	// Get declarator (name and possibly value)
	declaratorNode := node.ChildByFieldName("declarator")
	if declaratorNode == nil {
		return nil
	}

	nameNode := declaratorNode.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	fieldName := nameNode.Content(source)

	// Extract documentation and annotation
	comment, annotation := extractDocumentation(node, source)

	// Check if field is final and static (constant)
	isFinal := false
	isStatic := false

	if node.NamedChild(0).Type() == "modifiers" {
		modifiersNode := node.NamedChild(0)
		for i := uint32(0); i < modifiersNode.NamedChildCount(); i++ {
			modifier := modifiersNode.NamedChild(int(i))
			if modifier.Type() == "final" {
				isFinal = true
			}
			if modifier.Type() == "static" {
				isStatic = true
			}
		}
	}

	// Create field with location information
	field := &graph.Field{
		Name:       fieldName,
		Type:       fieldType,
		Comment:    comment.Text,
		Annotation: annotation.Text,
		IsExported: isNodePublic(node, source),
		IsStatic:   isStatic,
		IsConstant: isFinal && isStatic,
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	return field
}

// parseMethodDeclaration extracts method information from a class
func parseMethodDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *graph.Function {
	if node.Type() != "method_declaration" {
		return nil
	}

	// Get method name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	methodName := nameNode.Content(source)

	// Get return type
	typeNode := node.ChildByFieldName("type")
	var returnType *graph.Type
	if typeNode != nil {
		returnType = parseJavaType(typeNode, source, importMap)
	}

	// Extract documentation and annotation
	comment, annotation := extractDocumentation(node, source)

	// Extract type parameters (generics) for the method
	typeParams := extractTypeParameters(node, source)

	// Check if method is static
	isStatic := false
	if node.NamedChildCount() > 0 && node.NamedChild(0).Type() == "modifiers" {
		modifiersNode := node.NamedChild(0)
		for i := uint32(0); i < modifiersNode.NamedChildCount(); i++ {
			modifier := modifiersNode.NamedChild(int(i))
			if modifier.Type() == "static" {
				isStatic = true
				break
			}
		}
	}

	// Create method with location information
	method := &graph.Function{
		Name:       methodName,
		Comment:    comment,
		Annotation: annotation,
		IsExported: isNodePublic(node, source),
		Parameters: []*graph.Parameter{},
		TypeParams: typeParams,
		IsStatic:   isStatic,
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
		Signature: formatMethodSignature(methodName, node, source, importMap),
	}

	if returnType != nil {
		method.Results = []*graph.Parameter{
			{
				Type: returnType,
			},
		}
	}

	// Extract parameters
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		for i := uint32(0); i < parametersNode.NamedChildCount(); i++ {
			paramNode := parametersNode.NamedChild(int(i))

			// Handle regular parameters
			if paramNode.Type() == "formal_parameter" {
				paramTypeNode := paramNode.ChildByFieldName("type")
				paramNameNode := paramNode.ChildByFieldName("name")

				if paramTypeNode != nil && paramNameNode != nil {
					paramType := parseJavaType(paramTypeNode, source, importMap)
					paramName := paramNameNode.Content(source)

					method.Parameters = append(method.Parameters, &graph.Parameter{
						Name: paramName,
						Type: paramType,
					})
				}
			}

			// Handle variadic parameters
			if paramNode.Type() == "spread_parameter" {
				if paramNode.NamedChildCount() >= 2 {
					paramTypeNode := paramNode.NamedChild(0)
					paramDeclNode := paramNode.NamedChild(1)

					if paramTypeNode != nil && paramDeclNode != nil {
						paramNameNode := paramDeclNode.ChildByFieldName("name")

						if paramNameNode != nil {
							paramType := parseJavaType(paramTypeNode, source, importMap)
							paramName := paramNameNode.Content(source)

							// Mark this as variadic by making it a slice type
							if paramType != nil {
								paramType.Name = "[]" + paramType.Name
								paramType.Kind = reflect.Slice
							}

							method.Parameters = append(method.Parameters, &graph.Parameter{
								Name: paramName,
								Type: paramType,
							})
						}
					}
				}
			}
		}
	}

	// Extract method body with location information
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		method.Body = &graph.LocationNode{
			Text: bodyNode.Content(source),
			Location: graph.Location{
				Start: int(bodyNode.StartByte()),
				End:   int(bodyNode.EndByte()),
			},
		}
	}

	return method
}

// parseConstructorDeclaration extracts constructor information from a class
func parseConstructorDeclaration(node *sitter.Node, source []byte, className string, importMap map[string]string) *graph.Function {
	if node.Type() != "constructor_declaration" {
		return nil
	}

	// Extract documentation and annotation
	comment, annotation := extractDocumentation(node, source)

	// Get constructor name from nameNode
	nameNode := node.ChildByFieldName("name")
	var constructorName string
	if nameNode != nil {
		constructorName = nameNode.Content(source)
	} else {
		constructorName = className
	}

	// Extract type parameters (generics) for the constructor
	typeParams := extractTypeParameters(node, source)

	// Create constructor method with location information
	constructor := &graph.Function{
		Name:          className,
		Comment:       comment,
		Annotation:    annotation,
		IsExported:    isNodePublic(node, source),
		Parameters:    []*graph.Parameter{},
		TypeParams:    typeParams,
		IsConstructor: true,
		Location: &graph.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
		Signature: formatMethodSignature(constructorName, node, source, importMap),
	}

	// Constructors return the class type
	constructor.Results = []*graph.Parameter{
		{
			Type: &graph.Type{
				Name: className,
			},
		},
	}

	// Extract parameters
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		for i := uint32(0); i < parametersNode.NamedChildCount(); i++ {
			paramNode := parametersNode.NamedChild(int(i))

			if paramNode.Type() == "formal_parameter" {
				paramTypeNode := paramNode.ChildByFieldName("type")
				paramNameNode := paramNode.ChildByFieldName("name")

				if paramTypeNode != nil && paramNameNode != nil {
					paramType := parseJavaType(paramTypeNode, source, importMap)
					paramName := paramNameNode.Content(source)

					constructor.Parameters = append(constructor.Parameters, &graph.Parameter{
						Name: paramName,
						Type: paramType,
					})
				}
			}
		}
	}

	// Extract constructor body with location information
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		constructor.Body = &graph.LocationNode{
			Text: bodyNode.Content(source),
			Location: graph.Location{
				Start: int(bodyNode.StartByte()),
				End:   int(bodyNode.EndByte()),
			},
		}
	}

	return constructor
}

// formatMethodSignature creates a full signature for a method including return type and package information
func formatMethodSignature(name string, node *sitter.Node, source []byte, importMap map[string]string) string {
	var signature strings.Builder

	// Add return type for methods
	if node.Type() == "method_declaration" {
		typeNode := node.ChildByFieldName("type")
		if typeNode != nil {
			returnType := parseJavaType(typeNode, source, importMap)
			if returnType != nil {
				// Try to use fully qualified name for return type
				typeName := returnType.Name
				if returnType.PackagePath != "" && !strings.Contains(typeName, ".") {
					typeName = returnType.PackagePath + "." + typeName
				}
				signature.WriteString(typeName)
				signature.WriteString(" ")
			}
		}
	}

	// Add method name
	signature.WriteString(name)

	// Format type parameters
	typeParams := extractTypeParameters(node, source)
	if len(typeParams) > 0 {
		signature.WriteString("<")
		for i, param := range typeParams {
			if i > 0 {
				signature.WriteString(", ")
			}
			signature.WriteString(param.Name)
			if param.Constraint != "any" {
				signature.WriteString(" extends ")
				// Try to fully qualify constraint types
				constraintParts := strings.Split(param.Constraint, "&")
				for j, part := range constraintParts {
					part = strings.TrimSpace(part)
					if j > 0 {
						signature.WriteString(" & ")
					}
					// Try to fully qualify the constraint type
					if packagePath, ok := importMap[extractSimpleTypeName(part)]; ok {
						signature.WriteString(packagePath + "." + extractSimpleTypeName(part))
					} else {
						signature.WriteString(part)
					}
				}
			}
		}
		signature.WriteString(">")
	}

	// Format parameters
	signature.WriteString("(")
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		var params []string
		for i := uint32(0); i < parametersNode.NamedChildCount(); i++ {
			paramNode := parametersNode.NamedChild(int(i))

			if paramNode.Type() == "formal_parameter" {
				paramTypeNode := paramNode.ChildByFieldName("type")
				paramNameNode := paramNode.ChildByFieldName("name")

				if paramTypeNode != nil && paramNameNode != nil {
					paramType := parseJavaType(paramTypeNode, source, importMap)
					paramName := paramNameNode.Content(source)

					if paramType != nil {
						// Try to use fully qualified name for parameter type
						typeName := paramType.Name
						if paramType.PackagePath != "" && !strings.Contains(typeName, ".") {
							typeName = paramType.PackagePath + "." + typeName
						}
						params = append(params, typeName+" "+paramName)
					}
				}
			} else if paramNode.Type() == "spread_parameter" {
				// Handle variadic parameters
				if paramNode.NamedChildCount() >= 2 {
					paramTypeNode := paramNode.NamedChild(0)
					paramDeclNode := paramNode.NamedChild(1)

					if paramTypeNode != nil && paramDeclNode != nil {
						paramNameNode := paramDeclNode.ChildByFieldName("name")

						if paramNameNode != nil {
							paramType := parseJavaType(paramTypeNode, source, importMap)
							paramName := paramNameNode.Content(source)

							if paramType != nil {
								// Try to use fully qualified name
								typeName := paramType.Name
								if paramType.PackagePath != "" && !strings.Contains(typeName, ".") {
									typeName = paramType.PackagePath + "." + typeName
								}
								params = append(params, typeName+"... "+paramName)
							}
						}
					}
				}
			}
		}
		signature.WriteString(strings.Join(params, ", "))
	}
	signature.WriteString(")")

	// Add exceptions if present
	throwsNode := node.ChildByFieldName("throws")
	if throwsNode != nil {
		signature.WriteString(" throws ")
		var exceptions []string
		for i := uint32(0); i < throwsNode.NamedChildCount(); i++ {
			exceptionNode := throwsNode.NamedChild(int(i))
			exceptionName := exceptionNode.Content(source)

			// Try to fully qualify exception names
			if packagePath, ok := importMap[extractSimpleTypeName(exceptionName)]; ok {
				exceptionName = packagePath + "." + extractSimpleTypeName(exceptionName)
			}
			exceptions = append(exceptions, exceptionName)
		}
		signature.WriteString(strings.Join(exceptions, ", "))
	}

	return signature.String()
}

// parseAnnotationDeclaration extracts annotation declarations
func parseAnnotationDeclaration(node *sitter.Node, source []byte) string {
	if node.Type() != "annotation" && node.Type() != "marker_annotation" {
		return ""
	}

	return "@" + node.Content(source)
}

// isNodePublic checks if a node has the 'public' modifier
func isNodePublic(node *sitter.Node, source []byte) bool {
	if node.NamedChildCount() > 0 && node.NamedChild(0).Type() == "modifiers" {
		modifiersNode := node.NamedChild(0)
		for i := uint32(0); i < modifiersNode.NamedChildCount(); i++ {
			modifier := modifiersNode.NamedChild(int(i))
			if modifier.Type() == "public" {
				return true
			}
		}
	}
	return false
}
