import { useState } from "react"
import { AlertCircle, CheckCircle2, Clock3, Copy, ExternalLink, LoaderCircle, RefreshCw, RotateCcw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useOperations, useRetryOperation, type Operation } from "@/hooks/use-operations"

export function OperationsPage() {
  const { t, i18n } = useTranslation()
  const query = useOperations()
  const retry = useRetryOperation()
  const [expanded, setExpanded] = useState<string | null>(null)
  const copyRequestId = async (requestId: string) => {
    try {
      await navigator.clipboard.writeText(requestId)
      toast.success(t("operations.copied"))
    } catch {
      toast.error(t("operations.copyFailed"))
    }
  }

  return <div className="mx-auto w-full max-w-7xl space-y-5">
    <div className="flex items-center justify-between gap-3">
      <div><h1 className="text-2xl font-semibold">{t("operations.title")}</h1><p className="text-sm text-muted-foreground">{t("operations.description")}</p></div>
      <Button variant="outline" size="icon" title={t("common.refresh")} onClick={() => query.refetch()}><RefreshCw className="size-4" /></Button>
    </div>
    <div className="divide-y rounded-md border">
      {query.data?.map((item) => <div key={item.id} className="p-4">
        <button className="flex w-full items-start gap-3 text-left" onClick={() => setExpanded(expanded === item.id ? null : item.id)}>
          <StatusIcon status={item.status} />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2"><span className="font-medium">{t(`operations.actions.${item.action}`, { defaultValue: item.action })}</span><span className="break-all text-sm">{item.resourceName}</span><Badge variant="outline">{t(`operations.status.${item.status}`)}</Badge></div>
            <div className="mt-1 text-xs text-muted-foreground">{[item.cluster, item.namespace].filter(Boolean).join(" / ")} · {new Date(item.createdAt).toLocaleString(i18n.language)}</div>
            {(item.status === "queued" || item.status === "running") && <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-muted"><div className="h-full bg-blue-500 transition-all" style={{ width: `${item.progress}%` }} /></div>}
          </div>
        </button>
        {expanded === item.id && <div className="ml-8 mt-4 space-y-3 border-l pl-4 text-sm">
          <div><span className="text-muted-foreground">{t("operations.stage")}: </span>{t(`operations.stages.${item.stage}`, { defaultValue: item.stage })}</div>
          {item.errorMessage && <div className="space-y-2 rounded-md border border-destructive/40 bg-destructive/5 p-3"><div className="font-medium text-destructive">{item.errorCode}: {item.errorMessage}</div>{item.suggestions?.map((suggestion) => <div key={suggestion}>• {suggestion}</div>)}</div>}
          {item.events?.map((event) => <div key={event.id} className="flex justify-between gap-3"><span>{t(`operations.stages.${event.stage}`, { defaultValue: event.message })}</span><span className="shrink-0 text-xs text-muted-foreground">{event.progress}%</span></div>)}
          <div className="flex flex-wrap gap-2">
            {item.requestId && <Button variant="outline" size="sm" onClick={() => void copyRequestId(item.requestId!)}><Copy className="size-4" />{t("operations.copyRequestId")}</Button>}
            {item.retryable && <Button size="sm" disabled={retry.isPending} onClick={() => retry.mutate(item.id)}><RotateCcw className="size-4" />{t("common.retry")}</Button>}
            {item.rollbackAvailable && item.namespace && item.resourceName && <Button asChild variant="outline" size="sm"><Link to={`/package-releases/${item.namespace}/${item.resourceName}`}><ExternalLink className="size-4" />{t("operations.reviewRollback")}</Link></Button>}
          </div>
        </div>}
      </div>)}
      {query.isLoading && <div className="flex justify-center p-10"><LoaderCircle className="size-5 animate-spin" /></div>}
      {!query.isLoading && !query.data?.length && <div className="p-10 text-center text-sm text-muted-foreground">{t("operations.empty")}</div>}
    </div>
  </div>
}

function StatusIcon({ status }: { status: Operation["status"] }) {
  if (status === "succeeded") return <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-green-600" />
  if (status === "failed") return <AlertCircle className="mt-0.5 size-5 shrink-0 text-destructive" />
  if (status === "running") return <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin text-blue-600" />
  return <Clock3 className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
}
