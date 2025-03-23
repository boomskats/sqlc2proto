package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boomskats/sqlc2proto/internal/parser"
)

// DefaultConfigPaths contains the default paths to look for configuration files
var DefaultConfigPaths = []string{
	"sqlc2proto.yaml",
	"sqlc2proto.yml",
	".sqlc2proto.yaml",
	".sqlc2proto.yml",
}

// LoadConfigFile loads configuration from a YAML file
func LoadConfigFile(path string, cfg *Config, verbose bool) error {
	if verbose {
		fmt.Printf("Loading config from %s\n", path)
	}

	config, err := LoadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update config with values from file (only if set)
	if config.SQLCDir != "" {
		cfg.SQLCDir = config.SQLCDir
	}
	if config.ProtoOutputDir != "" {
		cfg.ProtoOutputDir = config.ProtoOutputDir
	}
	if config.ProtoPackageName != "" {
		cfg.ProtoPackageName = config.ProtoPackageName
	}
	if config.GoPackagePath != "" {
		cfg.GoPackagePath = config.GoPackagePath
	}
	if config.GenerateMappers {
		cfg.GenerateMappers = true
	}
	if config.GenerateServices {
		cfg.GenerateServices = true
	}
	if config.ServiceNaming != "" {
		cfg.ServiceNaming = config.ServiceNaming
	}
	if config.ServicePrefix != "" {
		cfg.ServicePrefix = config.ServicePrefix
	}
	if config.ServiceSuffix != "" {
		cfg.ServiceSuffix = config.ServiceSuffix
	}
	// Note: GenerateImpl field has been removed as Connect-RPC tooling
	// will generate the service implementation code from the proto definitions.
	if len(config.TypeMappings) > 0 {
		parser.AddCustomTypeMappings(config.TypeMappings)
	}
	if len(config.NullableTypeMappings) > 0 {
		parser.AddCustomNullableTypeMappings(config.NullableTypeMappings)
	}
	if config.ModuleName != "" {
		cfg.ModuleName = config.ModuleName
	}
	if config.ProtoGoImport != "" {
		cfg.ProtoGoImport = config.ProtoGoImport
	}
	if config.FieldStyle != "" {
		cfg.FieldStyle = config.FieldStyle
	}
	if config.IncludeFile != "" {
		cfg.IncludeFile = config.IncludeFile
	}

	return nil
}

// TryLoadDefaultConfig attempts to load configuration from default paths
func TryLoadDefaultConfig(cfg *Config, verbose bool) bool {
	for _, path := range DefaultConfigPaths {
		if _, err := os.Stat(path); err == nil {
			if err := LoadConfigFile(path, cfg, verbose); err != nil {
				fmt.Printf("Error loading config file: %v\n", err)
				os.Exit(1)
			}
			if verbose {
				fmt.Printf("Loaded config from %s\n", path)
			}
			return true
		}
	}
	return false
}

// InferGoPackage creates a reasonable default Go package path
func InferGoPackage(protoPackage string, moduleName string) string {
	// If moduleName is provided, use it as the base
	if moduleName != "" {
		return filepath.Join(moduleName, "proto")
	}

	// Fallback to a default pattern
	return fmt.Sprintf("github.com/yourusername/yourproject/gen/%s", protoPackage)
}

// GetModuleNameFromGoMod reads the first line of the go.mod file and extracts the module name
func GetModuleNameFromGoMod() (string, error) {
	// Check if go.mod exists
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return "", fmt.Errorf("go.mod file not found")
	}

	// Open the go.mod file
	file, err := os.Open("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod file: %w", err)
	}
	defer file.Close()

	// Read the first line
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", fmt.Errorf("go.mod file is empty")
	}

	// Parse the module name
	line := scanner.Text()
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "module ") {
		return "", fmt.Errorf("module declaration not found in go.mod")
	}

	moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
	if moduleName == "" {
		return "", fmt.Errorf("empty module name in go.mod")
	}

	return moduleName, nil
}

