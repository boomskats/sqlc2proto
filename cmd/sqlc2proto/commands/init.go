package commands

import (
	"fmt"
	"os"

	"github.com/boomskats/sqlc2proto/cmd/sqlc2proto/common"
	"github.com/boomskats/sqlc2proto/internal/generator"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new sqlc2proto configuration file",
		Long: `Creates a new sqlc2proto.yaml configuration file with default values.
You can then edit this file to customize the behavior of sqlc2proto.`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if configFile == "" {
				configFile = "sqlc2proto.yaml"
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

			// Try to parse go.mod file to get module name
			moduleName, err := common.ParseGoModFile()
			if err == nil {
				// If we found a module name, use it to set GoPackagePath
				config.ModuleName = moduleName
				config.GoPackagePath = fmt.Sprintf("%s/proto", moduleName)
				if verbose {
					fmt.Printf("Found module name in go.mod: %s\n", moduleName)
					fmt.Printf("Setting GoPackagePath to: %s\n", config.GoPackagePath)
				}
			}

			// Write config file with comments
			if err := common.WriteConfigWithComments(config, configFile); err != nil {
				fmt.Printf("Failed to write config file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Created config file %s\n", configFile)
			fmt.Println("You can now edit this file to customize sqlc2proto behavior.")
		},
	}

	initCmd.Flags().StringP("output", "o", "sqlc2proto.yaml", "Path to write the config file")

	return initCmd
}
