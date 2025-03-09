package java

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/viant/linager/inspector/info"
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

	scopeNode := importNode.ChildByFieldName("scope")
	nameNode := importNode.ChildByFieldName("name")

	if scopeNode != nil && nameNode != nil {
		imports[nameNode.Content(source)] = scopeNode.Content(source)
	}

	return imports
}

// parseClassDeclaration extracts class information from a Java source file
func parseClassDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *info.Type {
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
	classType := &info.Type{
		Name:       className,
		Kind:       reflect.Struct,
		IsExported: isNodePublic(node, source),
		Fields:     []info.Field{},
		Methods:    []info.Method{},
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract documentation and annotation using the helper function
	classType.Comment, classType.Annotation = extractDocumentation(node, source)

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
					classType.Fields = append(classType.Fields, *field)
				}

			case "method_declaration":
				method := parseMethodDeclaration(child, source, importMap)
				if method != nil {
					classType.Methods = append(classType.Methods, *method)
				}
			case "constructor_declaration":
				constructor := parseConstructorDeclaration(child, source, className, importMap)
				if constructor != nil {
					classType.Methods = append(classType.Methods, *constructor)
				}
			}
		}
	}

	return classType
}

// parseInterfaceDeclaration extracts interface information from a Java source file
func parseInterfaceDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *info.Type {
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
	interfaceType := &info.Type{
		Name:       interfaceName,
		Kind:       reflect.Interface,
		IsExported: isNodePublic(node, source),
		Methods:    []info.Method{},
		Location: &info.Location{
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

			// Add to extends list
			interfaceType.Extends = append(interfaceType.Extends, extendedName)

			// Try to resolve the package path for the extended interface
			if packagePath, ok := importMap[extractSimpleTypeName(extendedName)]; ok {
				interfaceType.PackagePath = packagePath
			}
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
					interfaceType.Methods = append(interfaceType.Methods, *method)
				}
			}
		}
	}

	return interfaceType
}

