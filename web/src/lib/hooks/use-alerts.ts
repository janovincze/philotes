import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { alertsApi, type CreateAlertRuleInput, type UpdateAlertRuleInput } from "@/lib/api/alerts"

export function useAlertSummary() {
  return useQuery({
    queryKey: ["alerts", "summary"],
    queryFn: () => alertsApi.getSummary(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })
}

export function useAlertRules(limit = 100, offset = 0) {
  return useQuery({
    queryKey: ["alerts", "rules", { limit, offset }],
    queryFn: () => alertsApi.listRules(limit, offset),
  })
}

export function useAlertRule(id: string) {
  return useQuery({
    queryKey: ["alerts", "rules", id],
    queryFn: () => alertsApi.getRule(id),
    enabled: !!id,
  })
}

export function useCreateAlertRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateAlertRuleInput) => alertsApi.createRule(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts", "rules"] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "summary"] })
    },
  })
}

export function useUpdateAlertRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAlertRuleInput }) =>
      alertsApi.updateRule(id, input),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["alerts", "rules"] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "rules", id] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "summary"] })
    },
  })
}

export function useDeleteAlertRule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => alertsApi.deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts", "rules"] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "summary"] })
    },
  })
}

export function useAlerts(status?: string, severity?: string, limit = 100, offset = 0) {
  return useQuery({
    queryKey: ["alerts", "instances", { status, severity, limit, offset }],
    queryFn: () => alertsApi.listAlerts(status, severity, limit, offset),
    refetchInterval: 10000, // Refresh every 10 seconds
  })
}

export function useAlert(id: string) {
  return useQuery({
    queryKey: ["alerts", "instances", id],
    queryFn: () => alertsApi.getAlert(id),
    enabled: !!id,
  })
}

export function useAcknowledgeAlert() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      acknowledgedBy,
      comment,
    }: {
      id: string
      acknowledgedBy: string
      comment?: string
    }) => alertsApi.acknowledgeAlert(id, acknowledgedBy, comment),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["alerts", "instances"] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "instances", id] })
      queryClient.invalidateQueries({ queryKey: ["alerts", "summary"] })
    },
  })
}
