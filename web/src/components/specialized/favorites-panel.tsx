import { useState, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  Star,
  ChevronDown,
  ChevronRight,
  X,
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
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"
import { useFavorites, useRemoveFavorite, useReorderFavorites, type Favorite } from "@/hooks/use-favorites"
import { useAuth } from "@/stores/auth-store"
import { readFavoritesPanelOpen, writeFavoritesPanelOpen } from "./favorites-panel-preference"
import { toast } from "sonner"

// ---------------------------------------------------------------------------
// Resource type -> icon mapping (mirrors the resource-ui-config entries)
// ---------------------------------------------------------------------------

const RESOURCE_ICONS: Record<string, LucideIcon> = {
  pods: Box,
  deployments: Server,
  statefulsets: Layers,
  daemonsets: Radio,
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

function resourceIcon(resourceType: string): LucideIcon {
  return RESOURCE_ICONS[resourceType.toLowerCase()] ?? Star
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildDetailPath(fav: Favorite): string {
  const params = new URLSearchParams()
  if (fav.namespace) params.set("namespace", fav.namespace)
  const qs = params.toString() ? `?${params.toString()}` : ""
  return `/${fav.resourceType}/${fav.resourceName}${qs}`
}

// ---------------------------------------------------------------------------
// FavoritesPanel
// ---------------------------------------------------------------------------

interface FavoritesPanelProps {
  /** Extra class names applied to the outer wrapper. */
  className?: string
}

/**
 * A collapsible sidebar section that lists the current user's favorited
 * Kubernetes resources. Supports click-to-navigate, remove-on-hover, and
 * drag-to-reorder.
 */
export function FavoritesPanel({ className }: FavoritesPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuth()
  const userID = user?.id ?? 0
  const [isOpen, setIsOpen] = useState(() => readFavoritesPanelOpen(userID))

  const { data: favorites = [], isLoading } = useFavorites()
  const removeMutation = useRemoveFavorite()
  const reorderMutation = useReorderFavorites()

  const handleOpenChange = useCallback((open: boolean) => {
    setIsOpen(open)
    writeFavoritesPanelOpen(userID, open)
  }, [userID])

  // Drag-and-drop state
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null)

  const handleNavigate = useCallback(
    (fav: Favorite) => {
      navigate(buildDetailPath(fav))
    },
    [navigate]
  )

  const handleRemove = useCallback(
    (e: React.MouseEvent, fav: Favorite) => {
      e.stopPropagation()
      removeMutation.mutate(fav.id, {
        onError: () => toast.error("Failed to remove favorite"),
      })
    },
    [removeMutation]
  )

  // Drag-and-drop handlers
  const handleDragStart = useCallback(
    (e: React.DragEvent, index: number) => {
      setDragIndex(index)
      e.dataTransfer.effectAllowed = "move"
    },
    []
  )

  const handleDragOver = useCallback(
    (e: React.DragEvent, index: number) => {
      e.preventDefault()
      e.dataTransfer.dropEffect = "move"
      setDragOverIndex(index)
    },
    []
  )

  const handleDrop = useCallback(
    (e: React.DragEvent, dropIndex: number) => {
      e.preventDefault()
      if (dragIndex === null || dragIndex === dropIndex) {
        setDragIndex(null)
        setDragOverIndex(null)
        return
      }

      const reordered = [...favorites]
      const [moved] = reordered.splice(dragIndex, 1)
      reordered.splice(dropIndex, 0, moved)

      const orderedIds = reordered.map((f) => f.id)
      reorderMutation.mutate(
        { orderedIds },
        { onError: () => toast.error("Failed to reorder favorites") }
      )

      setDragIndex(null)
      setDragOverIndex(null)
    },
    [dragIndex, favorites, reorderMutation]
  )

  const handleDragEnd = useCallback(() => {
    setDragIndex(null)
    setDragOverIndex(null)
  }, [])

  return (
    <Collapsible open={isOpen} onOpenChange={handleOpenChange} className={cn("group/collapsible", className)}>
      <SidebarGroup className="py-0">
        <SidebarGroupLabel asChild>
          <CollapsibleTrigger className="flex w-full items-center justify-between px-2 py-1 text-xs font-medium uppercase tracking-wider text-muted-foreground hover:text-foreground transition-colors">
            <span className="flex items-center gap-1.5">
              <Star className="size-3" />
              {t("nav.favorites")}
            </span>
            {isOpen ? (
              <ChevronDown className="size-3 transition-transform" />
            ) : (
              <ChevronRight className="size-3 transition-transform" />
            )}
          </CollapsibleTrigger>
        </SidebarGroupLabel>

        <CollapsibleContent>
          <SidebarGroupContent>
            <ScrollArea className="max-h-[300px]">
              <SidebarMenu>
                {isLoading && (
                  <div className="flex flex-col gap-1 px-2 py-1">
                    <Skeleton className="h-7 w-full" />
                    <Skeleton className="h-7 w-4/5" />
                    <Skeleton className="h-7 w-3/5" />
                  </div>
                )}

                {!isLoading && favorites.length === 0 && (
                  <li className="px-2 py-3 text-xs text-muted-foreground text-center">
                    {t("favorites.empty")}
                  </li>
                )}

                {!isLoading &&
                  favorites.map((fav, index) => {
                    const Icon = resourceIcon(fav.resourceType)
                    const isDragging = dragIndex === index
                    const isOver = dragOverIndex === index && dragIndex !== index

                    return (
                      <SidebarMenuItem
                        key={fav.id}
                        draggable
                        onDragStart={(e) => handleDragStart(e, index)}
                        onDragOver={(e) => handleDragOver(e, index)}
                        onDrop={(e) => handleDrop(e, index)}
                        onDragEnd={handleDragEnd}
                        className={cn(
                          "group/fav-item cursor-grab active:cursor-grabbing transition-opacity",
                          isDragging && "opacity-40",
                          isOver && "border-t-2 border-primary"
                        )}
                      >
                        <SidebarMenuButton
                          onClick={() => handleNavigate(fav)}
                          tooltip={`${fav.displayName}${fav.namespace ? ` (${fav.namespace})` : ""}`}
                          className="pr-7 relative"
                        >
                          <Icon className="size-4 shrink-0 text-blue-500" />
                          <div className="flex flex-col min-w-0">
                            <span className="truncate text-sm font-medium">
                              {fav.displayName}
                            </span>
                            {fav.namespace && (
                              <span className="truncate text-xs text-muted-foreground">
                                {fav.namespace}
                              </span>
                            )}
                          </div>
                        </SidebarMenuButton>

                        {/* Remove button — visible on hover */}
                        <Button
                          variant="ghost"
                          size="icon"
                          className={cn(
                            "absolute right-1 top-1/2 -translate-y-1/2 size-5 p-0",
                            "opacity-0 group-hover/fav-item:opacity-100 transition-opacity",
                            "text-muted-foreground hover:text-destructive hover:bg-transparent"
                          )}
                          onClick={(e) => handleRemove(e, fav)}
                          aria-label={`Remove ${fav.displayName} from favorites`}
                          title="Remove from favorites"
                        >
                          <X className="size-3" />
                        </Button>
                      </SidebarMenuItem>
                    )
                  })}
              </SidebarMenu>
            </ScrollArea>
          </SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  )
}
