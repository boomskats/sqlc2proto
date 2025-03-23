package parser

import (
	"maps"
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

// ConversionMapping maps Go types to conversion function templates
var ConversionMapping = map[string]ConversionFuncs{
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
}

// ConversionFuncs holds function templates for conversion
type ConversionFuncs struct {
	ToProto   string // Template for converting from Go to Proto
	FromProto string // Template for converting from Proto to Go
}

// AddCustomTypeMappings adds custom type mappings
func AddCustomTypeMappings(mappings map[string]string) {
	maps.Copy(TypeMapping, mappings)
}

// AddCustomNullableTypeMappings adds custom nullable type mappings
func AddCustomNullableTypeMappings(mappings map[string]string) {
	maps.Copy(NullableTypeMapping, mappings)
}

// GetTypeMapConfig returns a TypeMappingConfig based on the current mappings
func GetTypeMapConfig() TypeMappingConfig {
	return TypeMappingConfig{
		StandardTypes:    maps.Clone(TypeMapping),
		NullableTypes:    maps.Clone(NullableTypeMapping),
		CustomConverters: maps.Clone(ConversionMapping),
	}
}

// TypeMappingConfig holds mappings for Go to Protobuf types
type TypeMappingConfig struct {
	// Standard type mappings
	StandardTypes map[string]string
	// Nullable type mappings
	NullableTypes map[string]string
	// Custom conversion functions for special types
	CustomConverters map[string]ConversionFuncs
}
