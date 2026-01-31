// Package models provides request and response types for the API.
package models

import "time"

// QueryLayerStatus represents the status of the query layer.
type QueryLayerStatus struct {
	Available      bool             `json:"available"`
	TrinoVersion   string           `json:"trino_version,omitempty"`
	CoordinatorURL string           `json:"coordinator_url,omitempty"`
	Uptime         string           `json:"uptime,omitempty"`
	NodeCount      int              `json:"node_count,omitempty"`
	RunningQueries int              `json:"running_queries,omitempty"`
	QueuedQueries  int              `json:"queued_queries,omitempty"`
	BlockedQueries int              `json:"blocked_queries,omitempty"`
	ActiveWorkers  int              `json:"active_workers,omitempty"`
	MemoryUsage    *MemoryUsageInfo `json:"memory_usage,omitempty"`
	Error          string           `json:"error,omitempty"`
	CheckedAt      time.Time        `json:"checked_at"`
}

// MemoryUsageInfo represents memory usage statistics.
type MemoryUsageInfo struct {
	TotalBytes    int64   `json:"total_bytes"`
	ReservedBytes int64   `json:"reserved_bytes"`
	UsedBytes     int64   `json:"used_bytes"`
	UsagePercent  float64 `json:"usage_percent"`
}

// TrinoCatalog represents a Trino catalog.
type TrinoCatalog struct {
	Name          string `json:"name"`
	ConnectorName string `json:"connector_name,omitempty"`
}

// CatalogListResponse represents the response for listing catalogs.
type CatalogListResponse struct {
	Catalogs []TrinoCatalog `json:"catalogs"`
	Total    int            `json:"total"`
}

// TrinoSchema represents a schema within a catalog.
type TrinoSchema struct {
	Name    string `json:"name"`
	Catalog string `json:"catalog"`
}

// SchemaListResponse represents the response for listing schemas.
type SchemaListResponse struct {
	Schemas []TrinoSchema `json:"schemas"`
	Catalog string        `json:"catalog"`
	Total   int           `json:"total"`
}

// TrinoTable represents a table within a schema.
type TrinoTable struct {
	Name    string `json:"name"`
	Schema  string `json:"schema"`
	Catalog string `json:"catalog"`
	Type    string `json:"type,omitempty"` // TABLE, VIEW, etc.
}

// TableListResponse represents the response for listing tables.
type TableListResponse struct {
	Tables  []TrinoTable `json:"tables"`
	Catalog string       `json:"catalog"`
	Schema  string       `json:"schema"`
	Total   int          `json:"total"`
}

// TrinoColumn represents a column in a table.
type TrinoColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Comment  string `json:"comment,omitempty"`
}

// TableInfoResponse represents detailed table information.
type TableInfoResponse struct {
	Name       string            `json:"name"`
	Schema     string            `json:"schema"`
	Catalog    string            `json:"catalog"`
	Type       string            `json:"type"`
	Columns    []TrinoColumn     `json:"columns"`
	Comment    string            `json:"comment,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// QueryHealthResponse represents a health check response for the query layer.
type QueryHealthResponse struct {
	Status  string            `json:"status"` // healthy, unhealthy, unknown
	Message string            `json:"message,omitempty"`
	Details *QueryLayerStatus `json:"details,omitempty"`
}

// TrinoClusterInfo represents Trino cluster information from the /v1/info endpoint.
type TrinoClusterInfo struct {
	Starting    bool   `json:"starting"`
	Uptime      string `json:"uptime,omitempty"`
	NodeVersion struct {
		Version string `json:"version"`
	} `json:"nodeVersion,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// TrinoClusterStats represents Trino cluster statistics from the /v1/cluster endpoint.
type TrinoClusterStats struct {
	RunningQueries   int     `json:"runningQueries"`
	BlockedQueries   int     `json:"blockedQueries"`
	QueuedQueries    int     `json:"queuedQueries"`
	ActiveWorkers    int     `json:"activeWorkers"`
	RunningDrivers   int     `json:"runningDrivers"`
	ReservedMemory   float64 `json:"reservedMemory"`
	TotalInputRows   int64   `json:"totalInputRows"`
	TotalInputBytes  int64   `json:"totalInputBytes"`
	TotalCpuTimeSecs float64 `json:"totalCpuTimeSecs"`
}
