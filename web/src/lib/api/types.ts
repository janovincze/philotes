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

// Table Discovery Types

export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  primary_key: boolean
  default?: string
}

export interface TableInfo {
  schema: string
  name: string
  columns: ColumnInfo[]
}

export interface TableDiscoveryResponse {
  tables: TableInfo[]
  count: number
}

export interface ConnectionTestResult {
  success: boolean
  message: string
  latency_ms?: number
  server_info?: string
  error_detail?: string
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

export interface CreateTableMappingInput {
  schema?: string
  table: string
  enabled?: boolean
  config?: Record<string, unknown>
}

export interface CreatePipelineInput {
  name: string
  source_id: string
  tables?: CreateTableMappingInput[]
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

// Metrics Types

export interface PipelineMetrics {
  pipeline_id: string
  status: PipelineStatus
  events_processed: number
  events_per_second: number
  lag_seconds: number
  lag_p95_seconds: number
  buffer_depth: number
  error_count: number
  iceberg_commits: number
  iceberg_bytes_written: number
  last_event_at?: string
  uptime?: string
  tables?: TableMetrics[]
}

export interface TableMetrics {
  schema: string
  table: string
  events_processed: number
  lag_seconds: number
  last_event_at?: string
}

export interface MetricsDataPoint {
  timestamp: string
  events_per_second: number
  lag_seconds: number
  buffer_depth: number
  error_count: number
}

export interface MetricsHistory {
  pipeline_id: string
  time_range: string
  data_points: MetricsDataPoint[]
}

export interface PipelineMetricsResponse {
  metrics: PipelineMetrics
}

export interface MetricsHistoryResponse {
  history: MetricsHistory
}

// Scaling Types

export type ScalingTargetType = "cdc-worker" | "trino" | "risingwave" | "nodes"
export type ScalingAction = "scale_up" | "scale_down" | "scheduled" | "manual"
export type RuleOperator = "gt" | "lt" | "gte" | "lte" | "eq"

export interface ScalingRule {
  metric: string
  operator: RuleOperator
  threshold: number
  duration_seconds: number
  scale_by: number
}

export interface ScalingSchedule {
  cron_expression: string
  desired_replicas: number
  timezone: string
  enabled: boolean
}

export interface ScalingPolicy {
  id: string
  name: string
  target_type: ScalingTargetType
  target_id?: string
  min_replicas: number
  max_replicas: number
  cooldown_seconds: number
  max_hourly_cost?: number
  scale_to_zero: boolean
  enabled: boolean
  scale_up_rules: ScalingRule[]
  scale_down_rules: ScalingRule[]
  schedules: ScalingSchedule[]
  created_at: string
  updated_at: string
}

export interface ScalingState {
  policy_id: string
  current_replicas: number
  last_scale_time?: string
  last_scale_action?: string
  pending_conditions?: Record<string, string>
  updated_at: string
}

export interface ScalingHistory {
  id: string
  policy_id?: string
  policy_name: string
  action: ScalingAction
  target_type: ScalingTargetType
  target_id?: string
  previous_replicas: number
  new_replicas: number
  reason: string
  triggered_by: string
  dry_run: boolean
  executed_at: string
}

export interface ScalingEvaluationResult {
  policy_id: string
  should_scale: boolean
  recommended_action?: ScalingAction
  recommended_replicas?: number
  current_replicas: number
  reason: string
  triggered_rules: string[]
  dry_run: boolean
}

export interface CreateScalingPolicyInput {
  name: string
  target_type: ScalingTargetType
  target_id?: string
  min_replicas: number
  max_replicas: number
  cooldown_seconds?: number
  max_hourly_cost?: number
  scale_to_zero?: boolean
  enabled?: boolean
  scale_up_rules?: ScalingRule[]
  scale_down_rules?: ScalingRule[]
  schedules?: ScalingSchedule[]
}

export interface ScalingPolicyResponse {
  policy: ScalingPolicy
}

export interface ScalingPoliciesResponse {
  policies: ScalingPolicy[]
}

export interface ScalingStateResponse {
  state: ScalingState
}

export interface ScalingHistoryResponse {
  history: ScalingHistory[]
}

export interface ScalingEvaluationResponse {
  result: ScalingEvaluationResult
}

// Installer Types

export type DeploymentStatus =
  | "pending"
  | "provisioning"
  | "configuring"
  | "deploying"
  | "verifying"
  | "completed"
  | "failed"
  | "canceled"

export type DeploymentSize = "small" | "medium" | "large"

export interface ProviderRegion {
  id: string
  name: string
  location: string
  is_default?: boolean
  is_available: boolean
}

export interface ProviderSize {
  id: DeploymentSize
  name: string
  description: string
  monthly_cost_eur: number
  control_plane_type: string
  worker_type: string
  worker_count: number
  storage_size_gb: number
  vcpu: number
  memory_gb: number
}

export interface Provider {
  id: string
  name: string
  description: string
  logo_url?: string
  regions: ProviderRegion[]
  sizes: ProviderSize[]
}

export interface DeploymentConfig {
  domain?: string
  ssh_public_key?: string
  chart_version?: string
  worker_count?: number
  storage_size_gb?: number
}

export interface DeploymentOutput {
  control_plane_ip?: string
  load_balancer_ip?: string
  kubeconfig?: string
  dashboard_url?: string
  api_url?: string
}

export interface Deployment {
  id: string
  user_id?: string
  name: string
  provider: string
  region: string
  size: DeploymentSize
  status: DeploymentStatus
  environment: string
  config?: DeploymentConfig
  outputs?: DeploymentOutput
  pulumi_stack_name?: string
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface DeploymentLog {
  id: number
  deployment_id: string
  level: string
  step?: string
  message: string
  timestamp: string
}

export interface CostEstimate {
  provider: string
  size: string
  control_plane: number
  workers: number
  storage: number
  load_balancer: number
  total: number
  currency: string
}

export interface ProviderCredentials {
  // Hetzner
  hetzner_token?: string
  // Scaleway
  scaleway_access_key?: string
  scaleway_secret_key?: string
  scaleway_project_id?: string
  // OVH
  ovh_endpoint?: string
  ovh_application_key?: string
  ovh_application_secret?: string
  ovh_consumer_key?: string
  ovh_service_name?: string
  // Exoscale
  exoscale_api_key?: string
  exoscale_api_secret?: string
  // Contabo
  contabo_client_id?: string
  contabo_client_secret?: string
  contabo_api_user?: string
  contabo_api_password?: string
}

export interface CreateDeploymentInput {
  name: string
  provider: string
  region: string
  size: DeploymentSize
  environment?: string
  domain?: string
  ssh_public_key?: string
  chart_version?: string
  worker_count?: number
  storage_size_gb?: number
  credentials?: ProviderCredentials
}

// Installer API Responses

export interface ProvidersResponse {
  providers: Provider[]
}

export interface ProviderResponse {
  provider: Provider
}

export interface DeploymentsResponse {
  deployments: Deployment[]
  total_count: number
}

export interface DeploymentResponse {
  deployment: Deployment
}

export interface DeploymentLogsResponse {
  logs: DeploymentLog[]
  total_count: number
}

export interface CostEstimateResponse {
  estimate: CostEstimate
}
