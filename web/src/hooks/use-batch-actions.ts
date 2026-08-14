import { useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

interface BatchDeleteItem {
  resource: string
  name: string
  namespace: string
}

interface BatchRestartItem {
  kind: string
  name: string
  namespace: string
}

interface BatchResult {
  resource?: string
  kind?: string
  name: string
  namespace: string
  success: boolean
  error?: string
}

export function useBatchDelete(clusterID: string) {
  const queryClient = useQueryClient()

  return useMutation<BatchResult[], Error, BatchDeleteItem[]>({
    mutationFn: async (items) => {
      return api.post<BatchResult[]>(
        `/clusters/${clusterID}/resources/batch-delete`,
        items
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resources", clusterID] })
    },
  })
}

export function useBatchRestart(clusterID: string) {
  const queryClient = useQueryClient()

  return useMutation<BatchResult[], Error, BatchRestartItem[]>({
    mutationFn: async (items) => {
      return api.post<BatchResult[]>(
        `/clusters/${clusterID}/batch-restart`,
        items
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resources", clusterID] })
    },
  })
}
