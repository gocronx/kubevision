import { useState, useCallback } from "react"
import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

export interface Cluster {
  id: string | number
  name: string
  status?: string
  apiServer?: string
  version?: string
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
      const res = await api.get("/clusters")
      // The api interceptor unwraps ApiResponse.data, so res is the array directly
      return Array.isArray(res) ? (res as Cluster[]) : []
    },
    refetchInterval: 30_000,
  })
}

export function useCluster() {
  const {
    data: clusters = [],
    isLoading,
    isFetching: isFetchingClusters,
    refetch: refetchClusters,
  } = useClusterList()
  const [currentClusterID, setCurrentClusterIDState] = useState<string>(getStoredCluster)

  const setCurrentCluster = useCallback((clusterID: string | number) => {
    const id = String(clusterID)
    storeCluster(id)
    setCurrentClusterIDState(id)
  }, [])

  // Auto-select first cluster if none is selected
  const effectiveClusterID =
    currentClusterID && clusters.some((c) => String(c.id) === currentClusterID)
      ? currentClusterID
      : String(clusters[0]?.id ?? "")

  // Persist auto-selection
  if (effectiveClusterID && effectiveClusterID !== currentClusterID) {
    storeCluster(effectiveClusterID)
  }

  const selectedCluster = clusters.find(
    (cluster) => String(cluster.id) === effectiveClusterID
  )
  const isClusterHealthy = selectedCluster?.status !== "unhealthy"

  return {
    currentCluster: effectiveClusterID,
    clusters,
    selectedCluster,
    isClusterHealthy,
    setCurrentCluster,
    isLoading,
    isFetchingClusters,
    refetchClusters,
  }
}
