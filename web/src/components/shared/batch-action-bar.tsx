import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Trash2, RotateCcw, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useBatchDelete, useBatchRestart } from "@/hooks/use-batch-actions"
import { toast } from "sonner"

const RESTARTABLE_KINDS = new Set(["deployments", "statefulsets", "daemonsets"])

interface SelectedItem {
  resource: string
  name: string
  namespace: string
}

interface BatchActionBarProps {
  clusterID: string
  resource: string
  selectedItems: SelectedItem[]
  onClearSelection: () => void
  onComplete: () => void
}

export function BatchActionBar({
  clusterID,
  resource,
  selectedItems,
  onClearSelection,
  onComplete,
}: BatchActionBarProps) {
  const { t } = useTranslation()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [restartDialogOpen, setRestartDialogOpen] = useState(false)

  const batchDeleteMutation = useBatchDelete(clusterID)
  const batchRestartMutation = useBatchRestart(clusterID)

  const canRestart = RESTARTABLE_KINDS.has(resource)
  const count = selectedItems.length

  if (count === 0) return null

  function handleBatchDelete() {
    batchDeleteMutation.mutate(
      selectedItems.map((item) => ({
        resource: item.resource,
        name: item.name,
        namespace: item.namespace,
      })),
      {
        onSuccess: (results) => {
          const failed = results.filter((r) => !r.success)
          if (failed.length === 0) {
            toast.success(t("batch.deleteSuccess", { count }))
          } else {
            toast.error(t("batch.deletePartial", { success: count - failed.length, total: count }))
          }
          setDeleteDialogOpen(false)
          onClearSelection()
          onComplete()
        },
      }
    )
  }

  function handleBatchRestart() {
    batchRestartMutation.mutate(
      selectedItems.map((item) => ({
        kind: item.resource,
        name: item.name,
        namespace: item.namespace,
      })),
      {
        onSuccess: (results) => {
          const failed = results.filter((r) => !r.success)
          if (failed.length === 0) {
            toast.success(t("batch.restartSuccess", { count }))
          } else {
            toast.error(t("batch.restartPartial", { success: count - failed.length, total: count }))
          }
          setRestartDialogOpen(false)
          onClearSelection()
          onComplete()
        },
      }
    )
  }

  return (
    <>
      <div className="flex items-center gap-2 rounded-lg border bg-muted/50 px-4 py-2">
        <span className="text-sm font-medium">
          {t("batch.selected", { count })}
        </span>
        <div className="flex-1" />
        {canRestart && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setRestartDialogOpen(true)}
          >
            <RotateCcw className="size-4" />
            {t("batch.restart")}
          </Button>
        )}
        <Button
          variant="destructive"
          size="sm"
          onClick={() => setDeleteDialogOpen(true)}
        >
          <Trash2 className="size-4" />
          {t("batch.delete")}
        </Button>
        <Button variant="ghost" size="icon-xs" onClick={onClearSelection}>
          <X className="size-4" />
        </Button>
      </div>

      {/* Batch Delete Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("batch.deleteTitle")}</DialogTitle>
            <DialogDescription>
              {t("batch.deleteDescription", { count })}
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-40 overflow-auto rounded border p-2">
            {selectedItems.map((item) => (
              <div key={`${item.namespace}-${item.name}`} className="text-sm py-0.5 font-mono">
                {item.namespace ? `${item.namespace}/` : ""}{item.name}
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)} disabled={batchDeleteMutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleBatchDelete} disabled={batchDeleteMutation.isPending}>
              {batchDeleteMutation.isPending ? t("common.loading") : t("batch.confirmDelete", { count })}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Batch Restart Dialog */}
      <Dialog open={restartDialogOpen} onOpenChange={setRestartDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("batch.restartTitle")}</DialogTitle>
            <DialogDescription>
              {t("batch.restartDescription", { count })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRestartDialogOpen(false)} disabled={batchRestartMutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleBatchRestart} disabled={batchRestartMutation.isPending}>
              {batchRestartMutation.isPending ? t("common.loading") : t("batch.confirmRestart", { count })}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
