import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

interface ResourceListOptions {
  namespace?: string
  labelSelector?: string
  limit?: number
  enabled?: boolean
}

interface ResourceListMetadata {
  continue?: string
  remainingItemCount?: number
}

interface ResourceListResponse {
  items: Record<string, unknown>[]
  metadata?: ResourceListMetadata
}

interface ResourceMutationVariables {
  namespace?: string
  name?: string
  body?: Record<string, unknown>
}

/**
 * Hook for listing Kubernetes resources.
 */
export function useResourceList(
  clusterID: string,
  resource: string,
  options: ResourceListOptions = {}
) {
  const { namespace, labelSelector, limit, enabled = true } = options

  return useQuery<ResourceListResponse>({
    queryKey: ["resources", clusterID, resource, namespace ?? "", labelSelector ?? ""],
    queryFn: async () => {
      const params: Record<string, string | number> = {}
      if (namespace) params.namespace = namespace
      if (labelSelector) params.labelSelector = labelSelector
      if (limit) params.limit = limit

      const res = await api.get(
        `/clusters/${clusterID}/resources/${resource}`,
        { params }
      )
      // api interceptor unwraps ApiResponse.data
      const data = res as unknown as ResourceListResponse
      return {
        items: data.items ?? [],
        metadata: data.metadata,
      }
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
  name: string
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
      return res as unknown as Record<string, unknown>
    },
    enabled: !!clusterID && !!resource && !!name,
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
      queryClient.invalidateQueries({
        queryKey: ["resources", clusterID, resource],
      })
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
      queryClient.invalidateQueries({
        queryKey: ["resources", clusterID, resource],
      })
      if (variables.name) {
        queryClient.invalidateQueries({
          queryKey: ["resource", clusterID, resource, variables.namespace ?? "", variables.name],
        })
      }
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
      queryClient.invalidateQueries({
        queryKey: ["resources", clusterID, resource],
      })
    },
  })
}
