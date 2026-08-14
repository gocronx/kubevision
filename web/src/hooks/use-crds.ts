import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import api from "@/lib/api"
import { useCluster } from "@/hooks/use-cluster"

export interface CRDInfo {
  group: string
  version: string
  kind: string
  plural: string
  namespaced: boolean
  categories?: string[]
}

export function useCRDs() {
  const { currentCluster, isClusterHealthy } = useCluster()
  return useQuery<CRDInfo[]>({
    queryKey: ["crds", currentCluster],
    queryFn: async () => {
      if (!currentCluster) return []
      return (await api.get<CRDInfo[]>(`/clusters/${currentCluster}/crds`)) ?? []
    },
    enabled: isClusterHealthy && !!currentCluster,
  })
}

export function useRefreshCRDs() {
  const { currentCluster } = useCluster()
  const queryClient = useQueryClient()
  return useMutation<CRDInfo[], Error>({
    mutationFn: async () => {
      return api.post<CRDInfo[]>(`/clusters/${currentCluster}/crds/refresh`)
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["crds", currentCluster], data)
      toast.success("CRD list refreshed")
    },
    onError: () => {
      toast.error("Failed to refresh CRDs")
    },
  })
}
