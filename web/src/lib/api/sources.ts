import { apiClient } from "./client"
import type { Source, CreateSourceInput, TableDiscoveryResponse, ConnectionTestResult } from "./types"

export const sourcesApi = {
  /**
   * List all sources
   */
  list(): Promise<Source[]> {
    return apiClient.get<Source[]>("/api/v1/sources")
  },

  /**
   * Get a single source by ID
   */
  get(id: string): Promise<Source> {
    return apiClient.get<Source>(`/api/v1/sources/${id}`)
  },

  /**
   * Create a new source
   */
  create(input: CreateSourceInput): Promise<Source> {
    return apiClient.post<Source>("/api/v1/sources", input)
  },

  /**
   * Update an existing source
   */
  update(id: string, input: Partial<CreateSourceInput>): Promise<Source> {
    return apiClient.put<Source>(`/api/v1/sources/${id}`, input)
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
