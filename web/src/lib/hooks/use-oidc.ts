"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { oidcApi } from "@/lib/api"
import type {
  CreateOIDCProviderRequest,
  UpdateOIDCProviderRequest,
  OIDCCallbackRequest,
} from "@/lib/api/types"

// Query keys
export const oidcKeys = {
  all: ["oidc"] as const,
  enabledProviders: () => [...oidcKeys.all, "enabled-providers"] as const,
  providers: () => [...oidcKeys.all, "providers"] as const,
  provider: (id: string) => [...oidcKeys.providers(), id] as const,
}

/**
 * Hook to fetch enabled OIDC providers for login page.
 */
export function useEnabledOIDCProviders() {
  return useQuery({
    queryKey: oidcKeys.enabledProviders(),
    queryFn: () => oidcApi.listEnabledProviders(),
    staleTime: 5 * 60 * 1000, // 5 minutes - providers don't change often
  })
}

/**
 * Hook to fetch all OIDC providers (admin).
 */
export function useOIDCProviders() {
  return useQuery({
    queryKey: oidcKeys.providers(),
    queryFn: () => oidcApi.listProviders(),
  })
}

/**
 * Hook to fetch a single OIDC provider.
 */
export function useOIDCProvider(id: string) {
  return useQuery({
    queryKey: oidcKeys.provider(id),
    queryFn: () => oidcApi.getProvider(id),
    enabled: !!id,
  })
}

/**
 * Hook to create an OIDC provider.
 */
export function useCreateOIDCProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateOIDCProviderRequest) =>
      oidcApi.createProvider(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: oidcKeys.providers() })
      queryClient.invalidateQueries({ queryKey: oidcKeys.enabledProviders() })
    },
  })
}

/**
 * Hook to update an OIDC provider.
 */
export function useUpdateOIDCProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string
      request: UpdateOIDCProviderRequest
    }) => oidcApi.updateProvider(id, request),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: oidcKeys.provider(id) })
      queryClient.invalidateQueries({ queryKey: oidcKeys.providers() })
      queryClient.invalidateQueries({ queryKey: oidcKeys.enabledProviders() })
    },
  })
}

/**
 * Hook to delete an OIDC provider.
 */
export function useDeleteOIDCProvider() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => oidcApi.deleteProvider(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: oidcKeys.providers() })
      queryClient.invalidateQueries({ queryKey: oidcKeys.enabledProviders() })
    },
  })
}

/**
 * Hook to test an OIDC provider.
 */
export function useTestOIDCProvider() {
  return useMutation({
    mutationFn: (id: string) => oidcApi.testProvider(id),
  })
}

/**
 * Hook to start OIDC authorization flow.
 */
export function useOIDCAuthorize() {
  return useMutation({
    mutationFn: async ({ providerName }: { providerName: string }) => {
      const redirectUri = oidcApi.getCallbackUrl()
      return oidcApi.authorize(providerName, { redirect_uri: redirectUri })
    },
  })
}

/**
 * Hook to handle OIDC callback.
 */
export function useOIDCCallback() {
  return useMutation({
    mutationFn: (request: OIDCCallbackRequest) => oidcApi.callback(request),
  })
}

/**
 * Hook to start OIDC flow and redirect to provider.
 */
export function useStartOIDCFlow() {
  const authorize = useOIDCAuthorize()

  const startFlow = async (providerName: string) => {
    try {
      const response = await authorize.mutateAsync({ providerName })

      // Redirect to the provider's authorization page
      if (typeof window !== "undefined") {
        window.location.href = response.authorization_url
      }
    } catch (error) {
      console.error("Failed to start OIDC flow:", error)
      throw error
    }
  }

  return {
    startFlow,
    isLoading: authorize.isPending,
    error: authorize.error,
  }
}
