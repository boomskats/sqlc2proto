# sqlc2proto

`sqlc2proto` is a Go CLI to automatically generate Protocol Buffer definitions and mapping helpers from [sqlc](https://github.com/sqlc-dev/sqlc)-generated Go structs, with a focus on Connect-RPC compatibility.

## What it does

- Generates protobuf messages and services from sqlc-generated Go types and queries
- Maps Go types to appropriate Protocol Buffer types, with support for:
  - Standard Go types
  - PostgreSQL-specific types (Date, Timestamptz, Numeric, etc.)
  - Nullable types (sql.NullString, sql.NullInt32, etc.)
  - Binary data ([]byte → bytes)
  - UUID types
  - JSON data
  - Array types
- Generates helper functions to convert between sqlc and protobuf types

It also allows you to specify a subset of models and queries to codegen for, so you can codegen incrementally and avoid context bloat.

## Installation

```bash
go install github.com/boomskats/sqlc2proto@latest
```

### Requirements

- Go 1.21 or higher, tested on 1.24
- [sqlc](https://github.com/sqlc-dev/sqlc) for generating Go code from SQL
- [buf](https://github.com/bufbuild/buf) (optional, for generating Go code from Protocol Buffers)

## Quick Start

1. Initialize a configuration file:

```bash
sqlc2proto init
```

2. (Optional) Generate a template file for selecting models and queries:

```bash
sqlc2proto getincludes
```

3. (Optional) Edit the generated `sqlc2proto.includes.yaml` file to select which models and queries to include.

4. Generate Protocol Buffer definitions:

```bash
sqlc2proto generate
```

## Configuration

`sqlc2proto` can be configured using a YAML file (`sqlc2proto.yaml`):

```yaml
# Directory containing sqlc-generated files
sqlcDir: "./db/sqlc"

# Directory to output .proto files
protoDir: "./proto/gen"

# Package name for proto files
protoPackage: "api.v1"

# Go package path for generated proto code
goPackage: "github.com/yourusername/yourproject/proto"

# Generate conversion functions between sqlc and proto types
withMappers: true

# Module name for import paths
moduleName: "github.com/yourusername/yourproject"

# Import path for protobuf-generated Go code
protoGoImport: "github.com/yourusername/yourproject/proto"

# Field naming style: "json", "snake_case", or "original"
fieldStyle: "json"

# Path to file specifying which models and queries to include
includeFile: "sqlc2proto.includes.yaml"

# Custom type mappings
typeMappings:
  "CustomType": "string"
  "time.Time": "string"  # Override default
  "uuid.UUID": "bytes"   # Use bytes instead of string for UUIDs

# Custom nullable type mappings
nullableTypeMappings:
  "sql.NullString": "google.protobuf.StringValue"  # Use wrapper types
  "sql.NullInt64": "google.protobuf.Int64Value"
  "uuid.NullUUID": "bytes"
```

## Field Naming Styles

sqlc2proto supports three field naming styles:

1. **json** (default): Uses JSON tag names from sqlc structs
   - Example: `UserID` with `json:"user_id"` becomes `user_id` in protobuf

2. **snake_case**: Converts Go field names to snake_case
   - Example: `UserID` becomes `user_id` in protobuf

3. **original**: Preserves original Go field names
   - Example: `UserID` remains `UserID` in protobuf

Set your preference in the config file:
```yaml
fieldStyle: "json"  # or "snake_case" or "original"
```

Or use the command line flag:
```bash
sqlc2proto --field-style=json
```

## Selective Generation with Includes

sqlc2proto allows you to selectively generate Protocol Buffer definitions for specific models and queries:

1. Generate an includes template file:
```bash
sqlc2proto getincludes
```

2. Edit the generated `sqlc2proto.includes.yaml` file to select which models and queries to include:
```yaml
models:
- Book  # Include this model
# - Loan  # Exclude this model (commented out)
- Member  # Include this model

queries:
- GetBook  # Include this query
# - CreateLoan  # Exclude this query (commented out)
- ListBooks  # Include this query
```

3. Generate Protocol Buffer definitions for only the selected models and queries:
```bash
sqlc2proto generate
```

**Dependency Resolution**: Models used by included queries are automatically included, even if not explicitly selected. Use `--verbose` to see which models are included due to dependencies.

## Service Configuration

```yaml
# Service generation options
withServices: true

# Service naming strategy:
# - "entity": Group by entity (BookService, AuthorService)
# - "flat": One service for all methods
# - "custom": Custom naming
serviceNaming: "entity"

# Optional prefix/suffix for service names
servicePrefix: "API"  # Optional
serviceSuffix: "Service"  # Default
```

## Streaming Support

```yaml
serviceOptions:
  # Enable streaming for list methods (methods starting with "List")
  enableStreaming: true
```

When enabled, list methods are generated as server streaming RPCs:

```protobuf
// Without streaming
rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);

// With streaming
rpc ListBooks(ListBooksRequest) returns (stream Book);
```

## Pagination

```yaml
serviceOptions:
  # Add pagination fields to list methods
  includePagination: true
  
  # Customize field names
  pageSizeField: "limit"
  pageTokenField: "page_token"
  nextPageTokenField: "next_page_token"
  totalSizeField: "total_size"
```

## Command Line Usage

### Initialize Configuration

```bash
sqlc2proto init [--output=path/to/config.yaml]
```

### Generate Includes Template

```bash
sqlc2proto getincludes [flags]
```

Flags:
- `--output`: Output file path (default: from config or sqlc2proto.includes.yaml)
- `--force`: Overwrite existing file without confirmation
- `--verbose`: Enable verbose output

### Generate Protocol Buffers

```bash
sqlc2proto generate [flags]
```

Flags:
- `--sqlc-dir`: Directory containing sqlc-generated files
- `--proto-dir`: Directory to output .proto files
- `--package`: Package name for proto files
- `--go-package`: Go package path for generated proto code
- `--module`: Module name for import paths
- `--proto-go-import`: Import path for protobuf-generated Go code
- `--with-mappers`: Generate conversion functions
- `--field-style`: Field naming style ('json', 'snake_case', or 'original')
- `--include-file`: Path to file specifying which models and queries to include
- `--dry-run`: Show what would be generated without writing files
- `--verbose`: Enable verbose output

### Command-Line Examples

```bash
# Basic generation
sqlc2proto generate

# Custom directories and package
sqlc2proto generate --sqlc-dir=./internal/db --package=myapi.v1

# Selective generation with dependencies
sqlc2proto generate --include-file=api-includes.yaml --verbose

# Preview without writing files
sqlc2proto generate --dry-run --verbose

# Original Go field names in Proto
sqlc2proto generate --field-style=original
```

## Type Mappings

`sqlc2proto` automatically maps Go types from sqlc to appropriate Protocol Buffer types:

### Basic Types

| Go Type | Protocol Buffer Type |
|---------|---------------------|
| `string` | `string` |
| `int` | `int32` |
| `int16` | `int32` |
| `int32` | `int32` |
| `int64` | `int64` |
| `float32` | `float` |
| `float64` | `double` |
| `bool` | `bool` |
| `[]byte` | `bytes` |
| `time.Time` | `google.protobuf.Timestamp` |

### PostgreSQL-specific Types

| Go Type | Protocol Buffer Type |
|---------|---------------------|
| `pgtype.Date` | `google.protobuf.Timestamp` |
| `pgtype.Timestamptz` | `google.protobuf.Timestamp` |
| `pgtype.Text` | `string` |
| `pgtype.Numeric` | `string` |
| `pgtype.Interval` | `int64` |

### Nullable Types

| Go Type | Protocol Buffer Type |
|---------|---------------------|
| `sql.NullString` | `string` (optional) |
| `sql.NullInt64` | `int64` (optional) |
| `sql.NullBool` | `bool` (optional) |
| `sql.NullTime` | `google.protobuf.Timestamp` (optional) |
| `uuid.NullUUID` | `string` (optional) |

### Array Types

Array types map to repeated fields:
```go
[]string → repeated string
[]int32 → repeated int32
```

Special case: `[]byte` maps to `bytes` (not repeated), which is idiomatic in Protocol Buffers.

### Custom Type Conversions

For complex custom types, add the mapping in your config:

```yaml
typeMappings:
  "github.com/shopspring/decimal.Decimal": "string"
```

The tool will generate appropriate conversion functions in the mappers file.

## API Versioning Strategies

Create separate configurations for different API versions:

```bash
# Generate v1 API
sqlc2proto generate --config=api/v1/sqlc2proto.yaml

# Generate v2 API 
sqlc2proto generate --config=api/v2/sqlc2proto.yaml
```

Or use different package names:

```bash
sqlc2proto generate --include-file=includes.yaml --package=api.v1
sqlc2proto generate --include-file=includes.yaml --package=api.v2
```

## Complete Workflow Example

```bash
# 1. Generate Go code from SQL
sqlc generate

# 2. Create an includes template
sqlc2proto getincludes

# 3. Edit includes.yaml to select models/queries

# 4. Generate Protocol Buffers
sqlc2proto generate

# 5. Generate Go code from Protocol Buffers
buf generate

# 6. Use mappers to convert between types:
# db.Book → proto.Book: BookToProto(dbBook)
# proto.Book → db.Book: BookFromProto(protoBook)
```

## License

MIT