import { useTranslation } from "react-i18next"
import { Cpu, MemoryStick, type LucideIcon } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import type { ResourceUsage } from "./types"
import { formatMemory } from "./utils"

interface StatCardProps {
  title: string
  icon: LucideIcon
  value?: number
  subtitle?: string
  subtitleColor?: string
  isLoading: boolean
  iconClass?: string
}

export function StatCard({ title, icon: Icon, value, subtitle, subtitleColor, isLoading, iconClass }: StatCardProps) {
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
            {subtitle && <p className={`text-xs font-medium ${subtitleColor ?? "text-muted-foreground"}`}>{subtitle}</p>}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function getBarColorClass(percent: number): string {
  if (percent >= 90) return "[&>[data-slot=progress-indicator]]:bg-red-500"
  if (percent >= 60) return "[&>[data-slot=progress-indicator]]:bg-yellow-500"
  return "[&>[data-slot=progress-indicator]]:bg-blue-500"
}

function getTextColorClass(percent: number): string {
  if (percent >= 90) return "text-red-500"
  if (percent >= 60) return "text-yellow-500"
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

interface ResourceBarProps {
  label: string
  value: number
  total: number
  format: (value: number) => string
}

function ResourceBar({ label, value, total, format }: ResourceBarProps) {
  const percent = total > 0 ? (value / total) * 100 : 0
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-muted-foreground">{label}</span>
        <span className={`font-semibold ${getTextColorClass(percent)}`}>{percent.toFixed(0)}%</span>
      </div>
      <Progress
        value={Math.min(100, Math.max(0, percent))}
        max={100}
        className={`h-2.5 rounded-full ${getBarColorClass(percent)}`}
      />
      <p className="mt-0.5 text-xs text-muted-foreground">{format(value)} / {format(total)}</p>
    </div>
  )
}

export function ResourceUtilization({ resources, isLoading }: { resources: ResourceUsage; isLoading: boolean }) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader><Skeleton className="h-5 w-40" /></CardHeader>
        <CardContent className="grid gap-6 sm:grid-cols-2">
          {[0, 1].map((column) => (
            <div key={column} className="flex flex-col gap-3">
              <Skeleton className="h-4 w-24" />
              {[0, 1].map((row) => (
                <div key={row} className="flex flex-col gap-1.5">
                  <div className="flex justify-between"><Skeleton className="h-3 w-16" /><Skeleton className="h-3 w-8" /></div>
                  <Skeleton className="h-2 w-full" /><Skeleton className="h-3 w-28" />
                </div>
              ))}
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  const dimensions = [
    { label: "CPU", icon: Cpu, metric: resources.cpu, format: formatCPU },
    { label: t("overview.memory"), icon: MemoryStick, metric: resources.memory, format: formatMemory },
  ]

  return (
    <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm">
      <CardHeader className="flex flex-row items-center gap-2 p-5 pb-4">
        <CardTitle className="text-sm font-semibold">{t("overview.resource_usage")}</CardTitle>
      </CardHeader>
      <CardContent className="grid flex-1 gap-8 overflow-y-auto p-5 pt-0 sm:grid-cols-2">
        {dimensions.map(({ label, icon: Icon, metric, format }) => (
          <div key={label} className="flex flex-col gap-4">
            <div className="flex items-center gap-2"><Icon className="size-4 text-muted-foreground" /><span className="text-sm font-semibold">{label}</span></div>
            {resources.metricsAvailable
              ? <ResourceBar label={t("overview.actual_usage")} value={metric.usage} total={metric.allocatable} format={format} />
              : <p className="text-xs text-muted-foreground">{t("overview.metrics_unavailable")}</p>}
            <ResourceBar label={t("overview.requests")} value={metric.requests} total={metric.allocatable} format={format} />
            <ResourceBar label={t("overview.limits")} value={metric.limits} total={metric.allocatable} format={format} />
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
