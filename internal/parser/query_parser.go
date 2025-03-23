package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)


// ParseSQLCQuerierInterface parses the Querier interface in a sqlc-generated directory
func ParseSQLCQuerierInterface(dir string) ([]QueryMethod, error) {
	// Look for the file containing the Querier interface
	querierFile := findQuerierFile(dir)
	if querierFile == "" {
		return nil, fmt.Errorf("could not find querier.go file in %s", dir)
	}

	// Parse the file containing the Querier interface
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, querierFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse querier file: %w", err)
	}

	// Find the Querier interface
	var querierInterface *ast.InterfaceType
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check if this is the Querier interface
			if typeSpec.Name.Name == "Querier" {
				if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					querierInterface = interfaceType
					break
				}
			}
		}

		if querierInterface != nil {
			break
		}
	}

	if querierInterface == nil {
		return nil, fmt.Errorf("querier interface not found in %s", querierFile)
	}

	// Extract methods from the Querier interface
	var methods []QueryMethod
	for _, method := range querierInterface.Methods.List {
		if len(method.Names) == 0 {
			continue // Skip embedded interfaces
		}

		methodName := method.Names[0].Name

		// Parse the method's function signature
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		// Extract parameter types (excluding context)
		var paramTypes []ParamType
		if funcType.Params != nil && len(funcType.Params.List) > 0 {
			for _, param := range funcType.Params.List {
				if len(param.Names) == 0 {
					continue
				}

				// Skip context parameter
				typeStr := typeToString(param.Type)
				if typeStr == "context.Context" {
					continue
				}

				for _, name := range param.Names {
					paramTypes = append(paramTypes, ParamType{
						Name: name.Name,
						Type: typeStr,
					})
				}
			}
		}

		// Extract return type and check if it's an array
		var returnType string
		var isArray bool
		var queryType QueryType

		// Default to exec type
		queryType = QueryTypeExec

		if funcType.Results != nil && len(funcType.Results.List) > 0 {
			// Find first result (excluding error)
			for _, result := range funcType.Results.List {
				resultType := typeToString(result.Type)
				if resultType != "error" {
					// Check if it's an array
					if strings.HasPrefix(resultType, "[]") {
						isArray = true
						returnType = strings.TrimPrefix(resultType, "[]")
						queryType = QueryTypeMany
					} else {
						returnType = resultType
						queryType = QueryTypeOne
					}
					break
				}
			}
		}

		// Infer query type if not already determined
		if queryType == QueryTypeExec && (strings.HasPrefix(methodName, "Get") || 
		   strings.HasPrefix(methodName, "Find") || strings.HasPrefix(methodName, "Lookup")) {
			queryType = QueryTypeOne
		} else if queryType == QueryTypeExec && (strings.HasPrefix(methodName, "List") || 
		           strings.HasPrefix(methodName, "Search") || strings.HasPrefix(methodName, "Query")) {
			queryType = QueryTypeMany
		}

		// Extract comments
		comment := ""
		if method.Doc != nil {
			for _, c := range method.Doc.List {
				comment += strings.TrimSpace(strings.TrimPrefix(c.Text, "//")) + " "
			}
			comment = strings.TrimSpace(comment)
		}

		// Create the query method
		queryMethod := QueryMethod{
			Name:       methodName,
			Type:       queryType,
			ParamTypes: paramTypes,
			ReturnType: returnType,
			IsArray:    isArray,
			Comment:    comment,
		}

		methods = append(methods, queryMethod)
	}

	return methods, nil
}

// findQuerierFile finds the file containing the Querier interface
func findQuerierFile(dir string) string {
	// Common filenames for the Querier interface
	possibleNames := []string{
		"querier.go",
		"db.go",
		"interface.go",
	}

	for _, name := range possibleNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			// Check if this file contains the Querier interface
			if containsQuerierInterface(path) {
				return path
			}
		}
	}

	// If not found, search all Go files in the directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			path := filepath.Join(dir, entry.Name())
			if containsQuerierInterface(path) {
				return path
			}
		}
	}

	return ""
}

