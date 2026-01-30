import { apiClient } from "./client"
import type {
  ClusterHealthResponse,
  OnboardingProgressResponse,
  SaveOnboardingProgressRequest,
  DataVerificationRequest,
  DataVerificationResponse,
  RegisterRequest,
  RegisterResponse,
  AdminExistsResponse,
} from "./types"

export const onboardingApi = {
  /**
   * Get extended cluster health for onboarding wizard
   */
  getClusterHealth(): Promise<ClusterHealthResponse> {
    return apiClient.get<ClusterHealthResponse>("/api/v1/onboarding/cluster/health")
  },

  /**
   * Get current onboarding progress
   */
  getProgress(sessionId?: string): Promise<OnboardingProgressResponse> {
    return apiClient.get<OnboardingProgressResponse>(
      "/api/v1/onboarding/progress",
      sessionId ? { session_id: sessionId } : undefined
    )
  },

  /**
   * Save onboarding progress
   */
  saveProgress(data: SaveOnboardingProgressRequest): Promise<OnboardingProgressResponse> {
    return apiClient.post<OnboardingProgressResponse>("/api/v1/onboarding/progress", data)
  },

  /**
   * Verify data flow to Iceberg
   */
  verifyDataFlow(data: DataVerificationRequest): Promise<DataVerificationResponse> {
    return apiClient.post<DataVerificationResponse>("/api/v1/onboarding/data/verify", data)
  },

  /**
   * Check if admin user already exists
   */
  checkAdminExists(): Promise<AdminExistsResponse> {
    return apiClient.get<AdminExistsResponse>("/api/v1/onboarding/admin/exists")
  },

  /**
   * Register first admin user during onboarding
   */
  registerAdmin(data: RegisterRequest): Promise<RegisterResponse> {
    return apiClient.post<RegisterResponse>("/api/v1/auth/register", data)
  },
}
