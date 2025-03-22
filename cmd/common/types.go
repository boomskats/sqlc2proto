package common

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for code generation
type Config struct {
	// Basic configuration
	SQLCDir          string `yaml:"sqlcDir"`
	ProtoOutputDir   string `yaml:"protoDir"`
	ProtoPackageName string `yaml:"protoPackage"`
	GoPackagePath    string `yaml:"goPackage"`
	ModuleName       string `yaml:"moduleName"`
	ProtoGoImport    string `yaml:"protoGoImport"`

	// Type mapping configuration
	TypeMappings         map[string]string `yaml:"typeMappings"`
	NullableTypeMappings map[string]string `yaml:"nullableTypeMappings"`

	// Feature flags
	GenerateMappers  bool `yaml:"withMappers"`
	GenerateServices bool `yaml:"withServices"`

	// Field naming configuration
	FieldStyle string `yaml:"fieldStyle"` // "json", "snake_case", or "original"

	// Service naming configuration
	ServiceNaming string `yaml:"serviceNaming"` // "entity", "flat", or "custom"
	ServicePrefix string `yaml:"servicePrefix"` // Prefix for service names (e.g., "API")
	ServiceSuffix string `yaml:"serviceSuffix"` // Suffix for service names (e.g., "Service")

	// Extended service options
	ServiceOptions ServiceOptions `yaml:"serviceOptions"`
}

// ServiceOptions contains configuration options for service generation
type ServiceOptions struct {
	// Whether to include pagination in list methods
	IncludePagination bool `yaml:"includePagination"`

	// Whether to use separate proto files for each service
	SplitServices bool `yaml:"splitServices"`

	// Whether to generate streaming methods (for list operations)
	EnableStreaming bool `yaml:"enableStreaming"`

	// Pagination field names
	PageSizeField      string `yaml:"pageSizeField"`      // Default: "limit"
	PageTokenField     string `yaml:"pageTokenField"`     // Default: "page_token"
	NextPageTokenField string `yaml:"nextPageTokenField"` // Default: "next_page_token"
	TotalSizeField     string `yaml:"totalSizeField"`     // Default: "total_size"
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		SQLCDir:              "./db/sqlc",
		ProtoOutputDir:       "./proto/gen",
		ProtoPackageName:     "api.v0",
		GoPackagePath:        "",
		GenerateMappers:      false,
		GenerateServices:     false,
		ServiceNaming:        "entity",
		ServicePrefix:        "",
		ServiceSuffix:        "Service",
		ModuleName:           "",
		ProtoGoImport:        "",
		FieldStyle:           "json",
		TypeMappings:         map[string]string{},
		NullableTypeMappings: map[string]string{},
		ServiceOptions:       DefaultServiceOptions(),
	}
}

// DefaultServiceOptions returns default service options
func DefaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		IncludePagination:  true,
		SplitServices:      false,
		EnableStreaming:    false,
		PageSizeField:      "limit",
		PageTokenField:     "page_token",
		NextPageTokenField: "next_page_token",
		TotalSizeField:     "total_size",
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return config, err
}