// containsQuerierInterface checks if a file contains the Querier interface
func containsQuerierInterface(filePath string) bool {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return false
	}

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if typeSpec.Name.Name == "Querier" {
				_, ok := typeSpec.Type.(*ast.InterfaceType)
				return ok
			}
		}
	}

	return false
}

// typeToString converts an AST type expression to a string
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return "unknown"
	}
}

// GenerateServiceDefinitions creates service definitions from query methods
func GenerateServiceDefinitions(queryMethods []QueryMethod, messages []ProtoMessage) []ServiceDefinition {
	// Group methods by entity
	methodsByEntity := make(map[string][]QueryMethod)
	for _, method := range queryMethods {
		entity := inferEntityFromMethodName(method.Name)
		methodsByEntity[entity] = append(methodsByEntity[entity], method)
	}

	// Create a map of message names for quick lookup
	messageMap := make(map[string]ProtoMessage)
	for _, msg := range messages {
		messageMap[msg.Name] = msg
	}

	// Generate service definitions
	var services []ServiceDefinition
	for entity, methods := range methodsByEntity {
		service := ServiceDefinition{
			Name:        entity + "Service",
			Description: fmt.Sprintf("Service for %s operations", entity),
			Methods:     []ServiceMethod{},
		}

		for _, method := range methods {
			serviceMethod := ServiceMethod{
				Name:         method.Name,
				Description:  method.Comment,
				RequestType:  method.Name + "Request",
				ResponseType: method.Name + "Response",
				OriginalQuery: &method,
			}

			// Generate request fields based on parameter types
			if len(method.ParamTypes) > 0 {
				for i, param := range method.ParamTypes {
					// Check if the parameter type is a known message type
					if _, ok := messageMap[param.Type]; ok {
						// If it's a known message type, include it directly
						protoField := ProtoField{
							Name:     strcase.ToSnake(param.Type),
							Type:     param.Type,
							Number:   i + 1,
							Comment:  fmt.Sprintf("%s to process", param.Type),
						}
						serviceMethod.RequestFields = append(serviceMethod.RequestFields, protoField)
					} else {
						// For primitive types or unknown types, use the parameter name
						// Map Go type to Proto type
						protoType := mapGoTypeToProtoType(param.Type)
						
						protoField := ProtoField{
							Name:     strcase.ToSnake(param.Name),
							Type:     protoType,
							Number:   i + 1,
							Comment:  fmt.Sprintf("%s parameter", param.Name),
						}
						serviceMethod.RequestFields = append(serviceMethod.RequestFields, protoField)
					}
				}

				// Add pagination fields for list methods
				if strings.HasPrefix(method.Name, "List") {
					// Only add pagination if not already present
					hasLimit := false
					hasOffset := false
					for _, field := range serviceMethod.RequestFields {
						if field.Name == "limit" {
							hasLimit = true
						}
						if field.Name == "offset" {
							hasOffset = true
						}
					}

					if !hasLimit {
						serviceMethod.RequestFields = append(serviceMethod.RequestFields, ProtoField{
							Name:     "limit",
							Type:     "int32",
							Number:   len(serviceMethod.RequestFields) + 1,
							Comment:  "Maximum number of results to return",
						})
					}

					if !hasOffset {
						serviceMethod.RequestFields = append(serviceMethod.RequestFields, ProtoField{
							Name:     "page_token",
							Type:     "string",
							Number:   len(serviceMethod.RequestFields) + 1,
							Comment:  "Page token for pagination",
						})
					}
				}
			} else if strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "Delete") {
				// For Get and Delete methods without parameters, add an ID field
				serviceMethod.RequestFields = append(serviceMethod.RequestFields, ProtoField{
					Name:     strcase.ToSnake(entity) + "_id",
					Type:     "int32",
					Number:   1,
					Comment:  fmt.Sprintf("ID of the %s", entity),
				})
			}

			// Generate response fields based on return type
			if method.ReturnType != "" {
				if !method.IsArray {
					// For single result methods
					serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
						Name:     strcase.ToSnake(method.ReturnType),
						Type:     method.ReturnType,
						Number:   1,
						Comment:  fmt.Sprintf("The %s result", method.ReturnType),
					})
				} else {
					// For list/array result methods
					serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
						Name:       strcase.ToSnake(method.ReturnType) + "s",
						Type:       method.ReturnType,
						Number:     1,
						IsRepeated: true,
						Comment:    fmt.Sprintf("List of %s results", method.ReturnType),
					})

					// Add pagination metadata for list methods
					if strings.HasPrefix(method.Name, "List") {
						serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
							Name:     "next_page_token",
							Type:     "string",
							Number:   2,
							Comment:  "Token for retrieving the next page of results",
						})

						serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
							Name:     "total_size",
							Type:     "int32",
							Number:   3,
							Comment:  "Total number of results available",
						})
					}
				}
			} else if method.Type == QueryTypeExec {
				// For exec-type methods with no return value, add a success flag
				serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
					Name:     "success",
					Type:     "bool",
					Number:   1,
					Comment:  "Whether the operation was successful",
				})

				serviceMethod.ResponseFields = append(serviceMethod.ResponseFields, ProtoField{
					Name:     "affected_rows",
					Type:     "int32",
					Number:   2,
					Comment:  "Number of rows affected by the operation",
				})
			}

			service.Methods = append(service.Methods, serviceMethod)
		}

		services = append(services, service)
	}

	return services
}

