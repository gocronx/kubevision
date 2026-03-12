import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  Box, Server, Network, Monitor, AlertCircle, Plus, Layers,
  Cpu, MemoryStick, Info, AlertTriangle, XCircle,
  Database, GitBranch, Clock, CalendarClock, Globe, HardDrive,
  RefreshCw,
} from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
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

interface PodStatusDist {
  running: number
  pending: number
  succeeded: number
  failed: number
  unknown: number
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
  // Workloads
  statefulSets: number
  readyStatefulSets: number
  daemonSets: number
  readyDaemonSets: number
  jobs: number
  succeededJobs: number
  failedJobs: number
  cronJobs: number
  activeCronJobs: number
  ingresses: number
  // Storage
  persistentVolumes: number
  boundPVs: number
  availablePVs: number
  releasedPVs: number
  persistentVolumeClaims: number
  boundPVCs: number
  pendingPVCs: number
  totalStorageBytes: number
  usedStorageBytes: number
  // Pod status
  podStatusDistribution: PodStatusDist
}

// ---------------------------------------------------------------------------
// Data fetching hook
// ---------------------------------------------------------------------------

function useOverview(clusterID: string, refetchInterval: number | false) {
  return useQuery<OverviewData>({
    queryKey: ["overview", clusterID],
    queryFn: async () => {
      const res = await api.get(`/clusters/${clusterID}/overview`)
      return res as unknown as OverviewData
    },
    enabled: !!clusterID,
    refetchInterval,
  })
}

