package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boomskats/sqlc2proto/cmd/common"
	"github.com/boomskats/sqlc2proto/internal/includes"
	"github.com/boomskats/sqlc2proto/internal/parser"
	"github.com/spf13/cobra"
)

// NewGetIncludesCmd creates the getincludes command
func NewGetIncludesCmd() *cobra.Command {
	getIncludesCmd := &cobra.Command{
		Use:   "getincludes",
		Short: "Generate a template file for selecting models and queries",
		Long: `Generate a YAML file listing all available models and queries.
This file can be edited to select which models and queries to include in the generation process.

Example:
     sqlc2proto getincludes --output=./custom-includes.yaml
`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			verbose, _ := cmd.Flags().GetBool("verbose")
			outputPath, _ := cmd.Flags().GetString("output")
			force, _ := cmd.Flags().GetBool("force")

			// Try to load config file if specified or exists in default location
			if configFile != "" {
				if err := common.LoadConfigFile(configFile, &Config, verbose); err != nil {
					fmt.Printf("Error loading config file: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Try default config locations
				common.TryLoadDefaultConfig(&Config, verbose)
			}

			// If output path is not specified, use the one from config
			if outputPath == "" {
				outputPath = Config.IncludeFile
			}

			if verbose {
				fmt.Printf("Using output path: %s\n", outputPath)
			}

			// Check if the output file already exists
			if _, err := os.Stat(outputPath); err == nil && !force {
				// File exists, ask for confirmation
				fmt.Printf("File %s already exists. Overwrite? (y/N): ", outputPath)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Operation cancelled.")
					return
				}
			}

			// Process sqlc directory to find all models
			messages, err := parser.ProcessSQLCDirectory(Config.SQLCDir, Config.FieldStyle)
			if err != nil {
				fmt.Printf("Failed to process sqlc directory: %v\n", err)
				os.Exit(1)
			}

			// Extract model names
			var modelNames []string
			for _, msg := range messages {
				modelNames = append(modelNames, msg.Name)
			}

			if verbose {
				fmt.Printf("Found %d models in %s\n", len(modelNames), Config.SQLCDir)
				for _, model := range modelNames {
					fmt.Printf("  - %s\n", model)
				}
			}

			// Parse the Querier interface to find all queries
			var queryNames []string
			if Config.GenerateServices {
				queryMethods, err := parser.ParseSQLCQuerierInterface(Config.SQLCDir)
				if err != nil {
					if verbose {
						fmt.Printf("Warning: Failed to parse Querier interface: %v\n", err)
						fmt.Println("Make sure sqlc is configured with emit_interface: true")
						fmt.Println("No queries will be included in the template.")
					}
				} else {
					// Extract query names
					for _, method := range queryMethods {
						queryNames = append(queryNames, method.Name)
					}

					if verbose {
						fmt.Printf("Found %d queries in Querier interface\n", len(queryNames))
						for _, query := range queryNames {
							fmt.Printf("  - %s\n", query)
						}
					}
				}
			} else if verbose {
				fmt.Println("Service generation is disabled, no queries will be included in the template.")
			}

			// Ensure the output directory exists
			outputDir := filepath.Dir(outputPath)
			if outputDir != "." {
				if err := os.MkdirAll(outputDir, 0o755); err != nil {
					fmt.Printf("Failed to create output directory: %v\n", err)
					os.Exit(1)
				}
			}

			// Write the includes file with all entries commented out
			if err := includes.WriteIncludesFile(outputPath, modelNames, queryNames, true); err != nil {
				fmt.Printf("Failed to write includes file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Generated includes template at %s\n", outputPath)
			fmt.Println("Edit this file to uncomment the models and queries you want to include.")
			fmt.Println("Then run 'sqlc2proto generate' to generate Protocol Buffer definitions for the selected items.")
		},
	}

	// Add flags
	getIncludesCmd.Flags().String("output", "", "Output file path (default: value of includeFile in config or sqlc2proto.includes.yaml)")
	getIncludesCmd.Flags().Bool("force", false, "Overwrite existing file without confirmation")

	return getIncludesCmd
}
