import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { AlertTriangle, Loader2, PackagePlus, ShieldAlert } from "lucide-react"
import { toast } from "sonner"
import { usePackageChange, type PackageChangeInput, type PackagePreview } from "@/hooks/use-package-releases"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

interface Props { open: boolean; onOpenChange: (open: boolean) => void; cluster: string; operation: "install" | "upgrade"; releaseName?: string; namespace?: string; chart?: string; version?: string; source?: PackageChangeInput["source"]; initialValues?: Record<string, unknown>; autoPreview?: boolean; onSuccess?: () => void }

export function PackageChangeDialog({ open, onOpenChange, cluster, operation, releaseName = "", namespace = "default", chart = "", version = "", source, initialValues, autoPreview = false, onSuccess }: Props) {
  const { t } = useTranslation(); const change = usePackageChange(cluster, operation)
  const [form, setForm] = useState({ releaseName, namespace, chart, repoUrl: "", version, values: "{}", createNamespace: false, wait: true, atomic: true })
  const [preview, setPreview] = useState<PackagePreview | null>(null)
  useEffect(() => { if (open) { const nextForm = { releaseName, namespace, chart: source?.chart ?? chart, repoUrl: source?.repoUrl ?? "", version: source?.version ?? version, values: JSON.stringify(initialValues ?? {}, null, 2), createNamespace: false, wait: true, atomic: true }; setForm(nextForm); setPreview(null); change.preview.reset(); change.execute.reset(); if (autoPreview && source?.chart) change.preview.mutate({ releaseName: nextForm.releaseName.trim(), namespace: nextForm.namespace.trim(), source, values: initialValues ?? {}, createNamespace: false, wait: true, atomic: true, timeoutSeconds: 300 }, { onSuccess: setPreview }) } }, [open, releaseName, namespace, chart, version, source, initialValues, autoPreview]) // eslint-disable-line react-hooks/exhaustive-deps
  const set = (key: keyof typeof form, value: string | boolean) => { setForm((current) => ({ ...current, [key]: value })); setPreview(null) }
  const input = (): PackageChangeInput | null => {
    try {
      const values = JSON.parse(form.values) as unknown
      if (!values || Array.isArray(values) || typeof values !== "object") throw new Error()
      if (form.repoUrl.trim() && form.chart.includes("://")) {
        toast.error(t("packages.repositoryChartNameRequired"))
        return null
      }
      return { releaseName: form.releaseName.trim(), namespace: form.namespace.trim(), source: source ? { ...source, chart: source.chart || form.chart.trim(), version: form.version.trim() || source.version } : { chart: form.chart.trim(), repoUrl: form.repoUrl.trim() || undefined, version: form.version.trim() || undefined }, values: values as Record<string, unknown>, createNamespace: form.createNamespace, wait: form.wait, atomic: form.atomic, timeoutSeconds: 300 }
    } catch { toast.error(t("packages.invalidValues")); return null }
  }
  const runPreview = () => { const request = input(); if (request) change.preview.mutate(request, { onSuccess: setPreview }) }
  const execute = () => { const request = input(); if (!request || !preview?.confirmationToken) return; change.execute.mutate({ ...request, confirmationToken: preview.confirmationToken }, { onSuccess: () => { toast.success(t(`packages.${operation}Success`)); onOpenChange(false); onSuccess?.() }, onError: () => setPreview(null) }) }
  const busy = change.preview.isPending || change.execute.isPending
  return <Dialog open={open} onOpenChange={(next) => !busy && onOpenChange(next)}><DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl" onKeyDown={(event) => { if (event.key === "Enter" && event.target instanceof HTMLInputElement) event.preventDefault() }}>
    <DialogHeader><DialogTitle className="flex items-center gap-2"><PackagePlus className="size-5 text-green-600" />{t(`packages.${operation}Title`)}</DialogTitle><DialogDescription>{t("packages.changeDescription")}</DialogDescription></DialogHeader>
    <div className="grid gap-4 sm:grid-cols-2">
      <Field label={t("packages.releaseName")}><Input aria-label={t("packages.releaseName")} value={form.releaseName} disabled={operation === "upgrade"} onChange={(e) => set("releaseName", e.target.value)} /></Field>
      <Field label={t("common.namespace")}><Input aria-label={t("common.namespace")} value={form.namespace} disabled={operation === "upgrade"} onChange={(e) => set("namespace", e.target.value)} /></Field>
      <Field label={t("packages.chartReference")}><Input aria-label={t("packages.chartReference")} disabled={!!source?.uploadId || !!source?.repositoryId} placeholder={form.repoUrl ? "gocron" : "oci://registry.example/charts/app"} value={form.chart || (source?.uploadId ? t("packages.uploadedChart") : "")} onChange={(e) => set("chart", e.target.value)} /></Field>
      <Field label={t("packages.chartVersion")}><Input aria-label={t("packages.chartVersion")} placeholder="1.2.3" value={form.version} onChange={(e) => set("version", e.target.value)} /></Field>
      {!source && <div className="sm:col-span-2"><Field label={t("packages.repoUrl")}><Input aria-label={t("packages.repoUrl")} placeholder="https://charts.example.com" value={form.repoUrl} onChange={(e) => set("repoUrl", e.target.value)} /></Field></div>}
      <div className="sm:col-span-2"><Field label={t("packages.valuesJson")}><Textarea aria-label={t("packages.valuesJson")} className="min-h-40 font-mono text-xs" spellCheck={false} value={form.values} onChange={(e) => set("values", e.target.value)} /></Field></div>
      <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.wait} onChange={(e) => set("wait", e.target.checked)} />{t("packages.wait")}</label>
      <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.atomic} onChange={(e) => set("atomic", e.target.checked)} />{t("packages.atomic")}</label>
      {operation === "install" && <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.createNamespace} onChange={(e) => set("createNamespace", e.target.checked)} />{t("packages.createNamespace")}</label>}
    </div>
    {preview && <PreviewPanel preview={preview} />}
    <DialogFooter><Button variant="outline" disabled={busy} onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>{!preview ? <Button disabled={busy || !form.releaseName.trim() || !form.namespace.trim() || (!form.chart.trim() && !source?.uploadId)} onClick={runPreview}>{change.preview.isPending && <Loader2 className="size-4 animate-spin" />}{t("packages.preview")}</Button> : <Button disabled={busy || !preview.canExecute || !preview.confirmationToken} onClick={execute}>{change.execute.isPending && <Loader2 className="size-4 animate-spin" />}{t(`packages.confirm${operation === "install" ? "Install" : "Upgrade"}`)}</Button>}</DialogFooter>
  </DialogContent></Dialog>
}

function Field({ label, children }: { label: string; children: React.ReactNode }) { return <div className="space-y-1.5"><Label>{label}</Label>{children}</div> }
function PreviewPanel({ preview }: { preview: PackagePreview }) { const { t } = useTranslation(); return <div className="space-y-3 border-t pt-4"><div className="flex flex-wrap items-center justify-between gap-2"><div><div className="font-medium">{preview.chart} {preview.chartVersion}</div><div className="text-xs text-muted-foreground">{t("packages.resourceCount", { count: preview.resources.length })}</div></div>{!preview.canExecute && <span className="flex items-center gap-1 text-sm font-medium text-destructive"><ShieldAlert className="size-4" />{t("packages.adminRequired")}</span>}</div>{preview.risks.length > 0 && <div className="space-y-2">{preview.risks.map((risk, index) => <div className="flex gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 p-3 text-sm" key={`${risk.code}-${risk.resource}-${index}`}><AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" /><div><div className="font-medium">{risk.resource || risk.code}</div><div className="text-muted-foreground">{risk.message}</div></div></div>)}</div>}<details><summary className="cursor-pointer text-sm font-medium">{t("packages.renderedManifest")}</summary><pre className="mt-2 max-h-64 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">{preview.manifest}</pre></details></div> }