const REFRESH_OPTIONS = [
  { value: "0", label: "Off" },
  { value: "5000", label: "5s" },
  { value: "10000", label: "10s" },
  { value: "30000", label: "30s" },
  { value: "60000", label: "60s" },
] as const

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
    <Card className="py-0">
      <CardHeader className="flex flex-row items-center justify-between px-3 pt-2.5 pb-0.5">
        <CardTitle className="text-[11px] font-medium text-muted-foreground">{title}</CardTitle>
        <div className={`rounded p-0.5 ${iconClass ?? "bg-muted"}`}>
          <Icon className="size-3 text-foreground" />
        </div>
      </CardHeader>
      <CardContent className="px-3 pb-2.5">
        {isLoading || value === undefined ? (
          <>
            <Skeleton className="h-5 w-12" />
            <Skeleton className="mt-0.5 h-2.5 w-20" />
          </>
        ) : (
          <>
            <div className="text-xl font-bold leading-tight">{value}</div>
            {subtitle && (
              <p className={`text-[10px] font-medium leading-tight ${subtitleColor ?? "text-muted-foreground"}`}>{subtitle}</p>
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
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className={`font-medium ${textClass}`}>
          {pct.toFixed(0)}%
        </span>
      </div>
      <Progress
        value={pct}
        max={100}
        className={`h-1.5 ${barClass}`}
      />
      <p className="text-[11px] text-muted-foreground">
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
    <Card className="flex flex-col">
      <CardHeader className="flex flex-row items-center gap-2 px-4 pt-3 pb-2">
        <CardTitle className="text-sm">{t("overview.resource_usage")}</CardTitle>
      </CardHeader>
      <CardContent className="grid flex-1 gap-4 px-4 pb-3 sm:grid-cols-2">
        {/* CPU */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5">
            <Cpu className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">CPU</span>
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
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5">
            <MemoryStick className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{t("overview.memory")}</span>
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
    <Card className="flex min-h-0 flex-col gap-0 overflow-hidden py-0">
      <CardHeader className="shrink-0 px-3 pt-2 pb-1">
        <CardTitle className="text-sm">{t("overview.recent_events")}</CardTitle>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden px-3 pt-0 pb-2">
        {events.length === 0 ? (
          <p className="py-3 text-center text-xs text-muted-foreground">
            {t("overview.no_events")}
          </p>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
            {events.map((ev, i) => (
              <div
                key={`${ev.timestamp}-${ev.objectName}-${i}`}
                className="flex items-center gap-1.5 rounded border px-2 py-1"
              >
                <div className="shrink-0">{getEventIcon(ev.type)}</div>
                <span
                  className={`shrink-0 rounded px-1 py-px text-[8px] font-semibold uppercase leading-none ${getEventBadgeClass(
                    ev.type,
                  )}`}
                >
                  {ev.type}
                </span>
                <span className="min-w-0 flex-1 truncate text-[11px] text-muted-foreground">
                  <span className="font-medium text-foreground">{ev.reason}</span>{" "}
                  {ev.message}
                </span>
                <span className="shrink-0 text-[10px] text-muted-foreground/70">
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
// Storage overview component
// ---------------------------------------------------------------------------

interface StorageOverviewProps {
  data: OverviewData
  isLoading: boolean
}

function StorageOverview({ data, isLoading }: StorageOverviewProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-40" />
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </CardContent>
      </Card>
    )
  }

  const pvTotal = data.persistentVolumes
  const pvcTotal = data.persistentVolumeClaims
  const storagePct = data.totalStorageBytes > 0
    ? Math.min(100, (data.usedStorageBytes / data.totalStorageBytes) * 100)
    : 0

  return (
    <Card className="flex flex-col">
      <CardHeader className="flex flex-row items-center gap-2 px-4 pt-3 pb-2">
        <CardTitle className="text-sm">{t("overview.storage_overview")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-3 px-4 pb-3">
        {/* PV stats */}
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-1.5">
            <HardDrive className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{t("overview.pv")}</span>
            <span className="ml-auto text-[11px] text-muted-foreground">{pvTotal} total</span>
          </div>
          <div className="flex gap-3 text-[11px]">
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-green-500" />
              {t("overview.bound")}: {data.boundPVs}
            </span>
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-blue-500" />
              {t("overview.available")}: {data.availablePVs}
            </span>
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-yellow-500" />
              {t("overview.released")}: {data.releasedPVs}
            </span>
          </div>
        </div>

        {/* PVC stats */}
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-1.5">
            <HardDrive className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">{t("overview.pvc")}</span>
            <span className="ml-auto text-[11px] text-muted-foreground">{pvcTotal} total</span>
          </div>
          <div className="flex gap-3 text-[11px]">
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-green-500" />
              {t("overview.bound")}: {data.boundPVCs}
            </span>
            <span className="flex items-center gap-1">
              <span className="size-1.5 rounded-full bg-yellow-500" />
              {t("overview.pending")}: {data.pendingPVCs}
            </span>
          </div>
        </div>

        {/* Storage capacity bar */}
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">{t("overview.storage_allocation")}</span>
            <span className={`font-medium ${getTextColorClass(storagePct)}`}>
              {storagePct.toFixed(0)}%
            </span>
          </div>
          <Progress
            value={storagePct}
            max={100}
            className={`h-1.5 ${getBarColorClass(storagePct)}`}
          />
          <p className="text-[11px] text-muted-foreground">
            {t("overview.allocated")}: {formatMemory(data.usedStorageBytes)} / {t("overview.total_capacity")}: {formatMemory(data.totalStorageBytes)}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Pod status distribution — Donut chart
// ---------------------------------------------------------------------------

interface PodStatusBarProps {
  dist: PodStatusDist
  total: number
  isLoading: boolean
}

const DONUT_COLORS = ["#22c55e", "#eab308", "#3b82f6", "#ef4444", "#9ca3af"] // green, yellow, blue, red, gray

function PodStatusBar({ dist, total, isLoading }: PodStatusBarProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-40" />
        </CardHeader>
        <CardContent className="flex items-center justify-center gap-6">
          <Skeleton className="size-28 rounded-full" />
          <div className="flex flex-col gap-2">
            {[0, 1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-4 w-24" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const segments = [
    { label: t("overview.running"), count: dist.running, color: DONUT_COLORS[0], bgClass: "bg-green-500" },
    { label: t("overview.pending"), count: dist.pending, color: DONUT_COLORS[1], bgClass: "bg-yellow-500" },
    { label: t("overview.succeeded"), count: dist.succeeded, color: DONUT_COLORS[2], bgClass: "bg-blue-500" },
    { label: t("overview.failed"), count: dist.failed, color: DONUT_COLORS[3], bgClass: "bg-red-500" },
    { label: t("overview.unknown"), count: dist.unknown, color: DONUT_COLORS[4], bgClass: "bg-gray-400" },
  ]

  // SVG donut — uses viewBox so it scales with the container
  const vb = 100
  const strokeWidth = 14
  const radius = (vb - strokeWidth) / 2
  const cx = vb / 2
  const circumference = 2 * Math.PI * radius

  // Build arcs
  let cumulativeOffset = 0
  const arcs = segments
    .filter((s) => s.count > 0)
    .map((seg) => {
      const pct = total > 0 ? seg.count / total : 0
      const dashLength = pct * circumference
      const dashOffset = -cumulativeOffset
      cumulativeOffset += dashLength
      return { ...seg, dashLength, dashOffset }
    })

  return (
    <Card className="flex flex-col">
      <CardHeader className="px-3 pt-2.5 pb-1.5">
        <CardTitle className="text-sm">{t("overview.pod_status")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 items-center justify-center px-3 pb-2.5">
        <div className="flex items-center gap-4">
          {/* Donut chart */}
          <div className="relative aspect-square w-[110px] shrink-0">
            <svg viewBox={`0 0 ${vb} ${vb}`} className="size-full">
              <circle
                cx={cx}
                cy={cx}
                r={radius}
                fill="none"
                stroke="currentColor"
                className="text-muted"
                strokeWidth={strokeWidth}
              />
              {arcs.map((arc) => (
                <circle
                  key={arc.label}
                  cx={cx}
                  cy={cx}
                  r={radius}
                  fill="none"
                  stroke={arc.color}
                  strokeWidth={strokeWidth}
                  strokeDasharray={`${arc.dashLength} ${circumference - arc.dashLength}`}
                  strokeDashoffset={arc.dashOffset}
                  strokeLinecap="butt"
                  transform={`rotate(-90 ${cx} ${cx})`}
                />
              ))}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-lg font-bold leading-none">{total}</span>
              <span className="text-[9px] text-muted-foreground">Pods</span>
            </div>
          </div>

          {/* Legend */}
          <div className="flex flex-col gap-1.5 min-w-0">
            {segments.map((seg) => {
              const pct = total > 0 ? ((seg.count / total) * 100).toFixed(1) : "0.0"
              return (
                <div key={seg.label} className="flex items-center gap-1.5 text-[11px]">
                  <span className={`size-2 shrink-0 rounded-full ${seg.bgClass}`} />
                  <span className="truncate text-muted-foreground">{seg.label}</span>
                  <span className="ml-auto shrink-0 font-medium tabular-nums">{seg.count}</span>
                  <span className="w-9 shrink-0 text-right text-[10px] text-muted-foreground tabular-nums">{pct}%</span>
                </div>
              )
            })}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Loading skeleton for the whole page
// ---------------------------------------------------------------------------

function OverviewSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i} className="py-0">
            <CardHeader className="flex flex-row items-center justify-between px-3 pt-2.5 pb-0.5">
              <Skeleton className="h-2.5 w-14" />
              <Skeleton className="size-4 rounded" />
            </CardHeader>
            <CardContent className="px-3 pb-2.5">
              <Skeleton className="h-5 w-12" />
              <Skeleton className="mt-0.5 h-2.5 w-20" />
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
  const [refreshInterval, setRefreshInterval] = useState<string>("30000")
  const interval = refreshInterval === "0" ? false as const : Number(refreshInterval)
  const { data, isLoading, isError, error, refetch, isFetching } = useOverview(currentCluster, interval)
  const [showAddCluster, setShowAddCluster] = useState(false)

  const noClusters = !clustersLoading && clusters.length === 0

  return (
    <div className="flex h-full flex-col gap-2">
      <div className="flex shrink-0 items-center justify-between">
        <h1 className="text-lg font-bold tracking-tight">{t("overview.title")}</h1>
        {!noClusters && !clustersLoading && (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              className="size-8"
              onClick={() => refetch()}
              disabled={isFetching}
            >
              <RefreshCw className={`size-4 ${isFetching ? "animate-spin" : ""}`} />
            </Button>
            <Select value={refreshInterval} onValueChange={setRefreshInterval}>
              <SelectTrigger className="h-8 w-[80px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REFRESH_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

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
        <div className="flex min-h-0 flex-1 flex-col gap-2">
          {isError && (
            <div className="flex shrink-0 items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
              <AlertCircle className="size-4 shrink-0" />
              <span>
                {error instanceof Error
                  ? error.message
                  : t("overview.error")}
              </span>
            </div>
          )}

          {/* Row 1: Core resource stat cards - 5 columns */}
          <div className="grid shrink-0 gap-2 sm:grid-cols-2 lg:grid-cols-5">
            <StatCard
              title={t("overview.nodes")}
              icon={Monitor}
              value={data?.nodes}
              subtitle={
                data
                  ? data.nodes === 0
                    ? undefined
                    : data.readyNodes >= data.nodes
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
                  ? data.pods === 0
                    ? undefined
                    : data.runningPods >= data.pods
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
                  ? data.deployments === 0
                    ? undefined
                    : data.readyDeployments >= data.deployments
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
                data && data.services > 0
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
                  ? data.namespaces === 0
                    ? undefined
                    : data.activeNamespaces >= data.namespaces
                      ? t("overview.all_active")
                      : t("overview.active_count", { active: data.activeNamespaces, total: data.namespaces })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.activeNamespaces, data.namespaces) : undefined}
              isLoading={isLoading}
              iconClass="bg-teal-500/10"
            />
          </div>

          {/* Row 2: Workload stat cards - 5 columns */}
          <div className="grid shrink-0 gap-2 sm:grid-cols-2 lg:grid-cols-5">
            <StatCard
              title={t("overview.statefulsets")}
              icon={Database}
              value={data?.statefulSets}
              subtitle={
                data
                  ? data.statefulSets === 0
                    ? undefined
                    : data.readyStatefulSets >= data.statefulSets
                      ? t("overview.all_ready")
                      : t("overview.ready_count", { ready: data.readyStatefulSets, total: data.statefulSets })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.readyStatefulSets, data.statefulSets) : undefined}
              isLoading={isLoading}
              iconClass="bg-indigo-500/10"
            />
            <StatCard
              title={t("overview.daemonsets")}
              icon={GitBranch}
              value={data?.daemonSets}
              subtitle={
                data
                  ? data.daemonSets === 0
                    ? undefined
                    : data.readyDaemonSets >= data.daemonSets
                      ? t("overview.all_ready")
                      : t("overview.ready_count", { ready: data.readyDaemonSets, total: data.daemonSets })
                  : undefined
              }
              subtitleColor={data ? getSubtitleColor(data.readyDaemonSets, data.daemonSets) : undefined}
              isLoading={isLoading}
              iconClass="bg-cyan-500/10"
            />
            <StatCard
              title={t("overview.jobs")}
              icon={Clock}
              value={data?.jobs}
              subtitle={
                data
                  ? data.jobs === 0
                    ? undefined
                    : data.failedJobs === 0 && data.succeededJobs > 0
                      ? t("overview.all_succeeded")
                      : t("overview.succeeded_failed", { succeeded: data.succeededJobs, failed: data.failedJobs })
                  : undefined
              }
              subtitleColor={data ? (data.failedJobs > 0 ? "text-red-500" : "text-green-500") : undefined}
              isLoading={isLoading}
              iconClass="bg-amber-500/10"
            />
            <StatCard
              title={t("overview.cronjobs")}
              icon={CalendarClock}
              value={data?.cronJobs}
              subtitle={
                data
                  ? data.cronJobs === 0
                    ? undefined
                    : data.activeCronJobs > 0
                      ? t("overview.active_cronjobs", { active: data.activeCronJobs })
                      : t("overview.no_active")
                  : undefined
              }
              subtitleColor={data ? (data.activeCronJobs > 0 ? "text-blue-500" : "text-muted-foreground") : undefined}
              isLoading={isLoading}
              iconClass="bg-rose-500/10"
            />
            <StatCard
              title={t("overview.ingresses")}
              icon={Globe}
              value={data?.ingresses}
              subtitle={data && data.ingresses > 0 ? t("overview.all_active") : undefined}
              subtitleColor={data ? "text-green-500" : undefined}
              isLoading={isLoading}
              iconClass="bg-emerald-500/10"
            />
          </div>

          {/* Row 3: Resource Allocation + Storage Overview */}
          <div className="grid min-h-0 flex-1 grid-rows-[1fr] gap-2 overflow-hidden lg:grid-cols-2">
            <ResourceUtilization
              resources={data?.resources ?? { cpu: { allocatable: 0, requests: 0, limits: 0 }, memory: { allocatable: 0, requests: 0, limits: 0 } }}
              isLoading={isLoading}
            />
            <StorageOverview
              data={data ?? {} as OverviewData}
              isLoading={isLoading}
            />
          </div>

          {/* Row 4: Pod Status + Recent Events */}
          <div className="grid min-h-0 flex-1 grid-rows-[1fr] gap-2 overflow-hidden lg:grid-cols-2">
            <PodStatusBar
              dist={data?.podStatusDistribution ?? { running: 0, pending: 0, succeeded: 0, failed: 0, unknown: 0 }}
              total={data?.pods ?? 0}
              isLoading={isLoading}
            />
            <RecentEvents
              events={data?.recentEvents ?? []}
              isLoading={isLoading}
            />
          </div>
        </div>
      )}
    </div>
  )
}
