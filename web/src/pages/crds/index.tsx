import { useTranslation } from "react-i18next"
import { RefreshCw, Puzzle } from "lucide-react"
import { useCRDs, useRefreshCRDs } from "@/hooks/use-crds"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"

export function CRDListPage() {
  const { t } = useTranslation()
  const { data: crds, isLoading } = useCRDs()
  const refreshCRDs = useRefreshCRDs()

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("crd.title")}</h1>
          <p className="text-muted-foreground">{t("crd.description")}</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => refreshCRDs.mutate()}
          disabled={refreshCRDs.isPending}
        >
          <RefreshCw className={`mr-2 size-4 ${refreshCRDs.isPending ? "animate-spin" : ""}`} />
          {t("common.refresh")}
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="size-5 animate-spin" />
        </div>
      ) : !crds?.length ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Puzzle className="mb-2 size-10" />
          <p>{t("crd.empty")}</p>
        </div>
      ) : (
        <div className="max-w-full overflow-x-auto rounded-md border">
          <table className="w-full min-w-[42rem] text-sm">
            <thead className="bg-muted/50">
              <tr className="border-b">
                <th className="px-3 py-2 text-left font-medium">{t("crd.kind")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("crd.group")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("crd.version")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("crd.plural")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("crd.scope")}</th>
              </tr>
            </thead>
            <tbody>
              {crds.map((crd) => (
                <tr key={`${crd.group}/${crd.kind}`} className="border-b last:border-0">
                  <td className="px-3 py-2 font-medium">{crd.kind}</td>
                  <td className="px-3 py-2 text-muted-foreground">{crd.group}</td>
                  <td className="px-3 py-2">{crd.version}</td>
                  <td className="px-3 py-2 text-muted-foreground">{crd.plural}</td>
                  <td className="px-3 py-2">
                    <Badge variant={crd.namespaced ? "default" : "secondary"}>
                      {crd.namespaced ? t("crd.namespaced") : t("crd.clusterScoped")}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
