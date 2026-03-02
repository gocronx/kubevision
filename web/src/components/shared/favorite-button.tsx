import { useState, useCallback } from "react"
import { Star } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useToggleFavorite, type AddFavoritePayload } from "@/hooks/use-favorites"
import { toast } from "sonner"

interface FavoriteButtonProps {
  clusterId: string
  resourceType: string
  resourceName: string
  namespace?: string
  displayName?: string
  /** Whether the resource is currently favorited. */
  isFavorited: boolean
  /** Called after a successful toggle so the parent can update state. */
  onToggled?: (favorited: boolean) => void
  size?: "sm" | "default" | "lg" | "icon"
  className?: string
}

/**
 * A star icon button that toggles a Kubernetes resource's favorited status.
 * Applies an optimistic local state update so the icon flips immediately on
 * click, before the server confirms the result.
 */
export function FavoriteButton({
  clusterId,
  resourceType,
  resourceName,
  namespace = "",
  displayName,
  isFavorited,
  onToggled,
  size = "icon",
  className,
}: FavoriteButtonProps) {
  // Optimistic local state — starts from the prop and flips immediately on click.
  const [optimisticFavorited, setOptimisticFavorited] = useState(isFavorited)

  const toggleMutation = useToggleFavorite()

  const handleClick = useCallback(
    (e: React.MouseEvent) => {
      // Prevent the button click from bubbling to parent elements such as
      // table rows that navigate to the resource detail page.
      e.stopPropagation()
      e.preventDefault()

      const next = !optimisticFavorited
      setOptimisticFavorited(next)

      const payload: AddFavoritePayload = {
        clusterId,
        resourceType,
        resourceName,
        namespace: namespace || undefined,
        displayName: displayName || resourceName,
      }

      toggleMutation.mutate(payload, {
        onSuccess: (data) => {
          onToggled?.(data.favorited)
          toast.success(
            data.favorited
              ? `"${displayName ?? resourceName}" added to favorites`
              : `"${displayName ?? resourceName}" removed from favorites`
          )
        },
        onError: () => {
          // Revert optimistic update on failure.
          setOptimisticFavorited(!next)
          toast.error("Failed to update favorites")
        },
      })
    },
    [
      clusterId,
      displayName,
      namespace,
      optimisticFavorited,
      resourceName,
      resourceType,
      toggleMutation,
      onToggled,
    ]
  )

  // Keep local state in sync if the prop changes from outside (e.g., after a
  // list refresh).
  if (optimisticFavorited !== isFavorited && !toggleMutation.isPending) {
    setOptimisticFavorited(isFavorited)
  }

  return (
    <Button
      variant="ghost"
      size={size}
      onClick={handleClick}
      disabled={toggleMutation.isPending}
      aria-label={optimisticFavorited ? "Remove from favorites" : "Add to favorites"}
      title={optimisticFavorited ? "Remove from favorites" : "Add to favorites"}
      className={cn("transition-colors", className)}
    >
      <Star
        className={cn(
          "size-4 transition-colors",
          optimisticFavorited
            ? "fill-yellow-400 text-yellow-400"
            : "text-muted-foreground"
        )}
      />
    </Button>
  )
}
