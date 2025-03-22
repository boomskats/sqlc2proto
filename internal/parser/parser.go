package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

// TypeMapping maps Go types to Protobuf types
var TypeMapping = map[string]string{
	"string":             "string",
	"int":                "int32",
	"int32":              "int32",
	"int64":              "int64",
	"float32":            "float",
	"float64":            "double",
	"bool":               "bool",
	"[]byte":             "bytes",
	"time.Time":          "google.protobuf.Timestamp",
	"pgtype.Date":        "google.protobuf.Timestamp",
	"pgtype.Timestamptz": "google.protobuf.Timestamp",
	"pgtype.Text":        "string",
}

// NullableTypeMapping maps sqlc nullable types to Protobuf types
var NullableTypeMapping = map[string]string{
	"sql.NullString":  "string",
	"sql.NullInt32":   "int32",
	"sql.NullInt64":   "int64",
	"sql.NullFloat64": "double",
	"sql.NullBool":    "bool",
	"sql.NullTime":    "google.protobuf.Timestamp",
}

// ProtoMessage represents a Protobuf message
type ProtoMessage struct {
	Name         string
	Fields       []ProtoField
	Comments     string
	SQLCStruct   string
	ProtoPackage string
}

// ProtoField represents a field in a Protobuf message
type ProtoField struct {
	Name                  string
	Type                  string
	Number                int
	IsRepeated            bool
	IsOptional            bool
	Comment               string
	JSONName              string
	OriginalTag           string
	SQLCName              string
	ConversionCode        string
	ReverseConversionCode string
}

// AddCustomTypeMappings adds custom type mappings
func AddCustomTypeMappings(mappings map[string]string) {
	maps.Copy(TypeMapping, mappings)
}

// ProcessSQLCDirectory processes all Go files in the sqlc output directory
func ProcessSQLCDirectory(dir string) ([]ProtoMessage, error) {
	var messages []ProtoMessage

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			// Skip query files, only process model files
			if strings.Contains(path, "query") || strings.Contains(path, "querier") {
				return nil
			}

			fileMessages, err := processSQLCFile(path)
			if err != nil {
				return fmt.Errorf("error processing file %s: %v", path, err)
			}
			messages = append(messages, fileMessages...)
		}
		return nil
	})

	return messages, err
}

// processSQLCFile extracts message definitions from a sqlc-generated Go file
func processSQLCFile(filePath string) ([]ProtoMessage, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var messages []ProtoMessage

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

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Process struct
			message := ProtoMessage{
				Name:         typeSpec.Name.Name,
				SQLCStruct:   typeSpec.Name.Name,
				Comments:     extractComments(genDecl.Doc),
				ProtoPackage: "", // Will be set by the generator
			}

			// Process struct fields
			for i, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // Skip embedded fields
				}

				fieldName := field.Names[0].Name
				if !ast.IsExported(fieldName) {
					continue // Skip unexported fields
				}

				protoField := ProtoField{
					Name:     camelToSnake(fieldName),
					Number:   i + 1,
					Comment:  extractComments(field.Doc),
					SQLCName: fieldName,
				}

				// Extract type information
				typeStr := exprToTypeString(field.Type)
				isNullable := false
				isRepeated := false

				// Check if it's a slice (repeated)
				if strings.HasPrefix(typeStr, "[]") {
					isRepeated = true
					typeStr = strings.TrimPrefix(typeStr, "[]")
				}

				// Check if it's a nullable type
				if protoType, ok := NullableTypeMapping[typeStr]; ok {
					protoField.Type = protoType
					protoField.IsOptional = true
					isNullable = true

					// Set conversion code
					protoField.ConversionCode = generateNullableConversionCode(typeStr, protoField)
					protoField.ReverseConversionCode = generateReverseNullableConversionCode(typeStr, protoField)
				} else if protoType, ok := TypeMapping[typeStr]; ok {
					protoField.Type = protoType

					// Set conversion code for standard types
					protoField.ConversionCode = generateStandardConversionCode(typeStr, protoField)
					protoField.ReverseConversionCode = generateReverseStandardConversionCode(typeStr, protoField)
				} else {
					// Default to string for unknown types
					protoField.Type = "string"
					protoField.ConversionCode = fmt.Sprintf("in.%s", fieldName)
					protoField.ReverseConversionCode = fmt.Sprintf("in.%s", camelToSnake(fieldName))
				}

				// Set repeated flag
				protoField.IsRepeated = isRepeated

				// Extract JSON name and other tags
				if field.Tag != nil {
					tagValue := strings.Trim(field.Tag.Value, "`")
					protoField.OriginalTag = tagValue

					// Extract json tag
					if jsonTag := extractTag(tagValue, "json"); jsonTag != "" {
						jsonName := strings.Split(jsonTag, ",")[0]
						if jsonName != "-" {
							protoField.JSONName = jsonName
						}
					}

					// If the field is nullable but not marked as optional yet
					if !isNullable && strings.Contains(tagValue, "omitempty") {
						protoField.IsOptional = true
					}
				}

				message.Fields = append(message.Fields, protoField)
			}

			messages = append(messages, message)
		}
	}

	return messages, nil
}

// extractComments extracts comments from a comment group
func extractComments(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}
	var comments []string
	for _, comment := range commentGroup.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		comments = append(comments, text)
	}
	return strings.Join(comments, " ")
}

// exprToTypeString converts an ast expression to a type string
func exprToTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", exprToTypeString(t.X), t.Sel.Name)
	case *ast.StarExpr:
		return exprToTypeString(t.X) // Treat pointers as the base type
	case *ast.ArrayType:
		return "[]" + exprToTypeString(t.Elt)
	default:
		return "string" // Default for complex types
	}
}

