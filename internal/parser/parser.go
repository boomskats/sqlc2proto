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
	"int16":              "int32", // Added for smallint/int2
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
	"pgtype.Numeric":     "string", // Added for numeric/decimal
	"uuid.UUID":          "string", // Added for UUID
	"json.RawMessage":    "string", // Added for JSON
	"pgtype.Interval":    "int64",  // Added for interval
}

// NullableTypeMapping maps sqlc nullable types to Protobuf types
var NullableTypeMapping = map[string]string{
	"sql.NullString":  "string",
	"sql.NullInt16":   "int32", // Added for nullable smallint
	"sql.NullInt32":   "int32",
	"sql.NullInt64":   "int64",
	"sql.NullFloat64": "double",
	"sql.NullBool":    "bool",
	"sql.NullTime":    "google.protobuf.Timestamp",
	"uuid.NullUUID":   "string", // Added for nullable UUID
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

// AddCustomNullableTypeMappings adds custom nullable type mappings
func AddCustomNullableTypeMappings(mappings map[string]string) {
	maps.Copy(NullableTypeMapping, mappings)
}

// ProcessSQLCDirectory processes all Go files in the sqlc output directory
func ProcessSQLCDirectory(dir string, fieldStyle string) ([]ProtoMessage, error) {
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

			fileMessages, err := processSQLCFile(path, fieldStyle)
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
func processSQLCFile(filePath string, fieldStyle string) ([]ProtoMessage, error) {
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

				// Determine field name based on style
				var protoFieldName string

				// Extract JSON tag if present
				jsonTagName := ""
				if field.Tag != nil {
					tagValue := strings.Trim(field.Tag.Value, "`")
					if jsonTag := extractTag(tagValue, "json"); jsonTag != "" {
						jsonName := strings.Split(jsonTag, ",")[0]
						if jsonName != "-" {
							jsonTagName = jsonName
						}
					}
				}

				// Apply field naming style
				switch fieldStyle {
				case "json":
					// Use JSON tag if available, otherwise fall back to snake_case
					if jsonTagName != "" {
						protoFieldName = jsonTagName
					} else {
						protoFieldName = camelToSnake(fieldName)
					}
				case "snake_case":
					// Always convert to snake_case regardless of JSON tag
					protoFieldName = camelToSnake(fieldName)
				case "original":
					// Keep original Go field name
					protoFieldName = fieldName
				default:
					// Default to snake_case if style is not recognized
					protoFieldName = camelToSnake(fieldName)
				}

				protoField := ProtoField{
					Name:     protoFieldName,
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
					typeStr = strings.TrimPrefix(typeStr, "[]")

					// Special case for []byte which maps to bytes in protobuf (not repeated)
					if typeStr == "byte" {
						typeStr = "[]byte" // Restore the full type for TypeMapping lookup
					} else if typeStr == "[]byte" {
						// Special case for [][]byte which also maps to bytes in protobuf (not repeated)
						// This is for ByteaArray in PostgreSQL
						typeStr = "[]byte" // The inner []byte maps to bytes
						isRepeated = true
					} else {
						isRepeated = true
					}
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
	// Special cases for UUID and ULID which should not be treated as having ID suffix
	if s == "UUID" {
		return "uuid"
	}
	if s == "ULID" {
		return "ulid"
	}

	// Special case for ID at the end of the string
	s = strings.Replace(s, "ID", "Id", -1)

	// Use strcase package for consistent conversion
	return strcase.ToSnake(s)
}

// generateNullableConversionCode generates code to convert nullable SQL types to protobuf
func generateNullableConversionCode(sqlType string, field ProtoField) string {
	switch sqlType {
	case "sql.NullString":
		return fmt.Sprintf("nullStringToString(in.%s)", field.SQLCName)
	case "sql.NullInt16":
		return fmt.Sprintf("nullInt16ToInt32(in.%s)", field.SQLCName)
	case "sql.NullInt32":
		return fmt.Sprintf("nullInt32ToInt32(in.%s)", field.SQLCName)
	case "sql.NullInt64":
		return fmt.Sprintf("nullInt64ToInt64(in.%s)", field.SQLCName)
	case "sql.NullFloat64":
		return fmt.Sprintf("nullFloat64ToFloat64(in.%s)", field.SQLCName)
	case "sql.NullBool":
		return fmt.Sprintf("nullBoolToBool(in.%s)", field.SQLCName)
	case "sql.NullTime":
		return fmt.Sprintf("nullTimeToTimestamp(in.%s)", field.SQLCName)
	case "uuid.NullUUID":
		return fmt.Sprintf("nullUUIDToString(in.%s)", field.SQLCName)
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
		return fmt.Sprintf("stringToNullString(in.%s)", pascalName)
	case "sql.NullInt16":
		return fmt.Sprintf("int32ToNullInt16(in.%s)", pascalName)
	case "sql.NullInt32":
		return fmt.Sprintf("int32ToNullInt32(in.%s)", pascalName)
	case "sql.NullInt64":
		return fmt.Sprintf("int64ToNullInt64(in.%s)", pascalName)
	case "sql.NullFloat64":
		return fmt.Sprintf("float64ToNullFloat64(in.%s)", pascalName)
	case "sql.NullBool":
		return fmt.Sprintf("boolToNullBool(in.%s)", pascalName)
	case "sql.NullTime":
		return fmt.Sprintf("timestampToNullTime(in.%s)", pascalName)
	case "uuid.NullUUID":
		return fmt.Sprintf("stringToNullUUID(in.%s)", pascalName)
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
		return fmt.Sprintf("dateToTimestamp(in.%s)", field.SQLCName)
	case "pgtype.Timestamptz":
		return fmt.Sprintf("timestamptzToTimestamp(in.%s)", field.SQLCName)
	case "pgtype.Text":
		return fmt.Sprintf("pgtypeTextToString(in.%s)", field.SQLCName)
	case "pgtype.Numeric":
		return fmt.Sprintf("numericToString(in.%s)", field.SQLCName)
	case "uuid.UUID":
		return fmt.Sprintf("uuidToString(in.%s)", field.SQLCName)
	case "json.RawMessage":
		return fmt.Sprintf("jsonToString(in.%s)", field.SQLCName)
	case "pgtype.Interval":
		return fmt.Sprintf("intervalToInt64(in.%s)", field.SQLCName)
	case "int16":
		return fmt.Sprintf("int32(in.%s)", field.SQLCName)
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
		return fmt.Sprintf("timestampToDate(in.%s)", pascalName)
	case "pgtype.Timestamptz":
		return fmt.Sprintf("timestampToTimestamptz(in.%s)", pascalName)
	case "pgtype.Text":
		return fmt.Sprintf("stringToPgtypeText(in.%s)", pascalName)
	case "pgtype.Numeric":
		return fmt.Sprintf("stringToNumeric(in.%s)", pascalName)
	case "uuid.UUID":
		return fmt.Sprintf("stringToUUID(in.%s)", pascalName)
	case "json.RawMessage":
		return fmt.Sprintf("stringToJSON(in.%s)", pascalName)
	case "pgtype.Interval":
		return fmt.Sprintf("int64ToInterval(in.%s)", pascalName)
	case "int16":
		return fmt.Sprintf("int16(in.%s)", pascalName)
	default:
		return fmt.Sprintf("in.%s", pascalName)
	}
}

// GenerateHelperFunctions generates helper functions for type conversions
func GenerateHelperFunctions(messages []ProtoMessage) string {
	// Track which helper functions we need to generate
	needNullString := false
	needNullInt16 := false
	needNullInt32 := false
	needNullInt64 := false
	needNullFloat64 := false
	needNullBool := false
	needNullTime := false
	needPgtypeDate := false
	needPgtypeTimestamptz := false
	needPgtypeText := false
	needPgtypeNumeric := false
	needUUID := false
	needNullUUID := false
	needJSON := false
	needInterval := false

	// Check which types are used in the messages
	for _, msg := range messages {
		for _, field := range msg.Fields {
			switch {
			case strings.Contains(field.ConversionCode, "nullStringToString"):
				needNullString = true
			case strings.Contains(field.ConversionCode, "nullInt16ToInt32"):
				needNullInt16 = true
			case strings.Contains(field.ConversionCode, "nullInt32ToInt32"):
				needNullInt32 = true
			case strings.Contains(field.ConversionCode, "nullInt64ToInt64"):
				needNullInt64 = true
			case strings.Contains(field.ConversionCode, "nullFloat64ToFloat64"):
				needNullFloat64 = true
			case strings.Contains(field.ConversionCode, "nullBoolToBool"):
				needNullBool = true
			case strings.Contains(field.ConversionCode, "nullTimeToTimestamp"):
				needNullTime = true
			case strings.Contains(field.ConversionCode, "dateToTimestamp"):
				needPgtypeDate = true
			case strings.Contains(field.ConversionCode, "timestamptzToTimestamp"):
				needPgtypeTimestamptz = true
			case strings.Contains(field.ConversionCode, "pgtypeTextToString"):
				needPgtypeText = true
			case strings.Contains(field.ConversionCode, "numericToString"):
				needPgtypeNumeric = true
			case strings.Contains(field.ConversionCode, "uuidToString"):
				needUUID = true
			case strings.Contains(field.ConversionCode, "nullUUIDToString"):
				needNullUUID = true
			case strings.Contains(field.ConversionCode, "jsonToString"):
				needJSON = true
			case strings.Contains(field.ConversionCode, "intervalToInt64"):
				needInterval = true
			}

			switch {
			case strings.Contains(field.ReverseConversionCode, "stringToNullString"):
				needNullString = true
			case strings.Contains(field.ReverseConversionCode, "int32ToNullInt16"):
				needNullInt16 = true
			case strings.Contains(field.ReverseConversionCode, "int32ToNullInt32"):
				needNullInt32 = true
			case strings.Contains(field.ReverseConversionCode, "int64ToNullInt64"):
				needNullInt64 = true
			case strings.Contains(field.ReverseConversionCode, "float64ToNullFloat64"):
				needNullFloat64 = true
			case strings.Contains(field.ReverseConversionCode, "boolToNullBool"):
				needNullBool = true
			case strings.Contains(field.ReverseConversionCode, "timestampToNullTime"):
				needNullTime = true
			case strings.Contains(field.ReverseConversionCode, "timestampToDate"):
				needPgtypeDate = true
			case strings.Contains(field.ReverseConversionCode, "timestampToTimestamptz"):
				needPgtypeTimestamptz = true
			case strings.Contains(field.ReverseConversionCode, "stringToPgtypeText"):
				needPgtypeText = true
			case strings.Contains(field.ReverseConversionCode, "stringToNumeric"):
				needPgtypeNumeric = true
			case strings.Contains(field.ReverseConversionCode, "stringToUUID"):
				needUUID = true
			case strings.Contains(field.ReverseConversionCode, "stringToNullUUID"):
				needNullUUID = true
			case strings.Contains(field.ReverseConversionCode, "stringToJSON"):
				needJSON = true
			case strings.Contains(field.ReverseConversionCode, "int64ToInterval"):
				needInterval = true
			}
		}
	}

	var helpers []string

	// Add helper functions based on what's needed
	if needNullString {
		helpers = append(helpers, `
// Helper function to convert sql.NullString to string
func nullStringToString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// Helper function to convert string to sql.NullString
func stringToNullString(v string) sql.NullString {
	return sql.NullString{
		String: v,
		Valid:  v != "",
	}
}`)
	}

	if needNullInt32 {
		helpers = append(helpers, `
// Helper function to convert sql.NullInt32 to int32
func nullInt32ToInt32(v sql.NullInt32) int32 {
	if v.Valid {
		return v.Int32
	}
	return 0
}

// Helper function to convert int32 to sql.NullInt32
func int32ToNullInt32(v int32) sql.NullInt32 {
	return sql.NullInt32{
		Int32: v,
		Valid: v != 0,
	}
}`)
	}

	if needNullInt64 {
		helpers = append(helpers, `
// Helper function to convert sql.NullInt64 to int64
func nullInt64ToInt64(v sql.NullInt64) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

// Helper function to convert int64 to sql.NullInt64
func int64ToNullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: v,
		Valid: v != 0,
	}
}`)
	}

	if needNullFloat64 {
		helpers = append(helpers, `
// Helper function to convert sql.NullFloat64 to float64
func nullFloat64ToFloat64(v sql.NullFloat64) float64 {
	if v.Valid {
		return v.Float64
	}
	return 0
}

// Helper function to convert float64 to sql.NullFloat64
func float64ToNullFloat64(v float64) sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: v,
		Valid:   v != 0,
	}
}`)
	}

	if needNullBool {
		helpers = append(helpers, `
// Helper function to convert sql.NullBool to bool
func nullBoolToBool(v sql.NullBool) bool {
	if v.Valid {
		return v.Bool
	}
	return false
}

// Helper function to convert bool to sql.NullBool
func boolToNullBool(v bool) sql.NullBool {
	return sql.NullBool{
		Bool:  v,
		Valid: true,
	}
}`)
	}

	if needNullTime {
		helpers = append(helpers, `
// Helper function to convert sql.NullTime to *timestamppb.Timestamp
func nullTimeToTimestamp(v sql.NullTime) *timestamppb.Timestamp {
	if v.Valid {
		return timestamppb.New(v.Time)
	}
	return nil
}

// Helper function to convert *timestamppb.Timestamp to sql.NullTime
func timestampToNullTime(v *timestamppb.Timestamp) sql.NullTime {
	if v != nil {
		return sql.NullTime{
			Time:  v.AsTime(),
			Valid: true,
		}
	}
	return sql.NullTime{}
}`)
	}

	if needPgtypeDate {
		helpers = append(helpers, `
// Helper function to convert pgtype.Date to *timestamppb.Timestamp
func dateToTimestamp(v pgtype.Date) *timestamppb.Timestamp {
	t := v.Time
	return timestamppb.New(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
}

// Helper function to convert *timestamppb.Timestamp to pgtype.Date
func timestampToDate(v *timestamppb.Timestamp) pgtype.Date {
	return pgtype.Date{
		Time:  v.AsTime(),
		Valid: v != nil,
	}
}`)
	}

	if needPgtypeTimestamptz {
		helpers = append(helpers, `
// Helper function to convert pgtype.Timestamptz to *timestamppb.Timestamp
func timestamptzToTimestamp(v pgtype.Timestamptz) *timestamppb.Timestamp {
	if v.Valid {
		return timestamppb.New(v.Time)
	}
	return nil
}

// Helper function to convert *timestamppb.Timestamp to pgtype.Timestamptz
func timestampToTimestamptz(v *timestamppb.Timestamp) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  v.AsTime(),
		Valid: v != nil,
	}
}`)
	}

	if needPgtypeText {
		helpers = append(helpers, `
// Helper function to convert pgtype.Text to string
func pgtypeTextToString(v pgtype.Text) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// Helper function to convert string to pgtype.Text
func stringToPgtypeText(v string) pgtype.Text {
	return pgtype.Text{
		String: v,
		Valid:  v != "",
	}
}`)
	}

	if needNullInt16 {
		helpers = append(helpers, `
// Helper function to convert sql.NullInt16 to int32
func nullInt16ToInt32(v sql.NullInt16) int32 {
	if v.Valid {
		return int32(v.Int16)
	}
	return 0
}

// Helper function to convert int32 to sql.NullInt16
func int32ToNullInt16(v int32) sql.NullInt16 {
	return sql.NullInt16{
		Int16: int16(v),
		Valid: v != 0,
	}
}`)
	}

	if needPgtypeNumeric {
		helpers = append(helpers, `
// Helper function to convert pgtype.Numeric to string
func numericToString(v pgtype.Numeric) string {
	if v.Valid {
		return v.String()
	}
	return ""
}

// Helper function to convert string to pgtype.Numeric
func stringToNumeric(v string) pgtype.Numeric {
	var n pgtype.Numeric
	n.Set(v)
	return n
}`)
	}

	if needUUID {
		helpers = append(helpers, `
// Helper function to convert uuid.UUID to string
func uuidToString(v uuid.UUID) string {
	return v.String()
}

// Helper function to convert string to uuid.UUID
func stringToUUID(v string) uuid.UUID {
	u, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return u
}`)
	}

	if needNullUUID {
		helpers = append(helpers, `
// Helper function to convert uuid.NullUUID to string
func nullUUIDToString(v uuid.NullUUID) string {
	if v.Valid {
		return v.UUID.String()
	}
	return ""
}

// Helper function to convert string to uuid.NullUUID
func stringToNullUUID(v string) uuid.NullUUID {
	if v == "" {
		return uuid.NullUUID{}
	}
	u, err := uuid.Parse(v)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{
		UUID:  u,
		Valid: true,
	}
}`)
	}

	if needJSON {
		helpers = append(helpers, `
// Helper function to convert json.RawMessage to string
func jsonToString(v json.RawMessage) string {
	return string(v)
}

// Helper function to convert string to json.RawMessage
func stringToJSON(v string) json.RawMessage {
	return json.RawMessage(v)
}`)
	}

	if needInterval {
		helpers = append(helpers, `
// Helper function to convert pgtype.Interval to int64
func intervalToInt64(v pgtype.Interval) int64 {
	return v.Microseconds
}

// Helper function to convert int64 to pgtype.Interval
func int64ToInterval(v int64) pgtype.Interval {
	return pgtype.Interval{
		Microseconds: v,
		Valid:        true,
	}
}`)
	}

	return strings.Join(helpers, "\n")
}
