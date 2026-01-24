// Package writer provides Iceberg table writing functionality.
package writer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/janovincze/philotes/internal/cdc"
	"github.com/janovincze/philotes/internal/cdc/buffer"
	"github.com/janovincze/philotes/internal/iceberg"
	"github.com/janovincze/philotes/internal/iceberg/catalog"
	"github.com/janovincze/philotes/internal/iceberg/schema"
)

// Writer writes CDC events to Iceberg tables.
type Writer interface {
	// WriteEvents writes a batch of events to Iceberg.
	WriteEvents(ctx context.Context, events []buffer.BufferedEvent) error

	// Close releases any resources held by the writer.
	Close() error
}

// Config holds writer configuration.
type Config struct {
	// Catalog is the catalog configuration.
	Catalog catalog.Config

	// S3 is the S3/MinIO configuration.
	S3 S3Config

	// Bucket is the S3 bucket for data files.
	Bucket string

	// WarehousePath is the base path for table data within the bucket.
	WarehousePath string

	// DefaultNamespace is the default namespace for tables.
	DefaultNamespace string
}

// IcebergWriter implements Writer for Iceberg tables.
type IcebergWriter struct {
	catalog       catalog.Catalog
	s3            *MinIOClient
	parquet       *ParquetWriter
	schemaBuilder *schema.Builder
	logger        *slog.Logger
	config        Config

	// tableSchemas caches table schemas to avoid repeated lookups.
	tableSchemas map[string]iceberg.Schema
}

// NewIcebergWriter creates a new Iceberg writer.
func NewIcebergWriter(cfg Config, logger *slog.Logger) (*IcebergWriter, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create catalog client
	cat := catalog.NewRESTCatalog(cfg.Catalog, logger)

	// Create S3 client
	s3Client, err := NewMinIOClient(cfg.S3, logger)
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	return &IcebergWriter{
		catalog:       cat,
		s3:            s3Client,
		parquet:       NewParquetWriter(),
		schemaBuilder: schema.NewBuilder(),
		logger:        logger.With("component", "iceberg-writer"),
		config:        cfg,
		tableSchemas:  make(map[string]iceberg.Schema),
	}, nil
}

// WriteEvents writes a batch of buffered events to Iceberg.
func (w *IcebergWriter) WriteEvents(ctx context.Context, events []buffer.BufferedEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Group events by table
	eventsByTable := w.groupEventsByTable(events)

	// Process each table's events
	for tableKey, tableEvents := range eventsByTable {
		if err := w.writeTableEvents(ctx, tableKey, tableEvents); err != nil {
			return fmt.Errorf("write events for %s: %w", tableKey, err)
		}
	}

	return nil
}

// groupEventsByTable groups events by their source table.
func (w *IcebergWriter) groupEventsByTable(events []buffer.BufferedEvent) map[string][]buffer.BufferedEvent {
	grouped := make(map[string][]buffer.BufferedEvent)

	for _, event := range events {
		key := event.Event.FullyQualifiedTable()
		grouped[key] = append(grouped[key], event)
	}

	return grouped
}

// writeTableEvents writes events for a single table.
func (w *IcebergWriter) writeTableEvents(ctx context.Context, tableKey string, events []buffer.BufferedEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Parse table identifier
	namespace, tableName := w.parseTableKey(tableKey)

	// Ensure table exists with appropriate schema
	if err := w.ensureTable(ctx, namespace, tableName, events); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	// Write events to Parquet file
	result, err := w.parquet.WriteEvents(events)
	if err != nil {
		return fmt.Errorf("write parquet: %w", err)
	}

	// Determine the data path
	basePath := w.getTableDataPath(namespace, tableName)

	// Upload to S3
	key := fmt.Sprintf("%s/%s", basePath, result.FileName)
	if err := w.s3.Upload(ctx, w.config.Bucket, key, bytes.NewReader(result.Data), result.FileSizeInBytes, "application/octet-stream"); err != nil {
		return fmt.Errorf("upload parquet file: %w", err)
	}

	// Create data file metadata
	dataFile := iceberg.DataFile{
		FilePath:        fmt.Sprintf("s3://%s/%s", w.config.Bucket, key),
		FileFormat:      "parquet",
		RecordCount:     result.RecordCount,
		FileSizeInBytes: result.FileSizeInBytes,
	}

	// Commit snapshot to catalog
	if err := w.catalog.CommitSnapshot(ctx, namespace, tableName, []iceberg.DataFile{dataFile}); err != nil {
		// If commit fails, try to clean up the uploaded file
		w.logger.Warn("snapshot commit failed, cleaning up file",
			"error", err,
			"file", key,
		)
		_ = w.s3.Delete(ctx, w.config.Bucket, key)
		return fmt.Errorf("commit snapshot: %w", err)
	}

	w.logger.Info("events written to Iceberg",
		"table", tableKey,
		"records", result.RecordCount,
		"file_size", result.FileSizeInBytes,
	)

	return nil
}

// ensureTable ensures the table exists, creating it if necessary.
func (w *IcebergWriter) ensureTable(ctx context.Context, namespace, tableName string, events []buffer.BufferedEvent) error {
	tableKey := namespace + "." + tableName

	// Check if we have a cached schema
	if _, exists := w.tableSchemas[tableKey]; exists {
		return nil
	}

	// Check if table exists in catalog
	exists, err := w.catalog.TableExists(ctx, namespace, tableName)
	if err != nil {
		return fmt.Errorf("check table exists: %w", err)
	}

	if exists {
		// Load and cache the schema
		meta, err := w.catalog.LoadTable(ctx, namespace, tableName)
		if err != nil {
			return fmt.Errorf("load table metadata: %w", err)
		}
		if len(meta.Schemas) > 0 {
			w.tableSchemas[tableKey] = meta.Schemas[meta.CurrentSchemaID]
		}
		return nil
	}

	// Build schema from events
	cdcEvents := make([]cdc.Event, len(events))
	for i, e := range events {
		cdcEvents[i] = e.Event
	}
	tableSchema := w.schemaBuilder.BuildFromEvents(cdcEvents)

	// Create partition spec
	partitionSpec := schema.DefaultPartitionSpec(tableSchema)

	// Ensure bucket exists
	if err := w.s3.EnsureBucket(ctx, w.config.Bucket); err != nil {
		return fmt.Errorf("ensure bucket: %w", err)
	}

	// Create table
	if err := w.catalog.CreateTable(ctx, namespace, tableName, tableSchema, partitionSpec); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Cache the schema
	w.tableSchemas[tableKey] = tableSchema

	w.logger.Info("table created",
		"namespace", namespace,
		"table", tableName,
		"columns", len(tableSchema.Fields),
	)

	return nil
}

// parseTableKey parses a table key (schema.table) into namespace and table name.
func (w *IcebergWriter) parseTableKey(tableKey string) (namespace, tableName string) {
	// Use the source schema as the namespace, or default namespace if not specified
	parts := strings.SplitN(tableKey, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return w.config.DefaultNamespace, tableKey
}

// getTableDataPath returns the data path for a table.
func (w *IcebergWriter) getTableDataPath(namespace, tableName string) string {
	return fmt.Sprintf("%s/%s/%s/data", w.config.WarehousePath, namespace, tableName)
}

// Close releases resources.
func (w *IcebergWriter) Close() error {
	if err := w.catalog.Close(); err != nil {
		return fmt.Errorf("close catalog: %w", err)
	}
	return nil
}

// Ensure IcebergWriter implements Writer.
var _ Writer = (*IcebergWriter)(nil)
