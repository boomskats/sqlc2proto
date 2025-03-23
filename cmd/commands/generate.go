package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boomskats/sqlc2proto/cmd/common"
	"github.com/boomskats/sqlc2proto/internal/generator"
	"github.com/boomskats/sqlc2proto/internal/includes"
	"github.com/boomskats/sqlc2proto/internal/parser"
	"github.com/spf13/cobra"
)

// NewGenerateCmd creates the generate command
func NewGenerateCmd() *cobra.Command {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Protocol Buffers from sqlc structs",
		Long: `Generate Protocol Buffer definitions from sqlc-generated Go structs.

This command processes sqlc-generated Go structs and creates corresponding Protocol Buffer
definitions, with a focus on Connect-RPC compatibility.

Example:
	 sqlc2proto generate --sqlc-dir=./db/sqlc --proto-dir=./proto --package=api.v1 --with-mappers --with-services
`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Try to load config file if specified or exists in default location
			if configFile != "" {
				if err := common.LoadConfigFile(configFile, &Config, verbose); err != nil {
					fmt.Printf("Error loading config file: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Try default config locations (sqlc2proto.yaml, sqlc2proto.yml, .sqlc2proto.yaml, .sqlc2proto.yml)
				common.TryLoadDefaultConfig(&Config, verbose)
			}

			// If go package is still empty, try to parse go.mod file
			if Config.GoPackagePath == "" {
				// Try to parse go.mod file to get module name
				moduleName, err := common.GetModuleNameFromGoMod()
				if err == nil && Config.ModuleName == "" {
					// If we found a module name and it's not already set, use it
					Config.ModuleName = moduleName
					if verbose {
						fmt.Printf("Found module name in go.mod: %s\n", moduleName)
					}
				}
				// Now infer from proto package and moduleName (which might have been set from go.mod)
				Config.GoPackagePath = common.InferGoPackage(Config.ProtoPackageName, Config.ModuleName)
			}

			if verbose {
				common.PrintConfig(Config)
			}

			if dryRun {
				fmt.Println("Dry run - no files will be generated")
			}

			// Ensure output directory exists
			if !dryRun {
				if err := os.MkdirAll(Config.ProtoOutputDir, 0o755); err != nil {
					fmt.Printf("Failed to create output directory: %v\n", err)
					os.Exit(1)
				}
			}

			// Check if includeFile is specified and exists
			var includesData *includes.IncludesFile
			if Config.IncludeFile != "" {
				if verbose {
					fmt.Printf("Looking for includes file at %s\n", Config.IncludeFile)
				}

				// Try to load the includes file
				includesFile, err := includes.LoadIncludesFile(Config.IncludeFile)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Printf("Includes file %s not found. Run 'sqlc2proto getincludes' to generate it.\n", Config.IncludeFile)
						fmt.Println("Proceeding with generating all models and queries...")
					} else {
						fmt.Printf("Error loading includes file: %v\n", err)
						os.Exit(1) // Halt on parsing errors
					}
				} else {
					includesData = &includesFile
					if verbose {
						fmt.Printf("Loaded includes file with %d models and %d queries\n",
							len(includesFile.Models), len(includesFile.Queries))
					}
				}
			}

			// Process sqlc directory
			messages, err := parser.ProcessSQLCDirectory(Config.SQLCDir, Config.FieldStyle)
			if err != nil {
				fmt.Printf("Failed to process sqlc directory: %v\n", err)
				os.Exit(1)
			}

			// Parse the Querier interface if service generation is enabled
			var queryMethods []parser.QueryMethod
			if Config.GenerateServices {
				queryMethods, err = parser.ParseSQLCQuerierInterface(Config.SQLCDir)
				if err != nil {
					if verbose {
						fmt.Printf("Warning: Failed to parse Querier interface: %v\n", err)
						fmt.Println("Make sure sqlc is configured with emit_interface: true")
						fmt.Println("Skipping service generation...")
					}
				}
			}

			// Filter messages and queries based on includes file
			if includesData != nil && (len(includesData.Models) > 0 || len(includesData.Queries) > 0) {
				// Resolve dependencies for included queries
				resolvedIncludes := includes.ResolveDependencies(*includesData, queryMethods, messages)

				if verbose {
					// Log which models are included due to dependencies
					additions := includes.GetDependencyAdditions(*includesData, resolvedIncludes)
					if len(additions) > 0 {
						fmt.Println("Models included due to dependencies:")
						for _, model := range additions {
							fmt.Printf("  - %s\n", model)
						}
					}
				}

				// Filter messages
				var filteredMessages []parser.ProtoMessage
				for _, msg := range messages {
					if includes.IsModelIncluded(resolvedIncludes, msg.Name) {
						filteredMessages = append(filteredMessages, msg)
					}
				}
				messages = filteredMessages

				// Filter query methods
				if len(queryMethods) > 0 {
					var filteredQueryMethods []parser.QueryMethod
					for _, method := range queryMethods {
						if includes.IsQueryIncluded(*includesData, method.Name) {
							filteredQueryMethods = append(filteredQueryMethods, method)
						}
					}
					queryMethods = filteredQueryMethods
				}

				if verbose {
					fmt.Printf("After filtering: %d message types and %d query methods\n",
						len(messages), len(queryMethods))
				}
			}

			if verbose {
				fmt.Printf("Generating %d message types from %s\n", len(messages), Config.SQLCDir)
				for _, msg := range messages {
					fmt.Printf("  - %s (%d fields)\n", msg.Name, len(msg.Fields))
				}
			}

			// Generate proto file
			protoPath := filepath.Join(Config.ProtoOutputDir, "models.proto")
			if dryRun {
				fmt.Printf("Would generate proto file: %s\n", protoPath)
			} else {
				if err := generator.GenerateProtoFile(messages, Config, protoPath); err != nil {
					fmt.Printf("Failed to generate proto file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Generated Protobuf definitions in %s\n", protoPath)
			}

			// Generate mapper file if requested
			if Config.GenerateMappers {
				// Remove old mappers.go file if it exists (for backward compatibility)
				oldMapperPath := filepath.Join(Config.ProtoOutputDir, "mappers.go")
				if !dryRun {
					// Ignore error if file doesn't exist
					_ = os.Remove(oldMapperPath)
				}

				// Create mappers directory
				mappersDir := filepath.Join(Config.ProtoOutputDir, "mappers")
				if !dryRun {
					if err := os.MkdirAll(mappersDir, 0o755); err != nil {
						fmt.Printf("Failed to create mappers directory: %v\n", err)
						os.Exit(1)
					}
				}

				mapperPath := filepath.Join(mappersDir, "mappers.go")
				if dryRun {
					fmt.Printf("Would generate mapper file: %s\n", mapperPath)
				} else {
					if err := generator.GenerateMapperFile(messages, Config, mapperPath); err != nil {
						fmt.Printf("Failed to generate mapper file: %v\n", err)
						os.Exit(1)
					}
					fmt.Printf("Generated mapper functions in %s\n", mapperPath)
				}
			}

			// Generate service definitions if requested
			if Config.GenerateServices && len(queryMethods) > 0 {
				if verbose {
					fmt.Printf("Generating services for %d query methods\n", len(queryMethods))
					for _, method := range queryMethods {
						fmt.Printf("  - %s (returns %s)\n", method.Name, method.ReturnType)
					}
				}

				// Generate service definitions
				services := parser.GenerateServiceDefinitions(queryMethods, messages)

				// Generate service.proto file
				servicePath := filepath.Join(Config.ProtoOutputDir, "service.proto")
				if dryRun {
					fmt.Printf("Would generate service file: %s\n", servicePath)
				} else {
					if err := generator.GenerateServiceFile(services, Config, servicePath); err != nil {
						fmt.Printf("Failed to generate service file: %v\n", err)
						os.Exit(1)
					}
					fmt.Printf("Generated service definitions in %s\n", servicePath)
				}
			} else if Config.GenerateServices && len(queryMethods) == 0 {
				fmt.Println("No query methods found or selected. Skipping service generation.")
			}
		},
	}

	// Add flags to the generate command
	generateCmd.Flags().StringVar(&Config.SQLCDir, "sqlc-dir", Config.SQLCDir, "Directory containing sqlc-generated files")
	generateCmd.Flags().StringVar(&Config.ProtoOutputDir, "proto-dir", Config.ProtoOutputDir, "Directory to output .proto files")
	generateCmd.Flags().StringVar(&Config.ProtoPackageName, "package", Config.ProtoPackageName, "Package name for proto files")
	generateCmd.Flags().StringVar(&Config.GoPackagePath, "go-package", Config.GoPackagePath, "Go package path for generated proto code")
	generateCmd.Flags().StringVar(&Config.ModuleName, "module", Config.ModuleName, "Module name for import paths")
	generateCmd.Flags().StringVar(&Config.ProtoGoImport, "proto-go-import", Config.ProtoGoImport, "Import path for protobuf-generated Go code")
	generateCmd.Flags().BoolVar(&Config.GenerateMappers, "with-mappers", Config.GenerateMappers, "Generate conversion functions between sqlc and proto types")
	generateCmd.Flags().BoolVar(&Config.GenerateServices, "with-services", Config.GenerateServices, "Generate service definitions from sqlc queries")
	generateCmd.Flags().StringVar(&Config.FieldStyle, "field-style", Config.FieldStyle, "Field naming style: 'json' (use json tags), 'snake_case' (convert to snake_case), or 'original' (keep original casing)")
	generateCmd.Flags().StringVar(&Config.IncludeFile, "include-file", Config.IncludeFile, "Path to file specifying which models and queries to include")
	generateCmd.Flags().Bool("dry-run", false, "Show what would be generated without writing files")

	return generateCmd
}
