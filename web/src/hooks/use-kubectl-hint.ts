import { useMemo } from "react"

/**
 * Supported kubectl actions that map to real kubectl sub-commands.
 */
export type KubectlAction =
  | "get"
  | "describe"
  | "edit"
  | "delete"
  | "logs"
  | "exec"
  | "scale"
  | "apply"

/**
 * Parameters for generating a kubectl command string.
 *
 * - action          Sub-command to generate (get, describe, edit, …)
 * - resource        Kubernetes resource type, e.g. "pods", "deployments"
 * - name            Resource name; omit for list-scope operations
 * - namespace       Namespace; omit for cluster-scoped resources or all-namespace views
 * - clusterContext  kubeconfig context name; when provided, --context flag is appended
 * - extraArgs       Additional flags for specific actions:
 *                     logs  → { container?: string; follow?: boolean }
 *                     exec  → { container?: string; shellCmd?: string }
 *                     scale → { replicas?: number }
 */
export interface KubectlHintParams {
  action: KubectlAction
  resource: string
  name?: string
  namespace?: string
  clusterContext?: string
  extraArgs?: {
    container?: string
    follow?: boolean
    replicas?: number
    shellCmd?: string
  }
}

/**
 * Build the kubectl command string for a given set of parameters.
 * Pure function — no side effects, fully deterministic.
 */
export function buildKubectlCommand(params: KubectlHintParams): string {
  const {
    action,
    resource,
    name,
    namespace,
    clusterContext,
    extraArgs = {},
  } = params

  // Common flag fragments assembled per-need.
  const nsFlag = namespace ? `-n ${namespace}` : "--all-namespaces"
  const namespacedNsFlag = namespace ? `-n ${namespace}` : ""
  const contextFlag = clusterContext ? ` --context ${clusterContext}` : ""

  switch (action) {
    case "get": {
      // List view: no resource name; namespace or --all-namespaces.
      const ns = name ? (namespace ? `-n ${namespace}` : "") : nsFlag
      const target = name ? `${resource} ${name}` : resource
      return `kubectl get ${target}${ns ? ` ${ns}` : ""}${contextFlag}`
    }

    case "describe": {
      const target = name ? `${resource} ${name}` : resource
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      return `kubectl describe ${target}${ns}${contextFlag}`
    }

    case "edit": {
      const target = name ? `${resource} ${name}` : resource
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      return `kubectl edit ${target}${ns}${contextFlag}`
    }

    case "delete": {
      const target = name ? `${resource} ${name}` : resource
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      return `kubectl delete ${target}${ns}${contextFlag}`
    }

    case "logs": {
      const podName = name ?? "<pod-name>"
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      const container = extraArgs.container ? ` -c ${extraArgs.container}` : ""
      const follow = extraArgs.follow ? " -f" : ""
      return `kubectl logs ${podName}${ns}${container}${follow}${contextFlag}`
    }

    case "exec": {
      const podName = name ?? "<pod-name>"
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      const container = extraArgs.container ? ` -c ${extraArgs.container}` : ""
      const shell = extraArgs.shellCmd ?? "/bin/bash"
      return `kubectl exec -it ${podName}${ns}${container}${contextFlag} -- ${shell}`
    }

    case "scale": {
      const target = name ? `${resource} ${name}` : resource
      const ns = namespacedNsFlag ? ` ${namespacedNsFlag}` : ""
      const replicas =
        extraArgs.replicas !== undefined ? extraArgs.replicas : "<n>"
      return `kubectl scale ${target} --replicas=${replicas}${ns}${contextFlag}`
    }

    case "apply": {
      // apply doesn't address an individual resource by name on the CLI;
      // show the canonical file-based apply pattern instead.
      const fileName = name ? `${name}.yaml` : `${resource}.yaml`
      return `kubectl apply -f ${fileName}${contextFlag}`
    }

    default:
      return `kubectl ${action} ${resource}${contextFlag}`
  }
}

/**
 * React hook that returns a memoised kubectl command string.
 * Re-computes only when the relevant parameters change.
 */
export function useKubectlHint(params: KubectlHintParams): string {
  return useMemo(() => buildKubectlCommand(params), [
    // Stringify extraArgs so the memo dependency is stable across renders
    // without requiring callers to memoize the object themselves.
    params.action,
    params.resource,
    params.name,
    params.namespace,
    params.clusterContext,
    params.extraArgs?.container,
    params.extraArgs?.follow,
    params.extraArgs?.replicas,
    params.extraArgs?.shellCmd,
  ])
}
