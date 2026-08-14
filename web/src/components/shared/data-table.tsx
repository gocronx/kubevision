import { useState, useMemo, useCallback, type ReactNode } from "react"
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

export interface DataTableColumn<T = Record<string, unknown>> {
  key: string
  label: string
  sortable?: boolean
  render?: (item: T, index: number) => ReactNode
  headerRender?: () => ReactNode
  className?: string
}

interface SortState {
  key: string
  direction: "asc" | "desc"
}

interface DataTableProps<T = Record<string, unknown>> {
  columns: DataTableColumn<T>[]
  data: T[]
  isLoading?: boolean
  emptyMessage?: string
  onRowClick?: (item: T, index: number) => void
  getRowKey?: (item: T, index: number) => string
  /** Initial sort order applied when the table first renders. */
  defaultSort?: SortState | null
}

export function DataTable<T = Record<string, unknown>>({
  columns,
  data,
  isLoading = false,
  emptyMessage = "No data",
  onRowClick,
  getRowKey,
  defaultSort = null,
}: DataTableProps<T>) {
  const [sort, setSort] = useState<SortState | null>(defaultSort)

  const handleSort = useCallback((key: string) => {
    setSort((prev) => {
      if (prev?.key === key) {
        return prev.direction === "asc"
          ? { key, direction: "desc" }
          : null
      }
      return { key, direction: "asc" }
    })
  }, [])

  const sortedData = useMemo(() => {
    return [...data].sort((a, b) => {
      if (sort && columns.some((column) => column.key === sort.key)) {
        const primary = compareSortValues(
          getColumnSortValue(a, sort.key),
          getColumnSortValue(b, sort.key),
        )
        if (primary !== 0) {
          return sort.direction === "asc" ? primary : -primary
        }
      }

      return compareResourceIdentity(a, b)
    })
  }, [data, sort, columns])

  const getSortIcon = (key: string) => {
    if (sort?.key !== key) return <ArrowUpDown className="size-3 opacity-40" />
    if (sort.direction === "asc") return <ArrowUp className="size-3" />
    return <ArrowDown className="size-3" />
  }

  const tableStyle = { minWidth: `${Math.max(36, columns.length * 8)}rem` }

  if (isLoading) {
    return (
      <div className="max-w-full overflow-x-auto rounded-md border">
        <table className="w-full" style={tableStyle}>
          <thead>
            <tr className="border-b bg-muted/50">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className="px-4 py-3 text-left text-sm font-medium text-muted-foreground"
                >
                  {col.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 5 }).map((_, i) => (
              <tr key={i} className="border-b last:border-b-0">
                {columns.map((col) => (
                  <td key={col.key} className="px-4 py-3">
                    <Skeleton className="h-5 w-full max-w-[200px]" />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div className="max-w-full overflow-x-auto rounded-md border">
        <table className="w-full" style={tableStyle}>
          <thead>
            <tr className="border-b bg-muted/50">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className="px-4 py-3 text-left text-sm font-medium text-muted-foreground"
                >
                  {col.label}
                </th>
              ))}
            </tr>
          </thead>
        </table>
        <div className="flex items-center justify-center py-12">
          <p className="text-sm text-muted-foreground">{emptyMessage}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-full overflow-x-auto rounded-md border">
      <table className="w-full" style={tableStyle}>
        <thead>
          <tr className="border-b bg-muted/50">
            {columns.map((col) => (
              <th
                key={col.key}
                className={cn(
                  "px-4 py-3 text-left text-sm font-medium text-muted-foreground",
                  col.sortable && "cursor-pointer select-none hover:text-foreground",
                  col.className
                )}
                onClick={col.sortable ? () => handleSort(col.key) : undefined}
              >
                <div className="flex items-center gap-1">
                  {col.headerRender ? col.headerRender() : col.label}
                  {col.sortable && getSortIcon(col.key)}
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.map((item, index) => (
            <tr
              key={getRowKey ? getRowKey(item, index) : index}
              className={cn(
                "border-b last:border-b-0 transition-colors",
                onRowClick && "cursor-pointer hover:bg-muted/50"
              )}
              onClick={onRowClick ? () => onRowClick(item, index) : undefined}
            >
              {columns.map((col) => (
                <td
                  key={col.key}
                  className={cn("px-4 py-3 text-sm", col.className)}
                >
                  {col.render ? col.render(item, index) : null}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function compareSortValues(a: string | number, b: string | number): number {
  if (typeof a === "number" && typeof b === "number") {
    return a - b
  }
  return String(a).localeCompare(String(b))
}

// Informer-backed lists originate from a Go map and therefore have no stable
// iteration order. This identity comparator keeps rows deterministic across
// refreshes and also resolves ties in user-selected sorts.
function compareResourceIdentity(a: unknown, b: unknown): number {
  const aIdentity = getResourceIdentity(a)
  const bIdentity = getResourceIdentity(b)

  return aIdentity.namespace.localeCompare(bIdentity.namespace) ||
    aIdentity.name.localeCompare(bIdentity.name) ||
    aIdentity.uid.localeCompare(bIdentity.uid)
}

function getResourceIdentity(item: unknown) {
  if (!item || typeof item !== "object") {
    return { namespace: "", name: "", uid: "" }
  }
  const obj = item as Record<string, unknown>
  const meta = obj.metadata as Record<string, unknown> | undefined
  return {
    namespace: String(meta?.namespace ?? obj.namespace ?? ""),
    name: String(meta?.name ?? obj.name ?? ""),
    uid: String(meta?.uid ?? obj.uid ?? ""),
  }
}

/**
 * Helper to extract a sortable value from an item by column key.
 * This is used internally for sorting, relying on the rendered text content
 * would be complex, so we extract the raw metadata values.
 */
function getColumnSortValue(item: unknown, key: string): string | number {
  if (!item || typeof item !== "object") return ""
  const obj = item as Record<string, unknown>

  // Try common K8s paths
  const meta = obj.metadata as Record<string, unknown> | undefined

  switch (key) {
    case "name":
      return (meta?.name as string) ?? ""
    case "namespace":
      return (meta?.namespace as string) ?? ""
    case "age": {
      const ts = meta?.creationTimestamp as string | undefined
      return ts ? new Date(ts).getTime() : 0
    }
    case "lastSeen": {
      const ts = (obj["lastTimestamp"] as string) ?? (obj["eventTime"] as string) ?? ""
      return ts ? new Date(ts).getTime() : 0
    }
    default: {
      // Try to get nested value from common paths
      const status = obj.status as Record<string, unknown> | undefined
      const spec = obj.spec as Record<string, unknown> | undefined
      return (
        (status?.[key] as string) ??
        (spec?.[key] as string) ??
        (obj[key] as string) ??
        ""
      )
    }
  }
}
