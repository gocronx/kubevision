import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

export interface PackageResource { apiVersion: string; kind: string; namespace?: string; name: string }
export interface PackageRelease {
  name: string; namespace: string; revision: number; status: string
  chart: string; chartVersion: string; appVersion: string; updatedAt: string
  notes?: string; values?: Record<string, unknown>; resources?: PackageResource[]
}
export interface PackageChangeInput {
  releaseName: string; namespace: string; source: { chart?: string; repoUrl?: string; version?: string; repositoryId?: number; uploadId?: string }
  values?: Record<string, unknown>; createNamespace?: boolean; wait: boolean; atomic: boolean; timeoutSeconds: number; confirmationToken?: string
}
export interface PackagePreview {
  operation: "install" | "upgrade"; chart: string; chartVersion: string; appVersion?: string; digest: string
  manifest: string; resources: PackageResource[]; risks: Array<{ level: string; code: string; message: string; resource?: string }>
  canExecute: boolean; confirmationToken?: string; expiresAt?: string
}

const base = (cluster: string) => `/clusters/${cluster}/package-releases`

export function usePackageReleases(cluster: string, namespace: string, state: string) {
  return useQuery<PackageRelease[]>({
    queryKey: ["package-releases", cluster, namespace, state],
    queryFn: async () => (await api.get(base(cluster), { params: { namespace: namespace || undefined, state: state || undefined, limit: 200 } })) as unknown as PackageRelease[],
    enabled: !!cluster,
  })
}

export function usePackageRelease(cluster: string, namespace: string, name: string) {
  return useQuery<PackageRelease>({ queryKey: ["package-release", cluster, namespace, name], queryFn: async () => (await api.get(`${base(cluster)}/${namespace}/${name}`)) as unknown as PackageRelease, enabled: !!cluster && !!namespace && !!name })
}

export function usePackageHistory(cluster: string, namespace: string, name: string) {
  return useQuery<PackageRelease[]>({ queryKey: ["package-history", cluster, namespace, name], queryFn: async () => (await api.get(`${base(cluster)}/${namespace}/${name}/history`)) as unknown as PackageRelease[], enabled: !!cluster && !!namespace && !!name })
}

export function usePackageRollback(cluster: string, namespace: string, name: string) {
  const cache = useQueryClient()
  return useMutation({ mutationFn: (revision: number) => api.post(`${base(cluster)}/${namespace}/${name}/rollback`, { revision, wait: true, atomic: true, timeoutSeconds: 300 }), onSuccess: () => {
    cache.invalidateQueries({ queryKey: ["package-releases"] })
    cache.invalidateQueries({ queryKey: ["package-release", cluster, namespace, name] })
    cache.invalidateQueries({ queryKey: ["package-history", cluster, namespace, name] })
  } })
}

export function usePackageRemove(cluster: string, namespace: string, name: string) {
  const cache = useQueryClient()
  return useMutation({ mutationFn: ({ confirmation, keepHistory }: { confirmation: string; keepHistory: boolean }) => api.delete(`${base(cluster)}/${namespace}/${name}`, { data: { confirmation, keepHistory, wait: true, timeoutSeconds: 300 } }), onSuccess: () => cache.invalidateQueries({ queryKey: ["package-releases"] }) })
}

export function usePackageChange(cluster: string, operation: "install" | "upgrade") {
  const cache = useQueryClient()
  const preview = useMutation({ mutationFn: (input: PackageChangeInput) => api.post(`${base(cluster)}/preview/${operation}`, input, { timeout: 600_000 }) as unknown as Promise<PackagePreview> })
  const execute = useMutation({ mutationFn: (input: PackageChangeInput) => api.post(`${base(cluster)}/${operation}`, input, { timeout: 600_000 }), onSuccess: () => cache.invalidateQueries({ queryKey: ["package-releases"] }) })
  return { preview, execute }
}
