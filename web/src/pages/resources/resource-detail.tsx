import { useState, useMemo, useCallback } from "react"
import { useParams, useNavigate, useSearchParams, Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ChevronRight, Pencil, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { StatusBadge } from "@/components/shared/status-badge"
import { KubectlHint } from "@/components/specialized/kubectl-hint"
import { FavoriteButton } from "@/components/shared/favorite-button"
import { DryRunDialog } from "@/components/specialized/dry-run-dialog"
import { ResourceEvents } from "@/components/specialized/resource-events"
import { PodTerminal } from "@/components/specialized/pod-terminal"
import { PodLogs } from "@/components/specialized/pod-logs"
import { ImageTagEditor } from "@/components/specialized/image-tag-editor"
import { resourceUIConfig } from "@/config/resource-ui-config"
import {
  useResource,
  useUpdateResource,
  useDeleteResource,
  useDryRunUpdate,
  type PodMetrics,
} from "@/hooks/use-resource"
import { useCluster } from "@/hooks/use-cluster"
import { useCheckFavorite } from "@/hooks/use-favorites"
import { toYaml, getResourceStatus } from "@/lib/k8s-utils"
import { prepareSecretForEditing } from "@/lib/secret-edit"
import { toast } from "sonner"
import { ClusterUnavailable } from "@/components/shared/cluster-unavailable"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"
import {
  ResourceOverviewTab,
  ResourceYamlTab,
  type ResourceMetadata,
} from "./resource-detail-tabs"

export function ResourceDetailPage() {
  const { resource = "", name = "" } = useParams<{ resource: string; name: string }>()
  const [searchParams] = useSearchParams()
  const namespace = searchParams.get("namespace") ?? ""
  const { t } = useTranslation()
  const navigate = useNavigate()
  const {
    currentCluster,
    clusters,
    selectedCluster,
    isClusterHealthy,
    refetchClusters,
  } = useCluster()
  const { user } = useAuth()

  // Resolve the human-readable cluster name for the --context flag.
  const clusterContext = useMemo(
    () => clusters.find((c) => String(c.id) === String(currentCluster))?.name,
    [clusters, currentCluster]
  )

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editJson, setEditJson] = useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  // Dry-run preview dialog state.
  const [dryRunDialogOpen, setDryRunDialogOpen] = useState(false)

  const config = resourceUIConfig[resource]
  const displayName = t(`resourceTypes.${resource}`, { defaultValue: config?.displayName ?? resource })

  const { data, isLoading } = useResource(
    currentCluster,
    resource,
    namespace,
    name,
    isClusterHealthy,
    resource === "pods"
  )
  const updateMutation = useUpdateResource(currentCluster, resource)
  const deleteMutation = useDeleteResource(currentCluster, resource)
  const dryRunUpdateMutation = useDryRunUpdate(currentCluster, resource)

  const { data: favoriteCheck } = useCheckFavorite(
    { clusterId: currentCluster, resourceType: resource, name, namespace },
    isClusterHealthy && !!currentCluster && !!resource && !!name
  )

  const metadata = data?.metadata as ResourceMetadata | undefined

  const yamlContent = useMemo(() => {
    if (!data) return ""
    return toYaml(data)
  }, [data])

  const status = useMemo(() => {
    if (!data) return ""
    return getResourceStatus(resource, data)
  }, [data, resource])

  // Extract the list of container names from the Pod spec (only relevant for pods).
  const podContainers = useMemo<string[]>(() => {
    if (resource !== "pods" || !data) return []
    const spec = (data as Record<string, unknown>).spec as Record<string, unknown> | undefined
    if (!spec) return []
    const containers = spec.containers as Array<{ name: string }> | undefined
    const initContainers = spec.initContainers as Array<{ name: string }> | undefined
    return [
      ...(containers ?? []).map((c) => c.name),
      ...(initContainers ?? []).map((c) => c.name),
    ]
  }, [data, resource])

  const handleEditOpen = useCallback(() => {
    if (!data) return
    const editable = resource === "secrets" ? prepareSecretForEditing(data) : data
    setEditJson(JSON.stringify(editable, null, 2))
    dryRunUpdateMutation.reset()
    setEditDialogOpen(true)
  }, [data, dryRunUpdateMutation, resource])

  // Parse the edit textarea JSON; return null and toast on failure.
  const parseEditBody = useCallback((): Record<string, unknown> | null => {
    try {
      return JSON.parse(editJson)
    } catch {
      toast.error(t("resource.invalidJsonShort"))
      return null
    }
  }, [editJson, t])

  // "Preview Changes" in the edit dialog — triggers dry-run, opens preview dialog.
  const handlePreviewEdit = useCallback(() => {
    const body = parseEditBody()
    if (!body) return

    dryRunUpdateMutation.mutate(
      { name, namespace, body },
      {
        onSuccess: () => {
          setDryRunDialogOpen(true)
        },
        onError: () => {
          // Show the dialog even on error so the user can see validation details.
          setDryRunDialogOpen(true)
        },
      }
    )
  }, [parseEditBody, name, namespace, dryRunUpdateMutation])

  // "Apply" inside the preview dialog — performs the actual (non-dry-run) update.
  const handleApplyEdit = useCallback(() => {
    const body = parseEditBody()
    if (!body) return

    updateMutation.mutate(
      { name, namespace, body },
      {
        onSuccess: () => {
          toast.success(t("resource.updatedToast", { name }))
          setDryRunDialogOpen(false)
          setEditDialogOpen(false)
          setEditJson("")
          dryRunUpdateMutation.reset()
        },
      }
    )
  }, [parseEditBody, name, namespace, updateMutation, dryRunUpdateMutation, t])

  // "Save" directly (bypass dry-run preview).
  const handleEditSave = useCallback(() => {
    const body = parseEditBody()
    if (!body) return

    updateMutation.mutate(
      { name, namespace, body },
      {
        onSuccess: () => {
          toast.success(t("resource.updatedToast", { name }))
          setEditDialogOpen(false)
          setEditJson("")
        },
      }
    )
  }, [parseEditBody, name, namespace, updateMutation, t])

  const handleDelete = useCallback(() => {
    deleteMutation.mutate(
      { name, namespace },
      {
        onSuccess: () => {
          toast.success(t("resource.deletedToast", { name }))
          setDeleteDialogOpen(false)
          navigate(`/${resource}`)
        },
      }
    )
  }, [deleteMutation, name, namespace, navigate, resource, t])

  if (selectedCluster?.status === "unhealthy") {
    return (
      <div className="flex h-full flex-col gap-4">
        <h1 className="text-2xl font-bold tracking-tight">{displayName}</h1>
        <ClusterUnavailable
          cluster={selectedCluster}
          onCheckAgain={refetchClusters}
          canRemove={canAccessAdmin(user?.role ?? "")}
        />
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Skeleton className="h-4 w-20" />
          <ChevronRight className="size-3" />
          <Skeleton className="h-4 w-32" />
        </div>
        <Skeleton className="h-8 w-48" />
        <div className="flex flex-col gap-4">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 py-16">
        <p className="text-muted-foreground">{t("resource.notFound")}</p>
        <Button variant="outline" onClick={() => navigate(`/${resource}`)}>
          {t("common.back")}
        </Button>
      </div>
    )
  }

  return (
    <div className="flex min-w-0 flex-col gap-4 sm:gap-6">
      {/* Breadcrumb */}
      <nav className="flex min-w-0 items-center gap-2 text-sm text-muted-foreground">
        <Link
          to={`/${resource}`}
          className="hover:text-foreground transition-colors"
        >
          {displayName}
        </Link>
        <ChevronRight className="size-3" />
        <span className="min-w-0 truncate font-medium text-foreground">{name}</span>
      </nav>

      {/* kubectl hint — shows the equivalent describe command for this resource */}
      <KubectlHint
        action="describe"
        resource={resource}
        name={name}
        namespace={namespace || undefined}
        clusterContext={clusterContext}
      />

      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          {config?.icon && (
            <config.icon className={`size-7 ${config.iconColor}`} />
          )}
          <div className="min-w-0 flex-1">
            <h1 className="break-all text-xl font-bold tracking-tight sm:text-2xl">{name}</h1>
            {namespace && (
              <p className="text-sm text-muted-foreground">
                {t("resource.namespaceValue", { namespace })}
              </p>
            )}
          </div>
          {status && <StatusBadge status={status} className="sm:ml-2" />}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* Favorite toggle — only available when a cluster is selected */}
          {currentCluster && (
            <FavoriteButton
              clusterId={currentCluster}
              resourceType={resource}
              resourceName={name}
              namespace={namespace}
              displayName={name}
              isFavorited={favoriteCheck?.favorited ?? false}
            />
          )}
          <Button variant="outline" size="sm" onClick={handleEditOpen}>
            <Pencil className="size-4" />
            {t("common.edit")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setDeleteDialogOpen(true)}
          >
            <Trash2 className="size-4" />
            {t("common.delete")}
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">{t("common.overview")}</TabsTrigger>
          <TabsTrigger value="yaml">{t("common.yaml")}</TabsTrigger>
          <TabsTrigger value="events">{t("resource.events")}</TabsTrigger>
          {resource === "pods" && (
            <>
              <TabsTrigger value="logs">{t("pod.logs")}</TabsTrigger>
              <TabsTrigger value="terminal">{t("pod.terminal")}</TabsTrigger>
            </>
          )}
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="mt-4">
          <ResourceOverviewTab
            resource={resource}
            namespace={namespace}
            metadata={metadata}
            status={data.status}
            metrics={data.metrics as PodMetrics | undefined}
            metricsStatus={data.metricsStatus as string | undefined}
          />
        </TabsContent>

        {/* YAML Tab */}
        <TabsContent value="yaml" className="mt-4">
          <ResourceYamlTab content={yamlContent} name={name} namespace={namespace} />
        </TabsContent>

        {/* Events Tab */}
        <TabsContent value="events" className="mt-4">
          <ResourceEvents
            clusterID={currentCluster}
            resource={resource}
            name={name}
            namespace={namespace}
          />
        </TabsContent>

        {/* Logs Tab — Pods only */}
        {resource === "pods" && (
          <TabsContent value="logs" className="mt-4">
            <Card className="flex flex-col" style={{ height: "60vh" }}>
              <CardHeader className="pb-2 shrink-0">
                <CardTitle className="text-base">{t("pod.logs")}</CardTitle>
              </CardHeader>
              <CardContent className="flex-1 min-h-0 pb-4">
                <PodLogs
                  clusterId={currentCluster}
                  namespace={namespace}
                  podName={name}
                  containers={podContainers}
                />
              </CardContent>
            </Card>
          </TabsContent>
        )}

        {/* Terminal Tab — Pods only */}
        {resource === "pods" && (
          <TabsContent value="terminal" className="mt-4">
            <Card className="flex flex-col" style={{ height: "60vh" }}>
              <CardHeader className="pb-2 shrink-0">
                <CardTitle className="text-base">{t("pod.terminal")}</CardTitle>
              </CardHeader>
              <CardContent className="flex-1 min-h-0 pb-4">
                <PodTerminal
                  clusterId={currentCluster}
                  namespace={namespace}
                  podName={name}
                  containers={podContainers}
                />
              </CardContent>
            </Card>
          </TabsContent>
        )}
      </Tabs>

      {/* ------------------------------------------------------------------ */}
      {/* Edit Dialog                                                          */}
      {/* ------------------------------------------------------------------ */}
      <Dialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          setEditDialogOpen(open)
          if (!open) {
            setEditJson("")
            dryRunUpdateMutation.reset()
          }
        }}
      >
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t("resource.editTitle", { name })}</DialogTitle>
            <DialogDescription>
              {t("resource.editDescription")}
            </DialogDescription>
          </DialogHeader>
          <KubectlHint
            action="edit"
            resource={resource}
            name={name}
            namespace={namespace || undefined}
            clusterContext={clusterContext}
            defaultOpen
          />
          <ImageTagEditor json={editJson} onChange={setEditJson} />
          {resource === "secrets" && (
            <div className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground">
              {t("resource.secretEditHint")}
            </div>
          )}
          <ScrollArea className="max-h-[500px]">
            <textarea
              className="h-[400px] w-full rounded-md border bg-muted p-3 font-mono text-xs outline-none focus:ring-2 focus:ring-ring"
              value={editJson}
              onChange={(e) => setEditJson(e.target.value)}
            />
          </ScrollArea>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setEditDialogOpen(false)
                setEditJson("")
                dryRunUpdateMutation.reset()
              }}
              disabled={updateMutation.isPending || dryRunUpdateMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="secondary"
              onClick={handlePreviewEdit}
              disabled={updateMutation.isPending || dryRunUpdateMutation.isPending}
            >
              {dryRunUpdateMutation.isPending ? t("common.loading") : t("resource.previewChanges")}
            </Button>
            <Button
              onClick={handleEditSave}
              disabled={updateMutation.isPending || dryRunUpdateMutation.isPending}
            >
              {updateMutation.isPending ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ------------------------------------------------------------------ */}
      {/* Dry-run preview dialog (update)                                      */}
      {/* ------------------------------------------------------------------ */}
      <DryRunDialog
        open={dryRunDialogOpen}
        onOpenChange={(open) => {
          setDryRunDialogOpen(open)
          if (!open) dryRunUpdateMutation.reset()
        }}
        dryRunResult={dryRunUpdateMutation.data}
        isLoading={dryRunUpdateMutation.isPending}
        title={t("resource.previewUpdate", { name })}
        operation="update"
        onApply={handleApplyEdit}
        isApplying={updateMutation.isPending}
      />

      {/* ------------------------------------------------------------------ */}
      {/* Delete Dialog                                                        */}
      {/* ------------------------------------------------------------------ */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("resource.deleteTitle", { name })}</DialogTitle>
            <DialogDescription>
              {t(namespace ? "resource.deleteDescriptionWithNamespace" : "resource.deleteDescription", {
                type: displayName,
                name,
                namespace,
              })}
            </DialogDescription>
          </DialogHeader>
          <KubectlHint
            action="delete"
            resource={resource}
            name={name}
            namespace={namespace || undefined}
            clusterContext={clusterContext}
            defaultOpen
          />
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleteMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t("common.loading") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
