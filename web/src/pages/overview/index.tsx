import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  Box, Server, Network, Monitor, AlertCircle, Plus, Layers,
  Cpu, MemoryStick, Info, AlertTriangle, XCircle,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import type { LucideIcon } from "lucide-react"
import { useCluster } from "@/hooks/use-cluster"
import { AddClusterDialog } from "@/components/shared/add-cluster-dialog"
import { useState } from "react"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// API types
// ---------------------------------------------------------------------------

interface ResourceMetric {
  allocatable: number
  requests: number
  limits: number
}

interface ResourceUsage {
  cpu: ResourceMetric
  memory: ResourceMetric
}

interface EventSummary {
  type: string
  reason: string
  message: string
  objectKind: string
  objectName: string
  namespace?: string
  timestamp: string
}

interface OverviewData {
  pods: number
  runningPods: number
  deployments: number
  readyDeployments: number
  services: number
  nodes: number
  readyNodes: number
  namespaces: number
  activeNamespaces: number
  resources: ResourceUsage
  recentEvents: EventSummary[]
}

// ---------------------------------------------------------------------------
// Data fetching hook
// ---------------------------------------------------------------------------

function useOverview(clusterID: string) {
  return useQuery<OverviewData>({
    queryKey: ["overview", clusterID],
    queryFn: async () => {
      const res = await api.get(`/clusters/${clusterID}/overview`)
      return res as unknown as OverviewData
    },
    enabled: !!clusterID,
    refetchInterval: 30_000,
  })
}

// ---------------------------------------------------------------------------
// StatCard component (enhanced with ready/total subtitle)
// ---------------------------------------------------------------------------

interface StatCardProps {
  title: string
  icon: LucideIcon
  value?: number
  subtitle?: string
  subtitleColor?: string
  isLoading: boolean
  iconClass?: string
}

function getSubtitleColor(active: number, total: number): string {
  if (total === 0) return "text-muted-foreground"
  const ratio = active / total
  if (ratio >= 1) return "text-green-500"
  if (ratio >= 0.5) return "text-amber-500"
  return "text-red-500"
}

