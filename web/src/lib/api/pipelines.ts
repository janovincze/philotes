import { apiClient } from "./client"
import type { Pipeline, CreatePipelineInput, TableMapping } from "./types"

export const pipelinesApi = {
  /**
   * List all pipelines
   */
  async list(): Promise<Pipeline[]> {
    const resp = await apiClient.get<{ pipelines: Pipeline[]; total_count: number }>("/api/v1/pipelines")
    return resp.pipelines
  },

  /**
   * Get a single pipeline by ID
   */
  async get(id: string): Promise<Pipeline> {
    const resp = await apiClient.get<{ pipeline: Pipeline }>(`/api/v1/pipelines/${id}`)
    return resp.pipeline
  },

  /**
   * Create a new pipeline
   */
  async create(input: CreatePipelineInput): Promise<Pipeline> {
    const resp = await apiClient.post<{ pipeline: Pipeline }>("/api/v1/pipelines", input)
    return resp.pipeline
  },

  /**
   * Update an existing pipeline
   */
  async update(id: string, input: Partial<CreatePipelineInput>): Promise<Pipeline> {
    const resp = await apiClient.put<{ pipeline: Pipeline }>(`/api/v1/pipelines/${id}`, input)
    return resp.pipeline
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
  start(id: string): Promise<void> {
    return apiClient.post(`/api/v1/pipelines/${id}/start`)
  },

  /**
   * Stop a pipeline
   */
  stop(id: string): Promise<void> {
    return apiClient.post(`/api/v1/pipelines/${id}/stop`)
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
