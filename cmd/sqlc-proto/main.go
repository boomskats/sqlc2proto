package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/boomskats/sqlc-proto/internal/generator"
    "github.com/boomskats/sqlc-proto/internal/parser"
)

func main() {
    config := generator.Config{
        SQLCDir:          "./db/sqlc",
        ProtoOutputDir:   "./proto/gen",
        ProtoPackageName: "api.v1",
        GoPackagePath:    "",
        GenerateMappers:  false,
    }

    configFile := "sqlc-proto.yaml"

    rootCmd := &cobra.Command{
        Use:   "sqlc-proto",
        Short: "Generate Protocol Buffers from sqlc structs",
        Long: `sqlc-proto automatically generates Protocol Buffer definitions from sqlc-generated Go structs.
It can also generate functions to convert between sqlc and protobuf types.`,
        Run: func(cmd *cobra.Command, args []string) {
            // Try to load config file if it exists
            if _, err := os.Stat(configFile); err == nil {
                if cfg, err := generator.LoadConfig(configFile); err == nil {
                    // Override with config file values
                    if cfg.SQLCDir != "" {
                        config.SQLCDir = cfg.SQLCDir
                    }
                    if cfg.ProtoOutputDir != "" {
                        config.ProtoOutputDir = cfg.ProtoOutputDir
                    }
                    if cfg.ProtoPackageName != "" {
                        config.ProtoPackageName = cfg.ProtoPackageName
                    }
                    if cfg.GoPackagePath != "" {
                        config.GoPackagePath = cfg.GoPackagePath
                    }
                    if cfg.GenerateMappers {
                        config.GenerateMappers = true
                    }
                    if len(cfg.TypeMappings) > 0 {
                        parser.AddCustomTypeMappings(cfg.TypeMappings)
                    }
                }
            }

            // If go package is still empty, infer from proto package
            if config.GoPackagePath == "" {
                config.GoPackagePath = "github.com/yourusername/yourproject/gen/" + config.ProtoPackageName
            }

            // Ensure output directory exists
            if err := os.MkdirAll(config.ProtoOutputDir, 0755); err != nil {
                fmt.Printf("Failed to create output directory: %v\n", err)
                os.Exit(1)
            }

            // Process sqlc directory
            messages, err := parser.ProcessSQLCDirectory(config.SQLCDir)
            if err != nil {
                fmt.Printf("Failed to process sqlc directory: %v\n", err)
                os.Exit(1)
            }

            // Generate proto file
            protoPath := filepath.Join(config.ProtoOutputDir, "models.proto")
            if err := generator.GenerateProtoFile(messages, config, protoPath); err != nil {
                fmt.Printf("Failed to generate proto file: %v\n", err)
                os.Exit(1)
            }

            fmt.Printf("Successfully generated Protobuf definitions in %s\n", protoPath)

            // Generate mapper file if requested
            if config.GenerateMappers {
                mapperPath := filepath.Join(config.ProtoOutputDir, "mappers.go")
                if err := generator.GenerateMapperFile(messages, config, mapperPath); err != nil {
                    fmt.Printf("Failed to generate mapper file: %v\n", err)
                    os.Exit(1)
                }
                fmt.Printf("Successfully generated mapper functions in %s\n", mapperPath)
            }
        },
    }

    rootCmd.Flags().StringVar(&config.SQLCDir, "sqlc-dir", config.SQLCDir, "Directory containing sqlc-generated files")
    rootCmd.Flags().StringVar(&config.ProtoOutputDir, "proto-dir", config.ProtoOutputDir, "Directory to output .proto files")
    rootCmd.Flags().StringVar(&config.ProtoPackageName, "package", config.ProtoPackageName, "Package name for proto files")
    rootCmd.Flags().StringVar(&config.GoPackagePath, "go-package", config.GoPackagePath, "Go package path for generated proto code")
    rootCmd.Flags().BoolVar(&config.GenerateMappers, "with-mappers", config.GenerateMappers, "Generate conversion functions between sqlc and proto types")
    rootCmd.Flags().StringVar(&configFile, "config", configFile, "Path to configuration file")

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
