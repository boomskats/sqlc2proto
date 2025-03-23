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

// ========================================
// Type Definitions and Structures
// ========================================

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

// ParserConfig holds configuration for the parser
type ParserConfig struct {
	FieldStyle string
	TypeConfig TypeMappingConfig
}

// ========================================
// Default Type Mappings
// ========================================

// DefaultTypeMappingConfig returns the default type mapping configuration
func DefaultTypeMappingConfig() TypeMappingConfig {
	return TypeMappingConfig{
		StandardTypes: map[string]string{
			"string":             "string",
			"int":                "int32",
			"int16":              "int32",
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
			"pgtype.Numeric":     "string",
			"uuid.UUID":          "string",
			"json.RawMessage":    "string",
			"pgtype.Interval":    "int64",
		},
		NullableTypes: map[string]string{
			"sql.NullString":  "string",
			"sql.NullInt16":   "int32",
			"sql.NullInt32":   "int32",
			"sql.NullInt64":   "int64",
			"sql.NullFloat64": "double",
			"sql.NullBool":    "bool",
			"sql.NullTime":    "google.protobuf.Timestamp",
			"uuid.NullUUID":   "string",
		},
		CustomConverters: map[string]ConversionFuncs{
			"time.Time": {
				ToProto:   "timestamppb.New(%s)",
				FromProto: "%s.AsTime()",
			},
			"pgtype.Date": {
				ToProto:   "dateToTimestamp(%s)",
				FromProto: "timestampToDate(%s)",
			},
			"pgtype.Timestamptz": {
				ToProto:   "timestamptzToTimestamp(%s)",
				FromProto: "timestampToTimestamptz(%s)",
			},
			"pgtype.Text": {
				ToProto:   "pgtypeTextToString(%s)",
				FromProto: "stringToPgtypeText(%s)",
			},
			"pgtype.Numeric": {
				ToProto:   "numericToString(%s)",
				FromProto: "stringToNumeric(%s)",
			},
			"uuid.UUID": {
				ToProto:   "uuidToString(%s)",
				FromProto: "stringToUUID(%s)",
			},
			"json.RawMessage": {
				ToProto:   "jsonToString(%s)",
				FromProto: "stringToJSON(%s)",
			},
			"pgtype.Interval": {
				ToProto:   "intervalToInt64(%s)",
				FromProto: "int64ToInterval(%s)",
			},
			"int16": {
				ToProto:   "int32(%s)",
				FromProto: "int16(%s)",
			},
			"sql.NullString": {
				ToProto:   "nullStringToString(%s)",
				FromProto: "stringToNullString(%s)",
			},
			"sql.NullInt16": {
				ToProto:   "nullInt16ToInt32(%s)",
				FromProto: "int32ToNullInt16(%s)",
			},
			"sql.NullInt32": {
				ToProto:   "nullInt32ToInt32(%s)",
				FromProto: "int32ToNullInt32(%s)",
			},
			"sql.NullInt64": {
				ToProto:   "nullInt64ToInt64(%s)",
				FromProto: "int64ToNullInt64(%s)",
			},
			"sql.NullFloat64": {
				ToProto:   "nullFloat64ToFloat64(%s)",
				FromProto: "float64ToNullFloat64(%s)",
			},
			"sql.NullBool": {
				ToProto:   "nullBoolToBool(%s)",
				FromProto: "boolToNullBool(%s)",
			},
			"sql.NullTime": {
				ToProto:   "nullTimeToTimestamp(%s)",
				FromProto: "timestampToNullTime(%s)",
			},
			"uuid.NullUUID": {
				ToProto:   "nullUUIDToString(%s)",
				FromProto: "stringToNullUUID(%s)",
			},
		},
	}
}

// ========================================
// Public API Methods
// ========================================

// ProcessSQLCDirectory processes all Go files in the sqlc output directory
func ProcessSQLCDirectory(dir string, fieldStyle string) ([]ProtoMessage, error) {
	config := ParserConfig{
		FieldStyle: fieldStyle,
		TypeConfig: DefaultTypeMappingConfig(),
	}

	var messages []ProtoMessage

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			// Process all Go files except for querier.go and db.go
			// querier.go contains the interface and db.go contains the DB connection code
			filename := filepath.Base(path)
			if filename == "querier.go" || filename == "db.go" {
				return nil
			}

			fileMessages, err := processSQLCFile(path, config)
			if err != nil {
				return fmt.Errorf("error processing file %s: %v", path, err)
			}
			messages = append(messages, fileMessages...)
		}
		return nil
	})

	return messages, err
}

