package parser

import (
	"maps"
	"path/filepath"
	"testing"
)

func TestCustomTypeMappings(t *testing.T) {
	// Save original mappings
	originalStandardMappings := maps.Clone(TypeMapping)
	originalNullableMappings := maps.Clone(NullableTypeMapping)

	// Create a temporary test file with custom types
	tempFile := filepath.Join(t.TempDir(), "custom_types.go")

	// Write test content to file with custom types
	content := `
package db

import (
	"github.com/example/custom"
	"github.com/example/another"
)

type CustomTypes struct {
	CustomType     custom.Type     ` + "`json:\"custom_type\"`" + `
	AnotherType    another.Type    ` + "`json:\"another_type\"`" + `
	CustomNullable custom.Nullable ` + "`json:\"custom_nullable,omitempty\"`" + `
}
`

	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
		return
	}

	// Add custom type mappings
	customMappings := map[string]string{
		"custom.Type":  "custom.ProtoType",
		"another.Type": "another.ProtoType",
	}

	customNullableMappings := map[string]string{
		"custom.Nullable": "string",
	}

	// Add custom mappings to the global variables
	AddCustomTypeMappings(customMappings)
	AddCustomNullableTypeMappings(customNullableMappings)

	// Get a config that includes the custom mappings we just added
	config := ParserConfig{
		FieldStyle: "json",
		TypeConfig: GetTypeMapConfig(), // This should now include our custom mappings
	}

	// Process the file with the updated config
	messages, err := processSQLCFile(tempFile, config)
	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// We should have 1 message: CustomTypes
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Find CustomTypes message
	var customMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "CustomTypes" {
			customMsg = &messages[i]
			break
		}
	}

	if customMsg == nil {
		t.Fatalf("CustomTypes message not found")
	}

	// Check that fields are correctly mapped to custom types
	expectedFields := map[string]struct {
		Type       string
		IsOptional bool
	}{
		"custom_type":     {"custom.ProtoType", false},
		"another_type":    {"another.ProtoType", false},
		"custom_nullable": {"string", true},
	}

	if len(customMsg.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(customMsg.Fields))
	}

	for _, field := range customMsg.Fields {
		expected, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		if field.IsOptional != expected.IsOptional {
			t.Errorf("Field %s: expected IsOptional=%v, got %v", field.Name, expected.IsOptional, field.IsOptional)
		}
	}

	// Restore original mappings for other tests
	TypeMapping = originalStandardMappings
	NullableTypeMapping = originalNullableMappings
}

func TestTypeMapping(t *testing.T) {
	// Test that all standard types are correctly mapped

	// Create a temporary test file with all standard types
	tempFile := filepath.Join(t.TempDir(), "standard_types.go")

	// Write test content to file with all standard types
	content := `
package db

import (
	"time"
	"github.com/jackc/pgtype"
)

type AllStandardTypes struct {
	StringField    string             ` + "`json:\"string_field\"`" + `
	IntField       int                ` + "`json:\"int_field\"`" + `
	Int32Field     int32              ` + "`json:\"int32_field\"`" + `
	Int64Field     int64              ` + "`json:\"int64_field\"`" + `
	Float32Field   float32            ` + "`json:\"float32_field\"`" + `
	Float64Field   float64            ` + "`json:\"float64_field\"`" + `
	BoolField      bool               ` + "`json:\"bool_field\"`" + `
	BytesField     []byte             ` + "`json:\"bytes_field\"`" + `
	TimeField      time.Time          ` + "`json:\"time_field\"`" + `
	DateField      pgtype.Date        ` + "`json:\"date_field\"`" + `
	TimestampField pgtype.Timestamptz ` + "`json:\"timestamp_field\"`" + `
	TextField      pgtype.Text        ` + "`json:\"text_field\"`" + `
}
`

	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
		return
	}

	config := ParserConfig{
		FieldStyle: "json",
		TypeConfig: DefaultTypeMappingConfig(),
	}

	// Process the file
	messages, err := processSQLCFile(tempFile, config)
	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// We should have 1 message: AllStandardTypes
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Find AllStandardTypes message
	var standardMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "AllStandardTypes" {
			standardMsg = &messages[i]
			break
		}
	}

	if standardMsg == nil {
		t.Fatalf("AllStandardTypes message not found")
	}

	// Check that all fields are correctly mapped
	expectedFields := map[string]string{
		"string_field":    "string",
		"int_field":       "int32",
		"int32_field":     "int32",
		"int64_field":     "int64",
		"float32_field":   "float",
		"float64_field":   "double",
		"bool_field":      "bool",
		"bytes_field":     "bytes",
		"time_field":      "google.protobuf.Timestamp",
		"date_field":      "google.protobuf.Timestamp",
		"timestamp_field": "google.protobuf.Timestamp",
		"text_field":      "string",
	}

	if len(standardMsg.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(standardMsg.Fields))
	}

	for _, field := range standardMsg.Fields {
		expectedType, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expectedType, field.Type)
		}

		// Check that conversion code is generated
		if field.ConversionCode == "" {
			t.Errorf("No conversion code generated for %s field", field.Name)
		}

		if field.ReverseConversionCode == "" {
			t.Errorf("No reverse conversion code generated for %s field", field.Name)
		}
	}
}