function StatCard({ title, icon: Icon, value, subtitle, subtitleColor, isLoading, iconClass }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <div className={`rounded-md p-1.5 ${iconClass ?? "bg-muted"}`}>
          <Icon className="size-4 text-foreground" />
        </div>
      </CardHeader>
      <CardContent>
        {isLoading || value === undefined ? (
          <>
            <Skeleton className="h-8 w-20" />
            <Skeleton className="mt-1 h-3 w-28" />
          </>
        ) : (
          <>
            <div className="text-3xl font-bold">{value}</div>
            {subtitle && (
              <p className={`mt-1 text-xs font-medium ${subtitleColor ?? "text-muted-foreground"}`}>{subtitle}</p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Resource utilization bar
// ---------------------------------------------------------------------------

function getBarColorClass(pct: number): string {
  if (pct >= 90) return "[&>[data-slot=progress-indicator]]:bg-red-500"
  if (pct >= 60) return "[&>[data-slot=progress-indicator]]:bg-yellow-500"
  return "[&>[data-slot=progress-indicator]]:bg-blue-500"
}

function getTextColorClass(pct: number): string {
  if (pct >= 90) return "text-red-500"
  if (pct >= 60) return "text-yellow-500"
  return "text-blue-500"
}

function formatCPU(millicores: number): string {
  if (millicores === 0) return "0"
  if (millicores >= 1000) {
    const cores = millicores / 1000
    return cores % 1 === 0 ? `${cores} cores` : `${cores.toFixed(1)} cores`
  }
  return `${millicores}m`
}

function formatMemory(bytes: number): string {
  if (bytes === 0) return "0"
  const units = ["B", "Ki", "Mi", "Gi", "Ti"]
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  return `${value % 1 === 0 ? value : value.toFixed(1)} ${units[unitIndex]}`
}

interface ResourceBarProps {
  label: string
  value: number
  total: number
  formatFn: (v: number) => string
  /** Force a specific color variant instead of the dynamic threshold-based color. */
  forceColor?: "red" | "blue" | "yellow"
}

const forceBarColorMap: Record<string, string> = {
  red: "[&>[data-slot=progress-indicator]]:bg-red-500",
  blue: "[&>[data-slot=progress-indicator]]:bg-blue-500",
  yellow: "[&>[data-slot=progress-indicator]]:bg-yellow-500",
}
const forceTextColorMap: Record<string, string> = {
  red: "text-red-500",
  blue: "text-blue-500",
  yellow: "text-yellow-500",
}

function ResourceBar({ label, value, total, formatFn, forceColor }: ResourceBarProps) {
  const pct = total > 0 ? Math.min(100, (value / total) * 100) : 0
  const barClass = forceColor ? forceBarColorMap[forceColor] : getBarColorClass(pct)
  const textClass = forceColor ? forceTextColorMap[forceColor] : getTextColorClass(pct)
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className={`font-medium ${textClass}`}>
          {pct.toFixed(0)}%
        </span>
      </div>
      <Progress
        value={pct}
        max={100}
        className={`h-2 ${barClass}`}
      />
      <p className="text-xs text-muted-foreground">
        {formatFn(value)} / {formatFn(total)}
      </p>
    </div>
  )
}

interface ResourceUtilizationProps {
  resources: ResourceUsage
  isLoading: boolean
}

function ResourceUtilization({ resources, isLoading }: ResourceUtilizationProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-40" />
        </CardHeader>
        <CardContent className="grid gap-6 sm:grid-cols-2">
          {[0, 1].map((i) => (
            <div key={i} className="flex flex-col gap-3">
              <Skeleton className="h-4 w-24" />
              {[0, 1].map((j) => (
                <div key={j} className="flex flex-col gap-1.5">
                  <div className="flex justify-between">
                    <Skeleton className="h-3 w-16" />
                    <Skeleton className="h-3 w-8" />
                  </div>
                  <Skeleton className="h-2 w-full" />
                  <Skeleton className="h-3 w-28" />
                </div>
              ))}
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2 pb-4">
        <CardTitle className="text-base">{t("overview.resource_usage")}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-8 sm:grid-cols-2">
        {/* CPU */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <Cpu className="size-4 text-muted-foreground" />
            <span className="text-sm font-medium">CPU</span>
          </div>
          <ResourceBar
            label={t("overview.requests")}
            value={resources.cpu.requests}
            total={resources.cpu.allocatable}
            formatFn={formatCPU}
          />
          <ResourceBar
            label={t("overview.limits")}
            value={resources.cpu.limits}
            total={resources.cpu.allocatable}
            formatFn={formatCPU}
            forceColor="red"
          />
        </div>

        {/* Memory */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <MemoryStick className="size-4 text-muted-foreground" />
            <span className="text-sm font-medium">{t("overview.memory")}</span>
          </div>
          <ResourceBar
            label={t("overview.requests")}
            value={resources.memory.requests}
            total={resources.memory.allocatable}
            formatFn={formatMemory}
          />
          <ResourceBar
            label={t("overview.limits")}
            value={resources.memory.limits}
            total={resources.memory.allocatable}
            formatFn={formatMemory}
            forceColor="red"
          />
        </div>
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Recent events component
// ---------------------------------------------------------------------------

function getEventIcon(type: string) {
  switch (type) {
    case "Warning":
      return <AlertTriangle className="size-3.5 text-yellow-500" />
    case "Error":
      return <XCircle className="size-3.5 text-red-500" />
    default:
      return <Info className="size-3.5 text-blue-500" />
  }
}

function getEventBadgeClass(type: string): string {
  switch (type) {
    case "Warning":
      return "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
    case "Error":
      return "bg-red-500/10 text-red-600 dark:text-red-400"
    default:
      return "bg-blue-500/10 text-blue-600 dark:text-blue-400"
  }
}

function formatRelativeTime(timestamp: string): string {
  if (!timestamp) return ""
  const now = Date.now()
  const then = new Date(timestamp).getTime()
  const diffMs = now - then
  if (isNaN(diffMs) || diffMs < 0) return ""

  const seconds = Math.floor(diffMs / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

interface RecentEventsProps {
  events: EventSummary[]
  isLoading: boolean
}

function RecentEvents({ events, isLoading }: RecentEventsProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-3">
            {[0, 1, 2, 3, 4].map((i) => (
              <div key={i} className="flex items-start gap-3">
                <Skeleton className="size-4 shrink-0 rounded" />
                <div className="flex flex-1 flex-col gap-1">
                  <Skeleton className="h-3 w-3/4" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
                <Skeleton className="h-3 w-12" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-4">
        <CardTitle className="text-base">{t("overview.recent_events")}</CardTitle>
      </CardHeader>
      <CardContent>
        {events.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {t("overview.no_events")}
          </p>
        ) : (
          <div className="flex max-h-[320px] flex-col gap-3 overflow-y-auto pr-1">
            {events.map((ev, i) => (
              <div
                key={`${ev.timestamp}-${ev.objectName}-${i}`}
                className="flex items-start gap-3 rounded-md border p-3"
              >
                <div className="mt-0.5 shrink-0">{getEventIcon(ev.type)}</div>
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <div className="flex items-center gap-2">
                    <span
                      className={`inline-flex rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase ${getEventBadgeClass(
                        ev.type,
                      )}`}
                    >
                      {ev.type}
                    </span>
                    <span className="text-xs font-medium">{ev.reason}</span>
                  </div>
                  <p className="text-xs text-muted-foreground line-clamp-2">
                    {ev.message}
                  </p>
                  <p className="text-[11px] text-muted-foreground/70">
                    {ev.objectKind}
                    {ev.namespace ? `/${ev.namespace}` : ""}/{ev.objectName}
                  </p>
                </div>
                <span className="shrink-0 text-[11px] text-muted-foreground/70">
                  {formatRelativeTime(ev.timestamp)}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Loading skeleton for the whole page
// ---------------------------------------------------------------------------

function OverviewSkeleton() {
  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="size-7 rounded-md" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-20" />
              <Skeleton className="mt-1 h-3 w-28" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export function OverviewPage() {
  const { t } = useTranslation()
  const { currentCluster, clusters, isLoading: clustersLoading } = useCluster()
  const { data, isLoading, isError, error } = useOverview(currentCluster)
  const [showAddCluster, setShowAddCluster] = useState(false)

  const noClusters = !clustersLoading && clusters.length === 0

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold tracking-tight">{t("overview.title")}</h1>

      {clustersLoading ? (
        <OverviewSkeleton />
      ) : noClusters ? (
        <>
          <Card>
            <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
              <div className="rounded-full bg-muted p-4">
                <Server className="size-8 text-muted-foreground" />
              </div>
              <div>
                <h2 className="text-lg font-semibold">{t("overview.no_cluster")}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{t("overview.no_cluster_desc")}</p>
              </div>
              <Button onClick={() => setShowAddCluster(true)} className="mt-2">
                <Plus className="mr-2 size-4" />
                {t("overview.add_cluster")}
              </Button>
            </CardContent>
          </Card>
          <AddClusterDialog open={showAddCluster} onOpenChange={setShowAddCluster} />
        </>
      ) : (
        <>
          {isError && (
            <div className="flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
              <AlertCircle className="size-4 shrink-0" />
              <span>
                {error instanceof Error
                  ? error.message
                  : t("overview.error")}
              </span>
            </div>
          )}

          {/* Stat cards - 5 columns */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <StatCard
              title={t("overview.nodes")}
              icon={Monitor}
              value={data?.nodes}
              subtitle={
                data
                  ? data.readyNodes >= data.nodes
                    ? t("overview.all_ready")
                    : t("overview.ready_count", { ready: data.readyNodes, total: data.nodes })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.readyNodes, data.nodes) : undefined}
              isLoading={isLoading}
              iconClass="bg-blue-500/10"
            />
            <StatCard
              title={t("overview.pods")}
              icon={Box}
              value={data?.pods}
              subtitle={
                data
                  ? data.runningPods >= data.pods
                    ? t("overview.all_running")
                    : t("overview.running_count", { running: data.runningPods, total: data.pods })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.runningPods, data.pods) : undefined}
              isLoading={isLoading}
              iconClass="bg-green-500/10"
            />
            <StatCard
              title={t("overview.deployments")}
              icon={Server}
              value={data?.deployments}
              subtitle={
                data
                  ? data.readyDeployments >= data.deployments
                    ? t("overview.all_available")
                    : t("overview.available_count", { available: data.readyDeployments, total: data.deployments })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.readyDeployments, data.deployments) : undefined}
              isLoading={isLoading}
              iconClass="bg-purple-500/10"
            />
            <StatCard
              title={t("overview.services")}
              icon={Network}
              value={data?.services}
              subtitle={
                data
                  ? t("overview.all_active")
                  : undefined
              }
              subtitleColor={data ? "text-green-500" : undefined}
              isLoading={isLoading}
              iconClass="bg-orange-500/10"
            />
            <StatCard
              title={t("overview.namespaces")}
              icon={Layers}
              value={data?.namespaces}
              subtitle={
                data
                  ? data.activeNamespaces >= data.namespaces
                    ? t("overview.all_active")
                    : t("overview.active_count", { active: data.activeNamespaces, total: data.namespaces })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.activeNamespaces, data.namespaces) : undefined}
              isLoading={isLoading}
              iconClass="bg-teal-500/10"
            />
          </div>

          {/* Resource utilization & Recent events - 2 columns */}
          <div className="grid gap-6 lg:grid-cols-2">
            <ResourceUtilization
              resources={data?.resources ?? { cpu: { allocatable: 0, requests: 0, limits: 0 }, memory: { allocatable: 0, requests: 0, limits: 0 } }}
              isLoading={isLoading}
            />
            <RecentEvents
              events={data?.recentEvents ?? []}
              isLoading={isLoading}
            />
          </div>
        </>
      )}
    </div>
  )
}
