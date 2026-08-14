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

export function useTopology(clusterID: string, namespace: string, enabled = true) {
  return useQuery<TopologyData>({
    queryKey: ["topology", clusterID, namespace],
    queryFn: async () => {
      return api.get<TopologyData>(
        `/clusters/${clusterID}/namespaces/${namespace}/topology`
      )
    },
    enabled: enabled && !!clusterID && !!namespace,
    refetchInterval: enabled ? 30000 : false,
  })
}
