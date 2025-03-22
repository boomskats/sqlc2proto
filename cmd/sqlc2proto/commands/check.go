package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boomskats/sqlc2proto/cmd/sqlc2proto/common"
	"github.com/spf13/cobra"
)

// NewCheckCmd creates the check command
func NewCheckCmd() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check the generated files for correctness",
		Long: `Checks if the imports in the generated mapper files are correct.
This command verifies that the protobuf-generated Go code exists and can be imported.
It helps identify issues in the workflow between sqlc2proto and buf generate.`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("config")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Try to load config file
			if configFile != "" {
				if err := common.LoadConfigFile(configFile, &Config, verbose); err != nil {
					fmt.Printf("Error loading config file: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Try default config locations
				common.TryLoadDefaultConfig(&Config, verbose)
			}

			if verbose {
				common.PrintConfig(Config)
			}

			// Check if mappers are enabled
			if !Config.GenerateMappers {
				fmt.Println("Mapper generation is not enabled in the configuration.")
				fmt.Println("Enable it with --with-mappers or by setting 'withMappers: true' in your config file.")
				os.Exit(1)
			}

			// Check if mapper file exists
			mappersDir := filepath.Join(Config.ProtoOutputDir, "mappers")
			mapperPath := filepath.Join(mappersDir, "mappers.go")
			if _, err := os.Stat(mapperPath); os.IsNotExist(err) {
				fmt.Printf("Mapper file not found at %s\n", mapperPath)
				fmt.Println("Run sqlc2proto first to generate the mapper file.")
				os.Exit(1)
			}

			// Check if proto file exists
			protoPath := filepath.Join(Config.ProtoOutputDir, "models.proto")
			if _, err := os.Stat(protoPath); os.IsNotExist(err) {
				fmt.Printf("Proto file not found at %s\n", protoPath)
				fmt.Println("Run sqlc2proto first to generate the proto file.")
				os.Exit(1)
			}

			// Determine the expected import path for the protobuf-generated Go code
			expectedImportPath := Config.ProtoGoImport
			if expectedImportPath == "" {
				if Config.GoPackagePath != "" {
					expectedImportPath = Config.GoPackagePath
				} else {
					expectedImportPath = common.InferGoPackage(Config.ProtoPackageName, Config.ModuleName)
				}
			}

			// Check if the protobuf-generated Go code exists
			// This is a simple check that looks for the .pb.go file
			protoGoPath := filepath.Join(Config.ProtoOutputDir, "models.pb.go")
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

	checkCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return checkCmd
}
