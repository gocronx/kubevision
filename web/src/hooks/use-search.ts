import { useQuery } from "@tanstack/react-query"
import { useMemo, useState, useEffect, useRef } from "react"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// Types — mirror the backend SearchResponse shape
// ---------------------------------------------------------------------------

export interface SearchResultItem {
  name: string
  namespace?: string
  kind: string
  resourceType: string
  apiVersion: string
  labels?: Record<string, string>
}

export interface SearchResultGroup {
  resource_type: string
  items: SearchResultItem[]
  total: number
}

export interface SearchResponse {
  results: SearchResultGroup[]
  total: number
}

// ---------------------------------------------------------------------------
// API fetch
// ---------------------------------------------------------------------------

async function fetchSearch(
  clusterID: string,
  query: string,
  namespace?: string,
  types?: string[],
  limit?: number,
): Promise<SearchResponse> {
  const params: Record<string, string | number> = { q: query }
  if (namespace) params.namespace = namespace
  if (types && types.length > 0) params.types = types.join(",")
  if (limit) params.limit = limit

  const res = await api.get(`/clusters/${clusterID}/search`, { params })
  const data = res as unknown as SearchResponse
  return {
    results: data.results ?? [],
    total: data.total ?? 0,
  }
}

// ---------------------------------------------------------------------------
// Debounce hook
// ---------------------------------------------------------------------------

/** Returns a debounced copy of `value` that only updates after `delay` ms. */
export function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState<T>(value)

  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])

  return debounced
}

// ---------------------------------------------------------------------------
// useSearch hook
// ---------------------------------------------------------------------------

export interface UseSearchOptions {
  namespace?: string
  types?: string[]
  limit?: number
  /** Debounce delay in milliseconds (default: 300). */
  debounceMs?: number
  enabled?: boolean
}

/**
 * TanStack Query hook that performs a debounced global search across all
 * Kubernetes resource types in the specified cluster.
 *
 * Results are returned grouped by resource type, sorted by relevance.
 */
export function useSearch(
  clusterID: string,
  rawQuery: string,
  options: UseSearchOptions = {},
) {
  const {
    namespace,
    types,
    limit,
    debounceMs = 300,
    enabled = true,
  } = options

  const debouncedQuery = useDebounce(rawQuery.trim(), debounceMs)

  // Track whether the user has ever typed something so we don't flash "No
  // results" before the debounce resolves on the very first keystroke.
  const hasQuery = debouncedQuery.length > 0

  const query = useQuery<SearchResponse>({
    queryKey: ["search", clusterID, debouncedQuery, namespace ?? "", types?.join(",") ?? "", limit ?? 10],
    queryFn: () => fetchSearch(clusterID, debouncedQuery, namespace, types, limit),
    enabled: enabled && !!clusterID && hasQuery,
    // Keep previous results visible while a new search is in-flight so the
    // dropdown doesn't flicker empty between keystrokes.
    placeholderData: (prev) => prev,
    // Cache search results briefly; they reflect live cluster state.
    staleTime: 10_000,
  })

  // Expose a flat list of all results for convenience alongside the grouped view.
  const flatItems = useMemo<SearchResultItem[]>(() => {
    return (query.data?.results ?? []).flatMap((g) => g.items)
  }, [query.data])

  return {
    /** Grouped results by resource type. */
    groups: query.data?.results ?? [],
    /** Flat list of all matched items across all types. */
    items: flatItems,
    /** Grand total of all matches (before per-type limit). */
    total: query.data?.total ?? 0,
    isLoading: query.isFetching,
    isError: query.isError,
    /** The debounced query that was actually sent to the server. */
    activeQuery: debouncedQuery,
  }
}

// ---------------------------------------------------------------------------
// Pending indicator helper
// ---------------------------------------------------------------------------

/**
 * Returns true during the debounce window (the user has typed something but
 * the debounced value hasn't caught up yet).  Useful for showing a spinner
 * in the search input itself.
 */
export function useSearchPending(rawQuery: string, debounceMs = 300): boolean {
  const debouncedQuery = useDebounce(rawQuery.trim(), debounceMs)
  const trimmed = rawQuery.trim()
  // Pending when the live input differs from the debounced value.
  return trimmed !== debouncedQuery && trimmed.length > 0
}

// ---------------------------------------------------------------------------
// Keyboard shortcut helper
// ---------------------------------------------------------------------------

/**
 * Calls `onTrigger` when the user presses Cmd+K (Mac) or Ctrl+K (other).
 * Returns a cleanup function; pass a ref to prevent stale closures.
 */
export function useSearchShortcut(onTrigger: () => void): void {
  const handler = useRef(onTrigger)
  handler.current = onTrigger

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        handler.current()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [])
}
