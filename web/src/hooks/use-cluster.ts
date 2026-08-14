import {
  createContext,
  createElement,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

export interface Cluster {
  id: string | number
  name: string
  status?: string
  apiServer?: string
  version?: string
}

const CLUSTER_STORAGE_KEY = "kubevision-current-cluster"
const CLUSTER_SCOPED_QUERIES = new Set([
  "overview",
  "resources",
  "resource",
  "topology",
  "quota-summary",
  "crds",
  "search",
  "package-releases",
  "package-release",
  "package-history",
  "helm-repositories",
  "helm-repository-charts",
  "artifact-hub",
  "helm-upgrade-policies",
  "rollout-history",
])

interface ClusterSelectionContextValue {
  currentClusterID: string
  setCurrentClusterID: (clusterID: string | number) => void
}

const ClusterSelectionContext = createContext<ClusterSelectionContextValue | null>(null)

function getStoredCluster(): string {
  return localStorage.getItem(CLUSTER_STORAGE_KEY) ?? ""
}

function storeCluster(clusterID: string) {
  localStorage.setItem(CLUSTER_STORAGE_KEY, clusterID)
}

export function ClusterProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [currentClusterID, setCurrentClusterIDState] = useState(getStoredCluster)

  const setCurrentClusterID = useCallback((clusterID: string | number) => {
    const id = String(clusterID)
    if (id === currentClusterID) return

    void queryClient.invalidateQueries({
      predicate: (query) => (
        CLUSTER_SCOPED_QUERIES.has(String(query.queryKey[0])) &&
        String(query.queryKey[1] ?? "") === id
      ),
      refetchType: "none",
    })
    storeCluster(id)
    setCurrentClusterIDState(id)
  }, [currentClusterID, queryClient])

  const value = useMemo(() => ({ currentClusterID, setCurrentClusterID }), [
    currentClusterID,
    setCurrentClusterID,
  ])

  return createElement(ClusterSelectionContext.Provider, { value }, children)
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
  const selection = useContext(ClusterSelectionContext)
  if (!selection) {
    throw new Error("useCluster must be used within ClusterProvider")
  }
  const {
    data: clusters = [],
    isLoading,
    isFetching: isFetchingClusters,
    refetch: refetchClusters,
  } = useClusterList()
  const { currentClusterID, setCurrentClusterID } = selection

  // Auto-select first cluster if none is selected
  const effectiveClusterID =
    currentClusterID && clusters.some((c) => String(c.id) === currentClusterID)
      ? currentClusterID
      : String(clusters[0]?.id ?? "")

  useEffect(() => {
    if (effectiveClusterID && effectiveClusterID !== currentClusterID) {
      setCurrentClusterID(effectiveClusterID)
    }
  }, [currentClusterID, effectiveClusterID, setCurrentClusterID])

  const selectedCluster = clusters.find(
    (cluster) => String(cluster.id) === effectiveClusterID
  )
  const isClusterHealthy = selectedCluster?.status !== "unhealthy"

  return {
    currentCluster: effectiveClusterID,
    clusters,
    selectedCluster,
    isClusterHealthy,
    setCurrentCluster: setCurrentClusterID,
    isLoading,
    isFetchingClusters,
    refetchClusters,
  }
}