// inferEntityFromMethodName extracts the entity name from a method name
func inferEntityFromMethodName(methodName string) string {
	// Common prefixes for CRUD operations
	prefixes := []string{
		"Get", "List", "Create", "Update", "Delete", 
		"Find", "Search", "Count", "Lookup", "Add",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(methodName, prefix) {
			// Remove the prefix
			entity := strings.TrimPrefix(methodName, prefix)
			
			// Handle special cases with suffixes
			suffixes := []string{"ByID", "ById", "WithDetails", "WithRelations"}
			for _, suffix := range suffixes {
				entity = strings.TrimSuffix(entity, suffix)
			}
			
			// Handle plural forms for list operations
			if prefix == "List" && strings.HasSuffix(entity, "s") {
				entity = strings.TrimSuffix(entity, "s")
			}
			
			// If we have a valid entity name, return it
			if entity != "" {
				return entity
			}
		}
	}

	// If no entity could be inferred, use a default
	return "Resource"
}

// mapGoTypeToProtoType converts Go types to Protocol Buffer types
func mapGoTypeToProtoType(goType string) string {
	mapping := map[string]string{
		"int":           "int32",
		"int32":         "int32",
		"int64":         "int64",
		"uint":          "uint32",
		"uint32":        "uint32",
		"uint64":        "uint64",
		"float32":       "float",
		"float64":       "double",
		"bool":          "bool",
		"string":        "string",
		"[]byte":        "bytes",
		"time.Time":     "google.protobuf.Timestamp",
	}

	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		baseType := strings.TrimPrefix(goType, "*")
		if protoType, ok := mapping[baseType]; ok {
			return protoType
		}
		return baseType // Pass through as is
	}

	// Handle array types
	if strings.HasPrefix(goType, "[]") {
		// For arrays, we'll handle the repeated tag separately
		baseType := strings.TrimPrefix(goType, "[]")
		if protoType, ok := mapping[baseType]; ok {
			return protoType
		}
		return baseType // Pass through as is
	}

	// Direct mapping
	if protoType, ok := mapping[goType]; ok {
		return protoType
	}

	// If no mapping found, pass through as is
	return goType
}

// GenerateServiceProto generates a proto file containing service definitions
func GenerateServiceProto(services []ServiceDefinition, config interface{}, outputPath string) error {
	// Implementation depends on your template engine
	// This function would render a proto template with the service definitions
	return nil
}
