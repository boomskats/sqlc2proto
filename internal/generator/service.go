package generator

import (
	"strings"

	"github.com/boomskats/sqlc2proto/cmd/common"
	"github.com/boomskats/sqlc2proto/internal/parser"
)

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

	return nil
}
