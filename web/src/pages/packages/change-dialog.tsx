import { useEffect, useMemo, useRef, useState } from "react"
import { AlertCircle, AlertTriangle, Loader2, PackagePlus, ShieldAlert } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  usePackageChange,
  type PackageChangeInput,
  type PackagePreview,
} from "@/hooks/use-package-releases"
import type { Operation } from "@/hooks/use-operations"
import { PackageValuesEditor } from "./package-values-editor"

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  cluster: string
  operation: "install" | "upgrade"
  releaseName?: string
  namespace?: string
  chart?: string
  version?: string
  source?: PackageChangeInput["source"]
  initialValues?: Record<string, unknown>
  autoPreview?: boolean
  onSuccess?: (operation: Operation) => void
}

interface ChangeForm {
  releaseName: string
  namespace: string
  chart: string
  repoUrl: string
  version: string
  values: string
  createNamespace: boolean
  wait: boolean
  atomic: boolean
  timeoutSeconds: string
}

export function PackageChangeDialog({
  open,
  onOpenChange,
  ...props
}: Props) {
  if (!open) return <Dialog open={false} onOpenChange={onOpenChange} />

  const sessionKey = [
    props.operation,
    props.releaseName,
    props.namespace,
    props.chart,
    props.version,
    props.source?.chart,
    props.source?.repoUrl,
    props.source?.version,
    props.source?.repositoryId,
    props.source?.uploadId,
  ].join(":")

  return <PackageChangeDialogSession key={sessionKey} open onOpenChange={onOpenChange} {...props} />
}

