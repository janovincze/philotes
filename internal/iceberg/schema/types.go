// Package schema provides schema conversion utilities for Iceberg.
package schema

import (
	"strings"

	"github.com/janovincze/philotes/internal/iceberg"
)

// PostgresTypeMapping maps PostgreSQL types to Iceberg types.
var PostgresTypeMapping = map[string]iceberg.Type{
	// Integer types
	"smallint":  iceberg.TypeInt,
	"int2":      iceberg.TypeInt,
	"integer":   iceberg.TypeInt,
	"int":       iceberg.TypeInt,
	"int4":      iceberg.TypeInt,
	"bigint":    iceberg.TypeLong,
	"int8":      iceberg.TypeLong,
	"serial":    iceberg.TypeInt,
	"bigserial": iceberg.TypeLong,

	// Floating point types
	"real":             iceberg.TypeFloat,
	"float4":           iceberg.TypeFloat,
	"double precision": iceberg.TypeDouble,
	"float8":           iceberg.TypeDouble,
	"numeric":          iceberg.TypeDouble,
	"decimal":          iceberg.TypeDouble,

	// Boolean
	"boolean": iceberg.TypeBoolean,
	"bool":    iceberg.TypeBoolean,

	// String types
	"text":      iceberg.TypeString,
	"varchar":   iceberg.TypeString,
	"char":      iceberg.TypeString,
	"character": iceberg.TypeString,
	"name":      iceberg.TypeString,

	// Date/Time types
	"date":                        iceberg.TypeDate,
	"time":                        iceberg.TypeTime,
	"time without time zone":      iceberg.TypeTime,
	"time with time zone":         iceberg.TypeTime,
	"timestamp":                   iceberg.TypeTimestamp,
	"timestamp without time zone": iceberg.TypeTimestamp,
	"timestamp with time zone":    iceberg.TypeTimestamp,
	"timestamptz":                 iceberg.TypeTimestamp,

	// Binary types
	"bytea": iceberg.TypeBinary,

	// UUID
	"uuid": iceberg.TypeUUID,

	// JSON types (stored as string)
	"json":  iceberg.TypeString,
	"jsonb": iceberg.TypeString,

	// Other types mapped to string
	"inet":    iceberg.TypeString,
	"cidr":    iceberg.TypeString,
	"macaddr": iceberg.TypeString,
	"oid":     iceberg.TypeLong,
}

// MapPostgresToIceberg converts a PostgreSQL type name to an Iceberg type.
func MapPostgresToIceberg(pgType string) iceberg.Type {
	// Normalize the type name (lowercase, trim whitespace)
	normalized := strings.ToLower(strings.TrimSpace(pgType))

	// Handle array types (store as string/JSON)
	if strings.HasSuffix(normalized, "[]") {
		return iceberg.TypeString
	}

	// Handle varchar(n), char(n), numeric(p,s), etc.
	if idx := strings.Index(normalized, "("); idx > 0 {
		normalized = normalized[:idx]
	}

	// Look up in mapping
	if icebergType, ok := PostgresTypeMapping[normalized]; ok {
		return icebergType
	}

	// Default to string for unknown types
	return iceberg.TypeString
}

// InferTypeFromValue attempts to infer an Iceberg type from a Go value.
func InferTypeFromValue(value any) iceberg.Type {
	if value == nil {
		return iceberg.TypeString
	}

	switch value.(type) {
	case bool:
		return iceberg.TypeBoolean
	case int, int32:
		return iceberg.TypeInt
	case int64:
		return iceberg.TypeLong
	case float32:
		return iceberg.TypeFloat
	case float64:
		return iceberg.TypeDouble
	case string:
		return iceberg.TypeString
	case []byte:
		return iceberg.TypeBinary
	default:
		// For complex types (maps, slices), use string (JSON)
		return iceberg.TypeString
	}
}
