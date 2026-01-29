import { apiClient } from "./client"
import type {
  Provider,
  ProvidersResponse,
  ProviderResponse,
  Deployment,
  DeploymentsResponse,
  DeploymentResponse,
  DeploymentLog,
  DeploymentLogsResponse,
  CostEstimate,
  CostEstimateResponse,
  CreateDeploymentInput,
} from "./types"

const BASE_PATH = "/api/v1/installer"

export const installerApi = {
  // Provider endpoints
  listProviders(): Promise<Provider[]> {
    return apiClient
      .get<ProvidersResponse>(`${BASE_PATH}/providers`)
      .then((res) => res.providers)
  },

  getProvider(providerId: string): Promise<Provider> {
    return apiClient
      .get<ProviderResponse>(`${BASE_PATH}/providers/${providerId}`)
      .then((res) => res.provider)
  },

  getCostEstimate(
    providerId: string,
    size: "small" | "medium" | "large"
  ): Promise<CostEstimate> {
    return apiClient
      .get<CostEstimateResponse>(`${BASE_PATH}/providers/${providerId}/estimate`, {
        size,
      })
      .then((res) => res.estimate)
  },

  // Deployment endpoints
  listDeployments(): Promise<Deployment[]> {
    return apiClient
      .get<DeploymentsResponse>(`${BASE_PATH}/deployments`)
      .then((res) => res.deployments)
  },

  getDeployment(deploymentId: string): Promise<Deployment> {
    return apiClient
      .get<DeploymentResponse>(`${BASE_PATH}/deployments/${deploymentId}`)
      .then((res) => res.deployment)
  },

  createDeployment(input: CreateDeploymentInput): Promise<Deployment> {
    return apiClient
      .post<DeploymentResponse>(`${BASE_PATH}/deployments`, input)
      .then((res) => res.deployment)
  },

  cancelDeployment(deploymentId: string): Promise<void> {
    return apiClient.post(`${BASE_PATH}/deployments/${deploymentId}/cancel`)
  },

  deleteDeployment(deploymentId: string): Promise<void> {
    return apiClient.delete(`${BASE_PATH}/deployments/${deploymentId}`)
  },

  getDeploymentLogs(deploymentId: string, limit?: number): Promise<DeploymentLog[]> {
    return apiClient
      .get<DeploymentLogsResponse>(`${BASE_PATH}/deployments/${deploymentId}/logs`, {
        limit,
      })
      .then((res) => res.logs)
  },
}

// WebSocket connection helper for real-time logs
export function createDeploymentLogsWebSocket(
  deploymentId: string,
  onMessage: (message: DeploymentLogMessage) => void,
  onError?: (error: Event) => void,
  onClose?: () => void
): WebSocket {
  const wsUrl = getWebSocketUrl(`${BASE_PATH}/deployments/${deploymentId}/logs/stream`)
  const ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data) as DeploymentLogMessage
      onMessage(message)
    } catch (e) {
      console.error("Failed to parse WebSocket message:", e)
    }
  }

  ws.onerror = (error) => {
    console.error("WebSocket error:", error)
    onError?.(error)
  }

  ws.onclose = () => {
    onClose?.()
  }

  return ws
}

function getWebSocketUrl(path: string): string {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
  const url = new URL(path, apiUrl)
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:"
  return url.toString()
}

// WebSocket message types
export interface DeploymentLogMessage {
  type: "log" | "status" | "connected" | "error"
  deployment_id: string
  timestamp: string
  level?: string
  step?: string
  message?: string
  status?: string
}
