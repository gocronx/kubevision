import { Cpu, MemoryStick } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import type { ContainerMetrics, PodMetrics } from "@/hooks/use-resource"
import { formatBytes, formatCPU, usagePercent } from "@/lib/pod-metrics"

interface Props {
  metrics?: PodMetrics
  status?: string
}

export function PodResourceMetrics({ metrics, status }: Props) {
  const { t } = useTranslation()

  return (
    <Card className="md:col-span-2">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">{t("pod.resourceUsage")}</CardTitle>
          {metrics?.timestamp && (
            <span className="text-xs text-muted-foreground">
              {t("pod.metricsUpdated", { time: new Date(metrics.timestamp).toLocaleTimeString() })}
            </span>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        {!metrics ? (
          <p className="text-sm text-muted-foreground">
            {status === "pending" ? t("pod.metricsPending") : t("pod.metricsUnavailable")}
          </p>
        ) : (
          <>
            <div className="grid gap-5 sm:grid-cols-2">
              <UsageSummary
                icon={Cpu}
                label={t("pod.cpu")}
                used={metrics.cpuMilli}
                request={metrics.cpuRequestMilli}
                limit={metrics.cpuLimitMilli}
                format={formatCPU}
              />
              <UsageSummary
                icon={MemoryStick}
                label={t("pod.memory")}
                used={metrics.memoryBytes}
                request={metrics.memoryRequestBytes}
                limit={metrics.memoryLimitBytes}
                format={formatBytes}
              />
            </div>
            <div className="overflow-x-auto rounded-md border">
              <table className="w-full min-w-[30rem] text-sm">
                <thead className="bg-muted/40 text-left text-xs text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 font-medium">{t("pod.container")}</th>
                    <th className="px-3 py-2 font-medium">{t("pod.cpu")}</th>
                    <th className="px-3 py-2 font-medium">{t("pod.memory")}</th>
                  </tr>
                </thead>
                <tbody>
                  {metrics.containers.map((container) => (
                    <ContainerRow key={container.name} container={container} />
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

interface SummaryProps {
  icon: typeof Cpu
  label: string
  used: number
  request?: number
  limit?: number
  format: (value: number | undefined) => string
}

function UsageSummary({ icon: Icon, label, used, request, limit, format }: SummaryProps) {
  const { t } = useTranslation()
  const percent = usagePercent(used, limit)
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-sm font-medium"><Icon className="size-4" />{label}</div>
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-xl font-semibold">{format(used)}</span>
        <span className="text-xs text-muted-foreground">
          {limit ? t("pod.ofLimit", { limit: format(limit) }) : t("pod.noLimit")}
        </span>
      </div>
      {percent !== undefined && <Progress value={percent} max={100} className="h-1.5" />}
      <div className="text-xs text-muted-foreground">{t("pod.requestValue", { request: format(request) })}</div>
    </div>
  )
}

function ContainerRow({ container }: { container: ContainerMetrics }) {
  const { t } = useTranslation()
  return (
    <tr className="border-t">
      <td className="px-3 py-2 font-medium">{container.name}</td>
      <td className="px-3 py-2">
        <div>{formatCPU(container.cpuMilli)}</div>
        <div className="text-xs text-muted-foreground">{container.cpuLimitMilli ? t("pod.limitValue", { limit: formatCPU(container.cpuLimitMilli) }) : t("pod.noLimit")}</div>
      </td>
      <td className="px-3 py-2">
        <div>{formatBytes(container.memoryBytes)}</div>
        <div className="text-xs text-muted-foreground">{container.memoryLimitBytes ? t("pod.limitValue", { limit: formatBytes(container.memoryLimitBytes) }) : t("pod.noLimit")}</div>
      </td>
    </tr>
  )
}
