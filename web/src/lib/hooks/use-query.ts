"use client"

import { useMutation, useQuery, useQueries } from "@tanstack/react-query"
import { useMemo } from "react"
import { apiClient } from "@/lib/api/client"
import type {
  QueryExecuteRequest,
  QueryExecuteResponse,
  CatalogListResponse,
  SchemaListResponse,
  TableListResponse,
  TableInfoResponse,
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

// Get table info with columns
export function useTableInfo(catalog: string | null, schema: string | null, table: string | null) {
  return useQuery({
    queryKey: ["query", "tableInfo", catalog, schema, table],
    queryFn: async (): Promise<TableInfoResponse> => {
      return apiClient.get<TableInfoResponse>(
        `/query/catalogs/${catalog}/schemas/${schema}/tables/${table}`
      )
    },
    enabled: !!catalog && !!schema && !!table,
    staleTime: 60000,
  })
}

// Metadata for auto-complete suggestions
export interface AutoCompleteMetadata {
  catalogs: string[]
  schemas: Array<{ name: string; catalog: string }>
  tables: Array<{ name: string; schema: string; catalog: string; fullName: string }>
  columns: Array<{ name: string; table: string; schema: string; catalog: string; type: string }>
  isLoading: boolean
}

// Hook to fetch all metadata for auto-complete
export function useAutoCompleteMetadata(): AutoCompleteMetadata {
  const catalogsQuery = useCatalogs()
  const catalogs = catalogsQuery.data?.catalogs || []

  // Fetch schemas for each catalog
  const schemaQueries = useQueries({
    queries: catalogs.map((catalog) => ({
      queryKey: ["query", "schemas", catalog.name],
      queryFn: async (): Promise<SchemaListResponse> => {
        return apiClient.get<SchemaListResponse>(`/query/catalogs/${catalog.name}/schemas`)
      },
      staleTime: 60000,
      enabled: !!catalog.name,
    })),
  })

  // Build list of all schemas
  const allSchemas = useMemo(() => {
    const schemas: Array<{ name: string; catalog: string }> = []
    schemaQueries.forEach((query) => {
      if (query.data?.schemas) {
        query.data.schemas.forEach((schema) => {
          // Skip information_schema as it's not useful for user queries
          if (schema.name !== "information_schema") {
            schemas.push({ name: schema.name, catalog: schema.catalog })
          }
        })
      }
    })
    return schemas
  }, [schemaQueries])

  // Fetch tables for each catalog.schema combination
  const tableQueries = useQueries({
    queries: allSchemas.map(({ catalog, name: schema }) => ({
      queryKey: ["query", "tables", catalog, schema],
      queryFn: async (): Promise<TableListResponse> => {
        return apiClient.get<TableListResponse>(
          `/query/catalogs/${catalog}/schemas/${schema}/tables`
        )
      },
      staleTime: 60000,
      enabled: !!catalog && !!schema,
    })),
  })

  // Build list of all tables
  const allTables = useMemo(() => {
    const tables: Array<{ name: string; schema: string; catalog: string; fullName: string }> = []
    tableQueries.forEach((query) => {
      if (query.data?.tables) {
        query.data.tables.forEach((table) => {
          tables.push({
            name: table.name,
            schema: table.schema,
            catalog: table.catalog,
            fullName: `${table.catalog}.${table.schema}.${table.name}`,
          })
        })
      }
    })
    return tables
  }, [tableQueries])

  // Fetch columns for each table
  const columnQueries = useQueries({
    queries: allTables.map(({ catalog, schema, name }) => ({
      queryKey: ["query", "tableInfo", catalog, schema, name],
      queryFn: async (): Promise<TableInfoResponse> => {
        return apiClient.get<TableInfoResponse>(
          `/query/catalogs/${catalog}/schemas/${schema}/tables/${name}`
        )
      },
      staleTime: 60000,
      enabled: !!catalog && !!schema && !!name,
    })),
  })

  // Build list of all columns
  const allColumns = useMemo(() => {
    const columns: Array<{
      name: string
      table: string
      schema: string
      catalog: string
      type: string
    }> = []
    columnQueries.forEach((query) => {
      if (query.data?.columns) {
        query.data.columns.forEach((col) => {
          columns.push({
            name: col.name,
            table: query.data!.name,
            schema: query.data!.schema,
            catalog: query.data!.catalog,
            type: col.type,
          })
        })
      }
    })
    return columns
  }, [columnQueries])

  const isLoading =
    catalogsQuery.isLoading ||
    schemaQueries.some((q) => q.isLoading) ||
    tableQueries.some((q) => q.isLoading) ||
    columnQueries.some((q) => q.isLoading)

  return {
    catalogs: catalogs.map((c) => c.name),
    schemas: allSchemas,
    tables: allTables,
    columns: allColumns,
    isLoading,
  }
}
