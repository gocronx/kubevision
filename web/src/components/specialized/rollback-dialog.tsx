import { useState } from "react"
import { useTranslation } from "react-i18next"
import { History, TriangleAlert } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useRolloutHistory, useRollbackDeployment } from "@/hooks/use-resource-actions"

interface RollbackDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterID: string
  namespace: string
  name: string
}

export function RollbackDialog({
  open,
  onOpenChange,
  clusterID,
  namespace,
  name,
}: RollbackDialogProps) {
  const { t } = useTranslation()
  const [selectedRevision, setSelectedRevision] = useState<number | null>(null)

  const { data: history, isLoading: historyLoading } = useRolloutHistory(
    clusterID,
    namespace,
    name,
    open
  )
  const rollbackMutation = useRollbackDeployment(clusterID)

  function handleConfirm() {
    if (selectedRevision === null) return

    rollbackMutation.mutate(
      { namespace, name, revision: selectedRevision },
      {
        onSuccess: () => {
          toast.success(t("resource.rollbackToast", { name, revision: selectedRevision }))
          onOpenChange(false)
          setSelectedRevision(null)
        },
        onError: (err) => {
          toast.error(
            err instanceof Error ? err.message : t("resource.rollbackFailed")
          )
        },
      }
    )
  }

  function handleClose(isOpen: boolean) {
    if (!isOpen) {
      setSelectedRevision(null)
    }
    onOpenChange(isOpen)
  }

  // The "previous" entry is the second-to-last revision (if any).
  const sortedHistory = history ?? []
  const latestRevision =
    sortedHistory.length > 0
      ? sortedHistory[sortedHistory.length - 1].revision
      : null

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent
        className="sm:max-w-lg"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{t("resource.rollbackTitle", { name })}</DialogTitle>
          <DialogDescription>{t("resource.rollbackDescription")}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          {historyLoading ? (
            <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
              {t("resource.loadingHistory")}
            </div>
          ) : sortedHistory.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-sm text-muted-foreground">
              <History className="size-8 opacity-40" />
              <p>{t("resource.noHistory")}</p>
            </div>
          ) : (
            <ScrollArea className="max-h-64 rounded-md border">
              <div className="p-1">
                {/* Display history newest-first so the most useful entries are at the top. */}
                {[...sortedHistory].reverse().map((rev) => {
                  const isCurrent = rev.revision === latestRevision
                  const isSelected = rev.revision === selectedRevision

                  return (
                    <button
                      key={rev.revision}
                      type="button"
                      disabled={isCurrent || rollbackMutation.isPending}
                      onClick={() =>
                        !isCurrent && setSelectedRevision(rev.revision)
                      }
                      className={[
                        "flex w-full items-start gap-3 rounded-md px-3 py-2.5 text-left text-sm transition-colors",
                        "disabled:cursor-not-allowed disabled:opacity-60",
                        isSelected
                          ? "bg-primary text-primary-foreground"
                          : isCurrent
                            ? "bg-muted/50 text-muted-foreground"
                            : "hover:bg-muted/70",
                      ].join(" ")}
                    >
                      <span
                        className={`mt-0.5 min-w-[4rem] font-mono text-xs font-semibold ${
                          isSelected
                            ? "text-primary-foreground/80"
                            : "text-muted-foreground"
                        }`}
                      >
                        #{rev.revision}
                      </span>
                      <div className="flex flex-1 flex-col gap-0.5 overflow-hidden">
                        <span
                          className={`truncate font-medium ${
                            !rev.changeCause ? "italic opacity-60" : ""
                          }`}
                        >
                          {rev.changeCause || t("resource.noChangeCause")}
                        </span>
                        {isCurrent && (
                          <span
                            className={`text-xs ${
                              isSelected
                                ? "text-primary-foreground/70"
                                : "text-muted-foreground"
                            }`}
                          >
                            {t("resource.currentRevision")}
                          </span>
                        )}
                      </div>
                    </button>
                  )
                })}
              </div>
            </ScrollArea>
          )}

          {selectedRevision !== null && (
            <div className="flex items-start gap-2.5 rounded-md border border-amber-200 bg-amber-50 px-3 py-2.5 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-300">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <p>{t("resource.rollbackWarning", { revision: selectedRevision })}</p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleClose(false)}
            disabled={rollbackMutation.isPending}
          >
            {t("common.cancel")}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={
              selectedRevision === null ||
              rollbackMutation.isPending ||
              historyLoading
            }
          >
            {rollbackMutation.isPending
              ? t("resource.rollingBack")
              : selectedRevision !== null
                ? t("resource.rollbackTo", { revision: selectedRevision })
                : t("resource.rollback")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
