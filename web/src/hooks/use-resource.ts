import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api, { getWithMeta } from "@/lib/api"
import type { ApiMeta } from "@/lib/api"

interface ResourceListOptions {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  limit?: number
  enabled?: boolean
}

interface ResourceListResult {
  items: Record<string, unknown>[]
  meta?: ApiMeta
}

interface ResourceMutationVariables {
  namespace?: string
  name?: string
  body?: Record<string, unknown>
}

// --------------------------------------------------------------------------
// Dry-run types
// --------------------------------------------------------------------------

/** A single Kubernetes resource as returned by the API. */
export interface K8sResource {
  apiVersion: string
  kind: string
  name: string
  namespace?: string
  raw?: Record<string, unknown>
}

/** Response shape from the dry-run endpoints. */
export interface DryRunResult {
  /** The current live resource. Null for create dry-runs. */
  current?: K8sResource | null
  /** What the resource would look like after the operation. */
  proposed?: K8sResource | null
  /** Whether the API server accepted the dry-run without validation errors. */
  valid: boolean
  /** Validation error messages when valid is false. */
  errors?: string[]
}

interface DryRunVariables {
  body: Record<string, unknown>
  namespace?: string
  /** Required for update dry-runs only. */
  name?: string
}

/**
 * Hook for listing Kubernetes resources.
 */
export function useResourceList(
  clusterID: string,
  resource: string,
  options: ResourceListOptions = {}
) {
  const { namespace, labelSelector, fieldSelector, limit, enabled = true } = options

  return useQuery<ResourceListResult>({
    queryKey: ["resources", clusterID, resource, namespace ?? "", labelSelector ?? "", fieldSelector ?? ""],
    queryFn: async () => {
      const params: Record<string, string | number> = {}
      if (namespace) params.namespace = namespace
      if (labelSelector) params.labelSelector = labelSelector
      if (fieldSelector) params.fieldSelector = fieldSelector
      if (limit) params.limit = limit

      const res = await getWithMeta<Record<string, unknown>[]>(
        `/clusters/${clusterID}/resources/${resource}`,
        { params }
      )
      // Backend returns { name, namespace, apiVersion, kind, raw: { metadata, spec, status, ... } }
      // Frontend expects standard K8s structure { metadata, spec, status, ... }
      // Normalize: spread raw onto top level so both access patterns work.
      const rawItems = Array.isArray(res.data) ? res.data : []
      const items = rawItems.map((item) => {
        const raw = item.raw as Record<string, unknown> | undefined
        if (raw && typeof raw === "object") {
          return { ...raw, ...item }
        }
        return item
      })
      return { items, meta: res.meta }
    },
    enabled: enabled && !!clusterID && !!resource,
  })
}

/**
 * Hook for fetching a single Kubernetes resource.
 */
export function useResource(
  clusterID: string,
  resource: string,
  namespace: string,
  name: string,
  enabled = true
) {
  return useQuery<Record<string, unknown>>({
    queryKey: ["resource", clusterID, resource, namespace, name],
    queryFn: async () => {
      const params: Record<string, string> = {}
      if (namespace) params.namespace = namespace

      const res = await api.get(
        `/clusters/${clusterID}/resources/${resource}/${name}`,
        { params }
      )
      const item = res as unknown as Record<string, unknown>
      const raw = item.raw as Record<string, unknown> | undefined
      if (raw && typeof raw === "object") {
        return { ...raw, ...item }
      }
      return item
    },
    enabled: enabled && !!clusterID && !!resource && !!name,
  })
}

/**
 * Hook for creating a Kubernetes resource.
 */
export function useCreateResource(clusterID: string, resource: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (variables: ResourceMutationVariables) => {
      const params: Record<string, string> = {}
      if (variables.namespace) params.namespace = variables.namespace

      const res = await api.post(
        `/clusters/${clusterID}/resources/${resource}`,
        variables.body,
        { params }
      )
      return res as unknown as Record<string, unknown>
    },
    onSuccess: () => {
      // Delay to allow the informer cache to receive the watch event.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, resource],
        })
      }, 2000)
    },
  })
}

/**
 * Hook for updating a Kubernetes resource.
 */
export function useUpdateResource(clusterID: string, resource: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (variables: ResourceMutationVariables) => {
      const params: Record<string, string> = {}
      if (variables.namespace) params.namespace = variables.namespace

      const res = await api.put(
        `/clusters/${clusterID}/resources/${resource}/${variables.name}`,
        variables.body,
        { params }
      )
      return res as unknown as Record<string, unknown>
    },
    onSuccess: (_data, variables) => {
      // Delay to allow the informer cache to receive the watch event.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, resource],
        })
        if (variables.name) {
          queryClient.invalidateQueries({
            queryKey: ["resource", clusterID, resource, variables.namespace ?? "", variables.name],
          })
        }
      }, 2000)
    },
  })
}

/**
 * Hook for deleting a Kubernetes resource.
 */
export function useDeleteResource(clusterID: string, resource: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (variables: ResourceMutationVariables) => {
      const params: Record<string, string> = {}
      if (variables.namespace) params.namespace = variables.namespace

      await api.delete(
        `/clusters/${clusterID}/resources/${resource}/${variables.name}`,
        { params }
      )
    },
    onSuccess: () => {
      // Delay to allow the informer cache to receive the watch event.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, resource],
        })
      }, 2000)
    },
  })
}

/**
 * Hook for performing a server-side dry-run create.
 *
 * Returns what the Kubernetes API server would produce when creating the given
 * resource manifest (with defaults filled in) without actually persisting it.
 */
export function useDryRunCreate(clusterID: string, resource: string) {
  return useMutation<DryRunResult, Error, DryRunVariables>({
    mutationFn: async (variables) => {
      const params: Record<string, string> = {}
      if (variables.namespace) params.namespace = variables.namespace

      const res = await api.post(
        `/clusters/${clusterID}/resources/${resource}/dry-run`,
        variables.body,
        { params }
      )
      return res as unknown as DryRunResult
    },
  })
}

/**
 * Hook for performing a server-side dry-run update.
 *
 * Returns both the current live resource and what the resource would look like
 * after the update, without actually applying the change.
 */
export function useDryRunUpdate(clusterID: string, resource: string) {
  return useMutation<DryRunResult, Error, DryRunVariables>({
    mutationFn: async (variables) => {
      const params: Record<string, string> = {}
      if (variables.namespace) params.namespace = variables.namespace

      const res = await api.put(
        `/clusters/${clusterID}/resources/${resource}/${variables.name}/dry-run`,
        variables.body,
        { params }
      )
      return res as unknown as DryRunResult
    },
  })
}
