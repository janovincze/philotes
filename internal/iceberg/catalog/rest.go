package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/janovincze/philotes/internal/iceberg"
)

// RESTCatalog implements Catalog using the Iceberg REST API (Lakekeeper compatible).
type RESTCatalog struct {
	config Config
	client *http.Client
	logger *slog.Logger
}

// NewRESTCatalog creates a new REST catalog client.
func NewRESTCatalog(cfg Config, logger *slog.Logger) *RESTCatalog {
	if logger == nil {
		logger = slog.Default()
	}

	return &RESTCatalog{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "iceberg-catalog"),
	}
}

// CreateNamespace creates a new namespace if it doesn't exist.
func (c *RESTCatalog) CreateNamespace(ctx context.Context, namespace string, properties map[string]string) error {
	// Check if namespace already exists
	exists, err := c.NamespaceExists(ctx, namespace)
	if err != nil {
		return err
	}
	if exists {
		c.logger.Debug("namespace already exists", "namespace", namespace)
		return nil
	}

	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces", c.config.CatalogURL, c.config.Warehouse)

	body := namespaceRequest{
		Namespace:  []string{namespace},
		Properties: properties,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("create namespace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Namespace already exists, which is fine
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.parseError(resp)
	}

	c.logger.Info("namespace created", "namespace", namespace)
	return nil
}

// NamespaceExists checks if a namespace exists.
func (c *RESTCatalog) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces/%s", c.config.CatalogURL, c.config.Warehouse, namespace)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("check namespace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, c.parseError(resp)
}

// CreateTable creates a new Iceberg table.
func (c *RESTCatalog) CreateTable(ctx context.Context, namespace, table string, schema iceberg.Schema, partitionSpec iceberg.PartitionSpec) error {
	// Ensure namespace exists first
	if err := c.CreateNamespace(ctx, namespace, nil); err != nil {
		return fmt.Errorf("ensure namespace: %w", err)
	}

	// Check if table already exists
	exists, err := c.TableExists(ctx, namespace, table)
	if err != nil {
		return err
	}
	if exists {
		c.logger.Debug("table already exists", "namespace", namespace, "table", table)
		return nil
	}

	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces/%s/tables", c.config.CatalogURL, c.config.Warehouse, namespace)

	body := createTableRequest{
		Name:          table,
		Schema:        convertSchemaToREST(schema),
		PartitionSpec: convertPartitionSpecToREST(partitionSpec),
		WriteOrder:    nil,
		StageCreate:   false,
		Properties:    map[string]string{},
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("create table request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Table already exists
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.parseError(resp)
	}

	c.logger.Info("table created", "namespace", namespace, "table", table)
	return nil
}

// TableExists checks if a table exists.
func (c *RESTCatalog) TableExists(ctx context.Context, namespace, table string) (bool, error) {
	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces/%s/tables/%s", c.config.CatalogURL, c.config.Warehouse, namespace, table)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("check table request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, c.parseError(resp)
}

// LoadTable loads table metadata.
func (c *RESTCatalog) LoadTable(ctx context.Context, namespace, table string) (*iceberg.TableMetadata, error) {
	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces/%s/tables/%s", c.config.CatalogURL, c.config.Warehouse, namespace, table)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("load table request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result loadTableResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode table response: %w", err)
	}

	return convertRESTToMetadata(result), nil
}

// CommitSnapshot commits a new snapshot with data files to the table.
func (c *RESTCatalog) CommitSnapshot(ctx context.Context, namespace, table string, dataFiles []iceberg.DataFile) error {
	url := fmt.Sprintf("%s/catalog/v1/%s/namespaces/%s/tables/%s", c.config.CatalogURL, c.config.Warehouse, namespace, table)

	// Build the append operation
	updates := []tableUpdate{
		{
			Action: "append",
			AppendFiles: &appendFilesUpdate{
				DataFiles: convertDataFilesToREST(dataFiles),
			},
		},
	}

	body := commitTableRequest{
		Requirements: []tableRequirement{},
		Updates:      updates,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("commit snapshot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.parseError(resp)
	}

	c.logger.Debug("snapshot committed", "namespace", namespace, "table", table, "files", len(dataFiles))
	return nil
}

// Close releases resources.
func (c *RESTCatalog) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// doRequest performs an HTTP request with the given method, URL, and body.
func (c *RESTCatalog) doRequest(ctx context.Context, method, url string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.Token)
	}

	return c.client.Do(req)
}

