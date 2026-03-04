import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ScaleRequest {
  replicas: number
}

export interface RollbackRequest {
  /** Pass 0 (or omit) to roll back to the previous revision. */
  revision?: number
}

export interface RolloutRevision {
  revision: number
  changeCause: string
}

// ---------------------------------------------------------------------------
// Base URL helpers
// ---------------------------------------------------------------------------

function actionBase(clusterID: string, namespace: string, kind: string, name: string) {
  return `/clusters/${clusterID}/namespaces/${namespace}/${kind}/${name}`
}

function deploymentActionBase(clusterID: string, namespace: string, name: string) {
  return `/clusters/${clusterID}/namespaces/${namespace}/deployments/${name}`
}

// ---------------------------------------------------------------------------
// Scale
// ---------------------------------------------------------------------------

/**
 * Mutation to scale a workload (Deployment, StatefulSet, ReplicaSet).
 * Invalidates the resource list on success so the table refreshes automatically.
 */
export function useScaleResource(clusterID: string, kind: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      namespace,
      name,
      replicas,
    }: {
      namespace: string
      name: string
      replicas: number
    }) => {
      await api.put(`${actionBase(clusterID, namespace, kind, name)}/scale`, {
        replicas,
      } satisfies ScaleRequest)
    },
    onSuccess: (_data, { name, replicas }) => {
      // Optimistically update spec.replicas in all matching cached queries so
      // the UI reflects the change instantly (the backend informer cache may
      // lag behind the K8s API by a few hundred milliseconds).
      queryClient.setQueriesData<{ items: Record<string, unknown>[]; meta?: unknown }>(
        { queryKey: ["resources", clusterID, kind] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            items: old.items.map((item) => {
              const meta = item.metadata as Record<string, unknown> | undefined
              if (meta?.name !== name) return item
              const spec = (item.spec ?? {}) as Record<string, unknown>
              return { ...item, spec: { ...spec, replicas } }
            }),
          }
        },
      )
      // Also invalidate after a short delay to pick up the full state
      // (e.g. status.readyReplicas) once the informer has caught up.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, kind],
        })
      }, 2000)
    },
  })
}

// ---------------------------------------------------------------------------
// Restart
// ---------------------------------------------------------------------------

/**
 * Mutation to restart a workload (Deployment, StatefulSet, DaemonSet).
 */
export function useRestartResource(clusterID: string, kind: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      namespace,
      name,
    }: {
      namespace: string
      name: string
    }) => {
      await api.post(`${actionBase(clusterID, namespace, kind, name)}/restart`)
    },
    onSuccess: () => {
      // Delay to allow the informer cache to receive the watch event.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, kind],
        })
      }, 2000)
    },
  })
}

// ---------------------------------------------------------------------------
// Rollout history
// ---------------------------------------------------------------------------

/**
 * Query that fetches the rollout history for a Deployment.
 * Only runs when `enabled` is true (i.e., the rollback dialog is open).
 */
export function useRolloutHistory(
  clusterID: string,
  namespace: string,
  name: string,
  enabled: boolean
) {
  return useQuery<RolloutRevision[]>({
    queryKey: ["rollout-history", clusterID, namespace, name],
    queryFn: async () => {
      const res = await api.get(
        `${deploymentActionBase(clusterID, namespace, name)}/history`
      )
      return (res as unknown as RolloutRevision[]) ?? []
    },
    enabled: enabled && !!clusterID && !!namespace && !!name,
    // History does not change frequently; a 30-second stale time avoids
    // redundant requests when the dialog is repeatedly opened.
    staleTime: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

/**
 * Mutation to roll back a Deployment to a previous revision.
 */
export function useRollbackDeployment(clusterID: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      namespace,
      name,
      revision,
    }: {
      namespace: string
      name: string
      revision?: number
    }) => {
      await api.post(
        `${deploymentActionBase(clusterID, namespace, name)}/rollback`,
        { revision: revision ?? 0 } satisfies RollbackRequest
      )
    },
    onSuccess: () => {
      // Delay to allow the informer cache to receive the watch event.
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ["resources", clusterID, "deployments"],
        })
        queryClient.invalidateQueries({
          queryKey: ["rollout-history", clusterID],
        })
      }, 2000)
    },
  })
}
