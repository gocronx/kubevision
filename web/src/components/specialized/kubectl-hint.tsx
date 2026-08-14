import { useState, useCallback, useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { Terminal, ChevronDown, ChevronRight, Copy, Check } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { useKubectlHint, type KubectlHintParams } from "@/hooks/use-kubectl-hint"

export interface KubectlHintProps extends KubectlHintParams {
  /** Additional class names applied to the outer wrapper. */
  className?: string
  /** Start expanded. Defaults to false (collapsed). */
  defaultOpen?: boolean
}

/**
 * KubectlHint — a collapsible banner that shows the equivalent kubectl
 * command for the current dashboard view.
 *
 * Design constraints:
 * - Unobtrusive: collapsed by default, expands on demand.
 * - Monospace code block with a dark/muted code-like background.
 * - Copy-to-clipboard with tooltip "Copied!" feedback.
 */
export function KubectlHint({
  className,
  defaultOpen = false,
  ...params
}: KubectlHintProps) {
  const { t } = useTranslation()
  const command = useKubectlHint(params)

  const [open, setOpen] = useState(defaultOpen)
  const [copied, setCopied] = useState(false)
  const copiedTimerRef = useRef<number | null>(null)

  useEffect(() => () => {
    if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
  }, [])

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(command)
      setCopied(true)
      if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
      copiedTimerRef.current = window.setTimeout(() => {
        copiedTimerRef.current = null
        setCopied(false)
      }, 2000)
    } catch {
      // Clipboard API unavailable — silently ignore; the user can still
      // manually select and copy the displayed text.
    }
  }, [command])

  return (
    <Collapsible open={open} onOpenChange={setOpen} className={cn("w-full", className)}>
      {/* ── Trigger row ─────────────────────────────────────────────── */}
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex w-full items-center gap-2 rounded-md px-3 py-1.5",
            "text-xs font-medium text-muted-foreground",
            "hover:text-foreground hover:bg-muted/60",
            "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            open && "rounded-b-none bg-muted/40",
          )}
        >
          <Terminal className="size-3.5 shrink-0" />
          <span className="mr-auto">{t("kubectl.title")}</span>

          {/* Preview the command inline when collapsed */}
          {!open && (
            <code className="hidden max-w-[420px] truncate font-mono text-xs text-muted-foreground sm:block">
              {command}
            </code>
          )}

          {open ? (
            <ChevronDown className="size-3.5 shrink-0" />
          ) : (
            <ChevronRight className="size-3.5 shrink-0" />
          )}
        </button>
      </CollapsibleTrigger>

      {/* ── Expanded content ────────────────────────────────────────── */}
      <CollapsibleContent>
        <div
          className={cn(
            "flex items-center gap-2 rounded-b-md border border-t-0",
            "border-border/60 bg-zinc-950 px-3 py-2 dark:bg-zinc-900",
          )}
        >
          {/* Command text */}
          <pre className="flex-1 overflow-x-auto font-mono text-xs leading-relaxed text-green-400 dark:text-green-300 whitespace-pre">
            {command}
          </pre>

          {/* Copy button */}
          <TooltipProvider>
            <Tooltip open={copied ? true : undefined}>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={handleCopy}
                  aria-label={t("kubectl.copy")}
                  className="shrink-0 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800"
                >
                  {copied ? (
                    <Check className="size-3.5 text-green-400" />
                  ) : (
                    <Copy className="size-3.5" />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                {copied ? t("kubectl.copied") : t("kubectl.copy")}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