// parseEnumDeclaration extracts enum information from a Java source file
func parseEnumDeclaration(node *sitter.Node, source []byte) *info.Type {
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
	enumType := &info.Type{
		Name:       enumName,
		Kind:       reflect.Int,
		IsExported: isNodePublic(node, source),
		Location: &info.Location{
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
func parseAnnotationTypeDeclaration(node *sitter.Node, source []byte) *info.Type {
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
	annotationType := &info.Type{
		Name:       annotationName,
		Kind:       reflect.Interface, // Annotations are interfaces in Java
		IsExported: isNodePublic(node, source),
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	// Extract comment/documentation and annotation using the helper function
	annotationType.Comment, annotationType.Annotation = extractDocumentation(node, source)

	return annotationType
}

// parseFieldDeclaration extracts field information from a class
func parseFieldDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *info.Field {
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
	field := &info.Field{
		Name:       fieldName,
		Type:       fieldType,
		Comment:    comment.Text,
		Annotation: annotation.Text,
		IsExported: isNodePublic(node, source),

		IsConstant: isFinal && isStatic,
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	return field
}

// parseMethodDeclaration extracts method information from a class
func parseMethodDeclaration(node *sitter.Node, source []byte, importMap map[string]string) *info.Method {
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
	var returnType *info.Type
	if typeNode != nil {
		returnType = parseJavaType(typeNode, source, importMap)
	}

	// Extract documentation and annotation
	comment, annotation := extractDocumentation(node, source)

	// Extract type parameters (generics) for the method
	typeParams := extractTypeParameters(node, source)

	// Create method with location information
	method := &info.Method{
		Name:       methodName,
		Comment:    comment,
		Annotation: annotation,
		IsExported: isNodePublic(node, source),
		Parameters: []info.Parameter{},
		TypeParams: typeParams,
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
		Signature: formatMethodSignature(methodName, node, source, importMap),
	}

	if returnType != nil {
		method.Results = []info.Parameter{
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

					method.Parameters = append(method.Parameters, info.Parameter{
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

							method.Parameters = append(method.Parameters, info.Parameter{
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
		method.Body = &info.LocationNode{
			Text: bodyNode.Content(source),
			Location: info.Location{
				Start: int(bodyNode.StartByte()),
				End:   int(bodyNode.EndByte()),
			},
		}
	}

	return method
}

// parseConstructorDeclaration extracts constructor information from a class
func parseConstructorDeclaration(node *sitter.Node, source []byte, className string, importMap map[string]string) *info.Method {
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
	constructor := &info.Method{
		Name:       className,
		Comment:    comment,
		Annotation: annotation,
		IsExported: isNodePublic(node, source),
		Parameters: []info.Parameter{},
		TypeParams: typeParams,
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
		Signature: formatMethodSignature(constructorName, node, source, importMap),
	}

	// Constructors return the class type
	constructor.Results = []info.Parameter{
		{
			Type: &info.Type{
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

					constructor.Parameters = append(constructor.Parameters, info.Parameter{
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
		constructor.Body = &info.LocationNode{
			Text: bodyNode.Content(source),
			Location: info.Location{
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
				signature.WriteString(returnType.Name)
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
				signature.WriteString(param.Constraint)
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
						params = append(params, paramType.Name+" "+paramName)
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
			exceptions = append(exceptions, exceptionNode.Content(source))
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
	if node.NamedChild(0).Type() == "modifiers" {
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

// parseJavaType converts a Java type node to an info.Type
func parseJavaType(node *sitter.Node, source []byte, importMap map[string]string) *info.Type {
	// Create a basic type with location information
	typeInfo := &info.Type{
		Name: node.Content(source),
		Location: &info.Location{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
	}

	switch node.Type() {
	case "integral_type":
		if node.NamedChildCount() == 0 {
			return typeInfo
		}
		switch node.NamedChild(0).Type() {
		case "int":
			typeInfo.Name = "int32"
			typeInfo.Kind = reflect.Int32
		case "short":
			typeInfo.Name = "int16"
			typeInfo.Kind = reflect.Int16
		case "long":
			typeInfo.Name = "int64"
			typeInfo.Kind = reflect.Int64
		case "char":
			typeInfo.Name = "rune"
			typeInfo.Kind = reflect.Int32
		case "byte":
			typeInfo.Name = "byte"
			typeInfo.Kind = reflect.Uint8
		}

	case "floating_point_type":
		if node.NamedChildCount() == 0 {
			return typeInfo
		}
		switch node.NamedChild(0).Type() {
		case "float":
			typeInfo.Name = "float32"
			typeInfo.Kind = reflect.Float32
		case "double":
			typeInfo.Name = "float64"
			typeInfo.Kind = reflect.Float64
		}

	case "boolean_type":
		typeInfo.Name = "bool"
		typeInfo.Kind = reflect.Bool

	case "void_type":
		typeInfo.Name = "void"

	case "array_type":
		elemType := parseJavaType(node.NamedChild(0), source, importMap)
		if elemType != nil {
			typeInfo.Name = "[]" + elemType.Name
			typeInfo.Kind = reflect.Array
			typeInfo.ComponentType = elemType.Name

			// Transfer package information from element type
			typeInfo.PackagePath = elemType.PackagePath
		}

	case "type_identifier":
		typeName := node.Content(source)
		if typeName == "String" {
			typeInfo.Name = "string"
			typeInfo.Kind = reflect.String
			typeInfo.PackagePath = "java.lang"
		} else {
			typeInfo.Kind = reflect.Ptr

			// Look for import information to add full package path
			if packagePath, ok := importMap[typeName]; ok {
				typeInfo.PackagePath = packagePath
				// Create fully qualified name if we have package info
				typeInfo.Name = packagePath + "." + typeName
			}
		}
		typeInfo.PackagePath = importMap[typeName]

	case "scoped_type_identifier":
		typeInfo.Kind = reflect.Interface

		// Extract package from scoped identifier
		scopedName := node.Content(source)
		if lastDotIndex := strings.LastIndex(scopedName, "."); lastDotIndex != -1 {
			packagePath := scopedName[:lastDotIndex]
			typeInfo.PackagePath = packagePath
		}

	case "generic_type":
		if node.NamedChildCount() > 0 {
			baseTypeNode := node.NamedChild(0)
			baseTypeName := baseTypeNode.Content(source)
			typeInfo.Kind = reflect.Ptr

			// Try to find package info for the base type
			if packagePath, ok := importMap[baseTypeName]; ok {
				typeInfo.PackagePath = packagePath
				// Update name with full package path
				typeInfo.Name = packagePath + "." + baseTypeName
			} else {
				typeInfo.Name = baseTypeName
			}

			var typeParams []info.TypeParam

			// Process type parameters
			typeArgsNode := node.ChildByFieldName("type_arguments")
			if typeArgsNode != nil {
				var paramTypes []string
				for i := uint32(0); i < typeArgsNode.NamedChildCount(); i++ {
					paramNode := typeArgsNode.NamedChild(int(i))
					paramType := parseJavaType(paramNode, source, importMap)
					if paramType != nil {
						paramName := paramType.Name
						typeParams = append(typeParams, info.TypeParam{
							Name:       paramName,
							Constraint: "any",
						})
						paramTypes = append(paramTypes, paramName)
					}
				}

				// Add type parameters to name
				if len(paramTypes) > 0 {
					typeInfo.Name += "<" + strings.Join(paramTypes, ", ") + ">"
				}
			}

			typeInfo.TypeParams = typeParams
			typeInfo.PackagePath = importMap[baseTypeName]
		}
	}

	return typeInfo
}

// extractSimpleTypeName extracts the simple name from a possibly qualified name
// e.g., "java.util.List" -> "List"
func extractSimpleTypeName(qualifiedName string) string {
	lastDotIndex := strings.LastIndex(qualifiedName, ".")
	if lastDotIndex != -1 {
		return qualifiedName[lastDotIndex+1:]
	}
	return qualifiedName
}
