# Library Example

This example demonstrates how to use sqlc-proto to generate Protocol Buffer definitions and mappers from sqlc-generated Go code.

## Workflow

The complete workflow involves several steps:

1. Define your SQL schema in `schema.sql`
2. Define your SQL queries in `query.sql`
3. Configure sqlc in `sqlc.yaml`
4. Run sqlc to generate Go code: `sqlc generate`
5. Configure sqlc-proto in `sqlc-proto.yaml`
6. Run sqlc-proto to generate Proto definitions and mappers: `sqlc-proto generate`
7. Configure buf in `buf.yaml` and `buf.gen.yaml`
8. Run buf to generate Go code from Proto definitions: `buf generate`
9. Verify the imports: `sqlc-proto verify-imports`

## Configuration Files

### sqlc.yaml

This configures sqlc to generate Go code from your SQL schema and queries.

### sqlc-proto.yaml

This configures sqlc-proto to generate Proto definitions and mappers from the sqlc-generated Go code.

Key options:
- `sqlcDir`: Directory containing sqlc-generated Go code
- `protoDir`: Directory where Proto definitions will be generated
- `protoPackage`: Package name for Proto definitions
- `withMappers`: Whether to generate mapper functions
- `moduleName`: Module name for import paths
- `protoGoImport`: Import path for the protobuf-generated Go code (should match the output of buf generate)

### buf.yaml

This configures buf to lint and break Proto definitions.

### buf.gen.yaml

This configures buf to generate Go code from Proto definitions.

## Generated Files

- `db/sqlc/*.go`: Go code generated by sqlc
- `proto/models.proto`: Proto definitions generated by sqlc-proto
- `proto/mappers/mappers.go`: Mapper functions generated by sqlc-proto
- `proto/*.pb.go`: Go code generated by buf

## Important Note

The mapper functions in `proto/mappers/mappers.go` import the Go code generated by buf. If you see import errors, make sure to run `buf generate` after running sqlc-proto.

You can use the `verify-imports` command to check if the imports are correct:

```bash
go run ../../cmd/sqlc-proto/main.go verify-imports
```

This command will check if the protobuf-generated Go code exists and can be imported.
