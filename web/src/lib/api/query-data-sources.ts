import { apiClient } from "./client"
import type {
  QueryDataSource,
  CreateQueryDataSourceRequest,
  UpdateQueryDataSourceRequest,
  QueryDataSourceTestResult,
} from "./types"

export const queryDataSourcesApi = {
  async list(): Promise<QueryDataSource[]> {
    const resp = await apiClient.get<{ data_sources: QueryDataSource[]; total_count: number }>(
      "/api/v1/query/datasources"
    )
    return resp.data_sources
  },

  async get(id: string): Promise<QueryDataSource> {
    const resp = await apiClient.get<{ data_source: QueryDataSource }>(
      `/api/v1/query/datasources/${id}`
    )
    return resp.data_source
  },

  async create(input: CreateQueryDataSourceRequest): Promise<QueryDataSource> {
    const resp = await apiClient.post<{ data_source: QueryDataSource }>(
      "/api/v1/query/datasources",
      input
    )
    return resp.data_source
  },

  async update(id: string, input: UpdateQueryDataSourceRequest): Promise<QueryDataSource> {
    const resp = await apiClient.put<{ data_source: QueryDataSource }>(
      `/api/v1/query/datasources/${id}`,
      input
    )
    return resp.data_source
  },

  delete(id: string): Promise<void> {
    return apiClient.delete(`/api/v1/query/datasources/${id}`)
  },

  testConnection(id: string): Promise<QueryDataSourceTestResult> {
    return apiClient.post<QueryDataSourceTestResult>(
      `/api/v1/query/datasources/${id}/test`
    )
  },
}
