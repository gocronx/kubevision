import { useState, useMemo, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { Box, Server, Layers, Radio, Network, GitBranch } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { NamespaceSelector } from "@/components/shared/namespace-selector"
import { useCluster } from "@/hooks/use-cluster"
import { useTopology, type TopologyData } from "@/hooks/use-topology"
import { ClusterUnavailable } from "@/components/shared/cluster-unavailable"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"

const kindIcons: Record<string, typeof Box> = {
  pods: Box,
  deployments: Server,
  statefulsets: Layers,
  daemonsets: Radio,
  replicasets: GitBranch,
  services: Network,
}

const kindColors: Record<string, string> = {
  pods: "bg-blue-500/10 text-blue-700 border-blue-200 dark:text-blue-400 dark:border-blue-800",
  deployments: "bg-green-500/10 text-green-700 border-green-200 dark:text-green-400 dark:border-green-800",
  statefulsets: "bg-purple-500/10 text-purple-700 border-purple-200 dark:text-purple-400 dark:border-purple-800",
  daemonsets: "bg-orange-500/10 text-orange-700 border-orange-200 dark:text-orange-400 dark:border-orange-800",
  replicasets: "bg-gray-500/10 text-gray-700 border-gray-200 dark:text-gray-400 dark:border-gray-800",
  services: "bg-cyan-500/10 text-cyan-700 border-cyan-200 dark:text-cyan-400 dark:border-cyan-800",
}

const relationColors: Record<string, string> = {
  owns: "border-green-400 dark:border-green-600",
  selects: "border-cyan-400 dark:border-cyan-600",
}

export function TopologyPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const {
    currentCluster,
    selectedCluster,
    isClusterHealthy,
    refetchClusters,
  } = useCluster()
  const { user } = useAuth()
  const [namespace, setNamespace] = useState("")

  const { data, isLoading } = useTopology(currentCluster, namespace, isClusterHealthy)

  // Build adjacency list for a hierarchical layout
  const { layers, edgeList } = useMemo(() => {
    if (!data || data.nodes.length === 0) {
      return { layers: [] as TopologyData["nodes"][], edgeList: data?.edges ?? [] }
    }

    // Group nodes by kind in hierarchy order
    const kindOrder = ["services", "deployments", "statefulsets", "daemonsets", "replicasets", "pods"]
    const grouped = new Map<string, TopologyData["nodes"]>()
    for (const node of data.nodes) {
      const list = grouped.get(node.kind) ?? []
      list.push(node)
      grouped.set(node.kind, list)
    }

    const layers: TopologyData["nodes"][] = []
    for (const kind of kindOrder) {
      const nodes = grouped.get(kind)
      if (nodes && nodes.length > 0) {
        layers.push(nodes)
      }
    }
    // Add any remaining kinds not in the order
    for (const [kind, nodes] of grouped) {
      if (!kindOrder.includes(kind) && nodes.length > 0) {
        layers.push(nodes)
      }
    }

    return { layers, edgeList: data.edges }
  }, [data])

  const handleNodeClick = useCallback(
    (kind: string, name: string) => {
      const params = new URLSearchParams()
      if (namespace) params.set("namespace", namespace)
      navigate(`/${kind}/${name}?${params.toString()}`)
    },
    [navigate, namespace]
  )

  if (selectedCluster?.status === "unhealthy") {
    return (
      <div className="flex h-full flex-col gap-4">
        <h1 className="text-2xl font-bold tracking-tight">{t("topology.title")}</h1>
        <ClusterUnavailable
          cluster={selectedCluster}
          onCheckAgain={refetchClusters}
          canRemove={canAccessAdmin(user?.role ?? "")}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight">{t("topology.title")}</h1>
      </div>

      <div className="flex items-center gap-3">
        <NamespaceSelector
          clusterID={currentCluster}
          value={namespace}
          onChange={setNamespace}
        />
      </div>

      {!namespace && (
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <p className="text-sm text-muted-foreground">{t("topology.noData")}</p>
          </CardContent>
        </Card>
      )}

      {namespace && isLoading && (
        <div className="flex flex-col gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full" />
          ))}
        </div>
      )}

      {namespace && !isLoading && data && (
        <div className="flex flex-col gap-6">
          {/* Summary */}
          <div className="flex flex-wrap gap-2">
            {Object.entries(
              data.nodes.reduce<Record<string, number>>((acc, n) => {
                acc[n.kind] = (acc[n.kind] ?? 0) + 1
                return acc
              }, {})
            ).map(([kind, count]) => {
              const Icon = kindIcons[kind] ?? Box
              return (
                <Badge key={kind} variant="secondary" className="gap-1.5 py-1">
                  <Icon className="size-3.5" />
                  {kind}: {count}
                </Badge>
              )
            })}
            <Badge variant="outline" className="gap-1.5 py-1">
              {t("topology.relationships")}: {edgeList.length}
            </Badge>
          </div>

          {/* Hierarchical layers */}
          {layers.map((layer, layerIdx) => {
            const kind = layer[0]?.kind ?? ""
            const Icon = kindIcons[kind] ?? Box
            const colorClass = kindColors[kind] ?? "bg-muted"

            return (
              <Card key={layerIdx}>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <Icon className="size-4" />
                    {kind.charAt(0).toUpperCase() + kind.slice(1)}
                    <Badge variant="secondary" className="ml-1">{layer.length}</Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {layer.map((node) => {
                      // Find edges for this node
                      const incomingEdges = edgeList.filter((e) => e.target === node.id)
                      const outgoingEdges = edgeList.filter((e) => e.source === node.id)
                      const hasEdges = incomingEdges.length > 0 || outgoingEdges.length > 0

                      return (
                        <button
                          key={node.id}
                          onClick={() => handleNodeClick(node.kind, node.name)}
                          className={`flex flex-col items-start gap-1 rounded-lg border px-3 py-2 text-left transition-colors hover:bg-accent ${colorClass}`}
                        >
                          <div className="flex items-center gap-1.5">
                            <span className="text-sm font-medium truncate max-w-[200px]">{node.name}</span>
                            {node.status && (
                              <Badge
                                variant={node.status === "Running" || node.status === "Available" ? "default" : "secondary"}
                                className="text-[10px] px-1 py-0"
                              >
                                {node.status}
                              </Badge>
                            )}
                          </div>
                          {hasEdges && (
                            <div className="flex gap-1 flex-wrap">
                              {incomingEdges.map((edge, i) => (
                                <span
                                  key={`in-${i}`}
                                  className={`text-[10px] border rounded px-1 ${relationColors[edge.relation] ?? ""}`}
                                >
                                  ← {edge.source.split("/")[1]}
                                </span>
                              ))}
                              {outgoingEdges.map((edge, i) => (
                                <span
                                  key={`out-${i}`}
                                  className={`text-[10px] border rounded px-1 ${relationColors[edge.relation] ?? ""}`}
                                >
                                  → {edge.target.split("/")[1]}
                                </span>
                              ))}
                            </div>
                          )}
                        </button>
                      )
                    })}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