// GenerateHelperFunctions generates helper functions for type conversions
func GenerateHelperFunctions(messages []ProtoMessage) string {
	// This method analyzes which helper functions are needed based on the conversion code
	// in the messages and generates the corresponding function implementations

	// Track which helper functions we need to generate using a set
	neededHelpers := make(map[string]bool)

	// Analyze all messages to determine which helpers are needed
	for _, msg := range messages {
		for _, field := range msg.Fields {
			// Extract function names from conversion code
			extractHelperNames(field.ConversionCode, neededHelpers)
			extractHelperNames(field.ReverseConversionCode, neededHelpers)
		}
	}

	// Generate the helper functions that are needed
	return generateHelperFunctionsCode(neededHelpers)
}

// ========================================
// Internal Implementation Methods
// ========================================

// processSQLCFile extracts message definitions from a sqlc-generated Go file
func processSQLCFile(filePath string, config ParserConfig) ([]ProtoMessage, error) {
	// Parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Find and process struct type declarations
	var messages []ProtoMessage
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// Process each struct type
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Create a message for this struct
			message := ProtoMessage{
				Name:       typeSpec.Name.Name,
				SQLCStruct: typeSpec.Name.Name,
				Comments:   extractComments(genDecl.Doc),
			}

			// Process struct fields
			message.Fields = processStructFields(structType, message.Name, config)

			messages = append(messages, message)
		}
	}

	return messages, nil
}

// processStructFields extracts and processes the fields of a struct
func processStructFields(structType *ast.StructType, structName string, config ParserConfig) []ProtoField {
	var fields []ProtoField

	for i, field := range structType.Fields.List {
		// Skip embedded fields
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue // Skip unexported fields
		}

		// Extract field information
		protoField, ok := extractProtoField(field, fieldName, i+1, config)
		if !ok {
			// Skip fields that couldn't be processed
			continue
		}

		fields = append(fields, protoField)
	}

	return fields
}

// extractProtoField creates a ProtoField from an AST field
func extractProtoField(field *ast.Field, fieldName string, fieldNumber int, config ParserConfig) (ProtoField, bool) {
	// Start with default values
	protoField := ProtoField{
		Number:   fieldNumber,
		Comment:  extractComments(field.Doc),
		SQLCName: fieldName,
	}

	// Determine the proto field name based on style
	protoField.Name = getProtoFieldName(field, fieldName, config.FieldStyle)

	// Extract JSON name and tags
	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		protoField.OriginalTag = tagValue

		// Extract JSON name
		if jsonTag := extractTag(tagValue, "json"); jsonTag != "" {
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName != "-" {
				protoField.JSONName = jsonName
			}

			// If json tag has omitempty, mark as optional
			if strings.Contains(jsonTag, "omitempty") {
				protoField.IsOptional = true
			}
		}
	}

	// Process the field type
	if !processFieldType(field, &protoField, config.TypeConfig) {
		return ProtoField{}, false
	}

	return protoField, true
}

// getProtoFieldName determines the Proto field name based on the field style
func getProtoFieldName(field *ast.Field, fieldName string, fieldStyle string) string {
	// Extract JSON tag name if present
	jsonTagName := ""
	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		if jsonTag := extractTag(tagValue, "json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				jsonTagName = parts[0]
			}
		}
	}

	switch fieldStyle {
	case "json":
		// Use JSON tag if available, otherwise convert to snake_case
		if jsonTagName != "" {
			return jsonTagName
		}
		return camelToSnake(fieldName)
	case "snake_case":
		// Always use snake_case regardless of JSON tag
		return camelToSnake(fieldName)
	case "original":
		// Keep the original Go field name
		return fieldName
	default:
		// Default to snake_case
		return camelToSnake(fieldName)
	}
}