// PrintConfig prints the current configuration
func PrintConfig(cfg Config) {
	fmt.Println("Using configuration:")
	fmt.Printf("  SQLC Directory:    %s\n", cfg.SQLCDir)
	fmt.Printf("  Proto Directory:   %s\n", cfg.ProtoOutputDir)
	fmt.Printf("  Proto Package:     %s\n", cfg.ProtoPackageName)
	fmt.Printf("  Proto Go Import:   %s\n", cfg.ProtoGoImport)
	fmt.Printf("  Go Package:        %s\n", cfg.GoPackagePath)
	fmt.Printf("  Module Name:       %s\n", cfg.ModuleName)
	fmt.Printf("  Generate Mappers:  %t\n", cfg.GenerateMappers)
	fmt.Printf("  Generate Services: %t\n", cfg.GenerateServices)
	if cfg.GenerateServices {
		fmt.Printf("  Service Naming:    %s\n", cfg.ServiceNaming)
		if cfg.ServicePrefix != "" {
			fmt.Printf("  Service Prefix:    %s\n", cfg.ServicePrefix)
		}
		fmt.Printf("  Service Suffix:    %s\n", cfg.ServiceSuffix)
		// Note: Generate Impl has been removed as Connect-RPC tooling
		// will generate the service implementation code from the proto definitions.
	}
	fmt.Printf("  Field Style:       %s\n", cfg.FieldStyle)
	if cfg.IncludeFile != "" {
		fmt.Printf("  Include File:      %s\n", cfg.IncludeFile)
	}
}

// WriteConfigWithComments writes the configuration to a YAML file with comments
func WriteConfigWithComments(config Config, path string) error {
	// Create the content with comments
	content := `# sqlcDir is the directory containing sqlc-generated models.go
sqlcDir: "` + config.SQLCDir + `"
# protoDir is the target directory for the generated protobuf files
protoDir: "` + config.ProtoOutputDir + `"
# protoPackage is the package name for the generated protobuf files
protoPackage: "` + config.ProtoPackageName + `"
# goPackage is optional and will be derived from moduleName if not specified
`
	if config.GoPackagePath != "" {
		content += `goPackage: "` + config.GoPackagePath + `"
`
	} else {
		content += `# goPackage: "github.com/yourusername/yourproject/proto"
`
	}

	content += `withMappers: ` + fmt.Sprintf("%t", config.GenerateMappers) + `

# Service generation options
# withServices enables generation of service definitions from sqlc queries
withServices: ` + fmt.Sprintf("%t", config.GenerateServices) + `
# serviceNaming controls how services are named and organized
# Options: "entity" (group by entity), "flat" (one service), or "custom"
serviceNaming: "` + config.ServiceNaming + `"
# servicePrefix is an optional prefix for service names
` + (func() string {
		if config.ServicePrefix != "" {
			return `servicePrefix: "` + config.ServicePrefix + `"`
		}
		return `# servicePrefix: "API"`
	})() + `
# serviceSuffix is a suffix for service names (default: "Service")
serviceSuffix: "` + config.ServiceSuffix + `"
# Note: Service implementation generation has been removed as Connect-RPC tooling
# will generate the service implementation code from the proto definitions.
# moduleName is used to derive import paths for the generated code
`
	if config.ModuleName != "" {
		content += `moduleName: "` + config.ModuleName + `"
`
	} else {
		content += `# moduleName: "github.com/yourusername/yourproject"
`
	}

	content += `# protoGoImport specifies the import path for the protobuf-generated Go code
# This should match the go_package option in your buf.yaml or the output of buf generate
`
	if config.ProtoGoImport != "" {
		content += `protoGoImport: "` + config.ProtoGoImport + `"
`
	} else {
		content += `# protoGoImport: "github.com/yourusername/yourproject/proto"
`
	}

	content += `# fieldStyle controls how field names are generated in protobuf
# Options: "json" (use json tags), "snake_case" (convert to snake_case), or "original" (keep original casing)
fieldStyle: "` + config.FieldStyle + `"

# includeFile specifies the path to a file that lists which models and queries to include
# If not specified or the file doesn't exist, all models and queries will be included
` + (func() string {
		if config.IncludeFile != "" {
			return `includeFile: "` + config.IncludeFile + `"`
		}
		return `# includeFile: "sqlc2proto.includes.yaml"`
	})() + `

# typeMappings is a map of SQLC type names to protobuf type names
typeMappings:
`
	if len(config.TypeMappings) > 0 {
		for k, v := range config.TypeMappings {
			content += `  "` + k + `": "` + v + `"
`
		}
	} else {
		content += `#  "CustomType": "string"
#  "time.Time": "string"  # Override default
#  "uuid.UUID": "bytes"   # Use bytes instead of string for UUIDs
`
	}

	content += `
# nullableTypeMappings is a map of SQLC nullable type names to protobuf type names
nullableTypeMappings:
`
	if len(config.NullableTypeMappings) > 0 {
		for k, v := range config.NullableTypeMappings {
			content += `  "` + k + `": "` + v + `"
`
		}
	} else {
		content += `#  "sql.NullString": "google.protobuf.StringValue"  # Use wrapper types
#  "sql.NullInt64": "int64"
#  "uuid.NullUUID": "string"
`
	}

	// Write the content to the file
	return os.WriteFile(path, []byte(content), 0o644)
}
