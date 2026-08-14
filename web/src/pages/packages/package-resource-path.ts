import type { PackageResource } from "@/hooks/use-package-releases"

const packageResourceRoutes: Record<string, string> = {
  pod: "pods",
  deployment: "deployments",
  statefulset: "statefulsets",
  daemonset: "daemonsets",
  job: "jobs",
  cronjob: "cronjobs",
  service: "services",
  ingress: "ingresses",
  configmap: "configmaps",
  secret: "secrets",
  persistentvolume: "persistentvolumes",
  persistentvolumeclaim: "persistentvolumeclaims",
  storageclass: "storageclasses",
  namespace: "namespaces",
}

export function packageResourcePath(resource: PackageResource): string | null {
  const route = packageResourceRoutes[resource.kind.toLowerCase()]
  if (!route) return null
  const path = `/${route}/${encodeURIComponent(resource.name)}`
  return resource.namespace ? `${path}?namespace=${encodeURIComponent(resource.namespace)}` : path
}
