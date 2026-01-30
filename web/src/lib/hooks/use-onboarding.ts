import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  onboardingApi,
  type SaveOnboardingProgressRequest,
  type DataVerificationRequest,
  type RegisterRequest,
} from "@/lib/api"

/**
 * Hook to fetch cluster health for onboarding
 * Polls every 3 seconds by default
 */
export function useClusterHealth(enabled = true, refetchInterval = 3000) {
  return useQuery({
    queryKey: ["onboarding", "cluster-health"],
    queryFn: () => onboardingApi.getClusterHealth(),
    refetchInterval,
    enabled,
  })
}

/**
 * Hook to fetch onboarding progress
 */
export function useOnboardingProgress(sessionId?: string) {
  return useQuery({
    queryKey: ["onboarding", "progress", sessionId],
    queryFn: () => onboardingApi.getProgress(sessionId),
    staleTime: 0, // Always fetch fresh data
  })
}

/**
 * Hook to save onboarding progress
 */
export function useSaveOnboardingProgress() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: SaveOnboardingProgressRequest) =>
      onboardingApi.saveProgress(data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["onboarding", "progress", variables.session_id],
      })
    },
  })
}

/**
 * Hook to check if admin user exists
 */
export function useAdminExists() {
  return useQuery({
    queryKey: ["onboarding", "admin-exists"],
    queryFn: () => onboardingApi.checkAdminExists(),
    staleTime: 0,
  })
}

/**
 * Hook to register first admin user
 */
export function useRegisterAdmin() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: RegisterRequest) => onboardingApi.registerAdmin(data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["onboarding", "admin-exists"],
      })
    },
  })
}

/**
 * Hook to verify data flow to Iceberg
 */
export function useVerifyDataFlow() {
  return useMutation({
    mutationFn: (data: DataVerificationRequest) =>
      onboardingApi.verifyDataFlow(data),
  })
}

/**
 * Session ID management for anonymous onboarding
 */
const SESSION_STORAGE_KEY = "philotes_onboarding_session"

export function getOnboardingSessionId(): string {
  if (typeof window === "undefined") return ""

  let sessionId = sessionStorage.getItem(SESSION_STORAGE_KEY)
  if (!sessionId) {
    sessionId = crypto.randomUUID()
    sessionStorage.setItem(SESSION_STORAGE_KEY, sessionId)
  }
  return sessionId
}

export function clearOnboardingSession(): void {
  if (typeof window !== "undefined") {
    sessionStorage.removeItem(SESSION_STORAGE_KEY)
  }
}
