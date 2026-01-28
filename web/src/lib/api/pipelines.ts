import { apiClient } from "./client"
import type { Pipeline, CreatePipelineInput, TableMapping } from "./types"

export const pipelinesApi = {
  /**
   * List all pipelines
   */
  list(): Promise<Pipeline[]> {
    return apiClient.get<Pipeline[]>("/api/v1/pipelines")
  },

  /**
   * Get a single pipeline by ID
   */
  get(id: string): Promise<Pipeline> {
    return apiClient.get<Pipeline>(`/api/v1/pipelines/${id}`)
  },

  /**
   * Create a new pipeline
   */
  create(input: CreatePipelineInput): Promise<Pipeline> {
    return apiClient.post<Pipeline>("/api/v1/pipelines", input)
  },

  /**
   * Update an existing pipeline
   */
  update(id: string, input: Partial<CreatePipelineInput>): Promise<Pipeline> {
    return apiClient.put<Pipeline>(`/api/v1/pipelines/${id}`, input)
  },

  /**
   * Delete a pipeline
   */
  delete(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/pipelines/${id}`)
  },

  /**
   * Start a pipeline
   */
  start(id: string): Promise<Pipeline> {
    return apiClient.post<Pipeline>(`/api/v1/pipelines/${id}/start`)
  },

  /**
   * Stop a pipeline
   */
  stop(id: string): Promise<Pipeline> {
    return apiClient.post<Pipeline>(`/api/v1/pipelines/${id}/stop`)
  },

  /**
   * Get pipeline status
   */
  getStatus(id: string): Promise<Pipeline> {
    return apiClient.get<Pipeline>(`/api/v1/pipelines/${id}/status`)
  },

  /**
   * Add table mapping to pipeline
   */
  addTable(
    pipelineId: string,
    table: Omit<TableMapping, "id">
  ): Promise<TableMapping> {
    return apiClient.post<TableMapping>(
      `/api/v1/pipelines/${pipelineId}/tables`,
      table
    )
  },

  /**
   * Remove table mapping from pipeline
   */
  removeTable(pipelineId: string, tableId: string): Promise<void> {
    return apiClient.delete(`/api/v1/pipelines/${pipelineId}/tables/${tableId}`)
  },
}
