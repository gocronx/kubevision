/**
 * Kubernetes utility functions for formatting and extracting data
 * from unstructured K8s resource objects.
 */

/**
 * Format a timestamp into a relative age string (e.g., "5m", "2h", "3d").
 */
export function formatAge(timestamp: string): string {
  if (!timestamp) return "-"

  const now = Date.now()
  const created = new Date(timestamp).getTime()
  if (isNaN(created)) return "-"

  const diffMs = now - created
  if (diffMs < 0) return "0s"

  const seconds = Math.floor(diffMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `${days}d`
  if (hours > 0) return `${hours}h`
  if (minutes > 0) return `${minutes}m`
  return `${seconds}s`
}

/**
 * Extract a nested value from an object using a dot-separated path.
 * Supports array indexing with bracket notation (e.g., "containers[0].name").
 */
export function getNestedValue(obj: unknown, path: string): unknown {
  if (obj == null || !path) return undefined

  const parts = path.replace(/\[(\d+)]/g, ".$1").split(".")
  let current: unknown = obj

  for (const part of parts) {
    if (current == null || typeof current !== "object") return undefined
    current = (current as Record<string, unknown>)[part]
  }

  return current
}

interface K8sResource {
  metadata?: {
    name?: string
    namespace?: string
    creationTimestamp?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
    deletionTimestamp?: string
  }
  status?: {
    phase?: string
    conditions?: Array<{
      type: string
      status: string
      reason?: string
    }>
    containerStatuses?: Array<{
      ready: boolean
      restartCount: number
      state?: Record<string, unknown>
    }>
    readyReplicas?: number
    replicas?: number
    availableReplicas?: number
    updatedReplicas?: number
    currentNumberScheduled?: number
    desiredNumberScheduled?: number
    numberReady?: number
  }
  spec?: {
    type?: string
    clusterIP?: string
    ports?: Array<{
      port: number
      protocol?: string
      targetPort?: number | string
      nodePort?: number
    }>
    schedule?: string
    suspend?: boolean
    nodeName?: string
    rules?: Array<{
      host?: string
    }>
    accessModes?: string[]
    capacity?: {
      storage?: string
    }
    storageClassName?: string
    volumeName?: string
    provisioner?: string
    reclaimPolicy?: string
    volumeBindingMode?: string
    completions?: number
  }
  data?: Record<string, unknown>
  type?: string
}

/**
 * Determine the status string from a K8s resource based on its resource type.
 */
export function getResourceStatus(resource: string, item: K8sResource): string {
  if (item.metadata?.deletionTimestamp) return "Terminating"

  switch (resource) {
    case "pods": {
      const containerStatuses = item.status?.containerStatuses
      if (containerStatuses) {
        for (const cs of containerStatuses) {
          if (cs.state) {
            if (cs.state["waiting"]) {
              const waiting = cs.state["waiting"] as { reason?: string }
              if (waiting.reason) return waiting.reason
            }
            if (cs.state["terminated"]) {
              const terminated = cs.state["terminated"] as { reason?: string }
              if (terminated.reason) return terminated.reason
            }
          }
        }
      }
      return item.status?.phase ?? "Unknown"
    }
    case "deployments": {
      const available = item.status?.availableReplicas ?? 0
      const desired = item.status?.replicas ?? 0
      if (available === desired && desired > 0) return "Ready"
      if (available > 0) return "Progressing"
      return "Pending"
    }
    case "nodes": {
      const conditions = item.status?.conditions
      if (conditions) {
        const ready = conditions.find((c) => c.type === "Ready")
        if (ready) return ready.status === "True" ? "Ready" : "NotReady"
      }
      return "Unknown"
    }
    case "namespaces":
      return item.status?.phase ?? "Active"
    case "persistentvolumes":
    case "persistentvolumeclaims":
      return item.status?.phase ?? "Unknown"
    default:
      return item.status?.phase ?? "-"
  }
}

/**
 * Format a labels map into a comma-separated "key=value" string.
 */
export function formatLabels(labels: Record<string, string> | undefined | null): string {
  if (!labels || Object.keys(labels).length === 0) return "-"
  return Object.entries(labels)
    .map(([k, v]) => `${k}=${v}`)
    .join(", ")
}

/**
 * Convert a JavaScript object to a YAML-like display string.
 * This is a simple formatter that does not require a YAML library.
 */
export function toYaml(obj: unknown, indent: number = 0): string {
  const prefix = "  ".repeat(indent)

  if (obj === null || obj === undefined) {
    return "null"
  }

  if (typeof obj === "string") {
    if (obj.includes("\n") || obj.includes(":") || obj.includes("#") || obj.includes("'") || obj.includes('"')) {
      return `"${obj.replace(/\\/g, "\\\\").replace(/"/g, '\\"').replace(/\n/g, "\\n")}"`
    }
    if (obj === "" || obj === "true" || obj === "false" || !isNaN(Number(obj))) {
      return `"${obj}"`
    }
    return obj
  }

  if (typeof obj === "number" || typeof obj === "boolean") {
    return String(obj)
  }

  if (Array.isArray(obj)) {
    if (obj.length === 0) return "[]"
    const items = obj.map((item) => {
      if (typeof item === "object" && item !== null && !Array.isArray(item)) {
        const entries = Object.entries(item)
        if (entries.length === 0) return `${prefix}- {}`
        const [firstKey, firstVal] = entries[0]
        const firstLine = `${prefix}- ${firstKey}: ${toYaml(firstVal, indent + 2)}`
        const rest = entries.slice(1).map(([k, v]) => {
          const valStr = toYaml(v, indent + 2)
          if (typeof v === "object" && v !== null) {
            return `${prefix}  ${k}:\n${valStr}`
          }
          return `${prefix}  ${k}: ${valStr}`
        })
        return [firstLine, ...rest].join("\n")
      }
      return `${prefix}- ${toYaml(item, indent + 1)}`
    })
    return items.join("\n")
  }

  if (typeof obj === "object") {
    const entries = Object.entries(obj)
    if (entries.length === 0) return "{}"
    return entries
      .map(([key, val]) => {
        if (val === null || val === undefined) {
          return `${prefix}${key}: null`
        }
        if (typeof val === "object") {
          const nested = toYaml(val, indent + 1)
          if (Array.isArray(val) && val.length === 0) {
            return `${prefix}${key}: []`
          }
          if (!Array.isArray(val) && typeof val === "object" && Object.keys(val).length === 0) {
            return `${prefix}${key}: {}`
          }
          return `${prefix}${key}:\n${nested}`
        }
        return `${prefix}${key}: ${toYaml(val, indent)}`
      })
      .join("\n")
  }

  return String(obj)
}

/**
 * Extract a column value from a K8s resource based on the column key.
 * Maps well-known column keys to their paths in the K8s object structure.
 */
export function extractColumnValue(
  resource: string,
  item: K8sResource,
  columnKey: string
): string {
  switch (columnKey) {
    case "name":
      return item.metadata?.name ?? "-"
    case "namespace":
      return item.metadata?.namespace ?? "-"
    case "age":
      return formatAge(item.metadata?.creationTimestamp ?? "")
    case "status":
      return getResourceStatus(resource, item)
    case "node":
      return (item.spec?.nodeName as string) ?? "-"
    case "restarts": {
      const statuses = item.status?.containerStatuses
      if (!statuses) return "0"
      return String(statuses.reduce((sum, cs) => sum + (cs.restartCount ?? 0), 0))
    }
    case "ready": {
      if (resource === "pods") {
        const cs = item.status?.containerStatuses
        if (!cs) return "0/0"
        const readyCount = cs.filter((c) => c.ready).length
        return `${readyCount}/${cs.length}`
      }
      const readyReplicas = item.status?.readyReplicas ?? 0
      const replicas = item.status?.replicas ?? 0
      return `${readyReplicas}/${replicas}`
    }
    case "upToDate":
      return String(item.status?.updatedReplicas ?? 0)
    case "available":
      return String(item.status?.availableReplicas ?? 0)
    case "desired":
      return String(item.status?.desiredNumberScheduled ?? 0)
    case "current":
      return String(item.status?.currentNumberScheduled ?? 0)
    case "type":
      if (resource === "services") return item.spec?.type ?? "-"
      if (resource === "secrets") return item.type ?? "-"
      if (resource === "events") return (getNestedValue(item, "type") as string) ?? "-"
      return "-"
    case "clusterIP":
      return item.spec?.clusterIP ?? "-"
    case "ports": {
      const ports = item.spec?.ports
      if (!ports || ports.length === 0) return "-"
      return ports.map((p) => {
        const np = p.nodePort ? `:${p.nodePort}` : ""
        return `${p.port}${np}/${p.protocol ?? "TCP"}`
      }).join(", ")
    }
    case "schedule":
      return item.spec?.schedule ?? "-"
    case "suspend":
      return item.spec?.suspend ? "True" : "False"
    case "lastSchedule": {
      const lastScheduleTime = getNestedValue(item, "status.lastScheduleTime") as string | undefined
      return lastScheduleTime ? formatAge(lastScheduleTime) : "-"
    }
    case "hosts": {
      const rules = item.spec?.rules
      if (!rules) return "-"
      return rules.map((r) => r.host ?? "*").join(", ")
    }
    case "address": {
      const lbIngress = getNestedValue(item, "status.loadBalancer.ingress") as Array<{ ip?: string; hostname?: string }> | undefined
      if (!lbIngress || lbIngress.length === 0) return "-"
      return lbIngress.map((i) => i.ip ?? i.hostname ?? "").filter(Boolean).join(", ") || "-"
    }
    case "dataCount": {
      const data = item.data
      return String(data ? Object.keys(data).length : 0)
    }
    case "capacity": {
      if (resource === "persistentvolumes") {
        return (getNestedValue(item, "spec.capacity.storage") as string) ?? "-"
      }
      return (getNestedValue(item, "status.capacity.storage") as string) ?? "-"
    }
    case "accessModes": {
      const modes = item.spec?.accessModes
      return modes ? modes.join(", ") : "-"
    }
    case "reclaimPolicy": {
      if (resource === "storageclasses") {
        return (getNestedValue(item, "reclaimPolicy") as string) ?? "-"
      }
      return item.spec?.reclaimPolicy ?? "-"
    }
    case "volume":
      return item.spec?.volumeName ?? "-"
    case "provisioner":
      return item.spec?.provisioner ?? (getNestedValue(item, "provisioner") as string) ?? "-"
    case "volumeBindingMode":
      return item.spec?.volumeBindingMode ?? (getNestedValue(item, "volumeBindingMode") as string) ?? "-"
    case "roles": {
      const labels = item.metadata?.labels
      if (!labels) return "<none>"
      const roles = Object.keys(labels)
        .filter((k) => k.startsWith("node-role.kubernetes.io/"))
        .map((k) => k.replace("node-role.kubernetes.io/", ""))
      return roles.length > 0 ? roles.join(", ") : "<none>"
    }
    case "version":
      return (getNestedValue(item, "status.nodeInfo.kubeletVersion") as string) ?? "-"
    case "completions": {
      const succeeded = (getNestedValue(item, "status.succeeded") as number) ?? 0
      const completions = item.spec?.completions ?? 1
      return `${succeeded}/${completions}`
    }
    case "duration": {
      const startTime = getNestedValue(item, "status.startTime") as string | undefined
      const completionTime = getNestedValue(item, "status.completionTime") as string | undefined
      if (!startTime) return "-"
      const start = new Date(startTime).getTime()
      const end = completionTime ? new Date(completionTime).getTime() : Date.now()
      const diffSec = Math.floor((end - start) / 1000)
      if (diffSec < 60) return `${diffSec}s`
      if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m`
      return `${Math.floor(diffSec / 3600)}h`
    }
    case "reason":
      return (getNestedValue(item, "reason") as string) ?? "-"
    case "message":
      return (getNestedValue(item, "message") as string) ?? "-"
    case "object": {
      const kind = (getNestedValue(item, "involvedObject.kind") as string) ?? ""
      const name = (getNestedValue(item, "involvedObject.name") as string) ?? ""
      return kind && name ? `${kind}/${name}` : "-"
    }
    case "count":
      return String((getNestedValue(item, "count") as number) ?? "-")
    case "quotaStatus": {
      // Summarise hard/used for the ResourceQuota list view.
      // Format: "3 resources constrained"
      const hard = getNestedValue(item, "status.hard") as Record<string, string> | undefined
      if (!hard) return "-"
      const count = Object.keys(hard).length
      return `${count} resource${count !== 1 ? "s" : ""}`
    }
    case "limitsCount": {
      // Count the number of limit items defined in a LimitRange.
      const limits = getNestedValue(item, "spec.limits") as unknown[] | undefined
      if (!limits) return "0"
      return String(limits.length)
    }
    default:
      return String(getNestedValue(item, columnKey) ?? "-")
  }
}

/**
 * Determine whether a resource type is namespace-scoped.
 */
export function isNamespaced(resource: string): boolean {
  const clusterScoped = new Set([
    "nodes",
    "namespaces",
    "persistentvolumes",
    "storageclasses",
  ])
  return !clusterScoped.has(resource)
}
