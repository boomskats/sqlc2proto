package parser

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestFieldNameStyles(t *testing.T) {
	tests := []struct {
		name          string
		style         string
		goField       string
		jsonTag       string
		expectedProto string
	}{
		{"JSON Tag", "json", "UserID", "user_id", "user_id"},
		{"JSON Fallback", "json", "UserID", "", "user_id"},                  // Fallback to snake_case
		{"Snake Case", "snake_case", "UserID", "custom_name", "user_id"},    // JSON tag ignored
		{"Original", "original", "UserID", "user_id", "UserID"},             // Original preserved
		{"Default", "", "UserID", "user_id", "user_id"},                     // Default to json if style not specified
		{"JSON With Dash", "json", "UserEmail", "user-email", "user-email"}, // Preserve dashes in JSON tags
		{"Snake With Acronym", "snake_case", "APIKey", "", "api_key"},       // Handle acronyms

		// ID suffix handling
		{"ID Suffix JSON", "json", "CustomerID", "customer_id", "customer_id"},
		{"ID Suffix Snake", "snake_case", "CustomerID", "", "customer_id"},
		{"ID Suffix Original", "original", "CustomerID", "", "CustomerID"},
		{"ID Suffix Lowercase", "snake_case", "CustomerId", "", "customer_id"},

		// Longer words
		{"Long Field JSON", "json", "UserAuthenticationCredentials", "user_auth_credentials", "user_auth_credentials"},
		{"Long Field Snake", "snake_case", "UserAuthenticationCredentials", "", "user_authentication_credentials"},
		{"Long Field Original", "original", "UserAuthenticationCredentials", "", "UserAuthenticationCredentials"},

		// Mixed case handling
		{"Mixed Case JSON", "json", "orderItemQuantity", "order_item_quantity", "order_item_quantity"},
		{"Mixed Case Snake", "snake_case", "orderItemQuantity", "", "order_item_quantity"},
		{"Mixed Case Original", "original", "orderItemQuantity", "", "orderItemQuantity"},

		// Underscore in original field
		{"Underscore Field JSON", "json", "User_Name", "username", "username"},
		{"Underscore Field Snake", "snake_case", "User_Name", "", "user_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock field with the test data
			field := &ast.Field{
				Names: []*ast.Ident{
					{Name: tt.goField},
				},
				Type: &ast.Ident{Name: "string"},
			}

			// Add JSON tag if specified
			if tt.jsonTag != "" {
				field.Tag = &ast.BasicLit{
					Kind:  token.STRING,
					Value: "`json:\"" + tt.jsonTag + "\"`",
				}
			}

			// We don't need to create a ProtoField for this test

			// Extract JSON tag if present
			jsonTagName := ""
			if field.Tag != nil {
				tagValue := field.Tag.Value[1 : len(field.Tag.Value)-1] // Remove backticks
				if jsonTag := extractTag(tagValue, "json"); jsonTag != "" {
					jsonName := strings.Split(jsonTag, ",")[0]
					if jsonName != "-" {
						jsonTagName = jsonName
					}
				}
			}

			// Apply field naming style
			var protoFieldName string
			switch tt.style {
			case "json":
				// Use JSON tag if available, otherwise fall back to snake_case
				if jsonTagName != "" {
					protoFieldName = jsonTagName
				} else {
					protoFieldName = camelToSnake(tt.goField)
				}
			case "snake_case":
				// Always convert to snake_case regardless of JSON tag
				protoFieldName = camelToSnake(tt.goField)
			case "original":
				// Keep original Go field name
				protoFieldName = tt.goField
			default:
				// Default to json style if not specified
				if jsonTagName != "" {
					protoFieldName = jsonTagName
				} else {
					protoFieldName = camelToSnake(tt.goField)
				}
			}

			// Check if the field name matches the expected value
			if protoFieldName != tt.expectedProto {
				t.Errorf("Field name = %v, want %v", protoFieldName, tt.expectedProto)
			}
		})
	}
}
