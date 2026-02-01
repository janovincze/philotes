import { apiClient } from "./client"

// Types
export type AlertSeverity = "critical" | "warning" | "info"
export type AlertStatus = "firing" | "pending" | "resolved"
export type AlertOperator = ">" | ">=" | "<" | "<=" | "==" | "!="

export interface AlertRule {
  id: string
  name: string
  description?: string
  metric_name: string
  operator: AlertOperator
  threshold: number
  duration_seconds: number
  severity: AlertSeverity
  labels?: Record<string, string>
  annotations?: Record<string, string>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface AlertInstance {
  id: string
  rule_id: string
  fingerprint: string
  status: AlertStatus
  labels?: Record<string, string>
  annotations?: Record<string, string>
  current_value?: number
  fired_at: string
  resolved_at?: string
  acknowledged_at?: string
  acknowledged_by?: string
  created_at: string
  updated_at: string
  rule?: AlertRule
}

export interface AlertSummary {
  total_rules: number
  enabled_rules: number
  firing_alerts: number
  resolved_alerts: number
  active_silences: number
  total_channels: number
  enabled_channels: number
}

export interface CreateAlertRuleInput {
  name: string
  description?: string
  metric_name: string
  operator: AlertOperator
  threshold: number
  duration_seconds: number
  severity: AlertSeverity
  labels?: Record<string, string>
  annotations?: Record<string, string>
  enabled?: boolean
}

export interface UpdateAlertRuleInput {
  name?: string
  description?: string
  threshold?: number
  duration_seconds?: number
  severity?: AlertSeverity
  enabled?: boolean
}

// API Client
export const alertsApi = {
  // Summary
  getSummary(): Promise<AlertSummary> {
    return apiClient.get<AlertSummary>("/api/v1/alerts/summary")
  },

  // Alert Rules
  listRules(limit = 100, offset = 0): Promise<{ rules: AlertRule[]; total_count: number }> {
    return apiClient.get(`/api/v1/alerts/rules?limit=${limit}&offset=${offset}`)
  },

  getRule(id: string): Promise<{ rule: AlertRule }> {
    return apiClient.get(`/api/v1/alerts/rules/${id}`)
  },

  createRule(input: CreateAlertRuleInput): Promise<{ rule: AlertRule }> {
    return apiClient.post("/api/v1/alerts/rules", input)
  },

  updateRule(id: string, input: UpdateAlertRuleInput): Promise<{ rule: AlertRule }> {
    return apiClient.put(`/api/v1/alerts/rules/${id}`, input)
  },

  deleteRule(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/alerts/rules/${id}`)
  },

  // Alert Instances
  listAlerts(
    status?: string,
    severity?: string,
    limit = 100,
    offset = 0
  ): Promise<{ alerts: AlertInstance[]; total_count: number }> {
    const params = new URLSearchParams()
    params.set("limit", limit.toString())
    params.set("offset", offset.toString())
    if (status) params.set("status", status)
    if (severity) params.set("severity", severity)
    return apiClient.get(`/api/v1/alerts?${params.toString()}`)
  },

  getAlert(id: string): Promise<{ alert: AlertInstance }> {
    return apiClient.get(`/api/v1/alerts/${id}`)
  },

  acknowledgeAlert(id: string, acknowledgedBy: string, comment?: string): Promise<void> {
    return apiClient.post(`/api/v1/alerts/${id}/acknowledge`, {
      acknowledged_by: acknowledgedBy,
      comment,
    })
  },
}
