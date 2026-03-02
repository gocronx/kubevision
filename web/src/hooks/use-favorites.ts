import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

export interface Favorite {
  id: number
  createdAt: string
  userId: number
  clusterId: string
  namespace: string
  resourceType: string
  resourceName: string
  displayName: string
  sortOrder: number
}

export interface AddFavoritePayload {
  clusterId: string
  namespace?: string
  resourceType: string
  resourceName: string
  displayName?: string
}

export interface ReorderFavoritesPayload {
  orderedIds: number[]
}

export interface ToggleFavoriteResponse {
  favorited: boolean
  favorite?: Favorite
}

export interface CheckFavoriteResponse {
  favorited: boolean
  favorite?: Favorite
}

const FAVORITES_QUERY_KEY = ["favorites"] as const

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

export function useFavorites() {
  return useQuery<Favorite[]>({
    queryKey: FAVORITES_QUERY_KEY,
    queryFn: async () => {
      const res = await api.get("/favorites")
      return (res as unknown as Favorite[]) ?? []
    },
  })
}

// ---------------------------------------------------------------------------
// Add
// ---------------------------------------------------------------------------

export function useAddFavorite() {
  const queryClient = useQueryClient()

  return useMutation<Favorite, Error, AddFavoritePayload>({
    mutationFn: async (payload) => {
      const res = await api.post("/favorites", payload)
      return res as unknown as Favorite
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FAVORITES_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Remove
// ---------------------------------------------------------------------------

export function useRemoveFavorite() {
  const queryClient = useQueryClient()

  return useMutation<void, Error, number>({
    mutationFn: async (id) => {
      await api.delete(`/favorites/${id}`)
    },
    onMutate: async (id) => {
      // Optimistic update: remove immediately from cache.
      await queryClient.cancelQueries({ queryKey: FAVORITES_QUERY_KEY })
      const previous = queryClient.getQueryData<Favorite[]>(FAVORITES_QUERY_KEY)
      queryClient.setQueryData<Favorite[]>(FAVORITES_QUERY_KEY, (old) =>
        old ? old.filter((f) => f.id !== id) : []
      )
      return { previous }
    },
    onError: (_err, _id, context) => {
      // Roll back on failure.
      const ctx = context as { previous?: Favorite[] } | undefined
      if (ctx?.previous) {
        queryClient.setQueryData(FAVORITES_QUERY_KEY, ctx.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: FAVORITES_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Toggle
// ---------------------------------------------------------------------------

export function useToggleFavorite() {
  const queryClient = useQueryClient()

  return useMutation<ToggleFavoriteResponse, Error, AddFavoritePayload>({
    mutationFn: async (payload) => {
      const res = await api.post("/favorites/toggle", payload)
      return res as unknown as ToggleFavoriteResponse
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: FAVORITES_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Reorder
// ---------------------------------------------------------------------------

export function useReorderFavorites() {
  const queryClient = useQueryClient()

  return useMutation<void, Error, ReorderFavoritesPayload>({
    mutationFn: async (payload) => {
      await api.put("/favorites/reorder", payload)
    },
    onMutate: async (payload) => {
      // Optimistic update: reorder immediately in cache.
      await queryClient.cancelQueries({ queryKey: FAVORITES_QUERY_KEY })
      const previous = queryClient.getQueryData<Favorite[]>(FAVORITES_QUERY_KEY)
      queryClient.setQueryData<Favorite[]>(FAVORITES_QUERY_KEY, (old) => {
        if (!old) return old
        const indexMap = new Map(payload.orderedIds.map((id, i) => [id, i]))
        return [...old].sort(
          (a, b) => (indexMap.get(a.id) ?? 0) - (indexMap.get(b.id) ?? 0)
        )
      })
      return { previous }
    },
    onError: (_err, _payload, context) => {
      const ctx = context as { previous?: Favorite[] } | undefined
      if (ctx?.previous) {
        queryClient.setQueryData(FAVORITES_QUERY_KEY, ctx.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: FAVORITES_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Check (single resource)
// ---------------------------------------------------------------------------

export interface CheckFavoriteParams {
  clusterId: string
  resourceType: string
  name: string
  namespace?: string
}

export function useCheckFavorite(params: CheckFavoriteParams, enabled = true) {
  return useQuery<CheckFavoriteResponse>({
    queryKey: ["favorites", "check", params.clusterId, params.resourceType, params.name, params.namespace ?? ""],
    queryFn: async () => {
      const searchParams = new URLSearchParams({
        cluster_id: params.clusterId,
        resource_type: params.resourceType,
        name: params.name,
      })
      if (params.namespace) {
        searchParams.set("namespace", params.namespace)
      }
      const res = await api.get(`/favorites/check?${searchParams.toString()}`)
      return res as unknown as CheckFavoriteResponse
    },
    enabled: enabled && !!params.clusterId && !!params.resourceType && !!params.name,
  })
}
