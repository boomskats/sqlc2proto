package main

import (
	"github.com/boomskats/sqlc2proto/cmd/sqlc2proto/commands"
)

var (
	version = "dev" // will be set during build
)

func main() {
	// Set the version
	commands.Version = version

	// Execute the root command
	commands.Execute()
}
