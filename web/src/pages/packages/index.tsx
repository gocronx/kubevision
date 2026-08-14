import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { AlertCircle, Loader2, PackageOpen, Plus, RefreshCw } from "lucide-react"
import { useCluster } from "@/hooks/use-cluster"
import { usePackageReleases } from "@/hooks/use-package-releases"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { PackageChangeDialog } from "./change-dialog"
import { AutomationPanel, CatalogPanel, RepositoriesPanel, type InstallSelection } from "./helm-workspace"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function PackageReleasesPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { currentCluster } = useCluster()
  const [namespace, setNamespace] = useState("")
  const [state, setState] = useState("")
  const [installOpen, setInstallOpen] = useState(false)
  const [selection, setSelection] = useState<InstallSelection | null>(null)
  const { user } = useAuth(); const admin = canAccessAdmin(user?.role ?? "")
  const releases = usePackageReleases(currentCluster, namespace, state)
  return <div className="mx-auto w-full max-w-7xl space-y-5">
    <div className="flex flex-wrap items-center justify-between gap-3"><div><h1 className="text-2xl font-semibold">{t("packages.title")}</h1><p className="text-sm text-muted-foreground">{t("packages.count", { count: releases.data?.length ?? 0 })}</p></div><div className="flex gap-2"><Button onClick={() => setInstallOpen(true)}><Plus className="size-4" />{t("packages.install")}</Button><Button variant="outline" size="icon" title={t("packages.refresh")} onClick={() => releases.refetch()}><RefreshCw className="size-4" /></Button></div></div>
    <Tabs defaultValue="releases"><TabsList variant="line"><TabsTrigger value="releases">{t("helm.releases")}</TabsTrigger><TabsTrigger value="catalog">{t("helm.catalog")}</TabsTrigger>{admin&&<TabsTrigger value="repositories">{t("helm.repositories")}</TabsTrigger>}{admin&&<TabsTrigger value="automation">{t("helm.automation")}</TabsTrigger>}</TabsList>
      <TabsContent value="releases" className="space-y-4 pt-3">
        <div className="grid gap-3 sm:max-w-xl sm:grid-cols-2"><Input placeholder={t("common.namespace")} value={namespace} onChange={(e) => setNamespace(e.target.value)} /><Input placeholder={t("common.status")} value={state} onChange={(e) => setState(e.target.value)} /></div>
        {releases.isError ? (
          <div className="flex flex-col items-center gap-3 border border-destructive/30 p-8 text-center">
            <AlertCircle className="size-6 text-destructive" />
            <div><p className="text-sm font-medium">{t("packages.loadFailed")}</p><p className="mt-1 text-xs text-muted-foreground">{releases.error instanceof Error ? releases.error.message : t("packages.tryAgain")}</p></div>
            <Button variant="outline" onClick={() => releases.refetch()}><RefreshCw className="size-4" />{t("common.retry")}</Button>
          </div>
        ) : (
          <div className="max-w-full overflow-x-auto rounded-md border">
            <table className="w-full min-w-[48rem] text-sm"><thead className="bg-muted/50 text-left"><tr><th className="p-3">{t("packages.release")}</th><th className="p-3">{t("common.namespace")}</th><th className="p-3">{t("packages.chart")}</th><th className="p-3">{t("packages.revision")}</th><th className="p-3">{t("common.status")}</th><th className="p-3">{t("packages.updated")}</th></tr></thead><tbody>{releases.data?.map((item) => <tr key={`${item.namespace}/${item.name}`} className="border-t hover:bg-muted/30"><td className="p-3 font-medium"><Link className="flex items-center gap-2 hover:underline" to={`/package-releases/${item.namespace}/${item.name}`}><PackageOpen className="size-4 text-green-600" />{item.name}</Link></td><td className="p-3">{item.namespace}</td><td className="p-3">{item.chart} {item.chartVersion}</td><td className="p-3">{item.revision}</td><td className="p-3"><Badge variant="outline">{item.status}</Badge></td><td className="p-3 text-muted-foreground">{item.updatedAt ? new Date(item.updatedAt).toLocaleString(i18n.language) : "-"}</td></tr>)}</tbody></table>
            {releases.isLoading && <div className="flex justify-center p-10"><Loader2 className="size-5 animate-spin text-muted-foreground" /></div>}
            {!releases.isLoading && !releases.data?.length && <div className="flex flex-col items-center gap-3 p-10 text-center"><PackageOpen className="size-7 text-muted-foreground" /><div><p className="text-sm font-medium">{t("packages.empty")}</p><p className="mt-1 text-xs text-muted-foreground">{t("packages.emptyDescription")}</p></div><Button onClick={() => setInstallOpen(true)}><Plus className="size-4" />{t("packages.install")}</Button></div>}
          </div>
        )}
      </TabsContent>
      <TabsContent value="catalog" className="pt-3"><CatalogPanel cluster={currentCluster} onInstall={(item)=>{setSelection(item);setInstallOpen(true)}}/></TabsContent>
      {admin&&<TabsContent value="repositories" className="pt-3"><RepositoriesPanel cluster={currentCluster}/></TabsContent>}
      {admin&&<TabsContent value="automation" className="pt-3"><AutomationPanel cluster={currentCluster}/></TabsContent>}
    </Tabs>
    <PackageChangeDialog open={installOpen} onOpenChange={(open)=>{setInstallOpen(open);if(!open)setSelection(null)}} cluster={currentCluster} operation="install" namespace={namespace || "default"} source={selection?.source} chart={selection?.inspection.name} version={selection?.inspection.version} initialValues={selection?.inspection.values} onSuccess={(release) => { releases.refetch(); if (release) navigate(`/package-releases/${release.namespace}/${release.name}`) }} />
  </div>
}
