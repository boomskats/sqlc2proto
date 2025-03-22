Based on the provided code from the sqlc project, here are the default mappings from PostgreSQL types to Go types:

## Basic Types
- `bit`, `bit varying`: `[]byte`
- `boolean`: `bool`
- `smallint`, `smallserial`, `int2`: `int16`
- `integer`, `serial`, `int`, `int4`: `int32`
- `bigint`, `bigserial`, `int8`: `int64`
- `real`, `float4`: `float32`
- `double precision`, `float8`: `float64`
- `numeric`, `decimal`: Custom `pgtype.Numeric` or `string` based on configuration
- `money`: `string`
- `text`: `string`
- `citext`: `string`
- `varchar`, `character varying`: `string`
- `character`, `char`: `string`
- `bytea`: `[]byte`
- `timestamp`, `timestamptz`: `time.Time`
- `date`: `time.Time`
- `time`, `timetz`: `time.Time`
- `interval`: `pgtype.Interval`
- `blob`: `[]byte`
- `json`, `jsonb`: `json.RawMessage` or custom types based on configuration
- `uuid`: `uuid.UUID` (from github.com/google/uuid) or `string` based on configuration
- `inet`: `net.IP` or `netip.Addr` based on configuration
- `macaddr`: `net.HardwareAddr` or custom type based on configuration
- `ltree`: `string`
- `void`: Empty struct `struct{}`
- `any`: `any` (Go's empty interface)
- `name`: `string`
- `"char"`: `pgtype.QChar`

## Array Types
- All array types map to slices of their respective element types
  - Example: `integer[]` → `[]int32`
  - Example: `text[]` → `[]string`
