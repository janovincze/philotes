import { apiClient } from "./client"
import type { Source, CreateSourceInput } from "./types"

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
  testConnection(id: string): Promise<{ success: boolean; message: string }> {
    return apiClient.post<{ success: boolean; message: string }>(
      `/api/v1/sources/${id}/test`
    )
  },

  /**
   * Discover tables from source
   */
  discoverTables(id: string): Promise<string[]> {
    return apiClient.get<string[]>(`/api/v1/sources/${id}/tables`)
  },
}
