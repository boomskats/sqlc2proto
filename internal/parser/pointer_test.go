package parser

import (
	"path/filepath"
	"testing"
)

func TestPointerTypes(t *testing.T) {
	// Test processing the complex_types.go file which contains pointer types
	filePath := filepath.Join("testdata", "complex_types.go")

	config := ParserConfig{
		FieldStyle: "json",
		TypeConfig: DefaultTypeMappingConfig(),
	}

	messages, err := processSQLCFile(filePath, config)

	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// Find Order message which has a pointer field (ShippingInfo)
	var orderMsg *ProtoMessage
	for i := range messages {
		if messages[i].Name == "Order" {
			orderMsg = &messages[i]
			break
		}
	}

	if orderMsg == nil {
		t.Fatalf("Order message not found")
	}

	// Check that the ShippingInfo pointer field is correctly processed
	var shippingInfoField *ProtoField
	for i := range orderMsg.Fields {
		if orderMsg.Fields[i].Name == "shipping_info" {
			shippingInfoField = &orderMsg.Fields[i]
			break
		}
	}

	if shippingInfoField == nil {
		t.Fatalf("shipping_info field not found in Order message")
	}

	// Verify that the pointer field is marked as optional
	if !shippingInfoField.IsOptional {
		t.Errorf("Pointer field shipping_info should be marked as optional")
	}

	// Verify the type is correctly identified
	if shippingInfoField.Type != "string" {
		t.Errorf("Expected shipping_info type to be string, got %s", shippingInfoField.Type)
	}
}

// Helper functions are defined in testutil_test.go
