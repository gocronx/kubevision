import { useState, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

interface Cluster {
  id: string
  name: string
  status?: string
}

interface ClusterListResponse {
  items: Cluster[]
}

const CLUSTER_STORAGE_KEY = "kubevision-current-cluster"

function getStoredCluster(): string {
  return localStorage.getItem(CLUSTER_STORAGE_KEY) ?? ""
}

function storeCluster(clusterID: string) {
  localStorage.setItem(CLUSTER_STORAGE_KEY, clusterID)
}

export function useClusterList() {
  return useQuery<Cluster[]>({
    queryKey: ["clusters"],
    queryFn: async () => {
      const res = await api.get<ClusterListResponse>("/clusters")
      // The api interceptor unwraps ApiResponse.data, so res is the data directly
      const data = res as unknown as ClusterListResponse
      return data.items ?? []
    },
  })
}

export function useCluster() {
  const { data: clusters = [], isLoading } = useClusterList()
  const [currentClusterID, setCurrentClusterIDState] = useState<string>(getStoredCluster)

  const setCurrentCluster = useCallback((clusterID: string) => {
    storeCluster(clusterID)
    setCurrentClusterIDState(clusterID)
  }, [])

  // Auto-select first cluster if none is selected
  const effectiveClusterID =
    currentClusterID && clusters.some((c) => c.id === currentClusterID)
      ? currentClusterID
      : clusters[0]?.id ?? ""

  // Persist auto-selection
  if (effectiveClusterID && effectiveClusterID !== currentClusterID) {
    storeCluster(effectiveClusterID)
  }

  return {
    currentCluster: effectiveClusterID,
    clusters,
    setCurrentCluster,
    isLoading,
  }
}
