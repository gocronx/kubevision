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
      const res = await api.post(
        `/clusters/${clusterID}/resources/batch-delete`,
        items
      )
      return res as unknown as BatchResult[]
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
      const res = await api.post(
        `/clusters/${clusterID}/batch-restart`,
        items
      )
      return res as unknown as BatchResult[]
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resources", clusterID] })
    },
  })
}
