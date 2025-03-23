package generator

import (
	"bytes"
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

// Template content from embedded files
//
//go:embed proto.tmpl
var protoTemplate string

//go:embed mapper.tmpl
var mapperTemplate string

func init() {
	// Register custom template functions directly using strcase
	template.New("").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	})

}

// GenerateProtoFile generates a .proto file from message definitions
func GenerateProtoFile(messages []parser.ProtoMessage, config common.Config, outputPath string) error {
	tmpl, err := template.New("proto").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	}).Parse(protoTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Set ProtoPackage for each message
	for i := range messages {
		messages[i].ProtoPackage = config.ProtoPackageName
	}

	// Create template data
	data := struct {
		Messages        []parser.ProtoMessage
		PackageName     string
		GoPackagePath   string
		HasTimestampMsg bool
	}{
		Messages:    messages,
		PackageName: config.ProtoPackageName,
		GoPackagePath: func() string {
			// If GoPackagePath is explicitly set, use it
			if config.GoPackagePath != "" {
				return config.GoPackagePath
			}

			// Otherwise, derive it from moduleName and protoDir
			moduleName := config.ModuleName
			if moduleName == "" {
				moduleName = "github.com/boomskats/sqlc2proto"
			}

			protoDir := strings.TrimPrefix(config.ProtoOutputDir, "./")

			return filepath.Join(moduleName, protoDir)
		}(),
	}

	// Check if any message uses Timestamp
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if field.Type == "google.protobuf.Timestamp" {
				data.HasTimestampMsg = true
				break
			}
		}
		if data.HasTimestampMsg {
			break
		}
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

// GenerateMapperFile generates a Go file with conversion functions
func GenerateMapperFile(messages []parser.ProtoMessage, config common.Config, outputPath string) error {
	tmpl, err := template.New("mapper").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
	}).Parse(mapperTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create template data
	data := struct {
		Messages        []parser.ProtoMessage
		PackageName     string
		ProtoPackage    string
		ProtoImport     string
		DBImport        string
		HasTimestamp    bool
		HasPgType       bool
		HasPgConn       bool
		HelperFunctions string
	}{
		Messages:        messages,
		PackageName:     "mappers", // Use a different package name to avoid circular imports
		ProtoPackage:    config.ProtoPackageName,
		HelperFunctions: parser.GenerateHelperFunctions(messages),
		ProtoImport: func() string {
			// If ProtoGoImport is explicitly set, use it
			if config.ProtoGoImport != "" {
				return config.ProtoGoImport
			}

			// If GoPackagePath is explicitly set, use it
			if config.GoPackagePath != "" {
				return config.GoPackagePath
			}

			// Use a relative import path for the proto package
			// This assumes the proto package is in the same module as the mappers
			// and that mappers are in a subdirectory of the proto directory

			// Use a relative import path to go up one directory level
			return ".."
		}(),
		// Use the module name from config or default to github.com/boomskats/sqlc2proto
		DBImport: func() string {
			// Get module name
			moduleName := config.ModuleName
			if moduleName == "" {
				moduleName = "github.com/boomskats/sqlc2proto"
			}

			// Remove leading "./" if present in SQLCDir
			sqlcDir := strings.TrimPrefix(config.SQLCDir, "./")

			return filepath.Join(moduleName, sqlcDir)
		}(),
	}

	// Check if any message uses Timestamp, pgtype, or pgconn
	for _, msg := range messages {
		for _, field := range msg.Fields {
			// Check for Timestamp
			if field.Type == "google.protobuf.Timestamp" {
				data.HasTimestamp = true
			}

			// Check for pgtype
			if strings.HasPrefix(field.OriginalTag, "pgtype.") {
				data.HasPgType = true
			}

			// Check for pgconn
			if strings.HasPrefix(field.OriginalTag, "pgconn.") {
				data.HasPgConn = true
			}

			// If we've found all types, we can break early
			if data.HasTimestamp && data.HasPgType && data.HasPgConn {
				break
			}
		}

		// If we've found all types, we can break early
		if data.HasTimestamp && data.HasPgType && data.HasPgConn {
			break
		}
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
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write to file
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}
