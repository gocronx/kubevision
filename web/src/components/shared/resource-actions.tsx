import { useState } from "react"
import { useTranslation } from "react-i18next"
import { MoreHorizontal, Trash2, Pencil, Scale, RotateCcw, History } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useDeleteResource } from "@/hooks/use-resource"
import { ScaleDialog } from "@/components/specialized/scale-dialog"
import { RestartDialog } from "@/components/specialized/restart-dialog"
import { RollbackDialog } from "@/components/specialized/rollback-dialog"
import { toast } from "sonner"

// Resource kinds that support each lifecycle action.
const SCALABLE_KINDS = new Set(["deployments", "statefulsets", "replicasets"])
const RESTARTABLE_KINDS = new Set(["deployments", "statefulsets", "daemonsets", "pods"])
const ROLLBACKABLE_KINDS = new Set(["deployments"])

interface ResourceActionsProps {
  clusterID: string
  resource: string
  name: string
  namespace?: string
  /** Current replica count — passed to the scale dialog as the starting value. */
  currentReplicas?: number
  /**
   * When true the component renders a read-only view of the detail link only;
   * all mutating actions (edit, scale, restart, rollback, delete) are hidden.
   */
  readOnly?: boolean
  onEdit?: () => void
  onDeleted?: () => void
}

export function ResourceActions({
  clusterID,
  resource,
  name,
  namespace,
  currentReplicas = 0,
  readOnly = false,
  onEdit,
  onDeleted,
}: ResourceActionsProps) {
  const { t } = useTranslation()

  // Delete dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const deleteMutation = useDeleteResource(clusterID, resource)

  // Lifecycle action dialog state
  const [scaleDialogOpen, setScaleDialogOpen] = useState(false)
  const [restartDialogOpen, setRestartDialogOpen] = useState(false)
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false)

  const canScale = !readOnly && SCALABLE_KINDS.has(resource)
  const canRestart = !readOnly && RESTARTABLE_KINDS.has(resource)
  const canRollback = !readOnly && ROLLBACKABLE_KINDS.has(resource)
  const hasLifecycleActions = canScale || canRestart || canRollback

  function handleDelete() {
    deleteMutation.mutate(
      { name, namespace },
      {
        onSuccess: () => {
          toast.success(t("resource.deletedToast", { name }))
          setDeleteDialogOpen(false)
          onDeleted?.()
        },
      }
    )
  }

  // Read-only roles get no action button at all.
  if (readOnly) {
    return null
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="size-4" />
            <span className="sr-only">{t("common.actions")}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {/* Edit */}
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation()
              onEdit?.()
            }}
          >
            <Pencil className="size-4" />
            {t("common.edit")}
          </DropdownMenuItem>

          {/* Lifecycle actions — only shown for supported resource types */}
          {hasLifecycleActions && <DropdownMenuSeparator />}

          {canScale && (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                setScaleDialogOpen(true)
              }}
            >
              <Scale className="size-4" />
              {t("resource.scale")}
            </DropdownMenuItem>
          )}

          {canRestart && (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                setRestartDialogOpen(true)
              }}
            >
              <RotateCcw className="size-4" />
              {t("resource.restart")}
            </DropdownMenuItem>
          )}

          {canRollback && (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                setRollbackDialogOpen(true)
              }}
            >
              <History className="size-4" />
              {t("resource.rollback")}
            </DropdownMenuItem>
          )}

          <DropdownMenuSeparator />

          {/* Delete (always last, always destructive) */}
          <DropdownMenuItem
            variant="destructive"
            onClick={(e) => {
              e.stopPropagation()
              setDeleteDialogOpen(true)
            }}
          >
            <Trash2 className="size-4" />
            {t("common.delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Delete confirmation dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent
          onPointerDownOutside={(e) => e.preventDefault()}
          onInteractOutside={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>{t("resource.deleteTitle", { name })}</DialogTitle>
            <DialogDescription>
              {t(namespace ? "resource.deleteDescriptionWithNamespace" : "resource.deleteDescription", {
                type: t(`resourceTypes.${resource}`, { defaultValue: resource }),
                name,
                namespace,
              })}
            </DialogDescription>
          </DialogHeader>
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

      {/* Scale dialog — only rendered for scalable resource types */}
      {canScale && namespace && (
        <ScaleDialog
          open={scaleDialogOpen}
          onOpenChange={setScaleDialogOpen}
          clusterID={clusterID}
          kind={resource}
          namespace={namespace}
          name={name}
          currentReplicas={currentReplicas}
        />
      )}

      {/* Restart dialog — only rendered for restartable resource types */}
      {canRestart && namespace && (
        <RestartDialog
          open={restartDialogOpen}
          onOpenChange={setRestartDialogOpen}
          clusterID={clusterID}
          kind={resource}
          namespace={namespace}
          name={name}
        />
      )}

      {/* Rollback dialog — only rendered for Deployments */}
      {canRollback && namespace && (
        <RollbackDialog
          open={rollbackDialogOpen}
          onOpenChange={setRollbackDialogOpen}
          clusterID={clusterID}
          namespace={namespace}
          name={name}
        />
      )}
    </>
  )
}
