import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

export interface OperationEvent {
  id: number
  stage: string
  status: string
  message: string
  progress: number
  createdAt: string
}

export interface Operation {
  id: string
  createdAt: string
  completedAt?: string
  parentId?: string
  username: string
  kind: string
  action: string
  status: "queued" | "running" | "succeeded" | "failed"
  stage: string
  cluster?: string
  namespace?: string
  resourceName?: string
  progress: number
  errorCode?: string
  errorMessage?: string
  suggestions?: string[]
  requestId?: string
  retryable: boolean
  rollbackAvailable: boolean
  events?: OperationEvent[]
}

const active = (operation?: Operation) => operation?.status === "queued" || operation?.status === "running"

export function useOperations() {
  return useQuery<Operation[]>({
    queryKey: ["operations"],
    queryFn: () => api.get<Operation[]>("/operations"),
    refetchInterval: (query) => query.state.data?.some(active) ? 2_000 : false,
  })
}

export function useOperation(id: string) {
  return useQuery<Operation>({
    queryKey: ["operation", id],
    queryFn: () => api.get<Operation>(`/operations/${id}`),
    enabled: !!id,
    refetchInterval: (query) => active(query.state.data) ? 2_000 : false,
  })
}

export function useRetryOperation() {
  const cache = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.post<Operation>(`/operations/${id}/retry`),
    onSuccess: () => cache.invalidateQueries({ queryKey: ["operations"] }),
  })
}
