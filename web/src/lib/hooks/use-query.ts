"use client"

import { useMutation, useQuery } from "@tanstack/react-query"
import { apiClient } from "@/lib/api/client"
import type {
  QueryExecuteRequest,
  QueryExecuteResponse,
  CatalogListResponse,
  SchemaListResponse,
  TableListResponse,
} from "@/lib/api/types"

// Execute a SQL query
export function useQueryExecute() {
  return useMutation({
    mutationFn: async (request: QueryExecuteRequest): Promise<QueryExecuteResponse> => {
      const response = await apiClient.post<QueryExecuteResponse>("/query/execute", request)
      return response
    },
  })
}

// List available catalogs
export function useCatalogs() {
  return useQuery({
    queryKey: ["query", "catalogs"],
    queryFn: async (): Promise<CatalogListResponse> => {
      return apiClient.get<CatalogListResponse>("/query/catalogs")
    },
    staleTime: 60000, // Cache for 1 minute
  })
}

// List schemas in a catalog
export function useSchemas(catalog: string | null) {
  return useQuery({
    queryKey: ["query", "schemas", catalog],
    queryFn: async (): Promise<SchemaListResponse> => {
      return apiClient.get<SchemaListResponse>(`/query/catalogs/${catalog}/schemas`)
    },
    enabled: !!catalog,
    staleTime: 60000,
  })
}

// List tables in a schema
export function useTables(catalog: string | null, schema: string | null) {
  return useQuery({
    queryKey: ["query", "tables", catalog, schema],
    queryFn: async (): Promise<TableListResponse> => {
      return apiClient.get<TableListResponse>(`/query/catalogs/${catalog}/schemas/${schema}/tables`)
    },
    enabled: !!catalog && !!schema,
    staleTime: 60000,
  })
}
