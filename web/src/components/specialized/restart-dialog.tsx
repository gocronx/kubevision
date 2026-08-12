import { AlertTriangle } from "lucide-react"
import { useTranslation } from "react-i18next"
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
import { useRestartResource } from "@/hooks/use-resource-actions"

interface RestartDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterID: string
  /** Resource kind: "deployments" | "statefulsets" | "daemonsets" */
  kind: string
  namespace: string
  name: string
}

export function RestartDialog({
  open,
  onOpenChange,
  clusterID,
  kind,
  namespace,
  name,
}: RestartDialogProps) {
  const { t } = useTranslation()
  const restartMutation = useRestartResource(clusterID, kind)

  function handleConfirm() {
    restartMutation.mutate(
      { namespace, name },
      {
        onSuccess: () => {
          toast.success(t("resource.restartToast", { name }))
          onOpenChange(false)
        },
        onError: (err) => {
          toast.error(
            err instanceof Error ? err.message : t("resource.restartFailed")
          )
        },
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{t("resource.restartTitle", { name })}</DialogTitle>
          <DialogDescription>{t("resource.restartDescription", { name, namespace })}</DialogDescription>
        </DialogHeader>

        <div className="flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <p>{t("resource.restartWarning")}</p>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={restartMutation.isPending}
          >
            {t("common.cancel")}
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={restartMutation.isPending}
          >
            {restartMutation.isPending ? t("resource.restarting") : t("resource.restart")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
