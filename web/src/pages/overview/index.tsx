import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  Box, Server, Network, Monitor, AlertCircle, Plus, Layers,
  Cpu, MemoryStick,
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
import { ClusterUnavailable } from "@/components/shared/cluster-unavailable"
import { useState } from "react"
import api from "@/lib/api"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"

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

function useOverview(clusterID: string, enabled: boolean, refetchInterval: number | false) {
  return useQuery<OverviewData>({
    queryKey: ["overview", clusterID],
    queryFn: async () => {
      const res = await api.get(`/clusters/${clusterID}/overview`)
      return res as unknown as OverviewData
    },
    enabled: enabled && !!clusterID,
    refetchInterval: enabled ? refetchInterval : false,
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
    <Card className="flex h-full flex-col gap-2 rounded-lg border-border/40 py-0 shadow-sm transition-shadow duration-200 hover:shadow-md">
      <CardHeader className="flex flex-row items-center justify-between px-4 pt-3">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        <div className={`rounded-md p-1.5 ${iconClass ?? "bg-secondary text-secondary-foreground"}`}>
          <Icon className="size-4 opacity-80" />
        </div>
      </CardHeader>
      <CardContent className="px-4 pb-3">
        {isLoading || value === undefined ? (
          <div className="space-y-1.5">
            <Skeleton className="h-7 w-14" />
            <Skeleton className="h-3 w-24" />
          </div>
        ) : (
          <div className="flex flex-col gap-0.5">
            <div className="text-2xl font-bold">{value}</div>
            {subtitle && (
              <p className={`text-xs font-medium ${subtitleColor ?? "text-muted-foreground"}`}>{subtitle}</p>
            )}
          </div>
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
        <span className="text-muted-foreground font-medium">{label}</span>
        <span className={`font-semibold ${textClass}`}>
          {pct.toFixed(0)}%
        </span>
      </div>
      <Progress
        value={pct}
        max={100}
        className={`h-2.5 rounded-full ${barClass}`}
      />
      <p className="text-xs text-muted-foreground mt-0.5">
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
    <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
      <CardHeader className="flex flex-row items-center gap-2 p-5 pb-4">
        <CardTitle className="text-sm font-semibold">{t("overview.resource_usage")}</CardTitle>
      </CardHeader>
      <CardContent className="grid flex-1 gap-8 p-5 pt-0 sm:grid-cols-2 overflow-y-auto">
        {/* CPU */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <Cpu className="size-4 text-muted-foreground" />
            <span className="text-sm font-semibold">CPU</span>
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
            <span className="text-sm font-semibold">{t("overview.memory")}</span>
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

function getEventBadgeClass(type: string): string {
  switch (type) {
    case "Warning":
      return "bg-amber-500"
    case "Error":
      return "bg-rose-500"
    default:
      return "bg-blue-500"
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
    <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
      <CardHeader className="shrink-0 p-5 pb-3">
        <CardTitle className="text-sm font-semibold">{t("overview.recent_events")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col flex-1 min-h-0 overflow-hidden p-0">
        {events.length === 0 ? (
          <div className="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
            {t("overview.no_events")}
          </div>
        ) : (
          <div className="flex flex-col overflow-y-auto px-5 pb-5 flex-1 min-h-0 divide-y divide-border/50">
            {events.map((ev, i) => (
              <div
                key={`${ev.timestamp}-${ev.objectName}-${i}`}
                className="flex items-start gap-3 py-3"
              >
                <div className={`mt-1.5 size-2 shrink-0 rounded-full ${getEventBadgeClass(ev.type)}`} />
                <div className="min-w-0 flex-1 flex flex-col gap-0.5">
                  <span className="text-sm font-medium text-foreground truncate">
                    {ev.reason}
                  </span>
                  <span className="text-xs text-muted-foreground line-clamp-2">
                    {ev.message}
                  </span>
                </div>
                <span className="shrink-0 text-xs text-muted-foreground/70 whitespace-nowrap">
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
    <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
      <CardHeader className="flex flex-row items-center gap-2 p-5 pb-4">
        <CardTitle className="text-sm font-semibold">{t("overview.storage_overview")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-6 p-5 pt-0 overflow-y-auto">
        {/* PV stats */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <HardDrive className="size-4 text-muted-foreground" />
            <span className="text-sm font-semibold">{t("overview.pv")}</span>
            <span className="ml-auto text-xs text-muted-foreground">{pvTotal} total</span>
          </div>
          <div className="flex gap-4 text-xs">
            <span className="flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-emerald-500" />
              {t("overview.bound")}: <span className="font-medium">{data.boundPVs}</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-blue-500" />
              {t("overview.available")}: <span className="font-medium">{data.availablePVs}</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-amber-500" />
              {t("overview.released")}: <span className="font-medium">{data.releasedPVs}</span>
            </span>
          </div>
        </div>

        {/* PVC stats */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <HardDrive className="size-4 text-muted-foreground" />
            <span className="text-sm font-semibold">{t("overview.pvc")}</span>
            <span className="ml-auto text-xs text-muted-foreground">{pvcTotal} total</span>
          </div>
          <div className="flex gap-4 text-xs">
            <span className="flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-emerald-500" />
              {t("overview.bound")}: <span className="font-medium">{data.boundPVCs}</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-amber-500" />
              {t("overview.pending")}: <span className="font-medium">{data.pendingPVCs}</span>
            </span>
          </div>
        </div>

        {/* Storage capacity bar */}
        <div className="flex flex-col gap-1.5 pt-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground font-medium">{t("overview.storage_allocation")}</span>
            <span className={`font-semibold ${getTextColorClass(storagePct)}`}>
              {storagePct.toFixed(0)}%
            </span>
          </div>
          <Progress
            value={storagePct}
            max={100}
            className={`h-2.5 rounded-full ${getBarColorClass(storagePct)}`}
          />
          <p className="text-xs text-muted-foreground mt-0.5">
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
    <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
      <CardHeader className="p-5 pb-4 shrink-0">
        <CardTitle className="text-sm font-semibold">{t("overview.pod_status")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 items-center justify-center p-5 pt-0">
        <div className="flex items-center gap-10">
          {/* Donut chart */}
          <div className="relative aspect-square w-[130px] shrink-0">
            <svg viewBox={`0 0 ${vb} ${vb}`} className="size-full">
              <circle
                cx={cx}
                cy={cx}
                r={radius}
                fill="none"
                stroke="currentColor"
                className="text-muted/30"
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
                  strokeLinecap="round"
                  transform={`rotate(-90 ${cx} ${cx})`}
                  className="transition-all duration-1000 ease-out"
                />
              ))}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-3xl font-bold tracking-tight">{total}</span>
              <span className="text-xs font-medium text-muted-foreground mt-1">Pods</span>
            </div>
          </div>

          {/* Legend */}
          <div className="flex flex-col gap-3 min-w-[140px]">
            {segments.map((seg) => {
              const pct = total > 0 ? ((seg.count / total) * 100).toFixed(1) : "0.0"
              return (
                <div key={seg.label} className="flex items-center gap-2.5 text-sm">
                  <span className={`size-3 shrink-0 rounded-full ${seg.bgClass}`} />
                  <span className="truncate text-muted-foreground font-medium">{seg.label}</span>
                  <span className="ml-auto shrink-0 font-semibold tabular-nums">{seg.count}</span>
                  <span className="w-10 shrink-0 text-right text-xs text-muted-foreground tabular-nums">{pct}%</span>
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
// Workload & Network Summary
// ---------------------------------------------------------------------------

interface WorkloadSummaryProps {
  data: OverviewData
  isLoading: boolean
}

function WorkloadSummary({ data, isLoading }: WorkloadSummaryProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
        <CardHeader className="p-5 pb-4 shrink-0">
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent className="flex-1 p-5 pt-0">
          <div className="grid grid-cols-2 gap-4 mt-2">
            {[0, 1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="size-10 rounded-xl shrink-0" />
                <div className="space-y-1.5 min-w-0">
                  <Skeleton className="h-5 w-8" />
                  <Skeleton className="h-3 w-16" />
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const items = [
    { label: t("overview.statefulsets"), value: data?.statefulSets, icon: Database, color: "text-indigo-500", bg: "bg-indigo-500/10" },
    { label: t("overview.daemonsets"), value: data?.daemonSets, icon: GitBranch, color: "text-cyan-500", bg: "bg-cyan-500/10" },
    { label: t("overview.services"), value: data?.services, icon: Network, color: "text-orange-500", bg: "bg-orange-500/10" },
    { label: t("overview.ingresses"), value: data?.ingresses, icon: Globe, color: "text-emerald-500", bg: "bg-emerald-500/10" },
    { label: t("overview.jobs"), value: data?.jobs, icon: Clock, color: "text-amber-500", bg: "bg-amber-500/10" },
    { label: t("overview.cronjobs"), value: data?.cronJobs, icon: CalendarClock, color: "text-rose-500", bg: "bg-rose-500/10" },
  ]

  return (
    <Card className="flex flex-col rounded-2xl shadow-sm border-border/40 h-full">
      <CardHeader className="p-5 pb-4 shrink-0">
        <CardTitle className="text-sm font-semibold">{t("overview.workloads_network", "Workloads & Network")}</CardTitle>
      </CardHeader>
      <CardContent className="flex-1 p-5 pt-0 overflow-y-auto">
        <div className="grid grid-cols-2 gap-x-4 gap-y-6 mt-1">
          {items.map((item, i) => (
            <div key={i} className="flex items-center gap-3">
              <div className={`p-2.5 rounded-xl shrink-0 ${item.bg} ${item.color}`}>
                 <item.icon className="size-4" />
              </div>
              <div className="flex flex-col min-w-0">
                <span className="text-xl font-bold leading-none">{item.value ?? 0}</span>
                <span className="text-xs text-muted-foreground mt-1.5 truncate" title={item.label}>{item.label}</span>
              </div>
            </div>
          ))}
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
    <div className="flex flex-col gap-4 flex-1 min-h-0 overflow-hidden pb-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="gap-2 rounded-lg border-border/40 py-0 shadow-sm">
            <CardHeader className="flex flex-row items-center justify-between px-4 pt-3">
              <Skeleton className="h-3 w-14" />
              <Skeleton className="size-7 rounded-md" />
            </CardHeader>
            <CardContent className="px-4 pb-3">
              <Skeleton className="h-7 w-12" />
              <Skeleton className="mt-1.5 h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-12 lg:grid-rows-2 gap-4 flex-1 min-h-0">
        <Skeleton className="lg:col-span-4 min-h-0 h-full rounded-2xl" />
        <Skeleton className="lg:col-span-4 min-h-0 h-full rounded-2xl" />
        <Skeleton className="lg:col-span-4 lg:row-span-2 min-h-0 h-full rounded-2xl" />
        <Skeleton className="lg:col-span-4 min-h-0 h-full rounded-2xl" />
        <Skeleton className="lg:col-span-4 min-h-0 h-full rounded-2xl" />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export function OverviewPage() {
  const { t } = useTranslation()
  const {
    currentCluster,
    clusters,
    selectedCluster,
    isClusterHealthy,
    isLoading: clustersLoading,
    refetchClusters,
  } = useCluster()
  const { user } = useAuth()
  const [refreshInterval, setRefreshInterval] = useState<string>("30000")
  const interval = refreshInterval === "0" ? false as const : Number(refreshInterval)
  const { data, isLoading, isError, error, refetch, isFetching } = useOverview(
    currentCluster,
    isClusterHealthy,
    interval
  )
  const [showAddCluster, setShowAddCluster] = useState(false)

  const noClusters = !clustersLoading && clusters.length === 0

  return (
    <div className="flex h-full flex-col gap-2">
      <div className="flex shrink-0 items-center justify-between">
        <h1 className="text-lg font-bold tracking-tight">{t("overview.title")}</h1>
        {!noClusters && !clustersLoading && isClusterHealthy && (
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
      ) : selectedCluster?.status === "unhealthy" ? (
        <ClusterUnavailable
          cluster={selectedCluster}
          onCheckAgain={refetchClusters}
          canRemove={canAccessAdmin(user?.role ?? "")}
        />
      ) : (
        <div className="flex flex-col gap-4 flex-1 min-h-0 overflow-hidden pb-4">
          {isError && (
            <div className="flex shrink-0 items-center gap-3 rounded-xl border border-destructive/20 bg-destructive/10 p-5 text-sm text-destructive font-medium">
              <AlertCircle className="size-5 shrink-0" />
              <span>
                {error instanceof Error
                  ? error.message
                  : t("overview.error")}
              </span>
            </div>
          )}

          {/* Top KPI row - 4 key metrics */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
              iconClass="bg-blue-500/10 text-blue-600 dark:text-blue-400"
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
              iconClass="bg-green-500/10 text-green-600 dark:text-green-400"
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
              iconClass="bg-purple-500/10 text-purple-600 dark:text-purple-400"
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
              iconClass="bg-teal-500/10 text-teal-600 dark:text-teal-400"
            />
          </div>

          {/* Main Dashboard Grid: Enforce perfectly aligned 2x2 grid for left components, full height for right */}
          <div className="grid grid-cols-1 lg:grid-cols-12 lg:grid-rows-2 gap-4 flex-1 min-h-0">
            {/* Row 1, Col 1 */}
            <div className="lg:col-span-4 min-h-0 h-full">
              <ResourceUtilization
                resources={data?.resources ?? { cpu: { allocatable: 0, requests: 0, limits: 0 }, memory: { allocatable: 0, requests: 0, limits: 0 } }}
                isLoading={isLoading}
              />
            </div>

            {/* Row 1, Col 2 */}
            <div className="lg:col-span-4 min-h-0 h-full">
              <StorageOverview
                data={data ?? {} as OverviewData}
                isLoading={isLoading}
              />
            </div>

            {/* Right Column: Spans 2 rows */}
            <div className="lg:col-span-4 lg:row-span-2 min-h-0 h-full">
              <RecentEvents
                events={data?.recentEvents ?? []}
                isLoading={isLoading}
              />
            </div>

            {/* Row 2, Col 1 */}
            <div className="lg:col-span-4 min-h-0 h-full">
              <PodStatusBar
                dist={data?.podStatusDistribution ?? { running: 0, pending: 0, succeeded: 0, failed: 0, unknown: 0 }}
                total={data?.pods ?? 0}
                isLoading={isLoading}
              />
            </div>

            {/* Row 2, Col 2 */}
            <div className="lg:col-span-4 min-h-0 h-full">
              <WorkloadSummary
                data={data ?? {} as OverviewData}
                isLoading={isLoading}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
