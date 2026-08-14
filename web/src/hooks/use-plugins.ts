import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import api from "@/lib/api"

export interface PluginInfo {
  name: string
  description: string
  version: string
  type: string
  enabled: boolean
  healthy: boolean
}

export interface PluginConfig {
  id: number
  name: string
  pluginType: string
  clusterId: string
  enabled: boolean
  config: string
}

export interface PluginConfigPayload {
  enabled: boolean
  config: Record<string, string>
}

const PLUGINS_QUERY_KEY = ["plugins"] as const

export function usePlugins() {
  return useQuery<PluginInfo[]>({
    queryKey: PLUGINS_QUERY_KEY,
    queryFn: async () => {
      return (await api.get<PluginInfo[]>("/plugins")) ?? []
    },
  })
}

export function usePluginConfig(name: string) {
  return useQuery<PluginConfig>({
    queryKey: ["plugin-config", name],
    queryFn: async () => {
      return api.get<PluginConfig>(`/plugins/${name}`)
    },
    enabled: !!name,
  })
}

export function useConfigurePlugin() {
  const queryClient = useQueryClient()
  return useMutation<PluginConfig, Error, { name: string; payload: PluginConfigPayload }>({
    mutationFn: async ({ name, payload }) => {
      return api.put<PluginConfig>(`/plugins/${name}`, payload)
    },
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: PLUGINS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: ["plugin-config", name] })
      toast.success("Plugin configuration updated")
    },
  })
}

export function usePluginHealthCheck() {
  return useMutation<{ status: string }, Error, string>({
    mutationFn: async (name) => {
      return api.get<{ status: string }>(`/plugins/${name}/health`)
    },
    onSuccess: () => {
      toast.success("Plugin is healthy")
    },
    onError: () => {
      toast.error("Plugin health check failed")
    },
  })
}

// Prometheus-specific hooks
export interface PrometheusQueryResult {
  status: string
  data: unknown
}

export function usePrometheusQuery(query: string) {
  return useQuery<PrometheusQueryResult>({
    queryKey: ["prometheus-query", query],
    queryFn: async () => {
      return api.get<PrometheusQueryResult>(`/plugins/prometheus/query`, { params: { query } })
    },
    enabled: !!query,
  })
}

// Grafana-specific hooks
export interface GrafanaDashboard {
  id: number
  uid: string
  title: string
  url: string
  tags: string[]
}

export function useGrafanaDashboards() {
  return useQuery<GrafanaDashboard[]>({
    queryKey: ["grafana-dashboards"],
    queryFn: async () => {
      return (await api.get<GrafanaDashboard[]>("/plugins/grafana/dashboards")) ?? []
    },
  })
}

// ArgoCD-specific hooks
export interface ArgoCDApplication {
  name: string
  namespace: string
  project: string
  syncStatus: string
  healthStatus: string
  repoURL: string
  path: string
  targetRevision: string
  url: string
}

export function useArgoCDApplications() {
  return useQuery<ArgoCDApplication[]>({
    queryKey: ["argocd-applications"],
    queryFn: async () => {
      return (await api.get<ArgoCDApplication[]>("/plugins/argocd/applications")) ?? []
    },
  })
}
