import { useState, useEffect } from "react"
import { Minus, Plus } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useScaleResource } from "@/hooks/use-resource-actions"

interface ScaleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterID: string
  /** Resource kind: "deployments" | "statefulsets" | "replicasets" */
  kind: string
  namespace: string
  name: string
  /** The current replica count shown as the starting value. */
  currentReplicas: number
}

export function ScaleDialog({
  open,
  onOpenChange,
  clusterID,
  kind,
  namespace,
  name,
  currentReplicas,
}: ScaleDialogProps) {
  const [desired, setDesired] = useState(currentReplicas)
  const scaleMutation = useScaleResource(clusterID, kind)

  // Reset desired replicas whenever the dialog opens or the current value changes.
  useEffect(() => {
    if (open) {
      setDesired(currentReplicas)
    }
  }, [open, currentReplicas])

  function handleDecrement() {
    setDesired((prev) => Math.max(0, prev - 1))
  }

  function handleIncrement() {
    setDesired((prev) => prev + 1)
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const value = parseInt(e.target.value, 10)
    if (!isNaN(value) && value >= 0) {
      setDesired(value)
    } else if (e.target.value === "") {
      setDesired(0)
    }
  }

  function handleConfirm() {
    scaleMutation.mutate(
      { namespace, name, replicas: desired },
      {
        onSuccess: () => {
          toast.success(
            `${name} scaled to ${desired} replica${desired !== 1 ? "s" : ""}`
          )
          onOpenChange(false)
        },
        onError: (err) => {
          toast.error(
            err instanceof Error ? err.message : "Failed to scale resource"
          )
        },
      }
    )
  }

  const hasChanged = desired !== currentReplicas

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-sm"
        onPointerDownOutside={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Scale {name}</DialogTitle>
          <DialogDescription>
            Adjust the number of replicas for this{" "}
            {kind.slice(0, -1).toLowerCase()}.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-2">
          {/* Current vs desired summary */}
          <div className="flex items-center justify-between rounded-md border bg-muted/40 px-4 py-3 text-sm">
            <div className="flex flex-col gap-0.5">
              <span className="text-muted-foreground">Current</span>
              <span className="text-xl font-semibold">{currentReplicas}</span>
            </div>
            <div className="text-muted-foreground">→</div>
            <div className="flex flex-col items-end gap-0.5">
              <span className="text-muted-foreground">Desired</span>
              <span
                className={`text-xl font-semibold ${
                  desired > currentReplicas
                    ? "text-green-600 dark:text-green-400"
                    : desired < currentReplicas
                      ? "text-amber-600 dark:text-amber-400"
                      : ""
                }`}
              >
                {desired}
              </span>
            </div>
          </div>

          {/* Stepper control */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="replicas-input">Replicas</Label>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={handleDecrement}
                disabled={desired <= 0 || scaleMutation.isPending}
                aria-label="Decrease replicas"
              >
                <Minus className="size-4" />
              </Button>
              <Input
                id="replicas-input"
                type="number"
                min={0}
                value={desired}
                onChange={handleInputChange}
                disabled={scaleMutation.isPending}
                className="w-24 text-center [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
              />
              <Button
                variant="outline"
                size="icon"
                onClick={handleIncrement}
                disabled={scaleMutation.isPending}
                aria-label="Increase replicas"
              >
                <Plus className="size-4" />
              </Button>
            </div>
          </div>

          {desired === 0 && (
            <p className="text-sm text-amber-600 dark:text-amber-400">
              Scaling to 0 will stop all pods for this workload.
            </p>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={scaleMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={scaleMutation.isPending || !hasChanged}
          >
            {scaleMutation.isPending ? "Scaling..." : "Apply"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
