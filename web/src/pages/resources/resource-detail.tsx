import { useState, useMemo, useCallback } from "react"
import { useParams, useNavigate, useSearchParams, Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ChevronRight, Copy, Check, Pencil, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
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
import { PodTerminal } from "@/components/specialized/pod-terminal"
import { PodLogs } from "@/components/specialized/pod-logs"
import { resourceUIConfig } from "@/config/resource-ui-config"
import {
  useResource,
  useUpdateResource,
  useDeleteResource,
  useDryRunUpdate,
} from "@/hooks/use-resource"
import { useCluster } from "@/hooks/use-cluster"
import { useCheckFavorite } from "@/hooks/use-favorites"
import { toYaml, getResourceStatus, formatAge } from "@/lib/k8s-utils"
import { toast } from "sonner"

export function ResourceDetailPage() {
  const { resource = "", name = "" } = useParams<{ resource: string; name: string }>()
  const [searchParams] = useSearchParams()
  const namespace = searchParams.get("namespace") ?? ""
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { currentCluster, clusters } = useCluster()

  // Resolve the human-readable cluster name for the --context flag.
  const clusterContext = useMemo(
    () => clusters.find((c) => c.id === currentCluster)?.name,
    [clusters, currentCluster]
  )

  const [copied, setCopied] = useState(false)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editJson, setEditJson] = useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  // Dry-run preview dialog state.
  const [dryRunDialogOpen, setDryRunDialogOpen] = useState(false)

  const config = resourceUIConfig[resource]
  const displayName = config?.displayName ?? resource

  const { data, isLoading } = useResource(currentCluster, resource, namespace, name)
  const updateMutation = useUpdateResource(currentCluster, resource)
  const deleteMutation = useDeleteResource(currentCluster, resource)
  const dryRunUpdateMutation = useDryRunUpdate(currentCluster, resource)

  const { data: favoriteCheck } = useCheckFavorite(
    { clusterId: currentCluster, resourceType: resource, name, namespace },
    !!currentCluster && !!resource && !!name
  )

  const metadata = data?.metadata as {
    name?: string
    namespace?: string
    uid?: string
    creationTimestamp?: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
    resourceVersion?: string
  } | undefined

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

  const handleCopyYaml = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(yamlContent)
      setCopied(true)
      toast.success("Copied to clipboard")
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error("Failed to copy to clipboard")
    }
  }, [yamlContent])

  const handleEditOpen = useCallback(() => {
    if (!data) return
    setEditJson(JSON.stringify(data, null, 2))
    dryRunUpdateMutation.reset()
    setEditDialogOpen(true)
  }, [data, dryRunUpdateMutation])

  // Parse the edit textarea JSON; return null and toast on failure.
  const parseEditBody = useCallback((): Record<string, unknown> | null => {
    try {
      return JSON.parse(editJson)
    } catch {
      toast.error("Invalid JSON")
      return null
    }
  }, [editJson])

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
          toast.success(`${name} updated successfully`)
          setDryRunDialogOpen(false)
          setEditDialogOpen(false)
          dryRunUpdateMutation.reset()
        },
      }
    )
  }, [parseEditBody, name, namespace, updateMutation, dryRunUpdateMutation])

  // "Save" directly (bypass dry-run preview).
  const handleEditSave = useCallback(() => {
    const body = parseEditBody()
    if (!body) return

    updateMutation.mutate(
      { name, namespace, body },
      {
        onSuccess: () => {
          toast.success(`${name} updated successfully`)
          setEditDialogOpen(false)
        },
      }
    )
  }, [parseEditBody, name, namespace, updateMutation])

  const handleDelete = useCallback(() => {
    deleteMutation.mutate(
      { name, namespace },
      {
        onSuccess: () => {
          toast.success(`${name} deleted successfully`)
          setDeleteDialogOpen(false)
          navigate(`/${resource}`)
        },
      }
    )
  }, [deleteMutation, name, namespace, navigate, resource])

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
        <p className="text-muted-foreground">Resource not found</p>
        <Button variant="outline" onClick={() => navigate(`/${resource}`)}>
          {t("common.back")}
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link
          to={`/${resource}`}
          className="hover:text-foreground transition-colors"
        >
          {displayName}
        </Link>
        <ChevronRight className="size-3" />
        <span className="text-foreground font-medium">{name}</span>
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
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          {config?.icon && (
            <config.icon className="size-7 text-muted-foreground" />
          )}
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{name}</h1>
            {namespace && (
              <p className="text-sm text-muted-foreground">
                Namespace: {namespace}
              </p>
            )}
          </div>
          {status && <StatusBadge status={status} className="ml-2" />}
        </div>
        <div className="flex items-center gap-2">
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
          <TabsTrigger value="events">Events</TabsTrigger>
          {resource === "pods" && (
            <>
              <TabsTrigger value="logs">{t("pod.logs")}</TabsTrigger>
              <TabsTrigger value="terminal">{t("pod.terminal")}</TabsTrigger>
            </>
          )}
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="mt-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Metadata Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Metadata</CardTitle>
              </CardHeader>
              <CardContent>
                <dl className="grid gap-3">
                  <DetailRow label="Name" value={metadata?.name ?? "-"} />
                  {namespace && (
                    <DetailRow label="Namespace" value={namespace} />
                  )}
                  <DetailRow label="UID" value={metadata?.uid ?? "-"} mono />
                  <DetailRow
                    label="Created"
                    value={
                      metadata?.creationTimestamp
                        ? `${formatAge(metadata.creationTimestamp)} (${new Date(metadata.creationTimestamp).toLocaleString()})`
                        : "-"
                    }
                  />
                  <DetailRow
                    label="Resource Version"
                    value={metadata?.resourceVersion ?? "-"}
                    mono
                  />
                </dl>
              </CardContent>
            </Card>

            {/* Labels Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Labels</CardTitle>
              </CardHeader>
              <CardContent>
                {metadata?.labels && Object.keys(metadata.labels).length > 0 ? (
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(metadata.labels).map(([key, value]) => (
                      <Badge
                        key={key}
                        variant="secondary"
                        className="font-mono text-xs"
                      >
                        {key}={String(value)}
                      </Badge>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No labels</p>
                )}
              </CardContent>
            </Card>

            {/* Annotations Card */}
            <Card className="md:col-span-2">
              <CardHeader>
                <CardTitle className="text-base">Annotations</CardTitle>
              </CardHeader>
              <CardContent>
                {metadata?.annotations &&
                Object.keys(metadata.annotations).length > 0 ? (
                  <dl className="grid gap-2">
                    {Object.entries(metadata.annotations).map(([key, value]) => (
                      <div key={key} className="grid gap-1">
                        <dt className="font-mono text-xs text-muted-foreground break-all">
                          {key}
                        </dt>
                        <dd className="text-sm break-all">{value}</dd>
                      </div>
                    ))}
                  </dl>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No annotations
                  </p>
                )}
              </CardContent>
            </Card>

            {/* Status Card - show raw status object */}
            {!!data.status && (
              <Card className="md:col-span-2">
                <CardHeader>
                  <CardTitle className="text-base">Status</CardTitle>
                </CardHeader>
                <CardContent>
                  <ScrollArea className="max-h-[300px]">
                    <pre className="rounded-md bg-muted/50 p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
                      {toYaml(data.status)}
                    </pre>
                  </ScrollArea>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        {/* YAML Tab */}
        <TabsContent value="yaml" className="mt-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Resource YAML</CardTitle>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyYaml}
                >
                  {copied ? (
                    <Check className="size-4" />
                  ) : (
                    <Copy className="size-4" />
                  )}
                  {copied ? "Copied" : "Copy"}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <ScrollArea className="max-h-[600px]">
                <pre className="rounded-md bg-muted/50 p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
                  {yamlContent}
                </pre>
              </ScrollArea>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Events Tab */}
        <TabsContent value="events" className="mt-4">
          <Card>
            <CardContent className="flex items-center justify-center py-12">
              <p className="text-sm text-muted-foreground">
                Events coming soon
              </p>
            </CardContent>
          </Card>
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
          if (!open) dryRunUpdateMutation.reset()
        }}
      >
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Edit {name}</DialogTitle>
            <DialogDescription>
              Modify the resource JSON. Use &quot;Preview Changes&quot; to review the diff
              before applying, or &quot;Save&quot; to apply directly.
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
          <ScrollArea className="max-h-[500px]">
            <textarea
              className="h-[400px] w-full rounded-md border bg-muted/50 p-3 font-mono text-xs outline-none focus:ring-2 focus:ring-ring"
              value={editJson}
              onChange={(e) => setEditJson(e.target.value)}
            />
          </ScrollArea>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setEditDialogOpen(false)
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
              {dryRunUpdateMutation.isPending ? t("common.loading") : "Preview Changes"}
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
        title={`Preview Changes: ${name}`}
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
            <DialogTitle>Delete {name}?</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete {resource.slice(0, -1)} &quot;{name}&quot;
              {namespace ? ` in namespace "${namespace}"` : ""}? This action cannot
              be undone.
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

function DetailRow({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="grid grid-cols-[120px_1fr] gap-2">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd
        className={`text-sm break-all ${mono ? "font-mono text-xs" : ""}`}
      >
        {value}
      </dd>
    </div>
  )
}
