import { apiClient } from "./client"
import type {
  OIDCProvidersResponse,
  OIDCProviderResponse,
  OIDCProviderSummary,
  CreateOIDCProviderRequest,
  UpdateOIDCProviderRequest,
  OIDCAuthorizeRequest,
  OIDCAuthorizeResponse,
  OIDCCallbackRequest,
  OIDCCallbackResponse,
  OIDCTestResponse,
} from "./types"

const AUTH_PATH = "/api/v1/auth/oidc"
const SETTINGS_PATH = "/api/v1/settings/oidc"

/**
 * Get the base URL for building absolute URLs.
 * Returns empty string for SSR contexts.
 */
function getBaseUrl(): string {
  return typeof window !== "undefined" ? window.location.origin : ""
}

export const oidcApi = {
  // --- Public endpoints ---

  /**
   * List enabled OIDC providers for login page.
   */
  async listEnabledProviders(): Promise<OIDCProvidersResponse> {
    return apiClient.get<OIDCProvidersResponse>(`${AUTH_PATH}/providers`)
  },

  /**
   * Start OIDC authorization flow.
   * Returns the authorization URL to redirect the user to.
   */
  async authorize(
    providerName: string,
    request: OIDCAuthorizeRequest
  ): Promise<OIDCAuthorizeResponse> {
    return apiClient.post<OIDCAuthorizeResponse>(
      `${AUTH_PATH}/${providerName}/authorize`,
      request
    )
  },

  /**
   * Handle OIDC callback from identity provider.
   */
  async callback(request: OIDCCallbackRequest): Promise<OIDCCallbackResponse> {
    return apiClient.post<OIDCCallbackResponse>(`${AUTH_PATH}/callback`, request)
  },

  // --- Admin endpoints ---

  /**
   * List all OIDC providers (admin).
   */
  async listProviders(): Promise<OIDCProvidersResponse> {
    return apiClient.get<OIDCProvidersResponse>(`${SETTINGS_PATH}/providers`)
  },

  /**
   * Create a new OIDC provider.
   */
  async createProvider(
    request: CreateOIDCProviderRequest
  ): Promise<OIDCProviderResponse> {
    return apiClient.post<OIDCProviderResponse>(
      `${SETTINGS_PATH}/providers`,
      request
    )
  },

  /**
   * Get an OIDC provider by ID.
   */
  async getProvider(id: string): Promise<OIDCProviderResponse> {
    return apiClient.get<OIDCProviderResponse>(`${SETTINGS_PATH}/providers/${id}`)
  },

  /**
   * Update an OIDC provider.
   */
  async updateProvider(
    id: string,
    request: UpdateOIDCProviderRequest
  ): Promise<OIDCProviderResponse> {
    return apiClient.put<OIDCProviderResponse>(
      `${SETTINGS_PATH}/providers/${id}`,
      request
    )
  },

  /**
   * Delete an OIDC provider.
   */
  async deleteProvider(id: string): Promise<void> {
    return apiClient.delete(`${SETTINGS_PATH}/providers/${id}`)
  },

  /**
   * Test an OIDC provider's discovery endpoint.
   */
  async testProvider(id: string): Promise<OIDCTestResponse> {
    return apiClient.post<OIDCTestResponse>(
      `${SETTINGS_PATH}/providers/${id}/test`
    )
  },

  // --- URL helpers ---

  /**
   * Build the OIDC callback URL for the frontend.
   */
  getCallbackUrl(): string {
    return `${getBaseUrl()}/auth/oidc/callback`
  },

  /**
   * Get provider icon name based on provider type.
   */
  getProviderIcon(
    providerType: OIDCProviderSummary["provider_type"]
  ): string {
    switch (providerType) {
      case "google":
        return "google"
      case "okta":
        return "okta"
      case "azure_ad":
        return "microsoft"
      case "auth0":
        return "auth0"
      default:
        return "key"
    }
  },

  /**
   * Get display label for provider type.
   */
  getProviderTypeLabel(
    providerType: OIDCProviderSummary["provider_type"]
  ): string {
    switch (providerType) {
      case "google":
        return "Google"
      case "okta":
        return "Okta"
      case "azure_ad":
        return "Azure AD"
      case "auth0":
        return "Auth0"
      case "generic":
        return "Generic OIDC"
      default:
        return providerType
    }
  },
}
