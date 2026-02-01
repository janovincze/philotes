// Package catalog provides Iceberg catalog operations.
package catalog

import (
	"context"

	"github.com/janovincze/philotes/internal/iceberg"
)

// Catalog defines the interface for Iceberg catalog operations.
type Catalog interface {
	// EnsureWarehouse ensures the warehouse exists, creating it if necessary.
	EnsureWarehouse(ctx context.Context) error

	// WarehouseExists checks if the configured warehouse exists.
	WarehouseExists(ctx context.Context) (bool, error)

	// ListWarehouses lists all available warehouses.
	ListWarehouses(ctx context.Context) ([]Warehouse, error)

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

// Warehouse represents an Iceberg warehouse configuration.
type Warehouse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	ProjectID string            `json:"project_id"`
	Status    string            `json:"status"`
	Location  string            `json:"location,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// Config holds catalog configuration.
type Config struct {
	// CatalogURL is the REST catalog endpoint URL.
	CatalogURL string

	// Warehouse is the warehouse name/prefix.
	Warehouse string

	// ProjectID is the Lakekeeper project ID for warehouse management.
	ProjectID string

	// Credentials for authentication (optional).
	Token string

	// Storage configuration for warehouse creation.
	Storage StorageConfig
}

// StorageConfig holds S3/MinIO storage configuration.
type StorageConfig struct {
	// Type is the storage type (s3, azure, gcs).
	Type string

	// Bucket is the S3 bucket name.
	Bucket string

	// Endpoint is the S3-compatible endpoint (for MinIO).
	Endpoint string

	// Region is the AWS region.
	Region string

	// PathStyleAccess enables path-style access for MinIO.
	PathStyleAccess bool

	// AccessKeyID is the AWS access key.
	AccessKeyID string

	// SecretAccessKey is the AWS secret key.
	SecretAccessKey string
}
