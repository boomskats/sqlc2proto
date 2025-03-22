package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boomskats/sqlc2proto/internal/generator"
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
func LoadConfigFile(path string, cfg *generator.Config, verbose bool) error {
	if verbose {
		fmt.Printf("Loading config from %s\n", path)
	}

	config, err := generator.LoadConfig(path)
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
	if len(config.TypeMappings) > 0 {
		parser.AddCustomTypeMappings(config.TypeMappings)
	}
	if config.ModuleName != "" {
		cfg.ModuleName = config.ModuleName
	}
	if config.ProtoGoImport != "" {
		cfg.ProtoGoImport = config.ProtoGoImport
	}

	return nil
}

// TryLoadDefaultConfig attempts to load configuration from default paths
func TryLoadDefaultConfig(cfg *generator.Config, verbose bool) bool {
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

// ParseGoModFile reads the first line of the go.mod file and extracts the module name
func ParseGoModFile() (string, error) {
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
func PrintConfig(cfg generator.Config) {
	fmt.Println("Using configuration:")
	fmt.Printf("  SQLC Directory:    %s\n", cfg.SQLCDir)
	fmt.Printf("  Proto Directory:   %s\n", cfg.ProtoOutputDir)
	fmt.Printf("  Proto Package:     %s\n", cfg.ProtoPackageName)
	fmt.Printf("  Proto Go Import:   %s\n", cfg.ProtoGoImport)
	fmt.Printf("  Go Package:        %s\n", cfg.GoPackagePath)
	fmt.Printf("  Module Name:       %s\n", cfg.ModuleName)
	fmt.Printf("  Generate Mappers:  %t\n", cfg.GenerateMappers)
}

// WriteConfigWithComments writes the configuration to a YAML file with comments
func WriteConfigWithComments(config generator.Config, path string) error {
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

	content += `# typeMappings is a map of SQLC type names to protobuf type names
typeMappings:
`
	if len(config.TypeMappings) > 0 {
		for k, v := range config.TypeMappings {
			content += `  "` + k + `": "` + v + `"
`
		}
	} else {
		content += `#  "CustomType": "string"
`
	}

	// Write the content to the file
	return os.WriteFile(path, []byte(content), 0644)
}

// DefaultConfig returns a default configuration
func DefaultConfig() generator.Config {
	return generator.Config{
		SQLCDir:          "./db/sqlc",
		ProtoOutputDir:   "./proto/gen",
		ProtoPackageName: "api.v1",
		GoPackagePath:    "",
		GenerateMappers:  false,
		ModuleName:       "",
		ProtoGoImport:    "",
	}
}
