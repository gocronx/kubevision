import { useState, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  Box,
  Server,
  Layers,
  Radio,
  Play,
  Clock,
  Network,
  Globe,
  FileText,
  KeyRound,
  HardDrive,
  Database,
  Boxes,
  Monitor,
  FolderOpen,
  Activity,
  Search,
  Loader2,
  Package,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import {
  CommandDialog,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandSeparator,
} from "@/components/ui/command"
import { Badge } from "@/components/ui/badge"
import { useSearch, useSearchShortcut } from "@/hooks/use-search"
import type { SearchResultItem } from "@/hooks/use-search"
import { useCluster } from "@/hooks/use-cluster"
import { Button } from "@/components/ui/button"

// ---------------------------------------------------------------------------
// Resource type → icon mapping (reuses the same icons as the sidebar)
// ---------------------------------------------------------------------------

const resourceIcons: Record<string, LucideIcon> = {
  pods: Box,
  deployments: Server,
  statefulsets: Layers,
  daemonsets: Radio,
  replicasets: Layers,
  jobs: Play,
  cronjobs: Clock,
  services: Network,
  ingresses: Globe,
  configmaps: FileText,
  secrets: KeyRound,
  persistentvolumes: HardDrive,
  persistentvolumeclaims: Database,
  storageclasses: Boxes,
  nodes: Monitor,
  namespaces: FolderOpen,
  events: Activity,
}

function ResourceIcon({ resourceType }: { resourceType: string }) {
  const Icon = resourceIcons[resourceType] ?? Package
  return <Icon className="size-4 shrink-0 text-muted-foreground" />
}

// ---------------------------------------------------------------------------
// Human-readable resource type labels
// ---------------------------------------------------------------------------

const resourceLabels: Record<string, string> = {
  pods: "Pods",
  deployments: "Deployments",
  statefulsets: "StatefulSets",
  daemonsets: "DaemonSets",
  replicasets: "ReplicaSets",
  jobs: "Jobs",
  cronjobs: "CronJobs",
  services: "Services",
  ingresses: "Ingresses",
  configmaps: "ConfigMaps",
  secrets: "Secrets",
  persistentvolumes: "Persistent Volumes",
  persistentvolumeclaims: "PVCs",
  storageclasses: "Storage Classes",
  nodes: "Nodes",
  namespaces: "Namespaces",
  events: "Events",
}

function resourceLabel(resourceType: string): string {
  return resourceLabels[resourceType] ?? resourceType
}

// ---------------------------------------------------------------------------
// Single result row
// ---------------------------------------------------------------------------

interface ResultItemProps {
  item: SearchResultItem
  onSelect: (item: SearchResultItem) => void
}

function ResultItem({ item, onSelect }: ResultItemProps) {
  return (
    <CommandItem
      // cmdk uses the value for keyboard filtering; we disable its built-in
      // filter by setting shouldFilter=false on CommandList, so this is just
      // a stable identity key.
      value={`${item.resourceType}/${item.namespace ?? ""}/${item.name}`}
      onSelect={() => onSelect(item)}
      className="flex items-center gap-3 py-2"
    >
      <ResourceIcon resourceType={item.resourceType} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate font-medium text-sm leading-tight">
          {item.name}
        </span>
        {item.namespace && (
          <span className="truncate text-xs text-muted-foreground leading-tight">
            {item.namespace}
          </span>
        )}
      </div>
      <Badge variant="secondary" className="shrink-0 font-normal text-xs">
        {item.kind}
      </Badge>
    </CommandItem>
  )
}

// ---------------------------------------------------------------------------
// Main GlobalSearch component
// ---------------------------------------------------------------------------

export function GlobalSearch() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { currentCluster } = useCluster()

  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")

  // Register Cmd+K / Ctrl+K shortcut globally.
  useSearchShortcut(useCallback(() => setOpen(true), []))

  const { groups, total, isLoading, activeQuery } = useSearch(
    currentCluster,
    query,
    { limit: 8 },
  )

  // Navigate to the resource detail page and close the dialog.
  const handleSelect = useCallback(
    (item: SearchResultItem) => {
      setOpen(false)
      setQuery("")
      const params = new URLSearchParams()
      if (item.namespace) params.set("namespace", item.namespace)
      const qs = params.toString()
      navigate(`/${item.resourceType}/${item.name}${qs ? `?${qs}` : ""}`)
    },
    [navigate],
  )

  function handleOpenChange(value: boolean) {
    setOpen(value)
    if (!value) setQuery("")
  }

  const showResults = activeQuery.length > 0
  const showEmpty = showResults && !isLoading && groups.length === 0

  return (
    <>
      {/* Trigger button rendered in the header */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        className="relative h-8 w-48 justify-start gap-2 rounded-md text-muted-foreground text-sm font-normal shadow-none md:w-56 lg:w-64"
      >
        <Search className="size-4 shrink-0" />
        <span className="hidden sm:inline">{t("search.placeholder")}</span>
        <kbd className="pointer-events-none ml-auto hidden select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 sm:flex">
          <span className="text-xs">⌘</span>K
        </kbd>
      </Button>

      {/* Command dialog */}
      <CommandDialog
        open={open}
        onOpenChange={handleOpenChange}
        title={t("search.title")}
        description={t("search.description")}
        showCloseButton={false}
      >
        {/* Search input — we manage value ourselves so the debounce lives in
            useSearch, not inside cmdk's built-in filter. */}
        <div className="flex items-center border-b px-3 gap-2 h-12">
          {isLoading && showResults ? (
            <Loader2 className="size-4 shrink-0 animate-spin opacity-50" />
          ) : (
            <Search className="size-4 shrink-0 opacity-50" />
          )}
          <input
            autoFocus
            className="flex h-full w-full bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={t("search.inputPlaceholder")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {query && (
            <button
              className="shrink-0 text-muted-foreground hover:text-foreground"
              onClick={() => setQuery("")}
              aria-label="Clear search"
            >
              <span className="text-xs">✕</span>
            </button>
          )}
        </div>

        {/* Results list — shouldFilter=false because filtering is done server-side */}
        <CommandList className="max-h-[480px]">
          {showEmpty && (
            <CommandEmpty>{t("search.noResults")}</CommandEmpty>
          )}

          {!showResults && (
            <div className="py-8 text-center text-sm text-muted-foreground">
              {t("search.hint")}
            </div>
          )}

          {groups.map((group, idx) => (
            <div key={group.resource_type}>
              {idx > 0 && <CommandSeparator />}
              <CommandGroup
                heading={
                  <span className="flex items-center gap-1.5">
                    <ResourceIcon resourceType={group.resource_type} />
                    {resourceLabel(group.resource_type)}
                    {group.total > group.items.length && (
                      <span className="ml-auto text-xs text-muted-foreground">
                        {group.items.length}/{group.total}
                      </span>
                    )}
                  </span>
                }
              >
                {group.items.map((item) => (
                  <ResultItem
                    key={`${item.resourceType}/${item.namespace ?? ""}/${item.name}`}
                    item={item}
                    onSelect={handleSelect}
                  />
                ))}
              </CommandGroup>
            </div>
          ))}

          {/* Footer showing grand total when there are results */}
          {showResults && !isLoading && total > 0 && (
            <div className="border-t px-3 py-2 text-xs text-muted-foreground">
              {t("search.totalResults", { count: total })}
            </div>
          )}
        </CommandList>
      </CommandDialog>
    </>
  )
}
