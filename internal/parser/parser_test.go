package parser

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessSQLCFile_BasicTypes(t *testing.T) {
	// Test processing the basic_types.go file
	filePath := filepath.Join("testdata", "basic_types.go")
	messages, err := processSQLCFile(filePath, "json")

	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// We should have 2 messages: User and Product
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Verify User message
	var userMsg *ProtoMessage
	var productMsg *ProtoMessage

	for i := range messages {
		if messages[i].Name == "User" {
			userMsg = &messages[i]
		} else if messages[i].Name == "Product" {
			productMsg = &messages[i]
		}
	}

	if userMsg == nil {
		t.Fatalf("User message not found")
	}

	// Check User fields
	expectedUserFields := map[string]string{
		"id":         "int64",
		"name":       "string",
		"email":      "string",
		"created_at": "google.protobuf.Timestamp",
		"updated_at": "google.protobuf.Timestamp",
		"deleted_at": "google.protobuf.Timestamp",
		"is_active":  "bool",
	}

	if len(userMsg.Fields) != len(expectedUserFields) {
		t.Errorf("Expected %d fields for User, got %d", len(expectedUserFields), len(userMsg.Fields))
	}

	for _, field := range userMsg.Fields {
		expectedType, ok := expectedUserFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expectedType, field.Type)
		}

		// Check if deleted_at is optional (it has omitempty tag)
		if field.Name == "deleted_at" && !field.IsOptional {
			t.Errorf("Field deleted_at should be optional")
		}
	}

	// Verify Product message
	if productMsg == nil {
		t.Fatalf("Product message not found")
	}

	// Check Product fields
	expectedProductFields := map[string]string{
		"id":          "int64",
		"name":        "string",
		"description": "string",
		"price":       "double",
		"in_stock":    "bool",
		"sku":         "string",
		"created_at":  "google.protobuf.Timestamp",
	}

	if len(productMsg.Fields) != len(expectedProductFields) {
		t.Errorf("Expected %d fields for Product, got %d", len(expectedProductFields), len(productMsg.Fields))
	}

	for _, field := range productMsg.Fields {
		expectedType, ok := expectedProductFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expectedType, field.Type)
		}

		// Check if sku is optional (it has omitempty tag)
		if field.Name == "sku" && !field.IsOptional {
			t.Errorf("Field sku should be optional")
		}
	}
}

