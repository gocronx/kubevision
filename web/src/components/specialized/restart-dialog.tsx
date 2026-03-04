import { AlertTriangle } from "lucide-react"
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
  const restartMutation = useRestartResource(clusterID, kind)

  function handleConfirm() {
    restartMutation.mutate(
      { namespace, name },
      {
        onSuccess: () => {
          toast.success(`${name} restart initiated`)
          onOpenChange(false)
        },
        onError: (err) => {
          toast.error(
            err instanceof Error ? err.message : "Failed to restart resource"
          )
        },
      }
    )
  }

  const kindLabel = kind.slice(0, -1).toLowerCase()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Restart {name}?</DialogTitle>
          <DialogDescription>
            This will trigger a rolling restart of the {kindLabel}{" "}
            <span className="font-medium text-foreground">{name}</span> in
            namespace{" "}
            <span className="font-medium text-foreground">{namespace}</span>.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-300">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <p>
            Pods will be replaced one at a time according to the update strategy.
            There may be brief disruption during the rollout.
          </p>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={restartMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={restartMutation.isPending}
          >
            {restartMutation.isPending ? "Restarting..." : "Restart"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
