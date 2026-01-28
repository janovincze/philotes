import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { sourcesApi, type CreateSourceInput } from "@/lib/api"

export function useSources() {
  return useQuery({
    queryKey: ["sources"],
    queryFn: () => sourcesApi.list(),
  })
}

export function useSource(id: string) {
  return useQuery({
    queryKey: ["sources", id],
    queryFn: () => sourcesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateSourceInput) => sourcesApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sources"] })
    },
  })
}

export function useUpdateSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CreateSourceInput> }) =>
      sourcesApi.update(id, input),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["sources"] })
      queryClient.invalidateQueries({ queryKey: ["sources", id] })
    },
  })
}

export function useDeleteSource() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => sourcesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sources"] })
    },
  })
}

export function useTestSourceConnection() {
  return useMutation({
    mutationFn: (id: string) => sourcesApi.testConnection(id),
  })
}