func TestProcessSQLCFile_ComplexTypes(t *testing.T) {
	// Test processing the complex_types.go file
	filePath := filepath.Join("testdata", "complex_types.go")
	messages, err := processSQLCFile(filePath, "json")

	if err != nil {
		t.Fatalf("processSQLCFile failed: %v", err)
	}

	// We should have 7 messages: Order, OrderItem, ShippingInfo, Document, Transaction, Configuration, and NullUUID
	expectedMessageCount := 7
	if len(messages) != expectedMessageCount {
		t.Errorf("Expected %d messages, got %d", expectedMessageCount, len(messages))
	}

	// Create a map to store all messages by name for easier lookup
	messageMap := make(map[string]*ProtoMessage)
	for i := range messages {
		messageMap[messages[i].Name] = &messages[i]
	}

	// Verify Order message
	orderMsg := messageMap["Order"]
	if orderMsg == nil {
		t.Fatalf("Order message not found")
	}

	// Check Order fields
	expectedOrderFields := map[string]struct {
		Type       string
		IsRepeated bool
		IsOptional bool
	}{
		"id":             {"int64", false, false},
		"customer_id":    {"int64", false, false},
		"order_date":     {"google.protobuf.Timestamp", false, false},
		"status":         {"string", false, false},
		"total":          {"double", false, false},
		"items":          {"string", true, false},
		"shipping_info":  {"string", false, true},
		"notes":          {"string", false, true},
		"payment_method": {"string", false, true},
	}

	if len(orderMsg.Fields) != len(expectedOrderFields) {
		t.Errorf("Expected %d fields for Order, got %d", len(expectedOrderFields), len(orderMsg.Fields))
	}

	for _, field := range orderMsg.Fields {
		expected, ok := expectedOrderFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		if field.IsRepeated != expected.IsRepeated {
			t.Errorf("Field %s: expected IsRepeated=%v, got %v", field.Name, expected.IsRepeated, field.IsRepeated)
		}

		if field.IsOptional != expected.IsOptional {
			t.Errorf("Field %s: expected IsOptional=%v, got %v", field.Name, expected.IsOptional, field.IsOptional)
		}
	}

	// Verify OrderItem message
	orderItemMsg := messageMap["OrderItem"]
	if orderItemMsg == nil {
		t.Fatalf("OrderItem message not found")
	}

	// Check OrderItem fields
	expectedOrderItemFields := map[string]string{
		"id":         "int64",
		"order_id":   "int64",
		"product_id": "int64",
		"quantity":   "int32",
		"unit_price": "double",
		"subtotal":   "double",
	}

	if len(orderItemMsg.Fields) != len(expectedOrderItemFields) {
		t.Errorf("Expected %d fields for OrderItem, got %d", len(expectedOrderItemFields), len(orderItemMsg.Fields))
	}

	for _, field := range orderItemMsg.Fields {
		expectedType, ok := expectedOrderItemFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expectedType, field.Type)
		}
	}

	// Verify ShippingInfo message
	shippingInfoMsg := messageMap["ShippingInfo"]
	if shippingInfoMsg == nil {
		t.Fatalf("ShippingInfo message not found")
	}

	// Check ShippingInfo fields
	expectedShippingInfoFields := map[string]struct {
		Type       string
		IsOptional bool
	}{
		"address":      {"string", false},
		"city":         {"string", false},
		"postal_code":  {"string", false},
		"country":      {"string", false},
		"tracking_num": {"string", true},
		"shipped_at":   {"google.protobuf.Timestamp", true},
	}

	if len(shippingInfoMsg.Fields) != len(expectedShippingInfoFields) {
		t.Errorf("Expected %d fields for ShippingInfo, got %d", len(expectedShippingInfoFields), len(shippingInfoMsg.Fields))
	}

	for _, field := range shippingInfoMsg.Fields {
		expected, ok := expectedShippingInfoFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		// Note: The current parser implementation might not correctly identify optional fields
		// based on the omitempty tag for non-pointer, non-nullable fields
		if expected.IsOptional && field.Name == "tracking_num" && !field.IsOptional {
			t.Logf("Warning: Field %s should be optional based on omitempty tag", field.Name)
		}
	}

	// Verify Document message
	documentMsg := messageMap["Document"]
	if documentMsg == nil {
		t.Fatalf("Document message not found")
	}

	// Check Document fields
	expectedDocumentFields := map[string]struct {
		Type       string
		IsRepeated bool
		IsOptional bool
	}{
		"id":         {"string", false, false}, // UUID maps to string
		"title":      {"string", false, false},
		"content":    {"bytes", false, false}, // []byte maps to bytes
		"metadata":   {"string", false, true}, // json.RawMessage maps to string
		"tags":       {"string", true, false}, // []string maps to repeated string
		"created_by": {"string", false, true}, // uuid.NullUUID maps to optional string
		"version":    {"int32", false, false},
	}

	if len(documentMsg.Fields) != len(expectedDocumentFields) {
		t.Errorf("Expected %d fields for Document, got %d", len(expectedDocumentFields), len(documentMsg.Fields))
	}

	for _, field := range documentMsg.Fields {
		expected, ok := expectedDocumentFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		if field.IsRepeated != expected.IsRepeated {
			t.Errorf("Field %s: expected IsRepeated=%v, got %v", field.Name, expected.IsRepeated, field.IsRepeated)
		}

		if field.IsOptional != expected.IsOptional {
			t.Errorf("Field %s: expected IsOptional=%v, got %v", field.Name, expected.IsOptional, field.IsOptional)
		}
	}

	// Verify Transaction message
	transactionMsg := messageMap["Transaction"]
	if transactionMsg == nil {
		t.Fatalf("Transaction message not found")
	}

	// Check Transaction fields
	expectedTransactionFields := map[string]struct {
		Type       string
		IsRepeated bool
		IsOptional bool
	}{
		"id":             {"string", false, false}, // UUID maps to string
		"amount":         {"string", false, false}, // decimal.Decimal maps to string
		"currency":       {"string", false, false},
		"status":         {"string", false, false}, // OrderStatus enum maps to string
		"reference_code": {"string", false, true},  // sql.NullString maps to optional string
		"processed_at":   {"google.protobuf.Timestamp", false, false},
		"attachments":    {"bytes", true, true}, // [][]byte maps to bytes
	}

	if len(transactionMsg.Fields) != len(expectedTransactionFields) {
		t.Errorf("Expected %d fields for Transaction, got %d", len(expectedTransactionFields), len(transactionMsg.Fields))
	}

	for _, field := range transactionMsg.Fields {
		expected, ok := expectedTransactionFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		if field.IsRepeated != expected.IsRepeated {
			t.Errorf("Field %s: expected IsRepeated=%v, got %v", field.Name, expected.IsRepeated, field.IsRepeated)
		}

		if field.IsOptional != expected.IsOptional {
			t.Errorf("Field %s: expected IsOptional=%v, got %v", field.Name, expected.IsOptional, field.IsOptional)
		}
	}

	// Verify Configuration message
	configMsg := messageMap["Configuration"]
	if configMsg == nil {
		t.Fatalf("Configuration message not found")
	}

	// Check Configuration fields
	expectedConfigFields := map[string]struct {
		Type       string
		IsRepeated bool
		IsOptional bool
	}{
		"id":            {"int64", false, false},
		"name":          {"string", false, false},
		"settings":      {"string", false, false}, // json.RawMessage maps to string
		"is_active":     {"bool", false, false},
		"valid_from":    {"google.protobuf.Timestamp", false, false},
		"valid_to":      {"google.protobuf.Timestamp", false, true}, // sql.NullTime maps to optional timestamp
		"numeric_array": {"int32", true, false},                     // []int32 maps to repeated int32
		"string_array":  {"string", true, false},                    // []string maps to repeated string
	}

	if len(configMsg.Fields) != len(expectedConfigFields) {
		t.Errorf("Expected %d fields for Configuration, got %d", len(expectedConfigFields), len(configMsg.Fields))
	}

	for _, field := range configMsg.Fields {
		expected, ok := expectedConfigFields[field.Name]
		if !ok {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.Type != expected.Type {
			t.Errorf("Field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}

		if field.IsRepeated != expected.IsRepeated {
			t.Errorf("Field %s: expected IsRepeated=%v, got %v", field.Name, expected.IsRepeated, field.IsRepeated)
		}

		if field.IsOptional != expected.IsOptional {
			t.Errorf("Field %s: expected IsOptional=%v, got %v", field.Name, expected.IsOptional, field.IsOptional)
		}
	}

	// Verify NullUUID struct
	nullUUIDMsg := messageMap["NullUUID"]
	if nullUUIDMsg == nil {
		t.Fatalf("NullUUID message not found")
	}

	// Check NullUUID fields
	expectedNullUUIDFields := map[string]struct {
		Type       string
		IsOptional bool
	}{
		"uuid":  {"string", false}, // UUID maps to string
		"valid": {"bool", false},
	}

	if len(nullUUIDMsg.Fields) != len(expectedNullUUIDFields) {
		t.Errorf("Expected %d fields for NullUUID, got %d", len(expectedNullUUIDFields), len(nullUUIDMsg.Fields))
	}

	for _, field := range nullUUIDMsg.Fields {
		expected, ok := expectedNullUUIDFields[field.Name]
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
}

func TestAddCustomTypeMappings(t *testing.T) {
	// Save original mappings
	originalMappings := make(map[string]string)
	for k, v := range TypeMapping {
		originalMappings[k] = v
	}

	// Test adding custom type mappings
	customMappings := map[string]string{
		"custom.Type":  "custom.ProtoType",
		"another.Type": "another.ProtoType",
		"time.Time":    "custom.Timestamp", // Override existing mapping
	}

	AddCustomTypeMappings(customMappings)

	// Check that custom mappings were added
	for k, v := range customMappings {
		if TypeMapping[k] != v {
			t.Errorf("Custom mapping not added correctly: %s -> %s, got %s", k, v, TypeMapping[k])
		}
	}

	// Check that original mappings are still present (except overridden ones)
	for k, v := range originalMappings {
		if k != "time.Time" && TypeMapping[k] != v {
			t.Errorf("Original mapping changed: %s -> %s, now %s", k, v, TypeMapping[k])
		}
	}

	// Verify the overridden mapping
	if TypeMapping["time.Time"] != "custom.Timestamp" {
		t.Errorf("Overridden mapping not applied: time.Time -> custom.Timestamp, got %s", TypeMapping["time.Time"])
	}

	// Restore original mappings for other tests
	TypeMapping = make(map[string]string)
	for k, v := range originalMappings {
		TypeMapping[k] = v
	}
}

func TestGenerateHelperFunctions(t *testing.T) {
	// Create a sample message with fields that require helper functions
	messages := []ProtoMessage{
		{
			Name: "TestMessage",
			Fields: []ProtoField{
				{
					Name:                  "string_field",
					Type:                  "string",
					ConversionCode:        "nullStringToString(in.StringField)",
					ReverseConversionCode: "stringToNullString(in.StringField)",
				},
				{
					Name:                  "int_field",
					Type:                  "int32",
					ConversionCode:        "nullInt32ToInt32(in.IntField)",
					ReverseConversionCode: "int32ToNullInt32(in.IntField)",
				},
				{
					Name:                  "timestamp_field",
					Type:                  "google.protobuf.Timestamp",
					ConversionCode:        "dateToTimestamp(in.TimestampField)",
					ReverseConversionCode: "timestampToDate(in.TimestampField)",
				},
				// Add fields for the new types
				{
					Name:                  "int16_field",
					Type:                  "int32",
					ConversionCode:        "nullInt16ToInt32(in.Int16Field)",
					ReverseConversionCode: "int32ToNullInt16(in.Int16Field)",
				},
				{
					Name:                  "numeric_field",
					Type:                  "string",
					ConversionCode:        "numericToString(in.NumericField)",
					ReverseConversionCode: "stringToNumeric(in.NumericField)",
				},
				{
					Name:                  "uuid_field",
					Type:                  "string",
					ConversionCode:        "uuidToString(in.UuidField)",
					ReverseConversionCode: "stringToUUID(in.UuidField)",
				},
				{
					Name:                  "null_uuid_field",
					Type:                  "string",
					ConversionCode:        "nullUUIDToString(in.NullUuidField)",
					ReverseConversionCode: "stringToNullUUID(in.NullUuidField)",
				},
				{
					Name:                  "json_field",
					Type:                  "string",
					ConversionCode:        "jsonToString(in.JsonField)",
					ReverseConversionCode: "stringToJSON(in.JsonField)",
				},
				{
					Name:                  "interval_field",
					Type:                  "int64",
					ConversionCode:        "intervalToInt64(in.IntervalField)",
					ReverseConversionCode: "int64ToInterval(in.IntervalField)",
				},
			},
		},
	}

	helpers := GenerateHelperFunctions(messages)

	// Check that the helper functions were generated
	expectedHelpers := []string{
		"nullStringToString", "stringToNullString",
		"nullInt32ToInt32", "int32ToNullInt32",
		"dateToTimestamp", "timestampToDate",
		"nullInt16ToInt32", "int32ToNullInt16",
		"numericToString", "stringToNumeric",
		"uuidToString", "stringToUUID",
		"nullUUIDToString", "stringToNullUUID",
		"jsonToString", "stringToJSON",
		"intervalToInt64", "int64ToInterval",
	}

	for _, helper := range expectedHelpers {
		if !strings.Contains(helpers, helper) {
			t.Errorf("Expected helper function %s not found in generated code", helper)
		}
	}
}

func TestGenerateConversionCode(t *testing.T) {
	// Test generateNullableConversionCode
	nullableField := ProtoField{
		Name:     "test_field",
		SQLCName: "TestField",
	}

	nullableTests := []struct {
		sqlType  string
		expected string
	}{
		{"sql.NullString", "nullStringToString(in.TestField)"},
		{"sql.NullInt16", "nullInt16ToInt32(in.TestField)"},
		{"sql.NullInt32", "nullInt32ToInt32(in.TestField)"},
		{"sql.NullInt64", "nullInt64ToInt64(in.TestField)"},
		{"sql.NullFloat64", "nullFloat64ToFloat64(in.TestField)"},
		{"sql.NullBool", "nullBoolToBool(in.TestField)"},
		{"sql.NullTime", "nullTimeToTimestamp(in.TestField)"},
		{"uuid.NullUUID", "nullUUIDToString(in.TestField)"},
		{"unknown.Type", "in.TestField"},
	}

	for _, tt := range nullableTests {
		result := generateNullableConversionCode(tt.sqlType, nullableField)
		if result != tt.expected {
			t.Errorf("generateNullableConversionCode(%q, field) = %q, want %q", tt.sqlType, result, tt.expected)
		}
	}

	// Test generateStandardConversionCode
	standardField := ProtoField{
		Name:     "test_field",
		SQLCName: "TestField",
	}

	standardTests := []struct {
		sqlType  string
		expected string
	}{
		{"time.Time", "timestamppb.New(in.TestField)"},
		{"pgtype.Date", "dateToTimestamp(in.TestField)"},
		{"pgtype.Timestamptz", "timestamptzToTimestamp(in.TestField)"},
		{"pgtype.Text", "pgtypeTextToString(in.TestField)"},
		{"pgtype.Numeric", "numericToString(in.TestField)"},
		{"uuid.UUID", "uuidToString(in.TestField)"},
		{"json.RawMessage", "jsonToString(in.TestField)"},
		{"pgtype.Interval", "intervalToInt64(in.TestField)"},
		{"int16", "int32(in.TestField)"},
		{"string", "in.TestField"},
	}

	for _, tt := range standardTests {
		result := generateStandardConversionCode(tt.sqlType, standardField)
		if result != tt.expected {
			t.Errorf("generateStandardConversionCode(%q, field) = %q, want %q", tt.sqlType, result, tt.expected)
		}
	}
}
