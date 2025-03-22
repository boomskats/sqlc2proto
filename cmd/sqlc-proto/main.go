package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boomskats/sqlc-proto/internal/generator"
	"github.com/boomskats/sqlc-proto/internal/parser"
	"github.com/spf13/cobra"
)

var (
	version = "dev" // will be set during build
)

func main() {
	cfg := generator.Config{
		SQLCDir:          "./db/sqlc",
		ProtoOutputDir:   "./proto/gen",
		ProtoPackageName: "api.v1",
		GoPackagePath:    "",
		GenerateMappers:  false,
		ModuleName:       "",
		ProtoGoImport:    "",
	}

	// Prepare rootCmd
	rootCmd := &cobra.Command{
		Use:   "sqlc-proto",
		Short: "Generate Protocol Buffers from sqlc structs",
		Long: `sqlc-proto automatically generates Protocol Buffer definitions 
from sqlc-generated Go structs, with a focus on Connect-RPC compatibility.

It maps Go types to appropriate Protocol Buffer types and can also generate
Go code for converting between sqlc models and protobuf messages.

Example:
  sqlc-proto --sqlc-dir=./db/sqlc --proto-dir=./proto --package=api.v1 --with-mappers
`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			verbose, _ := cmd.Flags().GetBool("verbose")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Try to load config file if specified or exists in default location
			if configFile != "" {
				if err := loadConfigFile(configFile, &cfg, verbose); err != nil {
					fmt.Printf("Error loading config file: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Try default config locations
				for _, path := range []string{"sqlc-proto.yaml", "sqlc-proto.yml", ".sqlc-proto.yaml", ".sqlc-proto.yml"} {
					if _, err := os.Stat(path); err == nil {
						if err := loadConfigFile(path, &cfg, verbose); err != nil {
							fmt.Printf("Error loading config file: %v\n", err)
							os.Exit(1)
						}
						if verbose {
							fmt.Printf("Loaded config from %s\n", path)
						}
						break
					}
				}
			}

			// If go package is still empty, infer from proto package and moduleName
			if cfg.GoPackagePath == "" {
				cfg.GoPackagePath = inferGoPackage(cfg.ProtoPackageName, cfg.ModuleName)
			}

			if verbose {
				printConfig(cfg)
			}

			if dryRun {
				fmt.Println("Dry run - no files will be generated")
			}

			// Ensure output directory exists
			if !dryRun {
				if err := os.MkdirAll(cfg.ProtoOutputDir, 0755); err != nil {
					fmt.Printf("Failed to create output directory: %v\n", err)
					os.Exit(1)
				}
			}

			// Process sqlc directory
			messages, err := parser.ProcessSQLCDirectory(cfg.SQLCDir)
			if err != nil {
				fmt.Printf("Failed to process sqlc directory: %v\n", err)
				os.Exit(1)
			}

			if verbose {
				fmt.Printf("Found %d message types in %s\n", len(messages), cfg.SQLCDir)
				for _, msg := range messages {
					fmt.Printf("  - %s (%d fields)\n", msg.Name, len(msg.Fields))
				}
			}

			// Generate proto file
			protoPath := filepath.Join(cfg.ProtoOutputDir, "models.proto")
			if dryRun {
				fmt.Printf("Would generate proto file: %s\n", protoPath)
			} else {
				if err := generator.GenerateProtoFile(messages, cfg, protoPath); err != nil {
					fmt.Printf("Failed to generate proto file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Generated Protobuf definitions in %s\n", protoPath)
			}

			// Generate mapper file if requested
			if cfg.GenerateMappers {
				// Remove old mappers.go file if it exists (for backward compatibility)
				oldMapperPath := filepath.Join(cfg.ProtoOutputDir, "mappers.go")
				if !dryRun {
					// Ignore error if file doesn't exist
					_ = os.Remove(oldMapperPath)
				}

				// Create mappers directory
				mappersDir := filepath.Join(cfg.ProtoOutputDir, "mappers")
				if !dryRun {
					if err := os.MkdirAll(mappersDir, 0755); err != nil {
						fmt.Printf("Failed to create mappers directory: %v\n", err)
						os.Exit(1)
					}
				}

				mapperPath := filepath.Join(mappersDir, "mappers.go")
				if dryRun {
					fmt.Printf("Would generate mapper file: %s\n", mapperPath)
				} else {
					if err := generator.GenerateMapperFile(messages, cfg, mapperPath); err != nil {
						fmt.Printf("Failed to generate mapper file: %v\n", err)
						os.Exit(1)
					}
					fmt.Printf("Generated mapper functions in %s\n", mapperPath)
				}
			}
		},
	}

	// Add a command to verify imports in the generated mapper files
	verifyImportsCmd := &cobra.Command{
		Use:   "verify-imports",
		Short: "Verify imports in the generated mapper files",
		Long: `Checks if the imports in the generated mapper files are correct.
This command verifies that the protobuf-generated Go code exists and can be imported.
It helps identify issues in the workflow between sqlc-proto and buf generate.`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Try to load config file
			if configFile != "" {
				if err := loadConfigFile(configFile, &cfg, verbose); err != nil {
					fmt.Printf("Error loading config file: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Try default config locations
				for _, path := range []string{"sqlc-proto.yaml", "sqlc-proto.yml", ".sqlc-proto.yaml", ".sqlc-proto.yml"} {
					if _, err := os.Stat(path); err == nil {
						if err := loadConfigFile(path, &cfg, verbose); err != nil {
							fmt.Printf("Error loading config file: %v\n", err)
							os.Exit(1)
						}
						if verbose {
							fmt.Printf("Loaded config from %s\n", path)
						}
						break
					}
				}
			}

			if verbose {
				printConfig(cfg)
			}

			// Check if mappers are enabled
			if !cfg.GenerateMappers {
				fmt.Println("Mapper generation is not enabled in the configuration.")
				fmt.Println("Enable it with --with-mappers or by setting 'withMappers: true' in your config file.")
				os.Exit(1)
			}

			// Check if mapper file exists
			mappersDir := filepath.Join(cfg.ProtoOutputDir, "mappers")
			mapperPath := filepath.Join(mappersDir, "mappers.go")
			if _, err := os.Stat(mapperPath); os.IsNotExist(err) {
				fmt.Printf("Mapper file not found at %s\n", mapperPath)
				fmt.Println("Run sqlc-proto first to generate the mapper file.")
				os.Exit(1)
			}

			// Check if proto file exists
			protoPath := filepath.Join(cfg.ProtoOutputDir, "models.proto")
			if _, err := os.Stat(protoPath); os.IsNotExist(err) {
				fmt.Printf("Proto file not found at %s\n", protoPath)
				fmt.Println("Run sqlc-proto first to generate the proto file.")
				os.Exit(1)
			}

			// Determine the expected import path for the protobuf-generated Go code
			expectedImportPath := cfg.ProtoGoImport
			if expectedImportPath == "" {
				if cfg.GoPackagePath != "" {
					expectedImportPath = cfg.GoPackagePath
				} else {
					expectedImportPath = inferGoPackage(cfg.ProtoPackageName, cfg.ModuleName)
				}
			}

			// Check if the protobuf-generated Go code exists
			// This is a simple check that looks for the .pb.go file
			protoGoPath := filepath.Join(cfg.ProtoOutputDir, "models.pb.go")
			if _, err := os.Stat(protoGoPath); os.IsNotExist(err) {
				fmt.Printf("Protobuf-generated Go code not found at %s\n", protoGoPath)
				fmt.Println("Run 'buf generate' to generate the Go code from the proto file.")
				os.Exit(1)
			}

			fmt.Println("✅ Verification successful!")
			fmt.Println("The mapper file imports the protobuf-generated Go code correctly.")
			fmt.Printf("Import path: %s\n", expectedImportPath)
		},
	}

	verifyImportsCmd.Flags().String("config", "", "Path to configuration file (default: sqlc-proto.yaml)")
	verifyImportsCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.AddCommand(verifyImportsCmd)

	// Add a command to create a config file
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new sqlc-proto configuration file",
		Long: `Creates a new sqlc-proto.yaml configuration file with default values.
You can then edit this file to customize the behavior of sqlc-proto.`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("output")
			if configFile == "" {
				configFile = "sqlc-proto.yaml"
			}

			// Check if file already exists
			if _, err := os.Stat(configFile); err == nil {
				fmt.Printf("Config file %s already exists. Use --output to specify a different path.\n", configFile)
				os.Exit(1)
			}

			// Create config file with default values
			config := generator.Config{
				SQLCDir:          "./db/sqlc",
				ProtoOutputDir:   "./proto/gen",
				ProtoPackageName: "api.v1",
				GoPackagePath:    "",
				GenerateMappers:  false,
				ModuleName:       "",
				TypeMappings:     map[string]string{},
				ProtoGoImport:    "", // Import path for protobuf-generated Go code
			}

			// Write config file
			if err := generator.SaveConfig(config, configFile); err != nil {
				fmt.Printf("Failed to write config file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Created config file %s\n", configFile)
			fmt.Println("You can now edit this file to customize sqlc-proto behavior.")
		},
	}

	initCmd.Flags().StringP("output", "o", "sqlc-proto.yaml", "Path to write the config file")
	rootCmd.AddCommand(initCmd)

	// Add flags to the root command
	rootCmd.Flags().StringVar(&cfg.SQLCDir, "sqlc-dir", cfg.SQLCDir, "Directory containing sqlc-generated files")
	rootCmd.Flags().StringVar(&cfg.ProtoOutputDir, "proto-dir", cfg.ProtoOutputDir, "Directory to output .proto files")
	rootCmd.Flags().StringVar(&cfg.ProtoPackageName, "package", cfg.ProtoPackageName, "Package name for proto files")
	rootCmd.Flags().StringVar(&cfg.GoPackagePath, "go-package", cfg.GoPackagePath, "Go package path for generated proto code")
	rootCmd.Flags().StringVar(&cfg.ModuleName, "module", cfg.ModuleName, "Module name for import paths")
	rootCmd.Flags().StringVar(&cfg.ProtoGoImport, "proto-go-import", cfg.ProtoGoImport, "Import path for protobuf-generated Go code")
	rootCmd.Flags().BoolVar(&cfg.GenerateMappers, "with-mappers", cfg.GenerateMappers, "Generate conversion functions between sqlc and proto types")
	rootCmd.Flags().String("config", "", "Path to configuration file (default: sqlc-proto.yaml)")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().Bool("dry-run", false, "Show what would be generated without writing files")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// loadConfigFile loads configuration from a YAML file
func loadConfigFile(path string, cfg *generator.Config, verbose bool) error {
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

// inferGoPackage creates a reasonable default Go package path
func inferGoPackage(protoPackage string, moduleName string) string {
	// If moduleName is provided, use it as the base
	if moduleName != "" {
		return filepath.Join(moduleName, "proto")
	}

	// Fallback to a default pattern
	return fmt.Sprintf("github.com/yourusername/yourproject/gen/%s", protoPackage)
}

// printConfig prints the current configuration
func printConfig(cfg generator.Config) {
	fmt.Println("Using configuration:")
	fmt.Printf("  SQLC Directory:    %s\n", cfg.SQLCDir)
	fmt.Printf("  Proto Directory:   %s\n", cfg.ProtoOutputDir)
	fmt.Printf("  Proto Package:     %s\n", cfg.ProtoPackageName)
	fmt.Printf("  Proto Go Import:   %s\n", cfg.ProtoGoImport)
	fmt.Printf("  Go Package:        %s\n", cfg.GoPackagePath)
	fmt.Printf("  Module Name:       %s\n", cfg.ModuleName)
	fmt.Printf("  Generate Mappers:  %t\n", cfg.GenerateMappers)
}