function PackageChangeDialogSession({
  open,
  onOpenChange,
  cluster,
  operation,
  releaseName = "",
  namespace = "default",
  chart = "",
  version = "",
  source,
  initialValues,
  autoPreview = false,
  onSuccess,
}: Props) {
  const { t } = useTranslation()
  const change = usePackageChange(cluster, operation)
  const previewMutate = change.preview.mutate
  const [form, setForm] = useState<ChangeForm>(() => initialForm(releaseName, namespace, chart, version, source, initialValues))
  const [valuesMode, setValuesMode] = useState<"guided" | "json">(() => initialValues && Object.keys(initialValues).length > 0 ? "guided" : "json")
  const [preview, setPreview] = useState<PackagePreview | null>(null)
  const [operationError, setOperationError] = useState("")
  const autoPreviewRequestRef = useRef(
    autoPreview && source?.chart
      ? toRequest(initialForm(releaseName, namespace, chart, version, source, initialValues), source)
      : null,
  )

  useEffect(() => {
    const request = autoPreviewRequestRef.current
    if (!request) return
    autoPreviewRequestRef.current = null
    previewMutate(request, {
      onSuccess: setPreview,
      onError: (error) => setOperationError(errorMessage(error)),
    })
  }, [previewMutate])

  const parsedValues = useMemo(() => parseValues(form.values), [form.values])
  const set = <K extends keyof ChangeForm>(key: K, value: ChangeForm[K]) => {
    setForm((current) => ({ ...current, [key]: value }))
    setPreview(null)
    setOperationError("")
  }

  const input = (): PackageChangeInput | null => {
    if (!parsedValues) {
      toast.error(t("packages.invalidValues"))
      return null
    }
    if (form.repoUrl.trim() && form.chart.includes("://")) {
      toast.error(t("packages.repositoryChartNameRequired"))
      return null
    }
    const timeoutSeconds = Number(form.timeoutSeconds)
    if (!Number.isInteger(timeoutSeconds) || timeoutSeconds < 1 || timeoutSeconds > 1800) {
      toast.error(t("packages.invalidTimeout"))
      return null
    }
    return toRequest(form, source, parsedValues)
  }

  const runPreview = () => {
    const request = input()
    if (!request) return
    setOperationError("")
    change.preview.mutate(request, {
      onSuccess: setPreview,
      onError: (error) => setOperationError(errorMessage(error)),
    })
  }

  const execute = () => {
    const request = input()
    if (!request || !preview?.confirmationToken) return
    setOperationError("")
    change.execute.mutate(
      { ...request, confirmationToken: preview.confirmationToken },
      {
        onSuccess: (queuedOperation) => {
          toast.success(t("operations.queued"))
          onOpenChange(false)
          onSuccess?.(queuedOperation)
        },
        onError: (error) => {
          setOperationError(errorMessage(error))
          setPreview(null)
        },
      },
    )
  }

  const busy = change.preview.isPending || change.execute.isPending

  return (
    <Dialog open={open} onOpenChange={(next) => !busy && onOpenChange(next)}>
      <DialogContent
        className="max-h-[90vh] overflow-y-auto sm:max-w-3xl"
        onKeyDown={(event) => {
          if (event.key === "Enter" && event.target instanceof HTMLInputElement) event.preventDefault()
        }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <PackagePlus className="size-5 text-green-600" />
            {t(`packages.${operation}Title`)}
          </DialogTitle>
          <DialogDescription>{t("packages.changeDescription")}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("packages.releaseName")}>
            <Input aria-label={t("packages.releaseName")} value={form.releaseName} disabled={operation === "upgrade"} onChange={(event) => set("releaseName", event.target.value)} />
          </Field>
          <Field label={t("common.namespace")}>
            <Input aria-label={t("common.namespace")} value={form.namespace} disabled={operation === "upgrade"} onChange={(event) => set("namespace", event.target.value)} />
          </Field>
          <Field label={t("packages.chartReference")}>
            <Input
              aria-label={t("packages.chartReference")}
              disabled={!!source?.uploadId || !!source?.repositoryId}
              placeholder={form.repoUrl ? "gocron" : "oci://registry.example/charts/app"}
              value={form.chart || (source?.uploadId ? t("packages.uploadedChart") : "")}
              onChange={(event) => set("chart", event.target.value)}
            />
          </Field>
          <Field label={t("packages.chartVersion")}>
            <Input aria-label={t("packages.chartVersion")} placeholder="1.2.3" value={form.version} onChange={(event) => set("version", event.target.value)} />
          </Field>
          {!source && (
            <div className="sm:col-span-2">
              <Field label={t("packages.repoUrl")}>
                <Input aria-label={t("packages.repoUrl")} placeholder="https://charts.example.com" value={form.repoUrl} onChange={(event) => set("repoUrl", event.target.value)} />
              </Field>
            </div>
          )}
        </div>

        <section className="space-y-3 border-t pt-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <Label>{t("packages.values")}</Label>
            <div className="inline-flex rounded-md border p-0.5">
              <Button type="button" size="sm" variant={valuesMode === "guided" ? "secondary" : "ghost"} aria-pressed={valuesMode === "guided"} onClick={() => setValuesMode("guided")}>{t("packages.guidedMode")}</Button>
              <Button type="button" size="sm" variant={valuesMode === "json" ? "secondary" : "ghost"} aria-pressed={valuesMode === "json"} onClick={() => setValuesMode("json")}>{t("packages.jsonMode")}</Button>
            </div>
          </div>
          {valuesMode === "guided" && parsedValues ? (
            <PackageValuesEditor values={parsedValues} onChange={(values) => set("values", JSON.stringify(values, null, 2))} />
          ) : (
            <Textarea aria-label={t("packages.valuesJson")} className="min-h-48 font-mono text-xs" spellCheck={false} value={form.values} onChange={(event) => set("values", event.target.value)} />
          )}
          {valuesMode === "guided" && !parsedValues && <InlineError message={t("packages.invalidValues")} />}
        </section>

        <details className="border-t pt-3">
          <summary className="cursor-pointer text-sm font-medium">{t("packages.advancedOptions")}</summary>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <Field label={t("packages.timeoutSeconds")}>
              <Input aria-label={t("packages.timeoutSeconds")} type="number" min={1} max={1800} value={form.timeoutSeconds} onChange={(event) => set("timeoutSeconds", event.target.value)} />
            </Field>
            <div className="space-y-2 pt-1">
              <CheckOption checked={form.wait} onChange={(value) => set("wait", value)} label={t("packages.wait")} />
              <CheckOption checked={form.atomic} onChange={(value) => set("atomic", value)} label={t("packages.atomic")} />
              {operation === "install" && <CheckOption checked={form.createNamespace} onChange={(value) => set("createNamespace", value)} label={t("packages.createNamespace")} />}
            </div>
          </div>
        </details>

        {operationError && <InlineError message={operationError} />}
        {preview && <PreviewPanel preview={preview} releaseName={form.releaseName} namespace={form.namespace} />}

        <DialogFooter>
          <Button variant="outline" disabled={busy} onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          {!preview ? (
            <Button disabled={busy || !form.releaseName.trim() || !form.namespace.trim() || (!form.chart.trim() && !source?.uploadId)} onClick={runPreview}>
              {change.preview.isPending && <Loader2 className="size-4 animate-spin" />}
              {operationError ? t("packages.previewAgain") : t("packages.preview")}
            </Button>
          ) : (
            <Button disabled={busy || !preview.canExecute || !preview.confirmationToken} onClick={execute}>
              {change.execute.isPending && <Loader2 className="size-4 animate-spin" />}
              {t(`packages.confirm${operation === "install" ? "Install" : "Upgrade"}`)}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function initialForm(
  releaseName: string,
  namespace: string,
  chart: string,
  version: string,
  source?: PackageChangeInput["source"],
  initialValues?: Record<string, unknown>,
): ChangeForm {
  return {
    releaseName,
    namespace,
    chart: source?.chart ?? chart,
    repoUrl: source?.repoUrl ?? "",
    version: source?.version ?? version,
    values: JSON.stringify(initialValues ?? {}, null, 2),
    createNamespace: false,
    wait: true,
    atomic: true,
    timeoutSeconds: "300",
  }
}

function parseValues(raw: string): Record<string, unknown> | null {
  try {
    const values = JSON.parse(raw) as unknown
    return values && !Array.isArray(values) && typeof values === "object" ? values as Record<string, unknown> : null
  } catch {
    return null
  }
}

function toRequest(form: ChangeForm, source?: PackageChangeInput["source"], values = parseValues(form.values) ?? {}): PackageChangeInput {
  return {
    releaseName: form.releaseName.trim(),
    namespace: form.namespace.trim(),
    source: source
      ? { ...source, chart: source.chart || form.chart.trim(), version: form.version.trim() || source.version }
      : { chart: form.chart.trim(), repoUrl: form.repoUrl.trim() || undefined, version: form.version.trim() || undefined },
    values,
    createNamespace: form.createNamespace,
    wait: form.wait,
    atomic: form.atomic,
    timeoutSeconds: Number(form.timeoutSeconds),
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error && error.message ? error.message : "Request failed"
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="space-y-1.5"><Label>{label}</Label>{children}</div>
}

function CheckOption({ checked, onChange, label }: { checked: boolean; onChange: (checked: boolean) => void; label: string }) {
  return <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />{label}</label>
}

function InlineError({ message }: { message: string }) {
  return (
    <div className="flex gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive" role="alert">
      <AlertCircle className="mt-0.5 size-4 shrink-0" />
      <span className="break-words">{message}</span>
    </div>
  )
}

function PreviewPanel({ preview, releaseName, namespace }: { preview: PackagePreview; releaseName: string; namespace: string }) {
  const { t, i18n } = useTranslation()
  const kindCounts = preview.resources.reduce<Record<string, number>>((counts, resource) => {
    counts[resource.kind] = (counts[resource.kind] ?? 0) + 1
    return counts
  }, {})
  return (
    <section className="space-y-3 border-t pt-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <div className="font-medium">{releaseName} · {namespace}</div>
          <div className="text-xs text-muted-foreground">{preview.chart} {preview.chartVersion}{preview.appVersion ? ` · ${t("packages.appVersion")} ${preview.appVersion}` : ""}</div>
        </div>
        {!preview.canExecute && <span className="flex items-center gap-1 text-sm font-medium text-destructive"><ShieldAlert className="size-4" />{t("packages.adminRequired")}</span>}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {Object.entries(kindCounts).map(([kind, count]) => <Badge key={kind} variant="secondary">{kind} {count}</Badge>)}
        {preview.resources.length === 0 && <span className="text-sm text-muted-foreground">{t("packages.resourceCount", { count: 0 })}</span>}
      </div>
      {preview.risks.length === 0 ? (
        <p className="text-sm text-green-700 dark:text-green-400">{t("packages.noRisksDetected")}</p>
      ) : (
        <div className="space-y-2">
          {preview.risks.map((risk, index) => (
            <div className="flex gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 p-3 text-sm" key={`${risk.code}-${risk.resource}-${index}`}>
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" />
              <div><div className="font-medium">{risk.level.toUpperCase()} · {risk.resource || risk.code}</div><div className="text-muted-foreground">{risk.message}</div></div>
            </div>
          ))}
        </div>
      )}
      {preview.expiresAt && <p className="text-xs text-muted-foreground">{t("packages.previewExpires", { time: new Date(preview.expiresAt).toLocaleTimeString(i18n.language) })}</p>}
      <details><summary className="cursor-pointer text-sm font-medium">{t("packages.renderedManifest")}</summary><pre className="mt-2 max-h-64 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">{preview.manifest}</pre></details>
    </section>
  )
}
