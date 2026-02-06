import { apiClient } from "./client"
import type {
  AlertRule,
  AlertInstance,
  AlertRuleListResponse,
  AlertInstanceListResponse,
  AlertSummaryResponse,
  NotificationChannel,
  ChannelListResponse,
  CreateAlertRuleInput,
  CreateChannelInput,
} from "./types"

export const alertsApi = {
  // Alert Rules
  listRules(): Promise<AlertRuleListResponse> {
    return apiClient.get<AlertRuleListResponse>("/api/v1/alerts/rules")
  },

  getRule(id: string): Promise<{ rule: AlertRule }> {
    return apiClient.get<{ rule: AlertRule }>(`/api/v1/alerts/rules/${id}`)
  },

  createRule(input: CreateAlertRuleInput): Promise<{ rule: AlertRule }> {
    return apiClient.post<{ rule: AlertRule }>("/api/v1/alerts/rules", input)
  },

  updateRule(id: string, input: Partial<CreateAlertRuleInput>): Promise<{ rule: AlertRule }> {
    return apiClient.put<{ rule: AlertRule }>(`/api/v1/alerts/rules/${id}`, input)
  },

  deleteRule(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/alerts/rules/${id}`)
  },

  // Alert Instances
  listAlerts(): Promise<AlertInstanceListResponse> {
    return apiClient.get<AlertInstanceListResponse>("/api/v1/alerts")
  },

  getAlert(id: string): Promise<{ alert: AlertInstance }> {
    return apiClient.get<{ alert: AlertInstance }>(`/api/v1/alerts/${id}`)
  },

  acknowledgeAlert(id: string, acknowledgedBy: string, comment?: string): Promise<{ alert: AlertInstance }> {
    return apiClient.post<{ alert: AlertInstance }>(`/api/v1/alerts/${id}/acknowledge`, {
      acknowledged_by: acknowledgedBy,
      comment,
    })
  },

  getSummary(): Promise<AlertSummaryResponse> {
    return apiClient.get<AlertSummaryResponse>("/api/v1/alerts/summary")
  },

  // Notification Channels
  listChannels(): Promise<ChannelListResponse> {
    return apiClient.get<ChannelListResponse>("/api/v1/notifications/channels")
  },

  createChannel(input: CreateChannelInput): Promise<{ channel: NotificationChannel }> {
    return apiClient.post<{ channel: NotificationChannel }>("/api/v1/notifications/channels", input)
  },

  updateChannel(id: string, input: Partial<CreateChannelInput>): Promise<{ channel: NotificationChannel }> {
    return apiClient.put<{ channel: NotificationChannel }>(`/api/v1/notifications/channels/${id}`, input)
  },

  deleteChannel(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/notifications/channels/${id}`)
  },

  testChannel(id: string): Promise<{ success: boolean; message: string }> {
    return apiClient.post<{ success: boolean; message: string }>(`/api/v1/notifications/channels/${id}/test`)
  },
}
