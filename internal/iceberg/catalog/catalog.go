// Package catalog provides Iceberg catalog operations.
package catalog

import (
	"context"

	"github.com/janovincze/philotes/internal/iceberg"
)

// Catalog defines the interface for Iceberg catalog operations.
type Catalog interface {
	// EnsureWarehouse ensures the warehouse exists in the catalog, creating it if necessary.
	EnsureWarehouse(ctx context.Context) error

	// CreateNamespace creates a new namespace if it doesn't exist.
	CreateNamespace(ctx context.Context, namespace string, properties map[string]string) error

	// NamespaceExists checks if a namespace exists.
	NamespaceExists(ctx context.Context, namespace string) (bool, error)

	// CreateTable creates a new Iceberg table.
	CreateTable(ctx context.Context, namespace, table string, schema iceberg.Schema, partitionSpec iceberg.PartitionSpec) error

	// TableExists checks if a table exists.
	TableExists(ctx context.Context, namespace, table string) (bool, error)

	// LoadTable loads table metadata.
	LoadTable(ctx context.Context, namespace, table string) (*iceberg.TableMetadata, error)

	// CommitSnapshot commits a new snapshot to the table.
	CommitSnapshot(ctx context.Context, namespace, table string, dataFiles []iceberg.DataFile) error

	// Close releases any resources held by the catalog.
	Close() error
}

// StorageProfile holds S3-compatible storage configuration for warehouse creation.
type StorageProfile struct {
	// Bucket is the S3 bucket name.
	Bucket string
	// KeyPrefix is an optional path prefix within the bucket.
	KeyPrefix string
	// Region is the S3 region.
	Region string
	// Endpoint is the S3/MinIO endpoint URL.
	Endpoint string
	// AccessKey is the S3 access key.
	AccessKey string
	// SecretKey is the S3 secret key.
	SecretKey string
}

// Config holds catalog configuration.
type Config struct {
	// CatalogURL is the REST catalog endpoint URL.
	CatalogURL string

	// Warehouse is the warehouse name/prefix.
	Warehouse string

	// Credentials for authentication (optional).
	Token string

	// Storage holds S3-compatible storage configuration for warehouse bootstrap.
	Storage StorageProfile
}