// parseError parses an error response from the REST API.
func (c *RESTCatalog) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("catalog error (status %d): failed to read response body", resp.StatusCode)
	}
	return fmt.Errorf("catalog error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

// REST API request/response types

type namespaceRequest struct {
	Namespace  []string          `json:"namespace"`
	Properties map[string]string `json:"properties,omitempty"`
}

type createTableRequest struct {
	Name          string            `json:"name"`
	Schema        restSchema        `json:"schema"`
	PartitionSpec restPartitionSpec `json:"partition-spec,omitempty"`
	WriteOrder    any               `json:"write-order,omitempty"`
	StageCreate   bool              `json:"stage-create,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
}

type restSchema struct {
	Type     string      `json:"type"`
	SchemaID int         `json:"schema-id"`
	Fields   []restField `json:"fields"`
}

type restField struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Doc      string `json:"doc,omitempty"`
}

type restPartitionSpec struct {
	SpecID int                  `json:"spec-id"`
	Fields []restPartitionField `json:"fields,omitempty"`
}

type restPartitionField struct {
	SourceID  int    `json:"source-id"`
	FieldID   int    `json:"field-id"`
	Name      string `json:"name"`
	Transform string `json:"transform"`
}

type loadTableResponse struct {
	MetadataLocation string `json:"metadata-location"`
	Metadata         struct {
		FormatVersion     int                 `json:"format-version"`
		TableUUID         string              `json:"table-uuid"`
		Location          string              `json:"location"`
		LastUpdatedMs     int64               `json:"last-updated-ms"`
		LastColumnID      int                 `json:"last-column-id"`
		Schemas           []restSchema        `json:"schemas"`
		CurrentSchemaID   int                 `json:"current-schema-id"`
		PartitionSpecs    []restPartitionSpec `json:"partition-specs"`
		DefaultSpecID     int                 `json:"default-spec-id"`
		LastPartitionID   int                 `json:"last-partition-id"`
		Properties        map[string]string   `json:"properties"`
		CurrentSnapshotID int64               `json:"current-snapshot-id"`
	} `json:"metadata"`
}

type commitTableRequest struct {
	Requirements []tableRequirement `json:"requirements"`
	Updates      []tableUpdate      `json:"updates"`
}

type tableRequirement struct {
	Type string `json:"type"`
}

type tableUpdate struct {
	Action      string             `json:"action"`
	AppendFiles *appendFilesUpdate `json:"append,omitempty"`
}

type appendFilesUpdate struct {
	DataFiles []restDataFile `json:"data-files"`
}

type restDataFile struct {
	FilePath        string         `json:"file-path"`
	FileFormat      string         `json:"file-format"`
	RecordCount     int64          `json:"record-count"`
	FileSizeInBytes int64          `json:"file-size-in-bytes"`
	Partition       map[string]any `json:"partition,omitempty"`
}

// Conversion functions

func convertSchemaToREST(schema iceberg.Schema) restSchema {
	fields := make([]restField, len(schema.Fields))
	for i, f := range schema.Fields {
		fields[i] = restField{
			ID:       f.ID,
			Name:     f.Name,
			Type:     string(f.Type),
			Required: f.Required,
			Doc:      f.Doc,
		}
	}
	return restSchema{
		Type:     "struct",
		SchemaID: schema.SchemaID,
		Fields:   fields,
	}
}

func convertPartitionSpecToREST(spec iceberg.PartitionSpec) restPartitionSpec {
	fields := make([]restPartitionField, len(spec.Fields))
	for i, f := range spec.Fields {
		fields[i] = restPartitionField{
			SourceID:  f.SourceID,
			FieldID:   f.FieldID,
			Name:      f.Name,
			Transform: f.Transform,
		}
	}
	return restPartitionSpec{
		SpecID: spec.SpecID,
		Fields: fields,
	}
}

func convertDataFilesToREST(files []iceberg.DataFile) []restDataFile {
	result := make([]restDataFile, len(files))
	for i, f := range files {
		result[i] = restDataFile{
			FilePath:        f.FilePath,
			FileFormat:      f.FileFormat,
			RecordCount:     f.RecordCount,
			FileSizeInBytes: f.FileSizeInBytes,
			Partition:       f.PartitionData,
		}
	}
	return result
}

func convertRESTToMetadata(resp loadTableResponse) *iceberg.TableMetadata {
	schemas := make([]iceberg.Schema, len(resp.Metadata.Schemas))
	for i, s := range resp.Metadata.Schemas {
		fields := make([]iceberg.Field, len(s.Fields))
		for j, f := range s.Fields {
			fields[j] = iceberg.Field{
				ID:       f.ID,
				Name:     f.Name,
				Type:     iceberg.Type(f.Type),
				Required: f.Required,
				Doc:      f.Doc,
			}
		}
		schemas[i] = iceberg.Schema{
			SchemaID: s.SchemaID,
			Fields:   fields,
		}
	}

	partitionSpecs := make([]iceberg.PartitionSpec, len(resp.Metadata.PartitionSpecs))
	for i, ps := range resp.Metadata.PartitionSpecs {
		fields := make([]iceberg.PartitionField, len(ps.Fields))
		for j, f := range ps.Fields {
			fields[j] = iceberg.PartitionField{
				SourceID:  f.SourceID,
				FieldID:   f.FieldID,
				Name:      f.Name,
				Transform: f.Transform,
			}
		}
		partitionSpecs[i] = iceberg.PartitionSpec{
			SpecID: ps.SpecID,
			Fields: fields,
		}
	}

	return &iceberg.TableMetadata{
		FormatVersion:     resp.Metadata.FormatVersion,
		TableUUID:         resp.Metadata.TableUUID,
		Location:          resp.Metadata.Location,
		LastUpdatedMs:     resp.Metadata.LastUpdatedMs,
		LastColumnID:      resp.Metadata.LastColumnID,
		Schemas:           schemas,
		CurrentSchemaID:   resp.Metadata.CurrentSchemaID,
		PartitionSpecs:    partitionSpecs,
		DefaultSpecID:     resp.Metadata.DefaultSpecID,
		LastPartitionID:   resp.Metadata.LastPartitionID,
		Properties:        resp.Metadata.Properties,
		CurrentSnapshotID: resp.Metadata.CurrentSnapshotID,
	}
}

// Ensure RESTCatalog implements Catalog interface.
var _ Catalog = (*RESTCatalog)(nil)
