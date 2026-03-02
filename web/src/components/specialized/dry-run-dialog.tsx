import { useCallback } from "react"
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react"
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
import { YamlDiffViewer } from "@/components/specialized/yaml-diff-viewer"
import { type DryRunResult } from "@/hooks/use-resource"
import { toYaml } from "@/lib/k8s-utils"

// --------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------

export interface DryRunDialogProps {
  /** Controls dialog visibility. */
  open: boolean
  onOpenChange: (open: boolean) => void

  /**
   * The result from the dry-run API call. Pass `undefined` while the
   * request is in-flight (the dialog will show a loading state) or `null`
   * when the dialog has not been triggered yet.
   */
  dryRunResult: DryRunResult | undefined | null

  /** Whether the dry-run API call is still in-flight. */
  isLoading: boolean

  /**
   * Human-readable title for what is being previewed.
   * E.g. "Preview Create: my-deployment" or "Preview Update: my-configmap".
   */
  title?: string

  /**
   * Called when the user clicks "Apply" — the parent should then submit the
   * actual (non-dry-run) create/update request.
   */
  onApply: () => void

  /** Whether the Apply action is currently in-flight. */
  isApplying?: boolean

  /** Operation type — changes copy in the dialog. */
  operation?: "create" | "update"
}

// --------------------------------------------------------------------------
// Component
// --------------------------------------------------------------------------

/**
 * DryRunDialog shows the diff between the current resource state and the
 * proposed change produced by a Kubernetes server-side dry-run.
 *
 * Flow:
 *  1. User clicks "Preview Changes" in an edit/create form.
 *  2. Parent calls the dry-run API and passes the result here.
 *  3. This dialog shows the diff and asks the user to confirm or cancel.
 *  4. On "Apply" the parent submits the real change.
 */
export function DryRunDialog({
  open,
  onOpenChange,
  dryRunResult,
  isLoading,
  title = "Preview Changes",
  onApply,
  isApplying = false,
  operation = "update",
}: DryRunDialogProps) {
  const handleApply = useCallback(() => {
    onApply()
  }, [onApply])

  // Derive the YAML strings for the diff viewer.
  const originalYaml = dryRunResult?.current?.raw
    ? toYaml(dryRunResult.current.raw)
    : ""

  const proposedYaml = dryRunResult?.proposed?.raw
    ? toYaml(dryRunResult.proposed.raw)
    : ""

  const isValid = dryRunResult?.valid ?? false
  const errors = dryRunResult?.errors ?? []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {!isLoading && dryRunResult && (
              isValid ? (
                <CheckCircle2 className="size-5 text-green-500 shrink-0" />
              ) : (
                <AlertTriangle className="size-5 text-destructive shrink-0" />
              )
            )}
            {title}
          </DialogTitle>
          <DialogDescription>
            {operation === "create"
              ? "Review what the Kubernetes API server would create before applying."
              : "Review the diff between the current resource and your proposed changes before applying."}
          </DialogDescription>
        </DialogHeader>

        {/* Loading state */}
        {isLoading && (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <Loader2 className="size-8 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Running dry-run against the Kubernetes API server…
            </p>
          </div>
        )}

        {/* Validation errors */}
        {!isLoading && dryRunResult && !isValid && errors.length > 0 && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 p-4">
            <p className="mb-2 text-sm font-semibold text-destructive">
              Validation failed — the resource was rejected by the API server:
            </p>
            <ul className="list-inside list-disc space-y-1">
              {errors.map((msg, i) => (
                <li key={i} className="font-mono text-xs text-destructive/90 break-all">
                  {msg}
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Create preview: show proposed only */}
        {!isLoading && dryRunResult && isValid && operation === "create" && proposedYaml && (
          <div className="flex flex-col gap-2">
            <p className="text-sm text-muted-foreground">
              The following resource will be created (showing the API server response with defaults applied):
            </p>
            <ScrollArea className="max-h-[500px] rounded-md border">
              <pre className="p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
                {proposedYaml}
              </pre>
            </ScrollArea>
          </div>
        )}

        {/* Update preview: side-by-side diff */}
        {!isLoading && dryRunResult && isValid && operation === "update" && (
          <YamlDiffViewer
            original={originalYaml}
            proposed={proposedYaml}
            originalLabel="Current"
            proposedLabel="Proposed"
          />
        )}

        {/* Empty state: dialog opened but dry-run not yet triggered */}
        {!isLoading && !dryRunResult && (
          <div className="flex items-center justify-center py-12">
            <p className="text-sm text-muted-foreground">
              No preview available yet.
            </p>
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isApplying}
          >
            Cancel
          </Button>
          <Button
            onClick={handleApply}
            disabled={isLoading || isApplying || !isValid || !dryRunResult}
          >
            {isApplying ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Applying…
              </>
            ) : (
              "Apply"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
