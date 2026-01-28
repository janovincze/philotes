import { apiClient } from "./client"
import type { HealthResponse } from "./types"

export const healthApi = {
  /**
   * Get overall system health status
   */
  getHealth(): Promise<HealthResponse> {
    return apiClient.get<HealthResponse>("/health")
  },

  /**
   * Kubernetes liveness probe
   */
  getLiveness(): Promise<{ status: string }> {
    return apiClient.get<{ status: string }>("/health/live")
  },

  /**
   * Kubernetes readiness probe
   */
  getReadiness(): Promise<{ status: string }> {
    return apiClient.get<{ status: string }>("/health/ready")
  },
}
