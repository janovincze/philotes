import { apiClient } from "./client"
import type {
  OAuthAuthorizeRequest,
  OAuthAuthorizeResponse,
  OAuthProvidersResponse,
  CredentialListResponse,
  StoreCredentialRequest,
  StoreCredentialResponse,
  ProviderCredentials,
} from "./types"

const BASE_PATH = "/api/v1/installer"

/**
 * Get the base URL for building absolute URLs.
 * Returns empty string for SSR contexts.
 */
function getBaseUrl(): string {
  return typeof window !== "undefined" ? window.location.origin : ""
}

export const oauthApi = {
  /**
   * Get list of available OAuth providers with their configuration.
   */
  async getOAuthProviders(): Promise<OAuthProvidersResponse> {
    return apiClient.get<OAuthProvidersResponse>(`${BASE_PATH}/oauth/providers`)
  },

  /**
   * Start OAuth authorization flow for a provider.
   * Returns the authorization URL to redirect the user to.
   */
  async authorize(
    provider: string,
    request: OAuthAuthorizeRequest
  ): Promise<OAuthAuthorizeResponse> {
    return apiClient.post<OAuthAuthorizeResponse>(
      `${BASE_PATH}/oauth/${provider}/authorize`,
      request
    )
  },

  /**
   * Store manual API credentials for a provider.
   */
  async storeCredential(
    provider: string,
    credentials: ProviderCredentials,
    deploymentId?: string,
    expiresIn?: number
  ): Promise<StoreCredentialResponse> {
    const request: StoreCredentialRequest = {
      provider,
      credentials,
      deployment_id: deploymentId,
      expires_in: expiresIn,
    }
    return apiClient.post<StoreCredentialResponse>(
      `${BASE_PATH}/credentials/${provider}`,
      request
    )
  },

  /**
   * List all stored credentials for the current user.
   */
  async listCredentials(): Promise<CredentialListResponse> {
    return apiClient.get<CredentialListResponse>(`${BASE_PATH}/credentials`)
  },

  /**
   * Delete stored credentials for a provider.
   */
  async deleteCredential(provider: string): Promise<void> {
    return apiClient.delete(`${BASE_PATH}/credentials/${provider}`)
  },

  /**
   * Build the OAuth callback URL for a provider.
   * This is the URL that the OAuth provider will redirect to after authorization.
   */
  getCallbackUrl(provider: string): string {
    return `${getBaseUrl()}/api/v1/installer/oauth/${provider}/callback`
  },

  /**
   * Build the frontend redirect URL for OAuth callback handling.
   */
  getFrontendCallbackUrl(provider: string): string {
    return `${getBaseUrl()}/install/oauth/callback?provider=${provider}`
  },
}
