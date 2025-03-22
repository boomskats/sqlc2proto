package parser

import (
	"testing"
)

func TestAddCustomNullableTypeMappings(t *testing.T) {
	// Save original mappings to restore after test
	originalNullableTypeMappings := make(map[string]string)
	for k, v := range NullableTypeMapping {
		originalNullableTypeMappings[k] = v
	}
	defer func() {
		// Restore original mappings
		NullableTypeMapping = originalNullableTypeMappings
	}()

	// Test adding new mappings
	customNullableMappings := map[string]string{
		"CustomNullableType":                        "string",
		"github.com/shopspring/decimal.NullDecimal": "string",
		"sql.NullString":                            "google.protobuf.StringValue", // Override existing mapping
	}

	AddCustomNullableTypeMappings(customNullableMappings)

	// Verify new mappings were added
	if NullableTypeMapping["CustomNullableType"] != "string" {
		t.Errorf("Expected CustomNullableType to be mapped to string, got %s", NullableTypeMapping["CustomNullableType"])
	}
	if NullableTypeMapping["github.com/shopspring/decimal.NullDecimal"] != "string" {
		t.Errorf("Expected decimal.NullDecimal to be mapped to string, got %s", NullableTypeMapping["github.com/shopspring/decimal.NullDecimal"])
	}

	// Verify existing mapping was overridden
	if NullableTypeMapping["sql.NullString"] != "google.protobuf.StringValue" {
		t.Errorf("Expected sql.NullString to be overridden to google.protobuf.StringValue, got %s", NullableTypeMapping["sql.NullString"])
	}
}
