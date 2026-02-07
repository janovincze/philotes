import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { queryDataSourcesApi } from "@/lib/api"
import type { CreateQueryDataSourceRequest, UpdateQueryDataSourceRequest } from "@/lib/api/types"

export function useQueryDataSources() {
  return useQuery({
    queryKey: ["query-data-sources"],
    queryFn: () => queryDataSourcesApi.list(),
  })
}

export function useQueryDataSource(id: string) {
  return useQuery({
    queryKey: ["query-data-sources", id],
    queryFn: () => queryDataSourcesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateQueryDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateQueryDataSourceRequest) => queryDataSourcesApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["query-data-sources"] })
    },
  })
}

export function useUpdateQueryDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateQueryDataSourceRequest }) =>
      queryDataSourcesApi.update(id, input),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["query-data-sources"] })
      queryClient.invalidateQueries({ queryKey: ["query-data-sources", id] })
    },
  })
}

export function useDeleteQueryDataSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => queryDataSourcesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["query-data-sources"] })
    },
  })
}

export function useTestQueryDataSourceConnection() {
  return useMutation({
    mutationFn: (id: string) => queryDataSourcesApi.testConnection(id),
  })
}
