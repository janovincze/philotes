import { apiClient } from "./client"
import type { PipelineMetricsResponse, MetricsHistoryResponse } from "./types"

export const metricsApi = {
  /**
   * Get current metrics for a pipeline
   */
  getPipelineMetrics(pipelineId: string): Promise<PipelineMetricsResponse> {
    return apiClient.get<PipelineMetricsResponse>(
      `/api/v1/pipelines/${pipelineId}/metrics`
    )
  },

  /**
   * Get historical metrics for a pipeline
   * @param pipelineId - The pipeline ID
   * @param range - Time range (e.g., "15m", "1h", "6h", "24h", "7d")
   */
  getPipelineMetricsHistory(
    pipelineId: string,
    range: string = "1h"
  ): Promise<MetricsHistoryResponse> {
    return apiClient.get<MetricsHistoryResponse>(
      `/api/v1/pipelines/${pipelineId}/metrics/history`,
      { range }
    )
  },
}
