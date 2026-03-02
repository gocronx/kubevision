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
}

export function DataTable<T = Record<string, unknown>>({
  columns,
  data,
  isLoading = false,
  emptyMessage = "No data",
  onRowClick,
  getRowKey,
}: DataTableProps<T>) {
  const [sort, setSort] = useState<SortState | null>(null)

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
    if (!sort) return data

    return [...data].sort((a, b) => {
      const aCol = columns.find((c) => c.key === sort.key)
      if (!aCol?.render) return 0

      // We need string representations for comparison
      // For simple sorting, we compare the raw extracted values
      const aVal = String(getColumnSortValue(a, sort.key))
      const bVal = String(getColumnSortValue(b, sort.key))

      // Try numeric comparison first
      const aNum = Number(aVal)
      const bNum = Number(bVal)
      if (!isNaN(aNum) && !isNaN(bNum)) {
        return sort.direction === "asc" ? aNum - bNum : bNum - aNum
      }

      // Fallback to string comparison
      const cmp = aVal.localeCompare(bVal)
      return sort.direction === "asc" ? cmp : -cmp
    })
  }, [data, sort, columns])

  const getSortIcon = (key: string) => {
    if (sort?.key !== key) return <ArrowUpDown className="size-3 opacity-40" />
    if (sort.direction === "asc") return <ArrowUp className="size-3" />
    return <ArrowDown className="size-3" />
  }

  if (isLoading) {
    return (
      <div className="rounded-md border">
        <table className="w-full">
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
      <div className="rounded-md border">
        <table className="w-full">
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
    <div className="rounded-md border">
      <table className="w-full">
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
