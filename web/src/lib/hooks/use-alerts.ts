import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { alertsApi } from "@/lib/api"
import type { CreateAlertRuleInput, CreateChannelInput } from "@/lib/api/types"

// Alert Rules
export function useAlertRules() {
  return useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => alertsApi.listRules(),
  })
}

export function useCreateAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAlertRuleInput) => alertsApi.createRule(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] })
      queryClient.invalidateQueries({ queryKey: ["alert-summary"] })
    },
  })
}

export function useUpdateAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CreateAlertRuleInput> }) =>
      alertsApi.updateRule(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] })
      queryClient.invalidateQueries({ queryKey: ["alert-summary"] })
    },
  })
}

export function useDeleteAlertRule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => alertsApi.deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] })
      queryClient.invalidateQueries({ queryKey: ["alert-summary"] })
    },
  })
}

// Alert Instances
export function useAlerts() {
  return useQuery({
    queryKey: ["alerts"],
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
      queryClient.invalidateQueries({ queryKey: ["alerts"] })
      queryClient.invalidateQueries({ queryKey: ["alert-summary"] })
    },
  })
}

export function useAlertSummary() {
  return useQuery({
    queryKey: ["alert-summary"],
    queryFn: () => alertsApi.getSummary(),
    refetchInterval: 30000,
  })
}

// Notification Channels
export function useNotificationChannels() {
  return useQuery({
    queryKey: ["notification-channels"],
    queryFn: () => alertsApi.listChannels(),
  })
}

export function useCreateChannel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateChannelInput) => alertsApi.createChannel(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] })
    },
  })
}

export function useDeleteChannel() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => alertsApi.deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-channels"] })
    },
  })
}

export function useTestChannel() {
  return useMutation({
    mutationFn: (id: string) => alertsApi.testChannel(id),
  })
}
