// API Response Types

export type SourceStatus = "inactive" | "active" | "error"
export type PipelineStatus = "stopped" | "starting" | "running" | "stopping" | "error"
export type HealthStatus = "healthy" | "unhealthy" | "degraded" | "unknown"

export interface Source {
  id: string
  name: string
  type: "postgresql"
  host: string
  port: number
  database_name: string
  username: string
  ssl_mode: string
  status: SourceStatus
  created_at: string
  updated_at: string
}

export interface CreateSourceInput {
  name: string
  type: "postgresql"
  host: string
  port: number
  database_name: string
  username: string
  password: string
  ssl_mode?: string
}

export interface TableMapping {
  id: string
  source_table: string
  target_table: string
  primary_key_columns: string[]
  excluded_columns?: string[]
}

export interface Pipeline {
  id: string
  name: string
  source_id: string
  status: PipelineStatus
  config: Record<string, unknown>
  error_message?: string
  tables: TableMapping[]
  created_at: string
  updated_at: string
  started_at?: string
  stopped_at?: string
}

export interface CreatePipelineInput {
  name: string
  source_id: string
  config?: Record<string, unknown>
}

export interface ComponentHealth {
  name: string
  status: HealthStatus
  message?: string
  duration_ms?: number
  last_check?: string
  error?: string
}

export interface HealthResponse {
  status: HealthStatus
  components: Record<string, ComponentHealth>
  timestamp: string
}

export interface ApiError {
  type: string
  title: string
  status: number
  detail: string
  errors?: Array<{
    field: string
    message: string
  }>
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}
