package generator

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/boomskats/sqlc2proto/cmd/common"
	"github.com/boomskats/sqlc2proto/internal/parser"
	"github.com/iancoleman/strcase"
)

//go:embed service.tmpl
var serviceTemplate string

// GenerateServiceFile generates a service.proto file based on the configuration
func GenerateServiceFile(services []parser.ServiceDefinition, config common.Config, outputPath string) error {

	// Apply service naming configuration
	for i := range services {
		// Default name from entity (previously set)
		origName := services[i].Name

		switch config.ServiceNaming {
		case "flat":
			// Keep the original name without entity prefix
			// (Usually not desired, as it can cause name conflicts)
			// Do nothing as name is already set
		case "custom":
			// Apply custom prefix and suffix
			if config.ServicePrefix != "" {
				services[i].Name = config.ServicePrefix + origName
			}

			// Handle suffix replacement (default is "Service")
			if config.ServiceSuffix != "" && config.ServiceSuffix != "Service" {
				// Remove default "Service" suffix if present and add custom suffix
				if strings.HasSuffix(services[i].Name, "Service") {
					base := strings.TrimSuffix(services[i].Name, "Service")
					services[i].Name = base + config.ServiceSuffix
				} else {
					services[i].Name = services[i].Name + config.ServiceSuffix
				}
			}
		default: // "entity" (default)
			// Keep entity prefix and "Service" suffix
			// This is already the default from the parser
		}

		// Apply streaming options if enabled
		if config.ServiceOptions.EnableStreaming {
			for j := range services[i].Methods {
				method := &services[i].Methods[j]

				// Add streaming for list methods
				if strings.HasPrefix(method.Name, "List") {
					method.StreamingServer = true
				}
			}
		}

		// Apply pagination options
		if config.ServiceOptions.IncludePagination {
			for j := range services[i].Methods {
				method := &services[i].Methods[j]

				// Add pagination fields to list methods
				if strings.HasPrefix(method.Name, "List") {
					// Update request field names
					for k, field := range method.RequestFields {
						if field.Name == "limit" {
							method.RequestFields[k].Name = config.ServiceOptions.PageSizeField
						} else if field.Name == "page_token" {
							method.RequestFields[k].Name = config.ServiceOptions.PageTokenField
						}
					}

					// Update response field names
					for k, field := range method.ResponseFields {
						if field.Name == "next_page_token" {
							method.ResponseFields[k].Name = config.ServiceOptions.NextPageTokenField
						} else if field.Name == "total_size" {
							method.ResponseFields[k].Name = config.ServiceOptions.TotalSizeField
						}
					}
				}
			}
		}
	}

	// Parse the template
	tmpl, err := template.New("service").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	}).Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	// Check if any service method uses Timestamp
	hasTimestamp := false
	for _, service := range services {
		for _, method := range service.Methods {
			for _, field := range method.RequestFields {
				if field.Type == "google.protobuf.Timestamp" {
					hasTimestamp = true
					break
				}
			}
			if hasTimestamp {
				break
			}
			for _, field := range method.ResponseFields {
				if field.Type == "google.protobuf.Timestamp" {
					hasTimestamp = true
					break
				}
			}
			if hasTimestamp {
				break
			}
		}
		if hasTimestamp {
			break
		}
	}

	// Create template data
	data := struct {
		Services       []parser.ServiceDefinition
		PackageName    string
		GoPackagePath  string
		ModelsProtoRef string
		HasTimestamp   bool
	}{
		Services:      services,
		PackageName:   config.ProtoPackageName,
		GoPackagePath: config.GoPackagePath,
		ModelsProtoRef: func() string {
			// For buf compatibility, we need to use a path that works with buf's import resolution
			// Buf typically looks for imports relative to the root of the buf module

			// Use the proto output directory directly, just ensure it's in the correct format
			// by removing any leading "./" and using forward slashes
			// protoDir := strings.TrimPrefix(config.ProtoOutputDir, "./")
			protoDir := ""

			// Join with models.proto to get the full import path
			return filepath.Join(protoDir, "models.proto")
		}(),
		HasTimestamp: hasTimestamp,
	}

	// Ensure the parent directory exists
	if err = os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
