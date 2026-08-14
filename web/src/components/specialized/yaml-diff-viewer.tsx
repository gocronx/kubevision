import { useMemo } from "react"
import { cn } from "@/lib/utils"

// --------------------------------------------------------------------------
// Minimal line-level diff algorithm
// --------------------------------------------------------------------------

type DiffLineKind = "unchanged" | "added" | "removed"

interface DiffLine {
  kind: DiffLineKind
  leftNum: number | null  // 1-based line number in the left (original) text
  rightNum: number | null // 1-based line number in the right (proposed) text
  content: string
}

/**
 * Compute a line-level diff between two multi-line strings using the
 * patience-style longest common subsequence approach.
 *
 * Returns an array of DiffLine entries that can be rendered in a side-by-side
 * or unified view.
 */
function diffLines(original: string, proposed: string): DiffLine[] {
  const leftLines = original.split("\n")
  const rightLines = proposed.split("\n")

  // Build LCS table.
  const lLen = leftLines.length
  const rLen = rightLines.length
  const dp: number[][] = Array.from({ length: lLen + 1 }, () =>
    new Array(rLen + 1).fill(0)
  )

  for (let i = lLen - 1; i >= 0; i--) {
    for (let j = rLen - 1; j >= 0; j--) {
      if (leftLines[i] === rightLines[j]) {
        dp[i][j] = 1 + dp[i + 1][j + 1]
      } else {
        dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1])
      }
    }
  }

  // Backtrack to build diff entries.
  const result: DiffLine[] = []
  let i = 0
  let j = 0
  let leftNum = 1
  let rightNum = 1

  while (i < lLen || j < rLen) {
    if (i < lLen && j < rLen && leftLines[i] === rightLines[j]) {
      result.push({
        kind: "unchanged",
        leftNum: leftNum++,
        rightNum: rightNum++,
        content: leftLines[i],
      })
      i++
      j++
    } else if (j < rLen && (i >= lLen || dp[i][j + 1] >= dp[i + 1][j])) {
      result.push({
        kind: "added",
        leftNum: null,
        rightNum: rightNum++,
        content: rightLines[j],
      })
      j++
    } else {
      result.push({
        kind: "removed",
        leftNum: leftNum++,
        rightNum: null,
        content: leftLines[i],
      })
      i++
    }
  }

  return result
}

// --------------------------------------------------------------------------
// Rendering helpers
// --------------------------------------------------------------------------

const LINE_BG: Record<DiffLineKind, string> = {
  unchanged: "",
  added: "bg-green-500/10 dark:bg-green-500/15",
  removed: "bg-red-500/10 dark:bg-red-500/15",
}

const LINE_PREFIX: Record<DiffLineKind, string> = {
  unchanged: " ",
  added: "+",
  removed: "-",
}

const PREFIX_COLOR: Record<DiffLineKind, string> = {
  unchanged: "text-muted-foreground",
  added: "text-green-600 dark:text-green-400 font-semibold",
  removed: "text-red-600 dark:text-red-400 font-semibold",
}

// --------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------

export interface YamlDiffViewerProps {
  /** YAML / text for the left (original / current) pane. */
  original: string
  /** YAML / text for the right (proposed) pane. */
  proposed: string
  /** Label displayed above the left pane. Defaults to "Current". */
  originalLabel?: string
  /** Label displayed above the right pane. Defaults to "Proposed". */
  proposedLabel?: string
  /** Extra classes applied to the outer wrapper. */
  className?: string
}

// --------------------------------------------------------------------------
// Component
// --------------------------------------------------------------------------

/**
 * YamlDiffViewer renders a side-by-side, line-level diff between two YAML
 * strings.  No external diff library is required — the diff is computed
 * client-side using a simple LCS algorithm.
 *
 * Lines are colour-coded:
 *  - Red background with a "-" prefix: present in original, removed in proposed.
 *  - Green background with a "+" prefix: absent in original, added in proposed.
 *  - No background: unchanged.
 */