// processFieldType processes a field's type information
func processFieldType(field *ast.Field, protoField *ProtoField, typeConfig TypeMappingConfig) bool {
	// Extract type string from the AST
	typeStr := exprToTypeString(field.Type)

	// Handle array/slice types
	if strings.HasPrefix(typeStr, "[]") {
		return processArrayType(typeStr, protoField, typeConfig)
	}

	// Handle standard types
	return processStandardType(typeStr, protoField, typeConfig)
}

// processArrayType handles array/slice type fields
func processArrayType(typeStr string, protoField *ProtoField, typeConfig TypeMappingConfig) bool {
	// Remove the slice prefix
	elementType := strings.TrimPrefix(typeStr, "[]")

	// Special case for []byte which maps to bytes
	if elementType == "byte" {
		// Reset to full type for lookup
		typeStr = "[]byte"
		return processStandardType(typeStr, protoField, typeConfig)
	}

	// For normal slices, process the element type and mark as repeated
	if !processStandardType(elementType, protoField, typeConfig) {
		return false
	}

	protoField.IsRepeated = true
	return true
}

// processStandardType handles non-array field types
func processStandardType(typeStr string, protoField *ProtoField, typeConfig TypeMappingConfig) bool {
	// Check for nullable types first
	if protoType, ok := typeConfig.NullableTypes[typeStr]; ok {
		protoField.Type = protoType
		protoField.IsOptional = true

		// Set conversion code
		if converter, ok := typeConfig.CustomConverters[typeStr]; ok {
			protoField.ConversionCode = fmt.Sprintf(converter.ToProto, "in."+protoField.SQLCName)
			protoField.ReverseConversionCode = fmt.Sprintf(converter.FromProto, "in."+pascalCase(protoField.Name))
		} else {
			// Default conversion for nullable types
			protoField.ConversionCode = fmt.Sprintf("in.%s", protoField.SQLCName)
			protoField.ReverseConversionCode = fmt.Sprintf("in.%s", pascalCase(protoField.Name))
		}

		return true
	}

	// Then check standard types
	if protoType, ok := typeConfig.StandardTypes[typeStr]; ok {
		protoField.Type = protoType

		// Set conversion code
		if converter, ok := typeConfig.CustomConverters[typeStr]; ok {
			protoField.ConversionCode = fmt.Sprintf(converter.ToProto, "in."+protoField.SQLCName)
			protoField.ReverseConversionCode = fmt.Sprintf(converter.FromProto, "in."+pascalCase(protoField.Name))
		} else {
			// Default conversion for standard types
			protoField.ConversionCode = fmt.Sprintf("in.%s", protoField.SQLCName)
			protoField.ReverseConversionCode = fmt.Sprintf("in.%s", pascalCase(protoField.Name))
		}

		return true
	}

	// Default to string for unknown types
	protoField.Type = "string"
	protoField.ConversionCode = fmt.Sprintf("in.%s", protoField.SQLCName)
	protoField.ReverseConversionCode = fmt.Sprintf("in.%s", pascalCase(protoField.Name))

	return true
}

// ========================================
// Helper Functions
// ========================================

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

// exprToTypeString converts an AST expression to a type string
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
	for _, tag := range strings.Split(tagStr, " ") {
		if strings.HasPrefix(tag, key+":") {
			value := strings.TrimPrefix(tag, key+":")
			return strings.Trim(value, "\"")
		}
	}
	return ""
}

// camelToSnake converts a camelCase string to snake_case
func camelToSnake(s string) string {
	// Special cases for UUID and ULID
	if s == "UUID" {
		return "uuid"
	}
	if s == "ULID" {
		return "ulid"
	}

	// Special case for ID suffix
	s = strings.Replace(s, "ID", "Id", -1)

	// Use strcase for consistent conversion
	return strcase.ToSnake(s)
}

// pascalCase converts a string to PascalCase
func pascalCase(s string) string {
	return strcase.ToCamel(s)
}

