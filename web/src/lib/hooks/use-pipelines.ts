import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { pipelinesApi, type CreatePipelineInput } from "@/lib/api"

export function usePipelines(options?: { refetchInterval?: number | false }) {
  return useQuery({
    queryKey: ["pipelines"],
    queryFn: () => pipelinesApi.list(),
    refetchInterval: options?.refetchInterval,
  })
}

export function usePipeline(id: string) {
  return useQuery({
    queryKey: ["pipelines", id],
    queryFn: () => pipelinesApi.get(id),
    enabled: !!id,
  })
}

export function useCreatePipeline() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreatePipelineInput) => pipelinesApi.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["pipelines"] })
    },
  })
}

export function useUpdatePipeline() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: string
      input: Partial<CreatePipelineInput>
    }) => pipelinesApi.update(id, input),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ["pipelines"] })
      queryClient.invalidateQueries({ queryKey: ["pipelines", id] })
    },
  })
}

export function useDeletePipeline() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => pipelinesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["pipelines"] })
    },
  })
}

export function useStartPipeline() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => pipelinesApi.start(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["pipelines", id] })
      queryClient.invalidateQueries({ queryKey: ["pipelines"] })
    },
  })
}

export function useStopPipeline() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => pipelinesApi.stop(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["pipelines", id] })
      queryClient.invalidateQueries({ queryKey: ["pipelines"] })
    },
  })
}
