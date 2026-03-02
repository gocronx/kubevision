import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

interface TopologyNode {
  id: string
  kind: string
  name: string
  namespace?: string
  status?: string
  labels?: Record<string, string>
}

interface TopologyEdge {
  source: string
  target: string
  relation: string
}

export interface TopologyData {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
}

export function useTopology(clusterID: string, namespace: string) {
  return useQuery<TopologyData>({
    queryKey: ["topology", clusterID, namespace],
    queryFn: async () => {
      const res = await api.get(
        `/clusters/${clusterID}/namespaces/${namespace}/topology`
      )
      return res as unknown as TopologyData
    },
    enabled: !!clusterID && !!namespace,
    refetchInterval: 30000,
  })
}