export function YamlDiffViewer({
  original,
  proposed,
  originalLabel = "Current",
  proposedLabel = "Proposed",
  className,
}: YamlDiffViewerProps) {
  const diff = useMemo(() => diffLines(original, proposed), [original, proposed])

  // Separate lines per pane for side-by-side display.
  // Each pane has its own line number sequence; blank entries fill in the gaps.
  const leftRows: Array<{ lineNum: number | null; content: string; kind: DiffLineKind }> = []
  const rightRows: Array<{ lineNum: number | null; content: string; kind: DiffLineKind }> = []

  for (const line of diff) {
    if (line.kind === "unchanged") {
      leftRows.push({ lineNum: line.leftNum, content: line.content, kind: "unchanged" })
      rightRows.push({ lineNum: line.rightNum, content: line.content, kind: "unchanged" })
    } else if (line.kind === "removed") {
      leftRows.push({ lineNum: line.leftNum, content: line.content, kind: "removed" })
      rightRows.push({ lineNum: null, content: "", kind: "unchanged" })
    } else {
      leftRows.push({ lineNum: null, content: "", kind: "unchanged" })
      rightRows.push({ lineNum: line.rightNum, content: line.content, kind: "added" })
    }
  }

  const totalRows = leftRows.length

  const hasChanges = diff.some((l) => l.kind !== "unchanged")

  return (
    <div className={cn("flex flex-col overflow-x-auto rounded-md border", className)}>
      {/* Column headers */}
      <div className="grid min-w-[40rem] grid-cols-2 divide-x divide-border border-b bg-muted/60">
        <div className="px-4 py-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {originalLabel}
        </div>
        <div className="px-4 py-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {proposedLabel}
        </div>
      </div>

      {!hasChanges && (
        <div className="px-4 py-6 text-center text-sm text-muted-foreground">
          No differences found — the resource is identical.
        </div>
      )}

      {hasChanges && (
        <div className="max-h-[500px] min-w-[40rem] overflow-auto">
          <div className="grid grid-cols-2 divide-x divide-border">
            {/* Left pane */}
            <div className="font-mono text-xs leading-5 overflow-x-auto">
              {Array.from({ length: totalRows }, (_, idx) => {
                const row = leftRows[idx]
                return (
                  <div
                    key={idx}
                    className={cn(
                      "flex min-w-0 whitespace-pre",
                      LINE_BG[row.kind],
                    )}
                  >
                    {/* Line number gutter */}
                    <span
                      className="select-none w-10 shrink-0 pr-2 text-right text-muted-foreground/50 border-r border-border"
                      aria-hidden
                    >
                      {row.lineNum ?? ""}
                    </span>
                    {/* Diff prefix (+/-/space) */}
                    <span
                      className={cn(
                        "select-none w-5 shrink-0 text-center",
                        PREFIX_COLOR[row.kind],
                      )}
                      aria-hidden
                    >
                      {row.kind !== "unchanged" ? LINE_PREFIX[row.kind] : " "}
                    </span>
                    {/* Line content */}
                    <span className="flex-1 pl-1 break-all">{row.content}</span>
                  </div>
                )
              })}
            </div>

            {/* Right pane */}
            <div className="font-mono text-xs leading-5 overflow-x-auto">
              {Array.from({ length: totalRows }, (_, idx) => {
                const row = rightRows[idx]
                return (
                  <div
                    key={idx}
                    className={cn(
                      "flex min-w-0 whitespace-pre",
                      LINE_BG[row.kind],
                    )}
                  >
                    {/* Line number gutter */}
                    <span
                      className="select-none w-10 shrink-0 pr-2 text-right text-muted-foreground/50 border-r border-border"
                      aria-hidden
                    >
                      {row.lineNum ?? ""}
                    </span>
                    {/* Diff prefix */}
                    <span
                      className={cn(
                        "select-none w-5 shrink-0 text-center",
                        PREFIX_COLOR[row.kind],
                      )}
                      aria-hidden
                    >
                      {row.kind !== "unchanged" ? LINE_PREFIX[row.kind] : " "}
                    </span>
                    {/* Line content */}
                    <span className="flex-1 pl-1 break-all">{row.content}</span>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
