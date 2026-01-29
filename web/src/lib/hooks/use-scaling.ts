import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { scalingApi } from "../api/scaling"
import type { CreateScalingPolicyInput } from "../api/types"

/**
 * Hook to fetch all scaling policies
 */
export function useScalingPolicies() {
  return useQuery({
    queryKey: ["scaling", "policies"],
    queryFn: () => scalingApi.listPolicies(),
  })
}

/**
 * Hook to fetch a single scaling policy
 */
export function useScalingPolicy(id: string) {
  return useQuery({
    queryKey: ["scaling", "policies", id],
    queryFn: () => scalingApi.getPolicy(id),
    enabled: !!id,
  })
}

/**
 * Hook to fetch the current state of a scaling policy
 */
export function useScalingState(id: string, options?: { refetchInterval?: number | false }) {
  return useQuery({
    queryKey: ["scaling", "policies", id, "state"],
    queryFn: () => scalingApi.getPolicyState(id),
    enabled: !!id,
    refetchInterval: options?.refetchInterval ?? 10000, // Default 10s polling
  })
}

/**
 * Hook to fetch scaling history
 */
export function useScalingHistory(policyId?: string) {
  return useQuery({
    queryKey: ["scaling", "history", policyId],
    queryFn: () => scalingApi.listHistory(policyId),
  })
}

/**
 * Hook to create a new scaling policy
 */
export function useCreateScalingPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateScalingPolicyInput) => scalingApi.createPolicy(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies"] })
    },
  })
}

/**
 * Hook to update an existing scaling policy
 */
export function useUpdateScalingPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CreateScalingPolicyInput> }) =>
      scalingApi.updatePolicy(id, input),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies"] })
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies", id] })
    },
  })
}

/**
 * Hook to delete a scaling policy
 */
export function useDeleteScalingPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => scalingApi.deletePolicy(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies"] })
    },
  })
}

/**
 * Hook to enable a scaling policy
 */
export function useEnableScalingPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => scalingApi.enablePolicy(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies"] })
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies", id] })
    },
  })
}

/**
 * Hook to disable a scaling policy
 */
export function useDisableScalingPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => scalingApi.disablePolicy(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies"] })
      queryClient.invalidateQueries({ queryKey: ["scaling", "policies", id] })
    },
  })
}

/**
 * Hook to evaluate a scaling policy (preview/dry-run)
 */
export function useEvaluateScalingPolicy() {
  return useMutation({
    mutationFn: ({ id, dryRun = true }: { id: string; dryRun?: boolean }) =>
      scalingApi.evaluatePolicy(id, dryRun),
  })
}

// Re-export types for convenience
export type {
  ScalingPolicy,
  ScalingRule,
  ScalingSchedule,
  ScalingState,
  ScalingHistory,
  ScalingEvaluationResult,
  CreateScalingPolicyInput,
  ScalingTargetType,
  ScalingAction,
  RuleOperator,
} from "../api/types"
