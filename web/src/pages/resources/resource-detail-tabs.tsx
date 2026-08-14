import { useCallback, useEffect, useRef, useState } from "react"
import { Check, Copy, Download } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { PodResourceMetrics } from "@/components/specialized/pod-resource-metrics"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import type { PodMetrics } from "@/hooks/use-resource"
import { formatAge, toYaml } from "@/lib/k8s-utils"

export interface ResourceMetadata {
  name?: string
  namespace?: string
  uid?: string
  creationTimestamp?: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  resourceVersion?: string
}

interface ResourceOverviewTabProps {
  resource: string
  namespace: string
  metadata?: ResourceMetadata
  status?: unknown
  metrics?: PodMetrics
  metricsStatus?: string
}

export function ResourceOverviewTab({
  resource,
  namespace,
  metadata,
  status,
  metrics,
  metricsStatus,
}: ResourceOverviewTabProps) {
  const { t } = useTranslation()

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {resource === "pods" && <PodResourceMetrics metrics={metrics} status={metricsStatus} />}
      <Card>
        <CardHeader><CardTitle className="text-base">{t("resource.metadata")}</CardTitle></CardHeader>
        <CardContent>
          <dl className="grid gap-3">
            <DetailRow label={t("common.name")} value={metadata?.name ?? "-"} />
            {namespace && <DetailRow label={t("common.namespace")} value={namespace} />}
            <DetailRow label={t("resource.uid")} value={metadata?.uid ?? "-"} mono />
            <DetailRow
              label={t("resource.created")}
              value={metadata?.creationTimestamp
                ? `${formatAge(metadata.creationTimestamp)} (${new Date(metadata.creationTimestamp).toLocaleString(undefined)})`
                : "-"}
            />
            <DetailRow label={t("resource.resourceVersion")} value={metadata?.resourceVersion ?? "-"} mono />
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">{t("resource.labels")}</CardTitle></CardHeader>
        <CardContent>
          {metadata?.labels && Object.keys(metadata.labels).length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {Object.entries(metadata.labels).map(([key, value]) => (
                <Badge key={key} variant="secondary" className="font-mono text-xs">
                  {key}={String(value)}
                </Badge>
              ))}
            </div>
          ) : <p className="text-sm text-muted-foreground">{t("resource.noLabels")}</p>}
        </CardContent>
      </Card>

      <Card className="md:col-span-2">
        <CardHeader><CardTitle className="text-base">{t("resource.annotations")}</CardTitle></CardHeader>
        <CardContent>
          {metadata?.annotations && Object.keys(metadata.annotations).length > 0 ? (
            <dl className="grid gap-2">
              {Object.entries(metadata.annotations).map(([key, value]) => (
                <div key={key} className="grid gap-1">
                  <dt className="font-mono text-xs text-muted-foreground break-all">{key}</dt>
                  <dd className="text-sm break-all">{value}</dd>
                </div>
              ))}
            </dl>
          ) : <p className="text-sm text-muted-foreground">{t("resource.noAnnotations")}</p>}
        </CardContent>
      </Card>

      {status != null && (
        <Card className="md:col-span-2">
          <CardHeader><CardTitle className="text-base">{t("common.status")}</CardTitle></CardHeader>
          <CardContent>
            <ScrollArea className="max-h-[300px]">
              <pre className="rounded-md bg-muted p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
                {toYaml(status)}
              </pre>
            </ScrollArea>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface ResourceYamlTabProps {
  content: string
  name: string
  namespace: string
}

export function ResourceYamlTab({ content, name, namespace }: ResourceYamlTabProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const copiedTimerRef = useRef<number | null>(null)

  useEffect(() => () => {
    if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
  }, [])

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      toast.success(t("resource.copiedToast"))
      if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
      copiedTimerRef.current = window.setTimeout(() => {
        copiedTimerRef.current = null
        setCopied(false)
      }, 2000)
    } catch {
      toast.error(t("resource.copyFailed"))
    }
  }, [content, t])

  const handleDownload = useCallback(() => {
    const blob = new Blob([content], { type: "text/yaml;charset=utf-8" })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement("a")
    anchor.href = url
    anchor.download = `${namespace ? namespace + "-" : ""}${name}.yaml`
    anchor.click()
    URL.revokeObjectURL(url)
  }, [content, name, namespace])

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle className="text-base">{t("resource.resourceYaml")}</CardTitle>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={handleDownload}>
              <Download className="size-4" />
              {t("pod.download")}
            </Button>
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
              {copied ? t("resource.copied") : t("resource.copy")}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ScrollArea className="max-h-[600px]">
          <pre className="rounded-md bg-muted p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
            {content}
          </pre>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}

function DetailRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[120px_1fr] sm:gap-2">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className={`text-sm break-all ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
    </div>
  )
}