// extractHelperNames analyzes conversion code to identify helper function names
func extractHelperNames(code string, helpers map[string]bool) {
	// Simple regex-like approach to find function calls
	// A more robust approach would use proper regex or AST parsing

	// Common prefixes that indicate helper functions
	helperPrefixes := []string{
		"nullStringToString", "stringToNullString",
		"nullInt16ToInt32", "int32ToNullInt16",
		"nullInt32ToInt32", "int32ToNullInt32",
		"nullInt64ToInt64", "int64ToNullInt64",
		"nullFloat64ToFloat64", "float64ToNullFloat64",
		"nullBoolToBool", "boolToNullBool",
		"nullTimeToTimestamp", "timestampToNullTime",
		"dateToTimestamp", "timestampToDate",
		"timestamptzToTimestamp", "timestampToTimestamptz",
		"pgtypeTextToString", "stringToPgtypeText",
		"numericToString", "stringToNumeric",
		"uuidToString", "stringToUUID",
		"nullUUIDToString", "stringToNullUUID",
		"jsonToString", "stringToJSON",
		"intervalToInt64", "int64ToInterval",
	}

	for _, prefix := range helperPrefixes {
		if strings.Contains(code, prefix) {
			helpers[prefix] = true
		}
	}
}

// generateHelperFunctionsCode generates the code for helper functions
func generateHelperFunctionsCode(neededHelpers map[string]bool) string {
	// Map from helper name to implementation
	helperImplementations := map[string]string{
		// String helpers
		"nullStringToString": `
// Helper function to convert sql.NullString to string
func nullStringToString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}`,
		"stringToNullString": `
// Helper function to convert string to sql.NullString
func stringToNullString(v string) sql.NullString {
	return sql.NullString{
		String: v,
		Valid:  v != "",
	}
}`,
		// Int32 helpers
		"nullInt32ToInt32": `
// Helper function to convert sql.NullInt32 to int32
func nullInt32ToInt32(v sql.NullInt32) int32 {
	if v.Valid {
		return v.Int32
	}
	return 0
}`,
		"int32ToNullInt32": `
// Helper function to convert int32 to sql.NullInt32
func int32ToNullInt32(v int32) sql.NullInt32 {
	return sql.NullInt32{
		Int32: v,
		Valid: v != 0,
	}
}`,
		// Int16 helpers
		"nullInt16ToInt32": `
// Helper function to convert sql.NullInt16 to int32
func nullInt16ToInt32(v sql.NullInt16) int32 {
	if v.Valid {
		return int32(v.Int16)
	}
	return 0
}`,
		"int32ToNullInt16": `
// Helper function to convert int32 to sql.NullInt16
func int32ToNullInt16(v int32) sql.NullInt16 {
	return sql.NullInt16{
		Int16: int16(v),
		Valid: v != 0,
	}
}`,
		// Int64 helpers
		"nullInt64ToInt64": `
// Helper function to convert sql.NullInt64 to int64
func nullInt64ToInt64(v sql.NullInt64) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}`,
		"int64ToNullInt64": `
// Helper function to convert int64 to sql.NullInt64
func int64ToNullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: v,
		Valid: v != 0,
	}
}`,
		// Float64 helpers
		"nullFloat64ToFloat64": `
// Helper function to convert sql.NullFloat64 to float64
func nullFloat64ToFloat64(v sql.NullFloat64) float64 {
	if v.Valid {
		return v.Float64
	}
	return 0
}`,
		"float64ToNullFloat64": `
// Helper function to convert float64 to sql.NullFloat64
func float64ToNullFloat64(v float64) sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: v,
		Valid:   v != 0,
	}
}`,
		// Bool helpers
		"nullBoolToBool": `
// Helper function to convert sql.NullBool to bool
func nullBoolToBool(v sql.NullBool) bool {
	if v.Valid {
		return v.Bool
	}
	return false
}`,
		"boolToNullBool": `
// Helper function to convert bool to sql.NullBool
func boolToNullBool(v bool) sql.NullBool {
	return sql.NullBool{
		Bool:  v,
		Valid: true,
	}
}`,
		// Time helpers
		"nullTimeToTimestamp": `
// Helper function to convert sql.NullTime to *timestamppb.Timestamp
func nullTimeToTimestamp(v sql.NullTime) *timestamppb.Timestamp {
	if v.Valid {
		return timestamppb.New(v.Time)
	}
	return nil
}`,
		"timestampToNullTime": `
// Helper function to convert *timestamppb.Timestamp to sql.NullTime
func timestampToNullTime(v *timestamppb.Timestamp) sql.NullTime {
	if v != nil {
		return sql.NullTime{
			Time:  v.AsTime(),
			Valid: true,
		}
	}
	return sql.NullTime{}
}`,
		// PostgreSQL date helpers
		"dateToTimestamp": `
// Helper function to convert pgtype.Date to *timestamppb.Timestamp
func dateToTimestamp(v pgtype.Date) *timestamppb.Timestamp {
	t := v.Time
	return timestamppb.New(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC))
}`,
		"timestampToDate": `
// Helper function to convert *timestamppb.Timestamp to pgtype.Date
func timestampToDate(v *timestamppb.Timestamp) pgtype.Date {
	return pgtype.Date{
		Time:  v.AsTime(),
		Valid: v != nil,
	}
}`,
		// PostgreSQL timestamptz helpers
		"timestamptzToTimestamp": `
// Helper function to convert pgtype.Timestamptz to *timestamppb.Timestamp
func timestamptzToTimestamp(v pgtype.Timestamptz) *timestamppb.Timestamp {
	if v.Valid {
		return timestamppb.New(v.Time)
	}
	return nil
}`,
		"timestampToTimestamptz": `
// Helper function to convert *timestamppb.Timestamp to pgtype.Timestamptz
func timestampToTimestamptz(v *timestamppb.Timestamp) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  v.AsTime(),
		Valid: v != nil,
	}
}`,
		// PostgreSQL text helpers
		"pgtypeTextToString": `
// Helper function to convert pgtype.Text to string
func pgtypeTextToString(v pgtype.Text) string {
	if v.Valid {
		return v.String
	}
	return ""
}`,
		"stringToPgtypeText": `
// Helper function to convert string to pgtype.Text
func stringToPgtypeText(v string) pgtype.Text {
	return pgtype.Text{
		String: v,
		Valid:  v != "",
	}
}`,
		// PostgreSQL numeric helpers
		"numericToString": `
// Helper function to convert pgtype.Numeric to string
func numericToString(v pgtype.Numeric) string {
	if v.Valid {
		return v.String()
	}
	return ""
}`,
		"stringToNumeric": `
// Helper function to convert string to pgtype.Numeric
func stringToNumeric(v string) pgtype.Numeric {
	var n pgtype.Numeric
	n.Set(v)
	return n
}`,
		// UUID helpers
		"uuidToString": `
// Helper function to convert uuid.UUID to string
func uuidToString(v uuid.UUID) string {
	return v.String()
}`,
		"stringToUUID": `
// Helper function to convert string to uuid.UUID
func stringToUUID(v string) uuid.UUID {
	u, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return u
}`,
		// Nullable UUID helpers
		"nullUUIDToString": `
// Helper function to convert uuid.NullUUID to string
func nullUUIDToString(v uuid.NullUUID) string {
	if v.Valid {
		return v.UUID.String()
	}
	return ""
}`,
		"stringToNullUUID": `
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
}`,
		// JSON helpers
		"jsonToString": `
// Helper function to convert json.RawMessage to string
func jsonToString(v json.RawMessage) string {
	return string(v)
}`,
		"stringToJSON": `
// Helper function to convert string to json.RawMessage
func stringToJSON(v string) json.RawMessage {
	return json.RawMessage(v)
}`,
		// Interval helpers
		"intervalToInt64": `
// Helper function to convert pgtype.Interval to int64
func intervalToInt64(v pgtype.Interval) int64 {
	return v.Microseconds
}`,
		"int64ToInterval": `
// Helper function to convert int64 to pgtype.Interval
func int64ToInterval(v int64) pgtype.Interval {
	return pgtype.Interval{
		Microseconds: v,
		Valid:        true,
	}
}`,
		// CommandTag helpers
		"commandTagToString": `
// Helper function to convert pgconn.CommandTag to string
func commandTagToString(v pgconn.CommandTag) string {
	return v.String()
}`,
		"stringToCommandTag": `
// Helper function to convert string to pgconn.CommandTag
func stringToCommandTag(v string) pgconn.CommandTag {
	return pgconn.CommandTag(v)
}`,
	}

	// Build the output string with only the needed implementations
	var implementations []string
	for helperName, needed := range neededHelpers {
		if needed {
			if impl, ok := helperImplementations[helperName]; ok {
				implementations = append(implementations, impl)
			}
		}
	}

	return strings.Join(implementations, "\n")
}
