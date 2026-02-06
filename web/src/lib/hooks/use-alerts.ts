import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { alertsApi } from "@/lib/api"
import type { CreateAlertRuleInput, CreateChannelInput } from "@/lib/api/types"

const alertKeys = {
  rules: ["alert-rules"] as const,
  alerts: ["alerts"] as const,
  summary: ["alert-summary"] as const,
  channels: ["notification-channels"] as const,
}

// Alert Rules
export function useAlertRules() {
  return useQuery({
    queryKey: alertKeys.rules,
    queryFn: () => alertsApi.listRules(),
  })
}

export function useCreateAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAlertRuleInput) => alertsApi.createRule(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules })
      queryClient.invalidateQueries({ queryKey: alertKeys.summary })
    },
  })
}

export function useUpdateAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CreateAlertRuleInput> }) =>
      alertsApi.updateRule(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules })
      queryClient.invalidateQueries({ queryKey: alertKeys.summary })
    },
  })
}

export function useDeleteAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => alertsApi.deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.rules })
      queryClient.invalidateQueries({ queryKey: alertKeys.summary })
    },
  })
}

// Alert Instances
export function useAlerts() {
  return useQuery({
    queryKey: alertKeys.alerts,
    queryFn: () => alertsApi.listAlerts(),
    refetchInterval: 30000, // refresh every 30s
  })
}

export function useAcknowledgeAlert() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, acknowledgedBy, comment }: { id: string; acknowledgedBy: string; comment?: string }) =>
      alertsApi.acknowledgeAlert(id, acknowledgedBy, comment),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.alerts })
      queryClient.invalidateQueries({ queryKey: alertKeys.summary })
    },
  })
}

export function useAlertSummary() {
  return useQuery({
    queryKey: alertKeys.summary,
    queryFn: () => alertsApi.getSummary(),
    refetchInterval: 30000,
  })
}

// Notification Channels
export function useNotificationChannels() {
  return useQuery({
    queryKey: alertKeys.channels,
    queryFn: () => alertsApi.listChannels(),
  })
}

export function useCreateChannel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateChannelInput) => alertsApi.createChannel(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.channels })
    },
  })
}

export function useDeleteChannel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => alertsApi.deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: alertKeys.channels })
    },
  })
}

export function useTestChannel() {
  return useMutation({
    mutationFn: (id: string) => alertsApi.testChannel(id),
  })
}
