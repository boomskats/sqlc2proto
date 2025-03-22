package parser

import (
	"path/filepath"
	"testing"
)

func TestNullableTypes(t *testing.T) {
	// Test processing the basic_types.go file which contains nullable types
	filePath := filepath.Join("testdata", "basic_types.go")
	messages, err := processSQLCFile(filePath, "json")

	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// Find User message which has a nullable field (DeletedAt)
	var userMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "User" {
			userMsg = &messages[i]
			break
		}
	}

	if userMsg == nil {
		t.Fatalf("User message not found")
	}

	// Check that the DeletedAt nullable field is correctly processed
	var deletedAtField *ProtoField
	for i := range userMsg.Fields {
		if userMsg.Fields[i].Name == "deleted_at" {
			deletedAtField = &userMsg.Fields[i]
			break
		}
	}

	if deletedAtField == nil {
		t.Fatalf("deleted_at field not found in User message")
	}

	// Verify that the nullable field is marked as optional
	if !deletedAtField.IsOptional {
		t.Errorf("Nullable field deleted_at should be marked as optional")
	}

	// Verify the type is correctly mapped
	expectedType := "google.protobuf.Timestamp"
	if deletedAtField.Type != expectedType {
		t.Errorf("Expected deleted_at type to be %s, got %s", expectedType, deletedAtField.Type)
	}

	// Verify the conversion code is generated
	if deletedAtField.ConversionCode == "" {
		t.Errorf("No conversion code generated for deleted_at field")
	}

	// Find Product message which has a nullable field (SKU)
	var productMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "Product" {
			productMsg = &messages[i]
			break
		}
	}

	if productMsg == nil {
		t.Fatalf("Product message not found")
	}

	// Check that the SKU nullable field is correctly processed
	var skuField *ProtoField
	for i := range productMsg.Fields {
		if productMsg.Fields[i].Name == "sku" {
			skuField = &productMsg.Fields[i]
			break
		}
	}

	if skuField == nil {
		t.Fatalf("sku field not found in Product message")
	}

	// Verify that the nullable field is marked as optional
	if !skuField.IsOptional {
		t.Errorf("Nullable field sku should be marked as optional")
	}

	// Verify the type is correctly mapped
	expectedType = "string"
	if skuField.Type != expectedType {
		t.Errorf("Expected sku type to be %s, got %s", expectedType, skuField.Type)
	}
}

func TestNullableTypeMapping(t *testing.T) {
	// Test that all nullable types are correctly mapped

	// Create a temporary test file with all nullable types
	tempFile := filepath.Join(t.TempDir(), "nullable_types.go")

	// Write test content to file with all nullable types
	content := `
package db

import (
	"database/sql"
	"time"
)

type AllNullableTypes struct {
	NullString  sql.NullString  ` + "`json:\"null_string\"`" + `
	NullInt32   sql.NullInt32   ` + "`json:\"null_int32\"`" + `
	NullInt64   sql.NullInt64   ` + "`json:\"null_int64\"`" + `
	NullFloat64 sql.NullFloat64 ` + "`json:\"null_float64\"`" + `
	NullBool    sql.NullBool    ` + "`json:\"null_bool\"`" + `
	NullTime    sql.NullTime    ` + "`json:\"null_time\"`" + `
}
`

	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
		return
	}

	// Process the file
	messages, err := processSQLCFile(tempFile, "json")
	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// We should have 1 message: AllNullableTypes
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Find AllNullableTypes message
	var nullableMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "AllNullableTypes" {
			nullableMsg = &messages[i]
			break
		}
	}

	if nullableMsg == nil {
		t.Fatalf("AllNullableTypes message not found")
	}

	// Check that all fields are correctly mapped
	expectedFields := map[string]struct {
		Type       string
		IsOptional bool
	}{
		"null_string":  {"string", true},
		"null_int32":   {"int32", true},
		"null_int64":   {"int64", true},
		"null_float64": {"double", true},
		"null_bool":    {"bool", true},
		"null_time":    {"google.protobuf.Timestamp", true},
	}

	if len(nullableMsg.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(nullableMsg.Fields))
	}

	for _, field := range nullableMsg.Fields {
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

		// Check that conversion code is generated
		if field.ConversionCode == "" {
			t.Errorf("No conversion code generated for %s field", field.Name)
		}

		if field.ReverseConversionCode == "" {
			t.Errorf("No reverse conversion code generated for %s field", field.Name)
		}
	}
}

// Helper functions are defined in testutil_test.go
