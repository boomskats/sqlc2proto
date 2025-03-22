package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/boomskats/sqlc-proto/internal/parser"
	"gopkg.in/yaml.v3"

	"github.com/iancoleman/strcase"
)

// Config holds the configuration for code generation
type Config struct {
	SQLCDir          string            `yaml:"sqlcDir"`
	ProtoOutputDir   string            `yaml:"protoDir"`
	ProtoPackageName string            `yaml:"protoPackage"`
	GoPackagePath    string            `yaml:"goPackage"`
	GenerateMappers  bool              `yaml:"withMappers"`
	TypeMappings     map[string]string `yaml:"typeMappings"`
}

// SaveConfig saves the configuration to a YAML file
func SaveConfig(config Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func init() {
	// Register custom template functions directly using strcase
	template.New("").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	})
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(data, &config)
	return config, err
}

// GenerateProtoFile generates a .proto file from message definitions
func GenerateProtoFile(messages []parser.ProtoMessage, config Config, outputPath string) error {
	// Load template
	tmplContent, err := loadTemplate("proto.tmpl")
	if err != nil {
		// Fall back to embedded template
		tmplContent = defaultProtoTemplate
	}

	tmpl, err := template.New("proto").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	}).Parse(tmplContent)
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
		Messages:      messages,
		PackageName:   config.ProtoPackageName,
		GoPackagePath: config.GoPackagePath,
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
func GenerateMapperFile(messages []parser.ProtoMessage, config Config, outputPath string) error {
	// Load template
	tmplContent, err := loadTemplate("mapper.tmpl")
	if err != nil {
		// Fall back to embedded template
		tmplContent = defaultMapperTemplate
	}

	tmpl, err := template.New("mapper").Funcs(template.FuncMap{
		"camelCase":  strcase.ToLowerCamel,
		"pascalCase": strcase.ToCamel,
		"snakeCase":  strcase.ToSnake,
	}).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create template data
	data := struct {
		Messages     []parser.ProtoMessage
		PackageName  string
		ProtoPackage string
		ProtoImport  string
		HasTimestamp bool
	}{
		Messages:     messages,
		PackageName:  filepath.Base(config.ProtoOutputDir),
		ProtoPackage: config.ProtoPackageName,
		ProtoImport:  config.GoPackagePath,
	}

	// Check if any message uses Timestamp
	for _, msg := range messages {
		for _, field := range msg.Fields {
			if field.Type == "google.protobuf.Timestamp" {
				data.HasTimestamp = true
				break
			}
		}
		if data.HasTimestamp {
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

// loadTemplate loads a template file from the templates directory
func loadTemplate(name string) (string, error) {
	// First try the executable directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		templatePath := filepath.Join(exeDir, "templates", name)
		if data, err := os.ReadFile(templatePath); err == nil {
			return string(data), nil
		}
	}

	// Try current directory
	for _, dir := range []string{
		"templates",
		"../templates",
		"../../templates",
	} {
		templatePath := filepath.Join(dir, name)
		if data, err := os.ReadFile(templatePath); err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("template not found: %s", name)
}

// Default templates embedded in the binary
const defaultProtoTemplate = `syntax = "proto3";

package {{ .PackageName }};

option go_package = "{{ .GoPackagePath }}";

{{ if .HasTimestampMsg }}import "google/protobuf/timestamp.proto";{{ end }}
{{ range .Messages }}{{ if not (eq .Name "Queries") }}
{{ if .Comments }}// {{ .Comments }}{{ end }}
message {{ .Name }} {
  {{- range $i, $field := .Fields }}
  {{ if $field.Comment }}// {{ $field.Comment }}{{ end }}
  {{ if $field.IsRepeated }}repeated {{ end }}{{ $field.Type }} {{ $field.Name }} = {{ $field.Number }}{{ if $field.JSONName }} [json_name="{{ $field.JSONName }}"]{{ end }};
  {{- end }}
}
{{ end }}{{ end }}
`

const defaultMapperTemplate = `// Code generated by sqlc-proto; DO NOT EDIT.
package {{ .PackageName }}

import (
    "database/sql"
    {{ if .HasTimestamp }}
    "time"
    "google.golang.org/protobuf/types/known/timestamppb"
    "github.com/jackc/pgx/v5/pgtype"
    {{ end }}

    "{{ .ProtoImport }}"
    db "github.com/yourusername/yourproject/db/sqlc" // Replace with your actual SQLC path
)
{{ range .Messages }}{{ if not (eq .Name "Queries") }}

// ToProto converts a DB {{ .SQLCStruct }} to a Proto {{ .Name }}
func {{ .SQLCStruct }}ToProto(in *db.{{ .SQLCStruct }}) *{{ .ProtoPackage }}.{{ .Name }} {
    if in == nil {
        return nil
    }
    
    return &{{ .ProtoPackage }}.{{ .Name }}{
        {{- range .Fields }}
        {{ pascalCase .Name }}: {{ .ConversionCode }},
        {{- end }}
    }
}

// FromProto converts a Proto {{ .Name }} to a DB {{ .SQLCStruct }}
func {{ .SQLCStruct }}FromProto(in *{{ .ProtoPackage }}.{{ .Name }}) *db.{{ .SQLCStruct }} {
    if in == nil {
        return nil
    }
    
    return &db.{{ .SQLCStruct }}{
        {{- range .Fields }}
        {{ .SQLCName }}: {{ .ReverseConversionCode }},
        {{- end }}
    }
}
{{ end }}{{ end }}
`
