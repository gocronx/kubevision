import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { AlertCircle, Info, BarChart3 } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { NamespaceSelector } from "@/components/shared/namespace-selector"
import { ClusterUnavailable } from "@/components/shared/cluster-unavailable"
import { useCluster } from "@/hooks/use-cluster"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// API types
// ---------------------------------------------------------------------------

interface QuotaResourceEntry {
  name: string
  hard: Record<string, string>
  used: Record<string, string>
}

interface NamespaceQuotaSummary {
  namespace: string
  quotas: QuotaResourceEntry[]
}

interface QuotaSummaryResponse {
  namespaces: NamespaceQuotaSummary[]
}

// ---------------------------------------------------------------------------
// Resource quantity parsing helpers
// ---------------------------------------------------------------------------

/**
 * Parse a Kubernetes CPU quantity string into millicores.
 * Examples: "500m" -> 500, "1" -> 1000, "2.5" -> 2500
 */
function parseCPU(val: string): number {
  if (!val || val === "0") return 0
  if (val.endsWith("m")) return parseFloat(val.slice(0, -1))
  return parseFloat(val) * 1000
}

/**
 * Parse a Kubernetes memory quantity string into bytes.
 * Examples: "1Gi" -> 1073741824, "512Mi" -> 536870912, "1000" -> 1000
 */
function parseMemory(val: string): number {
  if (!val || val === "0") return 0
  const units: Record<string, number> = {
    Ki: 1024,
    Mi: 1024 ** 2,
    Gi: 1024 ** 3,
    Ti: 1024 ** 4,
    K: 1000,
    M: 1000 ** 2,
    G: 1000 ** 3,
    T: 1000 ** 4,
  }
  for (const [suffix, multiplier] of Object.entries(units)) {
    if (val.endsWith(suffix)) {
      return parseFloat(val.slice(0, -suffix.length)) * multiplier
    }
  }
  return parseFloat(val)
}

/**
 * Format bytes into a human-readable string.
 */
function formatBytes(bytes: number): string {
  if (bytes === 0) return "0"
  const units = ["B", "Ki", "Mi", "Gi", "Ti"]
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  return `${value % 1 === 0 ? value : value.toFixed(1)}${units[unitIndex]}`
}

/**
 * Format millicores into a human-readable CPU string.
 */
function formatCPU(millicores: number): string {
  if (millicores === 0) return "0"
  if (millicores % 1000 === 0) return `${millicores / 1000}`
  return `${millicores}m`
}

// ---------------------------------------------------------------------------
// Resource display configuration
// ---------------------------------------------------------------------------

interface ResourceDisplayConfig {
  label: string
  isMemory?: boolean
  isCPU?: boolean
}

const RESOURCE_DISPLAY: Record<string, ResourceDisplayConfig> = {
  "cpu": { label: "CPU", isCPU: true },
  "limits.cpu": { label: "CPU Limits", isCPU: true },
  "requests.cpu": { label: "CPU Requests", isCPU: true },
  "memory": { label: "Memory", isMemory: true },
  "limits.memory": { label: "Memory Limits", isMemory: true },
  "requests.memory": { label: "Memory Requests", isMemory: true },
  "pods": { label: "Pods" },
  "services": { label: "Services" },
  "configmaps": { label: "ConfigMaps" },
  "secrets": { label: "Secrets" },
  "persistentvolumeclaims": { label: "PVCs" },
  "requests.storage": { label: "Storage Requests", isMemory: true },
  "count/deployments.apps": { label: "Deployments" },
  "count/replicasets.apps": { label: "ReplicaSets" },
  "count/statefulsets.apps": { label: "StatefulSets" },
}

function getResourceLabel(key: string): string {
  return RESOURCE_DISPLAY[key]?.label ?? key
}

function parseQuantity(key: string, val: string): number {
  const config = RESOURCE_DISPLAY[key]
  if (config?.isCPU) return parseCPU(val)
  if (config?.isMemory) return parseMemory(val)
  return parseFloat(val) || 0
}

function formatQuantity(key: string, quantity: number): string {
  const config = RESOURCE_DISPLAY[key]
  if (config?.isCPU) return formatCPU(quantity)
  if (config?.isMemory) return formatBytes(quantity)
  return String(quantity)
}

// ---------------------------------------------------------------------------
// Color coding for progress bars
// ---------------------------------------------------------------------------

function getProgressBarClass(pct: number): string {
  // Override the default bg-primary set by the progress bar indicator
  if (pct >= 90) return "[&>[data-slot=progress-indicator]]:bg-red-500"
  if (pct >= 70) return "[&>[data-slot=progress-indicator]]:bg-yellow-500"
  return "[&>[data-slot=progress-indicator]]:bg-green-500"
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

interface QuotaResourceCardProps {
  resourceKey: string
  used: string
  hard: string
}

function QuotaResourceCard({ resourceKey, used, hard }: QuotaResourceCardProps) {
  const hardNum = parseQuantity(resourceKey, hard)
  const usedNum = parseQuantity(resourceKey, used)
  const pct = hardNum > 0 ? Math.min(100, (usedNum / hardNum) * 100) : 0

  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-foreground">
          {getResourceLabel(resourceKey)}
        </span>
        <span
          className={`text-xs font-semibold ${
            pct >= 90
              ? "text-red-500"
              : pct >= 70
              ? "text-yellow-500"
              : "text-green-600"
          }`}
        >
          {pct.toFixed(0)}%
        </span>
      </div>
      <Progress
        value={pct}
        max={100}
        className={`h-2 ${getProgressBarClass(pct)}`}
      />
      <p className="text-xs text-muted-foreground">
        {formatQuantity(resourceKey, usedNum)} / {formatQuantity(resourceKey, hardNum)}
        {" "}
        <span className="opacity-70">used</span>
      </p>
    </div>
  )
}