// extractTag extracts a specific tag from a struct tag string
func extractTag(tagStr string, key string) string {
	for tag := range strings.SplitSeq(tagStr, " ") {
		if strings.HasPrefix(tag, key+":") {
			value := strings.TrimPrefix(tag, key+":")
			return strings.Trim(value, "\"")
		}
	}
	return ""
}

// camelToSnake converts a camelCase string to snake_case
func camelToSnake(s string) string {
	// Special case for ID at the end of the string
	s = strings.Replace(s, "ID", "Id", -1)

	// Use strcase package for consistent conversion
	return strcase.ToSnake(s)
}

// generateNullableConversionCode generates code to convert nullable SQL types to protobuf
func generateNullableConversionCode(sqlType string, field ProtoField) string {
	switch sqlType {
	case "sql.NullString":
		return fmt.Sprintf("func() string { if in.%s.Valid { return in.%s.String } return \"\" }()", field.SQLCName, field.SQLCName)
	case "sql.NullInt32":
		return fmt.Sprintf("func() int32 { if in.%s.Valid { return in.%s.Int32 } return 0 }()", field.SQLCName, field.SQLCName)
	case "sql.NullInt64":
		return fmt.Sprintf("func() int64 { if in.%s.Valid { return in.%s.Int64 } return 0 }()", field.SQLCName, field.SQLCName)
	case "sql.NullFloat64":
		return fmt.Sprintf("func() float64 { if in.%s.Valid { return in.%s.Float64 } return 0 }()", field.SQLCName, field.SQLCName)
	case "sql.NullBool":
		return fmt.Sprintf("func() bool { if in.%s.Valid { return in.%s.Bool } return false }()", field.SQLCName, field.SQLCName)
	case "sql.NullTime":
		return fmt.Sprintf("func() *timestamppb.Timestamp { if in.%s.Valid { return timestamppb.New(in.%s.Time) } return nil }()", field.SQLCName, field.SQLCName)
	default:
		return fmt.Sprintf("in.%s", field.SQLCName)
	}
}

// generateReverseNullableConversionCode generates code to convert from protobuf to nullable SQL types
func generateReverseNullableConversionCode(sqlType string, field ProtoField) string {
	// Convert snake_case to PascalCase for protobuf field names
	pascalName := strcase.ToCamel(field.Name)

	switch sqlType {
	case "sql.NullString":
		return fmt.Sprintf("sql.NullString{String: in.%s, Valid: in.%s != \"\"}", pascalName, pascalName)
	case "sql.NullInt32":
		return fmt.Sprintf("sql.NullInt32{Int32: in.%s, Valid: in.%s != 0}", pascalName, pascalName)
	case "sql.NullInt64":
		return fmt.Sprintf("sql.NullInt64{Int64: in.%s, Valid: in.%s != 0}", pascalName, pascalName)
	case "sql.NullFloat64":
		return fmt.Sprintf("sql.NullFloat64{Float64: in.%s, Valid: in.%s != 0}", pascalName, pascalName)
	case "sql.NullBool":
		return fmt.Sprintf("sql.NullBool{Bool: in.%s, Valid: true}", pascalName)
	case "sql.NullTime":
		return fmt.Sprintf("func() sql.NullTime { if in.%s != nil { return sql.NullTime{Time: in.%s.AsTime(), Valid: true} } return sql.NullTime{} }()", pascalName, pascalName)
	default:
		return fmt.Sprintf("in.%s", pascalName)
	}
}

// generateStandardConversionCode generates code to convert standard types
func generateStandardConversionCode(sqlType string, field ProtoField) string {
	switch sqlType {
	case "time.Time":
		return fmt.Sprintf("timestamppb.New(in.%s)", field.SQLCName)
	case "pgtype.Date":
		return fmt.Sprintf("func() *timestamppb.Timestamp { t := in.%s.Time; return timestamppb.New(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)) }()", field.SQLCName)
	case "pgtype.Timestamptz":
		return fmt.Sprintf("func() *timestamppb.Timestamp { if in.%s.Valid { return timestamppb.New(in.%s.Time) } return nil }()", field.SQLCName, field.SQLCName)
	case "pgtype.Text":
		return fmt.Sprintf("func() string { if in.%s.Valid { return in.%s.String } return \"\" }()", field.SQLCName, field.SQLCName)
	default:
		return fmt.Sprintf("in.%s", field.SQLCName)
	}
}

// generateReverseStandardConversionCode generates code to convert from protobuf to standard types
func generateReverseStandardConversionCode(sqlType string, field ProtoField) string {
	// Convert snake_case to PascalCase for protobuf field names
	pascalName := strcase.ToCamel(field.Name)

	switch sqlType {
	case "time.Time":
		return fmt.Sprintf("in.%s.AsTime()", pascalName)
	case "pgtype.Date":
		return fmt.Sprintf("pgtype.Date{Time: in.%s.AsTime(), Valid: in.%s != nil}", pascalName, pascalName)
	case "pgtype.Timestamptz":
		return fmt.Sprintf("pgtype.Timestamptz{Time: in.%s.AsTime(), Valid: in.%s != nil}", pascalName, pascalName)
	case "pgtype.Text":
		return fmt.Sprintf("pgtype.Text{String: in.%s, Valid: in.%s != \"\"}", pascalName, pascalName)
	default:
		return fmt.Sprintf("in.%s", pascalName)
	}
}
