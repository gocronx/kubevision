import { useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { PackageOpen, RefreshCw } from "lucide-react"
import { useCluster } from "@/hooks/use-cluster"
import { usePackageReleases } from "@/hooks/use-package-releases"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"

export function PackageReleasesPage() {
  const { t, i18n } = useTranslation()
  const { currentCluster } = useCluster()
  const [namespace, setNamespace] = useState("")
  const [state, setState] = useState("")
  const releases = usePackageReleases(currentCluster, namespace, state)
  return <div className="mx-auto w-full max-w-7xl space-y-5 p-6">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h1 className="text-2xl font-semibold">{t("packages.title")}</h1><p className="text-sm text-muted-foreground">{t("packages.count", { count: releases.data?.length ?? 0 })}</p></div><Button variant="outline" size="icon" title={t("packages.refresh")} onClick={() => releases.refetch()}><RefreshCw className="size-4" /></Button></div>
    <div className="flex flex-wrap gap-3"><Input className="max-w-64" placeholder={t("common.namespace")} value={namespace} onChange={(e) => setNamespace(e.target.value)} /><Input className="max-w-64" placeholder={t("common.status")} value={state} onChange={(e) => setState(e.target.value)} /></div>
    <div className="overflow-hidden rounded-md border"><table className="w-full text-sm"><thead className="bg-muted/50 text-left"><tr><th className="p-3">{t("packages.release")}</th><th className="p-3">{t("common.namespace")}</th><th className="p-3">{t("packages.chart")}</th><th className="p-3">{t("packages.revision")}</th><th className="p-3">{t("common.status")}</th><th className="p-3">{t("packages.updated")}</th></tr></thead><tbody>{releases.data?.map((item) => <tr key={`${item.namespace}/${item.name}`} className="border-t hover:bg-muted/30"><td className="p-3 font-medium"><Link className="flex items-center gap-2 hover:underline" to={`/package-releases/${item.namespace}/${item.name}`}><PackageOpen className="size-4 text-green-600" />{item.name}</Link></td><td className="p-3">{item.namespace}</td><td className="p-3">{item.chart} {item.chartVersion}</td><td className="p-3">{item.revision}</td><td className="p-3"><Badge variant="outline">{item.status}</Badge></td><td className="p-3 text-muted-foreground">{item.updatedAt ? new Date(item.updatedAt).toLocaleString(i18n.language) : "-"}</td></tr>)}</tbody></table>{!releases.isLoading && !releases.data?.length && <div className="p-10 text-center text-sm text-muted-foreground">{t("packages.empty")}</div>}</div>
  </div>
}
