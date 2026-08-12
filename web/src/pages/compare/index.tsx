import { useState } from "react"
import { useTranslation } from "react-i18next"
import { GitCompareArrows, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { YamlDiffViewer } from "@/components/specialized/yaml-diff-viewer"
import { useCluster } from "@/hooks/use-cluster"
import { useCompareResources, type CompareTarget } from "@/hooks/use-compare"

// --------------------------------------------------------------------------
// Resource selector form (one side of the comparison)
// --------------------------------------------------------------------------

const RESOURCE_OPTIONS = [
  "pods", "deployments", "statefulsets", "daemonsets",
  "services", "ingresses", "configmaps", "secrets",
  "persistentvolumeclaims", "nodes", "namespaces",
]

interface ResourceSelectorProps {
  label: string
  value: CompareTarget
  onChange: (v: CompareTarget) => void
  clusters: { id: string | number; name: string }[]
}

function ResourceSelector({ label, value, onChange, clusters }: ResourceSelectorProps) {
  const { t } = useTranslation()

  const set = (field: keyof CompareTarget, v: string) =>
    onChange({ ...value, [field]: v })

  return (
    <Card className="flex-1">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {/* Cluster */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-xs">{t("compare.cluster")}</Label>
          <select
            value={value.cluster}
            onChange={(e) => set("cluster", e.target.value)}
            className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm text-foreground"
          >
            <option value="">{t("cluster.select")}</option>
            {clusters.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>

        {/* Namespace */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-xs">{t("common.namespace")}</Label>
          <Input
            value={value.namespace}
            onChange={(e) => set("namespace", e.target.value)}
            placeholder="default"
            className="h-8 text-sm"
          />
        </div>

        {/* Resource type */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-xs">{t("compare.resourceType")}</Label>
          <select
            value={value.resource}
            onChange={(e) => set("resource", e.target.value)}
            className="h-8 w-full rounded-md border border-input bg-background px-2 text-sm text-foreground"
          >
            <option value="">{t("compare.selectResource")}</option>
            {RESOURCE_OPTIONS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </div>

        {/* Resource name */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-xs">{t("common.name")}</Label>
          <Input
            value={value.name}
            onChange={(e) => set("name", e.target.value)}
            placeholder="my-deployment"
            className="h-8 text-sm"
          />
        </div>
      </CardContent>
    </Card>
  )
}

// --------------------------------------------------------------------------
// Compare page
// --------------------------------------------------------------------------

const EMPTY_TARGET: CompareTarget = {
  cluster: "",
  namespace: "",
  resource: "",
  name: "",
}

export function ComparePage() {
  const { t } = useTranslation()
  const { clusters } = useCluster()

  const [source, setSource] = useState<CompareTarget>(EMPTY_TARGET)
  const [target, setTarget] = useState<CompareTarget>(EMPTY_TARGET)

  const compareMutation = useCompareResources()
  const result = compareMutation.data

  const canCompare =
    !!source.cluster && !!source.resource && !!source.name &&
    !!target.cluster && !!target.resource && !!target.name

  const handleCompare = () => {
    compareMutation.mutate({ source, target })
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("compare.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("compare.description")}</p>
      </div>

      {/* Two-panel selector */}
      <div className="flex items-start gap-4">
        <ResourceSelector
          label={t("compare.source")}
          value={source}
          onChange={setSource}
          clusters={clusters}
        />

        <div className="flex items-center pt-16 shrink-0 text-muted-foreground">
          <ArrowRight className="size-5" />
        </div>

        <ResourceSelector
          label={t("compare.target")}
          value={target}
          onChange={setTarget}
          clusters={clusters}
        />
      </div>

      {/* Compare button */}
      <div className="flex justify-center">
        <Button
          onClick={handleCompare}
          disabled={!canCompare || compareMutation.isPending}
          className="gap-2"
          size="lg"
        >
          <GitCompareArrows className="size-4" />
          {compareMutation.isPending ? t("common.loading") : t("compare.compare")}
        </Button>
      </div>

      {/* Diff result */}
      {result && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span className="font-mono text-xs bg-muted px-2 py-0.5 rounded">{result.sourceRef}</span>
            <ArrowRight className="size-3" />
            <span className="font-mono text-xs bg-muted px-2 py-0.5 rounded">{result.targetRef}</span>
          </div>
          <YamlDiffViewer
            original={result.sourceYaml}
            proposed={result.targetYaml}
            originalLabel={t("compare.source")}
            proposedLabel={t("compare.target")}
            className="max-h-[600px]"
          />
        </div>
      )}

      {/* Error state */}
      {compareMutation.isError && (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {compareMutation.error?.message ?? t("compare.error")}
        </div>
      )}
    </div>
  )
}
