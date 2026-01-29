import { useQuery } from "@tanstack/react-query"
import { metricsApi } from "../api/metrics"

/**
 * Hook to fetch current pipeline metrics with automatic polling
 * @param pipelineId - The pipeline ID
 * @param options - Query options
 * @param options.enabled - Whether to enable the query (defaults to true if pipelineId is set)
 * @param options.refetchInterval - Polling interval in ms (defaults to 5000)
 */
export function usePipelineMetrics(
  pipelineId: string,
  options?: {
    enabled?: boolean
    refetchInterval?: number | false
  }
) {
  return useQuery({
    queryKey: ["pipelines", pipelineId, "metrics"],
    queryFn: () => metricsApi.getPipelineMetrics(pipelineId),
    enabled: options?.enabled ?? !!pipelineId,
    refetchInterval: options?.refetchInterval ?? 5000, // Default 5s polling
    staleTime: 2000, // Consider data stale after 2s
  })
}

/**
 * Hook to fetch historical pipeline metrics
 * @param pipelineId - The pipeline ID
 * @param timeRange - Time range (e.g., "15m", "1h", "6h", "24h", "7d")
 * @param options - Query options
 * @param options.enabled - Whether to enable the query
 */
export function usePipelineMetricsHistory(
  pipelineId: string,
  timeRange: string = "1h",
  options?: {
    enabled?: boolean
  }
) {
  return useQuery({
    queryKey: ["pipelines", pipelineId, "metrics", "history", timeRange],
    queryFn: () => metricsApi.getPipelineMetricsHistory(pipelineId, timeRange),
    enabled: options?.enabled ?? !!pipelineId,
    staleTime: 30000, // Consider data stale after 30s
    refetchInterval: 60000, // Refetch every 60s for historical data
  })
}

// Re-export types for convenience
export type { PipelineMetrics, TableMetrics, MetricsDataPoint, MetricsHistory } from "../api/types"
