import { useState, useMemo, useCallback } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { RefreshCw, Plus, Search } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { resourceUIConfig } from "@/config/resource-ui-config"
import { useResourceList, useCreateResource, useDryRunCreate } from "@/hooks/use-resource"
import { useCluster } from "@/hooks/use-cluster"
import { DataTable, type DataTableColumn } from "@/components/shared/data-table"
import { NamespaceSelector } from "@/components/shared/namespace-selector"
import { StatusBadge } from "@/components/shared/status-badge"
import { ResourceActions } from "@/components/shared/resource-actions"
import { BatchActionBar } from "@/components/shared/batch-action-bar"
import { KubectlHint } from "@/components/specialized/kubectl-hint"
import { extractColumnValue, isNamespaced } from "@/lib/k8s-utils"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { DryRunDialog } from "@/components/specialized/dry-run-dialog"
import { toast } from "sonner"

type K8sItem = Record<string, unknown>

const statusColumns = new Set(["status"])
const ageColumns = new Set(["age", "lastSchedule"])

export function ResourceListPage() {
  const { resource = "" } = useParams<{ resource: string }>()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { currentCluster, clusters } = useCluster()

  // Resolve the human-readable cluster name for the --context flag.
  const clusterContext = useMemo(
    () => clusters.find((c) => c.id === currentCluster)?.name,
    [clusters, currentCluster]
  )

  const [namespace, setNamespace] = useState("")
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set())

  // Create dialog state.
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [createYaml, setCreateYaml] = useState("")

  // Dry-run preview dialog state.
  const [dryRunDialogOpen, setDryRunDialogOpen] = useState(false)

  const config = resourceUIConfig[resource]
  const displayName = config?.displayName ?? resource
  const namespaced = isNamespaced(resource)

  const { data, isLoading } = useResourceList(currentCluster, resource, {
    namespace: namespaced ? namespace : undefined,
    enabled: !!currentCluster,
  })

  const createMutation = useCreateResource(currentCluster, resource)
  const dryRunCreateMutation = useDryRunCreate(currentCluster, resource)

  const items = useMemo(() => {
    const all = data?.items ?? []
    if (!searchQuery) return all
    const lowerQuery = searchQuery.toLowerCase()
    return all.filter((item) => {
      const meta = item.metadata as { name?: string } | undefined
      const name = meta?.name ?? ""
      return name.toLowerCase().includes(lowerQuery)
    })
  }, [data, searchQuery])

  const handleRowClick = useCallback(
    (item: K8sItem) => {
      const meta = item.metadata as { name?: string; namespace?: string } | undefined
      const name = meta?.name ?? ""
      const ns = meta?.namespace ?? ""
      if (!name) return
      const params = new URLSearchParams()
      if (ns) params.set("namespace", ns)
      navigate(`/${resource}/${name}?${params.toString()}`)
    },
    [navigate, resource]
  )

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: ["resources", currentCluster, resource],
    })
  }, [queryClient, currentCluster, resource])

  // Parse the raw JSON from the textarea; return null and toast on failure.
  const parseCreateBody = useCallback((): Record<string, unknown> | null => {
    try {
      return JSON.parse(createYaml)
    } catch {
      toast.error("Invalid JSON. Please provide valid resource JSON.")
      return null
    }
  }, [createYaml])

  // "Preview Changes" — run dry-run, then open the preview dialog.
  const handlePreviewCreate = useCallback(() => {
    const body = parseCreateBody()
    if (!body) return

    const ns =
      (body.metadata as Record<string, unknown> | undefined)?.namespace as string | undefined ??
      namespace

    dryRunCreateMutation.mutate(
      { namespace: ns, body },
      {
        onSuccess: () => {
          setDryRunDialogOpen(true)
        },
        onError: () => {
          // The API interceptor already showed an error toast; still open the
          // dialog so the user can see the validation failure.
          setDryRunDialogOpen(true)
        },
      }
    )
  }, [parseCreateBody, namespace, dryRunCreateMutation])

  // "Apply" inside the preview dialog — performs the actual (non-dry-run) create.
  const handleApplyCreate = useCallback(() => {
    const body = parseCreateBody()
    if (!body) return

    const ns =
      (body.metadata as Record<string, unknown> | undefined)?.namespace as string | undefined ??
      namespace

    createMutation.mutate(
      { namespace: ns, body },
      {
        onSuccess: () => {
          toast.success("Resource created successfully")
          setDryRunDialogOpen(false)
          setCreateDialogOpen(false)
          setCreateYaml("")
          dryRunCreateMutation.reset()
        },
      }
    )
  }, [parseCreateBody, namespace, createMutation, dryRunCreateMutation])

  // "Create" directly (bypass the dry-run preview).
  const handleCreate = useCallback(() => {
    const body = parseCreateBody()
    if (!body) return

    const ns =
      (body.metadata as Record<string, unknown> | undefined)?.namespace as string | undefined ??
      namespace

    createMutation.mutate(
      { namespace: ns, body },
      {
        onSuccess: () => {
          toast.success("Resource created successfully")
          setCreateDialogOpen(false)
          setCreateYaml("")
        },
      }
    )
  }, [parseCreateBody, namespace, createMutation])

  // Batch selection helpers
  const selectedItems = useMemo(() => {
    return items
      .filter((item) => {
        const meta = item.metadata as { uid?: string; name?: string; namespace?: string } | undefined
        const key = meta?.uid ?? `${meta?.namespace ?? ""}-${meta?.name ?? ""}`
        return selectedKeys.has(key)
      })
      .map((item) => {
        const meta = item.metadata as { name?: string; namespace?: string } | undefined
        return {
          resource,
          name: meta?.name ?? "",
          namespace: meta?.namespace ?? "",
        }
      })
  }, [items, selectedKeys, resource])

  const handleToggleSelect = useCallback((key: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }, [])

  const handleToggleAll = useCallback(() => {
    setSelectedKeys((prev) => {
      if (prev.size === items.length && items.length > 0) {
        return new Set()
      }
      const allKeys = items.map((item) => {
        const meta = item.metadata as { uid?: string; name?: string; namespace?: string } | undefined
        return meta?.uid ?? `${meta?.namespace ?? ""}-${meta?.name ?? ""}`
      })
      return new Set(allKeys)
    })
  }, [items])

  const tableColumns: DataTableColumn<K8sItem>[] = useMemo(() => {
    const cols = config?.columns ?? [
      { key: "name", label: "Name", sortable: true },
    ]

    // Checkbox column for batch selection
    const selectCol: DataTableColumn<K8sItem> = {
      key: "_select",
      label: "",
      sortable: false,
      className: "w-[40px]",
      headerRender: () => (
        <input
          type="checkbox"
          checked={selectedKeys.size > 0 && selectedKeys.size === items.length}
          ref={(el) => {
            if (el) el.indeterminate = selectedKeys.size > 0 && selectedKeys.size < items.length
          }}
          onChange={handleToggleAll}
          className="size-4 rounded border-muted-foreground"
          onClick={(e) => e.stopPropagation()}
        />
      ),
      render: (item: K8sItem) => {
        const meta = item.metadata as { uid?: string; name?: string; namespace?: string } | undefined
        const key = meta?.uid ?? `${meta?.namespace ?? ""}-${meta?.name ?? ""}`
        return (
          <input
            type="checkbox"
            checked={selectedKeys.has(key)}
            onChange={() => handleToggleSelect(key)}
            className="size-4 rounded border-muted-foreground"
            onClick={(e) => e.stopPropagation()}
          />
        )
      },
    }

    const mapped: DataTableColumn<K8sItem>[] = [selectCol, ...cols.map((col) => ({
      key: col.key,
      label: col.label,
      sortable: col.sortable,
      render: (item: K8sItem) => {
        const value = extractColumnValue(resource, item, col.key)

        if (statusColumns.has(col.key)) {
          return <StatusBadge status={value} />
        }

        if (ageColumns.has(col.key)) {
          return (
            <span className="text-muted-foreground">{value}</span>
          )
        }

        if (col.key === "name") {
          return (
            <span className="font-medium text-foreground">{value}</span>
          )
        }

        return <span>{value}</span>
      },
    }))]

    // Add actions column
    mapped.push({
      key: "_actions",
      label: t("common.actions"),
      sortable: false,
      className: "w-[60px]",
      render: (item: K8sItem) => {
        const meta = item.metadata as { name?: string; namespace?: string } | undefined
        const spec = item.spec as { replicas?: number } | undefined
        return (
          <ResourceActions
            clusterID={currentCluster}
            resource={resource}
            name={meta?.name ?? ""}
            namespace={meta?.namespace}
            currentReplicas={spec?.replicas ?? 0}
            onDeleted={handleRefresh}
          />
        )
      },
    })

    return mapped
  }, [config, resource, t, currentCluster, handleRefresh, selectedKeys, items, handleToggleAll, handleToggleSelect])

  const getRowKey = useCallback((item: K8sItem) => {
    const meta = item.metadata as { uid?: string; name?: string; namespace?: string } | undefined
    return meta?.uid ?? `${meta?.namespace ?? ""}-${meta?.name ?? ""}`
  }, [])

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {config?.icon && (
            <config.icon className="size-6 text-muted-foreground" />
          )}
          <h1 className="text-2xl font-bold tracking-tight">{displayName}</h1>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
          >
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
          <Button
            size="sm"
            onClick={() => setCreateDialogOpen(true)}
          >
            <Plus className="size-4" />
            {t("common.create")}
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-3">
        {namespaced && (
          <NamespaceSelector
            clusterID={currentCluster}
            value={namespace}
            onChange={setNamespace}
          />
        )}
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={`${t("common.search")} ${displayName.toLowerCase()}...`}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* kubectl hint — shows the equivalent get command for this view */}
      <KubectlHint
        action="get"
        resource={resource}
        namespace={namespaced ? namespace : undefined}
        clusterContext={clusterContext}
      />

      {/* Batch action bar — visible when items are selected */}
      <BatchActionBar
        clusterID={currentCluster}
        resource={resource}
        selectedItems={selectedItems}
        onClearSelection={() => setSelectedKeys(new Set())}
        onComplete={handleRefresh}
      />

      <DataTable
        columns={tableColumns}
        data={items}
        isLoading={isLoading}
        emptyMessage={t("common.noData")}
        onRowClick={handleRowClick}
        getRowKey={getRowKey}
      />

      {/* ------------------------------------------------------------------ */}
      {/* Create dialog                                                        */}
      {/* ------------------------------------------------------------------ */}
      <Dialog
        open={createDialogOpen}
        onOpenChange={(open) => {
          setCreateDialogOpen(open)
          if (!open) {
            setCreateYaml("")
            dryRunCreateMutation.reset()
          }
        }}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Create {config?.displayName ?? resource}</DialogTitle>
            <DialogDescription>
              Paste the resource JSON below. Use &quot;Preview Changes&quot; to validate
              against the API server before applying, or &quot;Create&quot; to apply directly.
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="max-h-[400px]">
            <textarea
              className="h-[300px] w-full rounded-md border bg-muted/50 p-3 font-mono text-sm outline-none focus:ring-2 focus:ring-ring"
              placeholder='{"apiVersion": "v1", "kind": "...", "metadata": {"name": "..."}, ...}'
              value={createYaml}
              onChange={(e) => setCreateYaml(e.target.value)}
            />
          </ScrollArea>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setCreateDialogOpen(false)
                setCreateYaml("")
                dryRunCreateMutation.reset()
              }}
              disabled={createMutation.isPending || dryRunCreateMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="secondary"
              onClick={handlePreviewCreate}
              disabled={
                createMutation.isPending ||
                dryRunCreateMutation.isPending ||
                !createYaml.trim()
              }
            >
              {dryRunCreateMutation.isPending ? t("common.loading") : "Preview Changes"}
            </Button>
            <Button
              onClick={handleCreate}
              disabled={
                createMutation.isPending ||
                dryRunCreateMutation.isPending ||
                !createYaml.trim()
              }
            >
              {createMutation.isPending ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ------------------------------------------------------------------ */}
      {/* Dry-run preview dialog (create)                                      */}
      {/* ------------------------------------------------------------------ */}
      <DryRunDialog
        open={dryRunDialogOpen}
        onOpenChange={(open) => {
          setDryRunDialogOpen(open)
          if (!open) dryRunCreateMutation.reset()
        }}
        dryRunResult={dryRunCreateMutation.data}
        isLoading={dryRunCreateMutation.isPending}
        title={`Preview Create: ${config?.displayName ?? resource}`}
        operation="create"
        onApply={handleApplyCreate}
        isApplying={createMutation.isPending}
      />
    </div>
  )
}
