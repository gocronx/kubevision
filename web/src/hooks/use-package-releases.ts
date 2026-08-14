import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"
import type { Operation } from "@/hooks/use-operations"

export interface PackageResource { apiVersion: string; kind: string; namespace?: string; name: string }
export interface PackageRelease {
  name: string; namespace: string; revision: number; status: string
  chart: string; chartVersion: string; appVersion: string; updatedAt: string
  notes?: string; values?: Record<string, unknown>; resources?: PackageResource[]; source?: PackageChangeInput["source"]
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
export interface PackageUpgradeCandidate {
  sourceRequired: boolean; available: boolean; currentVersion: string; latestVersion?: string; appVersion?: string; source?: PackageChangeInput["source"]
}

const base = (cluster: string) => `/clusters/${cluster}/package-releases`

export function usePackageReleases(cluster: string, namespace: string, state: string) {
  return useQuery<PackageRelease[]>({
    queryKey: ["package-releases", cluster, namespace, state],
    queryFn: () => api.get<PackageRelease[]>(base(cluster), { params: { namespace: namespace || undefined, state: state || undefined, limit: 200 } }),
    enabled: !!cluster,
  })
}

export function usePackageRelease(cluster: string, namespace: string, name: string) {
  return useQuery<PackageRelease>({ queryKey: ["package-release", cluster, namespace, name], queryFn: () => api.get<PackageRelease>(`${base(cluster)}/${namespace}/${name}`), enabled: !!cluster && !!namespace && !!name })
}

export function usePackageHistory(cluster: string, namespace: string, name: string) {
  return useQuery<PackageRelease[]>({ queryKey: ["package-history", cluster, namespace, name], queryFn: () => api.get<PackageRelease[]>(`${base(cluster)}/${namespace}/${name}/history`), enabled: !!cluster && !!namespace && !!name })
}

export function usePackageRollback(cluster: string, namespace: string, name: string) {
  const cache = useQueryClient()
  return useMutation({ mutationFn: (revision: number) => api.post<Operation>(`${base(cluster)}/${namespace}/${name}/rollback`, { revision, wait: true, atomic: true, timeoutSeconds: 300 }), onSuccess: () => {
    cache.invalidateQueries({ queryKey: ["package-releases"] })
    cache.invalidateQueries({ queryKey: ["package-release", cluster, namespace, name] })
    cache.invalidateQueries({ queryKey: ["package-history", cluster, namespace, name] })
  } })
}

export function usePackageRemove(cluster: string, namespace: string, name: string) {
  const cache = useQueryClient()
  return useMutation({ mutationFn: ({ confirmation, keepHistory }: { confirmation: string; keepHistory: boolean }) => api.delete<Operation>(`${base(cluster)}/${namespace}/${name}`, { data: { confirmation, keepHistory, wait: true, timeoutSeconds: 300 } }), onSuccess: () => cache.invalidateQueries({ queryKey: ["package-releases"] }) })
}

export function usePackageChange(cluster: string, operation: "install" | "upgrade") {
  const cache = useQueryClient()
  const preview = useMutation({ mutationFn: (input: PackageChangeInput) => api.post<PackagePreview>(`${base(cluster)}/preview/${operation}`, input, { timeout: 600_000 }) })
  const execute = useMutation({ mutationFn: (input: PackageChangeInput) => api.post<Operation>(`${base(cluster)}/${operation}`, input), onSuccess: () => cache.invalidateQueries({ queryKey: ["operations"] }) })
  return { preview, execute }
}

export function useCheckPackageUpgrade(cluster: string, namespace: string, name: string) {
  const cache = useQueryClient()
  return useMutation({
    mutationFn: (source?: PackageChangeInput["source"]) => api.post<PackageUpgradeCandidate>(`${base(cluster)}/${namespace}/${name}/check-upgrade`, source ? { source } : {}, { timeout: 60_000 }),
    onSuccess: () => cache.invalidateQueries({ queryKey: ["package-release", cluster, namespace, name] }),
  })
}
