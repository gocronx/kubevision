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
      const res = await api.get("/plugins")
      return (res as unknown as PluginInfo[]) ?? []
    },
  })
}

export function usePluginConfig(name: string) {
  return useQuery<PluginConfig>({
    queryKey: ["plugin-config", name],
    queryFn: async () => {
      const res = await api.get(`/plugins/${name}`)
      return res as unknown as PluginConfig
    },
    enabled: !!name,
  })
}

export function useConfigurePlugin() {
  const queryClient = useQueryClient()
  return useMutation<PluginConfig, Error, { name: string; payload: PluginConfigPayload }>({
    mutationFn: async ({ name, payload }) => {
      const res = await api.put(`/plugins/${name}`, payload)
      return res as unknown as PluginConfig
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
      const res = await api.get(`/plugins/${name}/health`)
      return res as unknown as { status: string }
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
      const res = await api.get(`/plugins/prometheus/query`, { params: { query } })
      return res as unknown as PrometheusQueryResult
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
      const res = await api.get("/plugins/grafana/dashboards")
      return (res as unknown as GrafanaDashboard[]) ?? []
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
      const res = await api.get("/plugins/argocd/applications")
      return (res as unknown as ArgoCDApplication[]) ?? []
    },
  })
}
