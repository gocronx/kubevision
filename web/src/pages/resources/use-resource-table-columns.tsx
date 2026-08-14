import { useMemo, type Dispatch, type SetStateAction } from "react"
import { useTranslation } from "react-i18next"
import { FavoriteButton } from "@/components/shared/favorite-button"
import { ResourceActions } from "@/components/shared/resource-actions"
import { StatusBadge } from "@/components/shared/status-badge"
import type { DataTableColumn } from "@/components/shared/data-table"
import type { ResourceUIConfig } from "@/config/resource-ui-config"
import type { Favorite } from "@/hooks/use-favorites"
import type { PodMetrics } from "@/hooks/use-resource"
import { extractColumnValue } from "@/lib/k8s-utils"
import { formatBytes, formatCPU } from "@/lib/pod-metrics"

export type K8sItem = Record<string, unknown>

interface ResourceMetadata {
  uid?: string
  name?: string
  namespace?: string
}

interface ResourceTableColumnsOptions {
  config?: ResourceUIConfig
  resource: string
  clusterID: string
  items: K8sItem[]
  selectedKeys: Set<string>
  setSelectedKeys: Dispatch<SetStateAction<Set<string>>>
  canMutate: boolean
  favorites: Favorite[]
  onRefresh: () => void
  onEdit: (name: string, namespace?: string) => void
}

const statusColumns = new Set(["status"])
const ageColumns = new Set(["age", "lastSchedule"])

export function getResourceRowKey(item: K8sItem): string {
  const metadata = item.metadata as ResourceMetadata | undefined
  return metadata?.uid ?? `${metadata?.namespace ?? ""}-${metadata?.name ?? ""}`
}

export function useResourceTableColumns({
  config,
  resource,
  clusterID,
  items,
  selectedKeys,
  setSelectedKeys,
  canMutate,
  favorites,
  onRefresh,
  onEdit,
}: ResourceTableColumnsOptions): DataTableColumn<K8sItem>[] {
  const { t } = useTranslation()

  return useMemo(() => {
    const toggleSelect = (key: string) => {
      setSelectedKeys((previous) => {
        const next = new Set(previous)
        if (next.has(key)) next.delete(key)
        else next.add(key)
        return next
      })
    }

    const toggleAll = () => {
      setSelectedKeys((previous) => {
        if (previous.size === items.length && items.length > 0) return new Set()
        return new Set(items.map(getResourceRowKey))
      })
    }

    const selectionColumn: DataTableColumn<K8sItem> = {
      key: "_select",
      label: "",
      sortable: false,
      className: "w-[40px]",
      headerRender: () => (
        <input
          type="checkbox"
          checked={selectedKeys.size > 0 && selectedKeys.size === items.length}
          ref={(element) => {
            if (element) element.indeterminate = selectedKeys.size > 0 && selectedKeys.size < items.length
          }}
          onChange={toggleAll}
          className="size-4 rounded border-muted-foreground"
          onClick={(event) => event.stopPropagation()}
        />
      ),
      render: (item) => {
        const key = getResourceRowKey(item)
        return (
          <input
            type="checkbox"
            checked={selectedKeys.has(key)}
            onChange={() => toggleSelect(key)}
            className="size-4 rounded border-muted-foreground"
            onClick={(event) => event.stopPropagation()}
          />
        )
      },
    }

    const configuredColumns = config?.columns ?? [{ key: "name", label: "Name", sortable: true }]
    const columns: DataTableColumn<K8sItem>[] = [
      ...(canMutate ? [selectionColumn] : []),
      ...configuredColumns.map((column) => ({
        key: column.key,
        label: t(`resourceColumns.${column.key}`, { defaultValue: column.label }),
        sortable: column.sortable,
        render: (item: K8sItem) => {
          const value = extractColumnValue(resource, item, column.key)
          if (statusColumns.has(column.key)) return <StatusBadge status={value} />
          if (ageColumns.has(column.key)) return <span className="text-muted-foreground">{value}</span>
          if (resource === "pods" && (column.key === "cpu" || column.key === "memory")) {
            const metrics = item.metrics as PodMetrics | undefined
            if (!metrics) return <span className="text-muted-foreground">-</span>
            return <span className="tabular-nums">{column.key === "cpu" ? formatCPU(metrics.cpuMilli) : formatBytes(metrics.memoryBytes)}</span>
          }
          if (column.key === "name") return <span className="font-medium text-foreground">{value}</span>
          return <span>{value}</span>
        },
      })),
    ]

    columns.push({
      key: "_actions",
      label: t("common.actions"),
      sortable: false,
      className: "w-[90px]",
      render: (item) => {
        const metadata = item.metadata as ResourceMetadata | undefined
        const spec = item.spec as { replicas?: number } | undefined
        const name = metadata?.name ?? ""
        const namespace = metadata?.namespace ?? ""
        const isFavorited = favorites.some((favorite) =>
          String(favorite.clusterId) === String(clusterID) &&
          favorite.resourceType === resource &&
          favorite.resourceName === name &&
          (favorite.namespace ?? "") === namespace
        )

        return (
          <div className="flex items-center gap-0.5">
            <FavoriteButton
              clusterId={String(clusterID)}
              resourceType={resource}
              resourceName={name}
              namespace={namespace}
              isFavorited={isFavorited}
              size="icon"
              className="size-7"
            />
            <ResourceActions
              clusterID={clusterID}
              resource={resource}
              name={name}
              namespace={metadata?.namespace}
              currentReplicas={spec?.replicas ?? 0}
              readOnly={!canMutate}
              onDeleted={onRefresh}
              onEdit={() => onEdit(name, metadata?.namespace)}
            />
          </div>
        )
      },
    })

    return columns
  }, [canMutate, clusterID, config, favorites, items, onEdit, onRefresh, resource, selectedKeys, setSelectedKeys, t])
}
