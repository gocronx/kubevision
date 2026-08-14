import { useTranslation } from "react-i18next"
import { HardDrive } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type { EventSummary, OverviewData } from "./types"
import { formatMemory } from "./utils"

function eventBadgeClass(type: string): string {
  if (type === "Warning") return "bg-amber-500"
  if (type === "Error") return "bg-rose-500"
  return "bg-blue-500"
}

function relativeTime(timestamp: string, locale: string): string {
  const then = new Date(timestamp).getTime()
  const seconds = Math.floor((then - Date.now()) / 1000)
  if (!timestamp || Number.isNaN(then) || seconds > 0) return ""

  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "auto" })
  if (seconds > -60) return formatter.format(seconds, "second")
  const minutes = Math.ceil(seconds / 60)
  if (minutes > -60) return formatter.format(minutes, "minute")
  const hours = Math.ceil(minutes / 60)
  if (hours > -24) return formatter.format(hours, "hour")
  return formatter.format(Math.ceil(hours / 24), "day")
}

export function RecentEvents({ events, isLoading }: { events: EventSummary[]; isLoading: boolean }) {
  const { t, i18n } = useTranslation()

  if (isLoading) {
    return (
      <Card>
        <CardHeader><Skeleton className="h-5 w-32" /></CardHeader>
        <CardContent>
          <div className="flex flex-col gap-3">
            {[0, 1, 2, 3, 4].map((item) => (
              <div key={item} className="flex items-start gap-3">
                <Skeleton className="size-4 shrink-0 rounded" />
                <div className="flex flex-1 flex-col gap-1"><Skeleton className="h-3 w-3/4" /><Skeleton className="h-3 w-1/2" /></div>
                <Skeleton className="h-3 w-12" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm">
      <CardHeader className="shrink-0 p-5 pb-3"><CardTitle className="text-sm font-semibold">{t("overview.recent_events")}</CardTitle></CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col overflow-hidden p-0">
        {events.length === 0 ? (
          <div className="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">{t("overview.no_events")}</div>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col divide-y divide-border/50 overflow-y-auto px-5 pb-5">
            {events.map((event, index) => (
              <div key={`${event.timestamp}-${event.objectName}-${index}`} className="flex items-start gap-3 py-3">
                <div className={`mt-1.5 size-2 shrink-0 rounded-full ${eventBadgeClass(event.type)}`} />
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="truncate text-sm font-medium text-foreground">{event.reason}</span>
                  <span className="line-clamp-2 text-xs text-muted-foreground">{event.message}</span>
                </div>
                <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground/70">{relativeTime(event.timestamp, i18n.language)}</span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function StorageOverview({ data, isLoading }: { data?: OverviewData; isLoading: boolean }) {
  const { t } = useTranslation()

  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader><Skeleton className="h-5 w-40" /></CardHeader>
        <CardContent className="flex flex-col gap-4">{[0, 1, 2].map((item) => <Skeleton key={item} className="h-12 w-full" />)}</CardContent>
      </Card>
    )
  }

  const allocatedStorage = data.allocatedStorageBytes ?? data.usedStorageBytes
  return (
    <Card className="flex h-full flex-col rounded-2xl border-border/40 shadow-sm">
      <CardHeader className="flex flex-row items-center gap-2 p-5 pb-4"><CardTitle className="text-sm font-semibold">{t("overview.storage_overview")}</CardTitle></CardHeader>
      <CardContent className="flex flex-1 flex-col gap-6 overflow-y-auto p-5 pt-0">
        <VolumeStatus
          icon={<HardDrive className="size-4 text-muted-foreground" />}
          label={t("overview.pv")}
          total={data.persistentVolumes}
          statuses={[
            { label: t("overview.bound"), value: data.boundPVs, color: "bg-emerald-500" },
            { label: t("overview.available"), value: data.availablePVs, color: "bg-blue-500" },
            { label: t("overview.released"), value: data.releasedPVs, color: "bg-amber-500" },
          ]}
        />
        <VolumeStatus
          icon={<HardDrive className="size-4 text-muted-foreground" />}
          label={t("overview.pvc")}
          total={data.persistentVolumeClaims}
          statuses={[
            { label: t("overview.bound"), value: data.boundPVCs, color: "bg-emerald-500" },
            { label: t("overview.pending"), value: data.pendingPVCs, color: "bg-amber-500" },
          ]}
        />
        <div className="grid grid-cols-2 gap-4 border-t border-border/50 pt-4">
          <Capacity label={t("overview.bound_capacity")} value={allocatedStorage} />
          <Capacity label={t("overview.provisioned_capacity")} value={data.totalStorageBytes} />
          <p className="col-span-2 text-xs text-muted-foreground">{t("overview.volume_usage_unavailable")}</p>
        </div>
      </CardContent>
    </Card>
  )
}

interface VolumeStatusProps {
  icon: React.ReactNode
  label: string
  total: number
  statuses: Array<{ label: string; value: number; color: string }>
}

function VolumeStatus({ icon, label, total, statuses }: VolumeStatusProps) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">{icon}<span className="text-sm font-semibold">{label}</span><span className="ml-auto text-xs text-muted-foreground">{t("overview.total_count", { total })}</span></div>
      <div className="flex flex-wrap gap-x-4 gap-y-2 text-xs">
        {statuses.map((status) => (
          <span key={status.label} className="flex items-center gap-1.5"><span className={`size-2 rounded-full ${status.color}`} />{status.label}: <span className="font-medium">{status.value}</span></span>
        ))}
      </div>
    </div>
  )
}

function Capacity({ label, value }: { label: string; value: number }) {
  return <div><p className="text-xs text-muted-foreground">{label}</p><p className="mt-1 text-sm font-semibold">{formatMemory(value)}</p></div>
}
