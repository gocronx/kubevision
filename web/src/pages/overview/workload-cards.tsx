import { useTranslation } from "react-i18next"
import { CalendarClock, Clock, Database, GitBranch, Globe, Network } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { OverviewData, PodStatusDist } from "./types"

const DONUT_COLORS = ["#22c55e", "#eab308", "#3b82f6", "#ef4444", "#9ca3af"]

export function PodStatusBar({ dist, total, isLoading }: { dist: PodStatusDist; total: number; isLoading: boolean }) {
  const { t } = useTranslation()
  if (isLoading) {
    return <Card><CardHeader><Skeleton className="h-5 w-40" /></CardHeader><CardContent className="flex items-center justify-center gap-6"><Skeleton className="size-28 rounded-full" /><div className="flex flex-col gap-2">{[0, 1, 2, 3, 4].map((item) => <Skeleton key={item} className="h-4 w-24" />)}</div></CardContent></Card>
  }

  const segments = [
    { label: t("overview.running"), count: dist.running, color: DONUT_COLORS[0], className: "bg-green-500" },
    { label: t("overview.pending"), count: dist.pending, color: DONUT_COLORS[1], className: "bg-yellow-500" },
    { label: t("overview.succeeded"), count: dist.succeeded, color: DONUT_COLORS[2], className: "bg-blue-500" },
    { label: t("overview.failed"), count: dist.failed, color: DONUT_COLORS[3], className: "bg-red-500" },
    { label: t("overview.unknown"), count: dist.unknown, color: DONUT_COLORS[4], className: "bg-gray-400" },
  ]
  const strokeWidth = 14
  const radius = (100 - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  let cumulativeOffset = 0
  const arcs = segments.filter((segment) => segment.count > 0).map((segment) => {
    const dashLength = (total > 0 ? segment.count / total : 0) * circumference
    const dashOffset = -cumulativeOffset
    cumulativeOffset += dashLength
    return { ...segment, dashLength, dashOffset }
  })

  return (
    <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm">
      <CardHeader className="shrink-0 p-5 pb-4"><CardTitle className="text-sm font-semibold">{t("overview.pod_status")}</CardTitle></CardHeader>
      <CardContent className="flex flex-1 items-center justify-center p-4 pt-0 sm:p-5 sm:pt-0">
        <div className="flex w-full flex-col items-center gap-6 sm:w-auto sm:flex-row sm:gap-10">
          <div className="relative aspect-square w-28 shrink-0 sm:w-[130px]">
            <svg viewBox="0 0 100 100" className="size-full">
              <circle cx={50} cy={50} r={radius} fill="none" stroke="currentColor" className="text-muted/30" strokeWidth={strokeWidth} />
              {arcs.map((arc) => <circle key={arc.label} cx={50} cy={50} r={radius} fill="none" stroke={arc.color} strokeWidth={strokeWidth} strokeDasharray={`${arc.dashLength} ${circumference - arc.dashLength}`} strokeDashoffset={arc.dashOffset} strokeLinecap="round" transform="rotate(-90 50 50)" className="transition-all duration-1000 ease-out" />)}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-3xl font-bold tracking-tight">{total}</span><span className="mt-1 text-xs font-medium text-muted-foreground">{t("overview.pods")}</span></div>
          </div>
          <div className="flex w-full min-w-0 max-w-64 flex-col gap-3 sm:min-w-[140px]">
            {segments.map((segment) => <div key={segment.label} className="flex items-center gap-2.5 text-sm"><span className={`size-3 shrink-0 rounded-full ${segment.className}`} /><span className="truncate font-medium text-muted-foreground">{segment.label}</span><span className="ml-auto shrink-0 font-semibold tabular-nums">{segment.count}</span><span className="w-10 shrink-0 text-right text-xs text-muted-foreground tabular-nums">{(total > 0 ? segment.count / total * 100 : 0).toFixed(1)}%</span></div>)}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function WorkloadSummary({ data, isLoading }: { data?: OverviewData; isLoading: boolean }) {
  const { t } = useTranslation()
  if (isLoading || !data) {
    return <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm"><CardHeader className="shrink-0 p-5 pb-4"><Skeleton className="h-5 w-32" /></CardHeader><CardContent className="flex-1 p-5 pt-0"><div className="mt-2 grid grid-cols-2 gap-4">{[0, 1, 2, 3, 4, 5].map((item) => <div key={item} className="flex items-center gap-3"><Skeleton className="size-10 shrink-0 rounded-xl" /><div className="min-w-0 space-y-1.5"><Skeleton className="h-5 w-8" /><Skeleton className="h-3 w-16" /></div></div>)}</div></CardContent></Card>
  }
  const items = [
    { label: t("overview.statefulsets"), value: data.statefulSets, icon: Database, color: "text-indigo-500", bg: "bg-indigo-500/10" },
    { label: t("overview.daemonsets"), value: data.daemonSets, icon: GitBranch, color: "text-cyan-500", bg: "bg-cyan-500/10" },
    { label: t("overview.services"), value: data.services, icon: Network, color: "text-orange-500", bg: "bg-orange-500/10" },
    { label: t("overview.ingresses"), value: data.ingresses, icon: Globe, color: "text-emerald-500", bg: "bg-emerald-500/10" },
    { label: t("overview.jobs"), value: data.jobs, icon: Clock, color: "text-amber-500", bg: "bg-amber-500/10" },
    { label: t("overview.cronjobs"), value: data.cronJobs, icon: CalendarClock, color: "text-rose-500", bg: "bg-rose-500/10" },
  ]
  return (
    <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm"><CardHeader className="shrink-0 p-5 pb-4"><CardTitle className="text-sm font-semibold">{t("overview.workloads_network", "Workloads & Network")}</CardTitle></CardHeader><CardContent className="flex-1 overflow-y-auto p-5 pt-0"><div className="mt-1 grid grid-cols-2 gap-x-4 gap-y-6">{items.map((item) => <div key={item.label} className="flex items-center gap-3"><div className={`shrink-0 rounded-xl p-2.5 ${item.bg} ${item.color}`}><item.icon className="size-4" /></div><div className="flex min-w-0 flex-col"><span className="text-xl font-bold leading-none">{item.value}</span><span className="mt-1.5 truncate text-xs text-muted-foreground" title={item.label}>{item.label}</span></div></div>)}</div></CardContent></Card>
  )
}

export function OverviewSkeleton() {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden pb-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">{Array.from({ length: 4 }).map((_, item) => <Card key={item} className="gap-2 rounded-lg border-border/40 py-0 shadow-sm"><CardHeader className="flex flex-row items-center justify-between px-4 pt-3"><Skeleton className="h-3 w-14" /><Skeleton className="size-7 rounded-md" /></CardHeader><CardContent className="px-4 pb-3"><Skeleton className="h-7 w-12" /><Skeleton className="mt-1.5 h-3 w-20" /></CardContent></Card>)}</div>
      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-12 lg:grid-rows-2"><Skeleton className="h-full min-h-0 rounded-2xl lg:col-span-4" /><Skeleton className="h-full min-h-0 rounded-2xl lg:col-span-4" /><Skeleton className="h-full min-h-0 rounded-2xl lg:col-span-4 lg:row-span-2" /><Skeleton className="h-full min-h-0 rounded-2xl lg:col-span-4" /><Skeleton className="h-full min-h-0 rounded-2xl lg:col-span-4" /></div>
    </div>
  )
}
