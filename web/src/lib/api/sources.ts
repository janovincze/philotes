import { apiClient } from "./client"
import type { Source, CreateSourceInput, TableDiscoveryResponse, ConnectionTestResult } from "./types"

export const sourcesApi = {
  /**
   * List all sources
   */
  async list(): Promise<Source[]> {
    const resp = await apiClient.get<{ sources: Source[]; total_count: number }>("/api/v1/sources")
    return resp.sources
  },

  /**
   * Get a single source by ID
   */
  async get(id: string): Promise<Source> {
    const resp = await apiClient.get<{ source: Source }>(`/api/v1/sources/${id}`)
    return resp.source
  },

  /**
   * Create a new source
   */
  async create(input: CreateSourceInput): Promise<Source> {
    const resp = await apiClient.post<{ source: Source }>("/api/v1/sources", input)
    return resp.source
  },

  /**
   * Update an existing source
   */
  async update(id: string, input: Partial<CreateSourceInput>): Promise<Source> {
    const resp = await apiClient.put<{ source: Source }>(`/api/v1/sources/${id}`, input)
    return resp.source
  },

  /**
   * Delete a source
   */
  delete(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/sources/${id}`)
  },

  /**
   * Test source connection
   */
  testConnection(id: string): Promise<ConnectionTestResult> {
    return apiClient.post<ConnectionTestResult>(
      `/api/v1/sources/${id}/test`
    )
  },

  /**
   * Discover tables from source
   */
  discoverTables(id: string, schema?: string): Promise<TableDiscoveryResponse> {
    const params = schema ? `?schema=${encodeURIComponent(schema)}` : ""
    return apiClient.get<TableDiscoveryResponse>(`/api/v1/sources/${id}/tables${params}`)
  },
}