interface SingleQuotaCardProps {
  quota: QuotaResourceEntry
}

function SingleQuotaCard({ quota }: SingleQuotaCardProps) {
  const resourceKeys = Object.keys(quota.hard)

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">{quota.name}</CardTitle>
        <CardDescription>
          {resourceKeys.length} resource{resourceKeys.length !== 1 ? "s" : ""} constrained
        </CardDescription>
      </CardHeader>
      <CardContent>
        {resourceKeys.length === 0 ? (
          <p className="text-sm text-muted-foreground">No resource limits defined.</p>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {resourceKeys.map((key) => (
              <QuotaResourceCard
                key={key}
                resourceKey={key}
                used={quota.used[key] ?? "0"}
                hard={quota.hard[key] ?? "0"}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Loading skeleton
// ---------------------------------------------------------------------------

function QuotaOverviewSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      {[0, 1].map((i) => (
        <Card key={i}>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-48" />
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {[0, 1, 2, 3].map((j) => (
                <div key={j} className="flex flex-col gap-2 rounded-lg border p-4">
                  <div className="flex justify-between">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-4 w-8" />
                  </div>
                  <Skeleton className="h-2 w-full" />
                  <Skeleton className="h-3 w-28" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Data fetching hook
// ---------------------------------------------------------------------------

function useQuotaSummary(clusterID: string, namespace: string, enabled: boolean) {
  return useQuery<QuotaSummaryResponse>({
    queryKey: ["quota-summary", clusterID, namespace],
    queryFn: async () => {
      const params: Record<string, string> = {}
      if (namespace) params.namespace = namespace
      const res = await api.get(`/clusters/${clusterID}/quota-summary`, { params })
      return res as unknown as QuotaSummaryResponse
    },
    enabled: enabled && !!clusterID,
    refetchInterval: enabled ? 30_000 : false,
  })
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function QuotaOverview() {
  const { t } = useTranslation()
  const {
    currentCluster,
    selectedCluster,
    isClusterHealthy,
    refetchClusters,
  } = useCluster()
  const { user } = useAuth()
  const [namespace, setNamespace] = useState("")

  const { data, isLoading, isError, error } = useQuotaSummary(
    currentCluster,
    namespace,
    isClusterHealthy
  )

  const namespaceSummaries = data?.namespaces ?? []
  const totalQuotas = namespaceSummaries.reduce((acc, ns) => acc + ns.quotas.length, 0)

  if (selectedCluster?.status === "unhealthy") {
    return (
      <div className="flex h-full flex-col gap-4">
        <h1 className="text-2xl font-bold tracking-tight">{t("quota.title")}</h1>
        <ClusterUnavailable
          cluster={selectedCluster}
          onCheckAgain={refetchClusters}
          canRemove={canAccessAdmin(user?.role ?? "")}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col gap-1">
        <div className="flex items-center gap-2">
          <BarChart3 className="size-6 text-muted-foreground" />
          <h1 className="text-2xl font-bold tracking-tight">
            {t("quota.title")}
          </h1>
        </div>
        <p className="text-sm text-muted-foreground">
          {t("quota.description")}
        </p>
      </div>

      {/* Namespace selector */}
      <div className="flex items-center gap-3">
        <NamespaceSelector
          clusterID={currentCluster}
          value={namespace}
          onChange={setNamespace}
        />
        {!isLoading && !isError && (
          <span className="text-sm text-muted-foreground">
            {totalQuotas === 0
              ? t("quota.noQuotas")
              : `${totalQuotas} quota${totalQuotas !== 1 ? "s" : ""} found`}
          </span>
        )}
      </div>

      {/* Content */}
      {isLoading && <QuotaOverviewSkeleton />}

      {isError && (
        <div className="flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />
          <span>
            {error instanceof Error
              ? error.message
              : "Failed to load quota data."}
          </span>
        </div>
      )}

      {!isLoading && !isError && totalQuotas === 0 && (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16 text-center">
          <Info className="size-10 text-muted-foreground" />
          <div>
            <p className="font-medium">{t("quota.noQuotasDefined")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {namespace
                ? t("quota.noQuotasInNamespace", { namespace })
                : t("quota.noQuotasInCluster")}
            </p>
          </div>
        </div>
      )}

      {!isLoading &&
        !isError &&
        namespaceSummaries.map((ns) => {
          if (ns.quotas.length === 0) return null
          return (
            <div key={ns.namespace} className="flex flex-col gap-3">
              {/* Only show namespace label when showing all namespaces */}
              {!namespace && (
                <h2 className="text-lg font-semibold tracking-tight">
                  Namespace: <span className="text-primary">{ns.namespace}</span>
                </h2>
              )}
              {ns.quotas.map((quota) => (
                <SingleQuotaCard key={quota.name} quota={quota} />
              ))}
            </div>
          )
        })}
    </div>
  )
}
