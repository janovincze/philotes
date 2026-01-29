"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { oauthApi } from "@/lib/api"
import type {
  OAuthAuthorizeRequest,
  ProviderCredentials,
} from "@/lib/api/types"

// Query keys
export const oauthKeys = {
  all: ["oauth"] as const,
  providers: () => [...oauthKeys.all, "providers"] as const,
  credentials: () => [...oauthKeys.all, "credentials"] as const,
}

/**
 * Hook to fetch available OAuth providers.
 */
export function useOAuthProviders() {
  return useQuery({
    queryKey: oauthKeys.providers(),
    queryFn: () => oauthApi.getOAuthProviders(),
    staleTime: 5 * 60 * 1000, // 5 minutes - providers don't change often
  })
}

/**
 * Hook to fetch stored credentials.
 */
export function useCredentials() {
  return useQuery({
    queryKey: oauthKeys.credentials(),
    queryFn: () => oauthApi.listCredentials(),
  })
}

/**
 * Hook to start OAuth authorization flow.
 * Returns the authorization URL to redirect the user to.
 */
export function useOAuthAuthorize() {
  return useMutation({
    mutationFn: async ({
      provider,
      redirectUri,
      sessionId,
    }: {
      provider: string
      redirectUri: string
      sessionId?: string
    }) => {
      const request: OAuthAuthorizeRequest = {
        redirect_uri: redirectUri,
        session_id: sessionId,
      }
      return oauthApi.authorize(provider, request)
    },
  })
}

/**
 * Hook to store manual API credentials.
 */
export function useStoreCredential() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      provider,
      credentials,
      deploymentId,
      expiresIn,
    }: {
      provider: string
      credentials: ProviderCredentials
      deploymentId?: string
      expiresIn?: number
    }) => {
      return oauthApi.storeCredential(provider, credentials, deploymentId, expiresIn)
    },
    onSuccess: () => {
      // Invalidate credentials list
      queryClient.invalidateQueries({ queryKey: oauthKeys.credentials() })
    },
  })
}

/**
 * Hook to delete stored credentials.
 */
export function useDeleteCredential() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (provider: string) => {
      return oauthApi.deleteCredential(provider)
    },
    onSuccess: () => {
      // Invalidate credentials list
      queryClient.invalidateQueries({ queryKey: oauthKeys.credentials() })
    },
  })
}

/**
 * Check if a provider has stored credentials.
 */
export function useHasCredentials(provider: string) {
  const { data, isLoading, error } = useCredentials()

  const hasCredentials = data?.credentials?.some(
    (cred) => cred.provider === provider
  ) ?? false

  return {
    hasCredentials,
    isLoading,
    error,
  }
}

/**
 * Start OAuth flow by redirecting to the provider.
 */
export function useStartOAuthFlow() {
  const authorize = useOAuthAuthorize()

  const startFlow = async (provider: string) => {
    // Build the redirect URI (frontend callback page)
    const redirectUri = oauthApi.getFrontendCallbackUrl(provider)

    // Generate a session ID for unauthenticated users
    const sessionId = getOrCreateSessionId()

    try {
      const response = await authorize.mutateAsync({
        provider,
        redirectUri,
        sessionId,
      })

      // Redirect to the provider's authorization page
      if (typeof window !== "undefined") {
        window.location.href = response.authorization_url
      }
    } catch (error) {
      console.error("Failed to start OAuth flow:", error)
      throw error
    }
  }

  return {
    startFlow,
    isLoading: authorize.isPending,
    error: authorize.error,
  }
}

/**
 * Get or create a session ID for tracking OAuth state.
 */
function getOrCreateSessionId(): string {
  if (typeof window === "undefined") return ""

  const storageKey = "philotes_oauth_session"
  let sessionId = sessionStorage.getItem(storageKey)

  if (!sessionId) {
    sessionId = crypto.randomUUID()
    sessionStorage.setItem(storageKey, sessionId)
  }

  return sessionId
}
