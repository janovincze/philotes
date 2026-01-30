import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useEffect, useRef, useCallback, useState } from "react"
import {
  installerApi,
  createDeploymentLogsWebSocket,
} from "@/lib/api"
import type {
  DeploymentLogMessage,
  CreateDeploymentInput,
  DeploymentSize,
  StepUpdate,
  ProgressUpdate,
} from "@/lib/api/types"

// Provider hooks

export function useProviders() {
  return useQuery({
    queryKey: ["installer", "providers"],
    queryFn: () => installerApi.listProviders(),
  })
}

export function useProvider(providerId: string) {
  return useQuery({
    queryKey: ["installer", "providers", providerId],
    queryFn: () => installerApi.getProvider(providerId),
    enabled: !!providerId,
  })
}

export function useCostEstimate(providerId: string, size: DeploymentSize) {
  return useQuery({
    queryKey: ["installer", "providers", providerId, "estimate", size],
    queryFn: () => installerApi.getCostEstimate(providerId, size),
    enabled: !!providerId && !!size,
  })
}

// Deployment hooks

export function useDeployments() {
  return useQuery({
    queryKey: ["installer", "deployments"],
    queryFn: () => installerApi.listDeployments(),
  })
}

export function useDeployment(deploymentId: string) {
  return useQuery({
    queryKey: ["installer", "deployments", deploymentId],
    queryFn: () => installerApi.getDeployment(deploymentId),
    enabled: !!deploymentId,
    // Refetch more frequently for active deployments
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (
        status === "pending" ||
        status === "provisioning" ||
        status === "configuring" ||
        status === "deploying" ||
        status === "verifying"
      ) {
        return 5000 // Refetch every 5 seconds for active deployments
      }
      return false
    },
  })
}

export function useCreateDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateDeploymentInput) => installerApi.createDeployment(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["installer", "deployments"] })
    },
  })
}

export function useCancelDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (deploymentId: string) => installerApi.cancelDeployment(deploymentId),
    onSuccess: (_, deploymentId) => {
      queryClient.invalidateQueries({ queryKey: ["installer", "deployments"] })
      queryClient.invalidateQueries({
        queryKey: ["installer", "deployments", deploymentId],
      })
    },
  })
}

export function useDeleteDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (deploymentId: string) => installerApi.deleteDeployment(deploymentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["installer", "deployments"] })
    },
  })
}

export function useDeploymentLogs(deploymentId: string, limit?: number) {
  return useQuery({
    queryKey: ["installer", "deployments", deploymentId, "logs", limit],
    queryFn: () => installerApi.getDeploymentLogs(deploymentId, limit),
    enabled: !!deploymentId,
  })
}

// Progress tracking hooks

export function useDeploymentProgress(deploymentId: string) {
  return useQuery({
    queryKey: ["installer", "deployments", deploymentId, "progress"],
    queryFn: () => installerApi.getDeploymentProgress(deploymentId),
    enabled: !!deploymentId,
    refetchInterval: (query) => {
      // Refetch while deployment is active
      const progress = query.state.data
      if (progress && progress.overall_progress < 100) {
        return 3000 // Refetch every 3 seconds
      }
      return false
    },
  })
}

export function useRetryDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (deploymentId: string) => installerApi.retryDeployment(deploymentId),
    onSuccess: (_, deploymentId) => {
      queryClient.invalidateQueries({ queryKey: ["installer", "deployments"] })
      queryClient.invalidateQueries({
        queryKey: ["installer", "deployments", deploymentId],
      })
      queryClient.invalidateQueries({
        queryKey: ["installer", "deployments", deploymentId, "progress"],
      })
    },
  })
}

export function useCleanupResources(deploymentId: string) {
  return useQuery({
    queryKey: ["installer", "deployments", deploymentId, "cleanup-preview"],
    queryFn: () => installerApi.getCleanupResources(deploymentId),
    enabled: !!deploymentId,
  })
}

export function useRetryInfo(deploymentId: string) {
  return useQuery({
    queryKey: ["installer", "deployments", deploymentId, "retry-info"],
    queryFn: () => installerApi.getRetryInfo(deploymentId),
    enabled: !!deploymentId,
  })
}

// WebSocket hook for real-time logs and progress

export function useDeploymentLogsStream(deploymentId: string) {
  const [logs, setLogs] = useState<DeploymentLogMessage[]>([])
  const [status, setStatus] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState<ProgressUpdate | null>(null)
  const [stepUpdates, setStepUpdates] = useState<Map<string, StepUpdate>>(new Map())
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    if (!deploymentId || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setError(null)

    const ws = createDeploymentLogsWebSocket(
      deploymentId,
      (message) => {
        switch (message.type) {
          case "log":
            setLogs((prev) => [...prev, message])
            break
          case "status":
            setStatus(message.status || null)
            break
          case "connected":
            setConnected(true)
            break
          case "progress":
            if (message.progress) {
              setProgress(message.progress)
            }
            break
          case "step":
            if (message.step_update) {
              setStepUpdates((prev) => {
                const newMap = new Map(prev)
                newMap.set(message.step_update!.step_id, message.step_update!)
                return newMap
              })
            }
            break
          case "error":
            setLogs((prev) => [...prev, message])
            break
        }
      },
      (event: Event) => {
        setError(`WebSocket connection error: ${event.type}`)
        setConnected(false)
      },
      () => {
        setConnected(false)
      }
    )

    wsRef.current = ws
  }, [deploymentId])

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setConnected(false)
  }, [])

  const clearLogs = useCallback(() => {
    setLogs([])
    setStepUpdates(new Map())
  }, [])

  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  return {
    logs,
    status,
    connected,
    error,
    progress,
    stepUpdates,
    connect,
    disconnect,
    clearLogs,
  }
}
