import { useEffect, useCallback, useRef } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { wsClient } from "@/lib/ws"
import type { ResourceEvent } from "@/lib/ws"
import { useAuth } from "@/stores/auth-store"

/**
 * useWebSocket connects to the WebSocket server when authenticated and manages
 * the connection lifecycle. Should be called once at the app level.
 */
export function useWebSocket() {
  const { isAuthenticated } = useAuth()
  const queryClient = useQueryClient()

  // Handle incoming resource events by invalidating TanStack Query caches.
  const handleEvent = useCallback(
    (event: ResourceEvent) => {
      // Invalidate the list query for the affected resource type.
      queryClient.invalidateQueries({
        queryKey: ["resources", event.clusterId, event.resource],
      })

      // For MODIFIED events, also invalidate the specific resource detail query.
      if (event.type === "MODIFIED" || event.type === "DELETED") {
        queryClient.invalidateQueries({
          queryKey: [
            "resource",
            event.clusterId,
            event.resource,
            event.namespace,
            event.name,
          ],
        })
      }
    },
    [queryClient],
  )

  useEffect(() => {
    if (!isAuthenticated) {
      wsClient.disconnect()
      return
    }

    // Build WebSocket URL from current location.
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/watch`
    wsClient.connect(wsUrl)
    wsClient.onEvent(handleEvent)

    return () => {
      wsClient.offEvent(handleEvent)
      wsClient.disconnect()
    }
  }, [isAuthenticated, handleEvent])
}

/**
 * useResourceSubscription subscribes to WebSocket events for a specific
 * cluster + resource type. Automatically subscribes on mount and unsubscribes
 * on unmount.
 */
export function useResourceSubscription(
  clusterId: string | undefined,
  resource: string | undefined,
) {
  const subRef = useRef<{ clusterId: string; resource: string } | null>(null)

  useEffect(() => {
    if (!clusterId || !resource) return

    wsClient.subscribe(clusterId, resource)
    subRef.current = { clusterId, resource }

    return () => {
      if (subRef.current) {
        wsClient.unsubscribe(subRef.current.clusterId, subRef.current.resource)
        subRef.current = null
      }
    }
  }, [clusterId, resource])
}
