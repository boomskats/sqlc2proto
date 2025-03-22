package commands

import (
	"fmt"
	"os"

	"github.com/boomskats/sqlc2proto/internal/generator"
	"github.com/spf13/cobra"
)

// customHelpTemplate is a custom help template that displays commands in our desired order
const customHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// customCommandTemplate is a custom template for displaying commands in a specific order
const customCommandTemplate = `{{if .HasAvailableSubCommands}}
Available Commands:
  {{- if (eq .Name "sqlc2proto")}}
  help        Help about any command
  init        Initialize a new sqlc2proto configuration file
  generate    Generate Protocol Buffers from sqlc structs
  check       Check the generated files for correctness
  completion  Generate the autocompletion script for the specified shell
  {{- else}}
  {{- range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}
  {{- end}}{{end}}
  {{- end}}

{{end}}`

var (
	// Version will be set during build
	Version = "dev"

	// Config holds the global configuration
	Config = generator.Config{
		SQLCDir:          "./db/sqlc",
		ProtoOutputDir:   "./proto/gen",
		ProtoPackageName: "api.v1",
		GoPackagePath:    "",
		GenerateMappers:  false,
		ModuleName:       "",
		ProtoGoImport:    "",
	}
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sqlc2proto",
		Short: "Generate Protocol Buffers from sqlc structs",
		Long: `sqlc2proto automatically generates Protocol Buffer definitions
from sqlc-generated Go structs, with a focus on Connect-RPC compatibility.

It maps Go types to appropriate Protocol Buffer types and can also generate
Go code for converting between sqlc models and protobuf messages.

Example:
	 sqlc2proto generate --sqlc-dir=./db/sqlc --proto-dir=./proto --package=api.v1 --with-mappers
`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			// Just display help information by default
			cmd.Help()
		},
	}

	// Add global flags to the root command
	rootCmd.PersistentFlags().String("config", "", "Path to configuration file (default: sqlc2proto.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Create commands
	helpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type ` + rootCmd.Name() + ` help [path to command] for full details.`,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("Unknown help topic %#q\n", args)
				c.Root().Usage()
			} else {
				cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
				cmd.Help()
			}
		},
	}
	initCmd := NewInitCmd()
	generateCmd := NewGenerateCmd()
	checkCmd := NewCheckCmd()

	// Add commands to root in the order we want them to appear
	rootCmd.AddCommand(helpCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)

	// Set custom help template
	rootCmd.SetHelpTemplate(customHelpTemplate)
	rootCmd.SetUsageTemplate(customCommandTemplate)

	// Override the default help command with our custom one
	rootCmd.SetHelpCommand(helpCmd)

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
