package ai

import (
	"context"
	"fmt"

	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/plugin/prometheus"
	"github.com/gocronx/kubevision/internal/repository"
)

// mutationTools are the tools that change cluster state and therefore require
// explicit user approval before execution.
var mutationTools = map[string]bool{
	"create_resource": true,
	"update_resource": true,
	"patch_resource":  true,
	"delete_resource": true,
}

func isMutation(tool string) bool { return mutationTools[tool] }

// promGetter lazily resolves the Prometheus plugin, which may be unconfigured.
type promGetter func() (*prometheus.Plugin, bool)

// executor runs tool calls against a single cluster. It is created per chat
// request once the target cluster has been resolved.
type executor struct {
	k8s         repository.K8sResourceRepo
	clusterMgr  *cluster.Manager
	registry    *resource.Registry
	prom        promGetter
	clusterName string
}

// execute dispatches a tool call to its implementation. Mutation tools are
// executed here too, but only the handler/agent calls this for mutations after
// the user has approved them.
func (e *executor) execute(ctx context.Context, tool string, args map[string]any) (string, error) {
	switch tool {
	case "get_resource":
		return e.getResource(ctx, args)
	case "list_resources":
		return e.listResources(ctx, args)
	case "get_pod_logs":
		return e.getPodLogs(ctx, args)
	case "get_cluster_overview":
		return e.getClusterOverview(ctx)
	case "query_prometheus":
		return e.queryPrometheus(ctx, args)
	case "create_resource":
		return e.createResource(ctx, args)
	case "update_resource":
		return e.updateResource(ctx, args)
	case "patch_resource":
		return e.patchResource(ctx, args)
	case "delete_resource":
		return e.deleteResource(ctx, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

// toolDefinitions returns the function schemas advertised to the model.
func toolDefinitions() []Tool {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	kindProp := str("Lowercase plural Kubernetes resource name, e.g. pods, deployments, services, nodes.")
	nsProp := str("Namespace. Omit for cluster-scoped resources or to query all namespaces where applicable.")

	def := func(name, desc string, props map[string]any, required []string) Tool {
		return Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        name,
				Description: desc,
				Parameters: map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		}
	}

	return []Tool{
		def("get_resource", "Fetch a single Kubernetes resource as YAML.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "name": str("Resource name.")},
			[]string{"kind", "name"}),

		def("list_resources", "List Kubernetes resources of a kind, optionally filtered by namespace and label selector.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "labelSelector": str("Optional label selector, e.g. app=nginx.")},
			[]string{"kind"}),

		def("get_pod_logs", "Read recent logs from a pod container.",
			map[string]any{
				"namespace": nsProp,
				"name":      str("Pod name."),
				"container": str("Container name. Defaults to the first container."),
				"tailLines": map[string]any{"type": "integer", "description": "Number of trailing lines to read (default 200)."},
				"previous":  map[string]any{"type": "boolean", "description": "Read logs from the previous terminated container instance."},
			},
			[]string{"namespace", "name"}),

		def("get_cluster_overview", "Summarize cluster health: node, namespace, pod and deployment counts.",
			map[string]any{}, []string{}),

		def("query_prometheus", "Run an instant PromQL query against the configured Prometheus.",
			map[string]any{"query": str("PromQL expression.")}, []string{"query"}),

		def("create_resource", "Create a resource from YAML. Requires user approval before it runs.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "yaml": str("Full resource manifest in YAML.")},
			[]string{"kind", "yaml"}),

		def("update_resource", "Replace an existing resource from YAML. Requires user approval before it runs.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "name": str("Resource name."), "yaml": str("Full updated manifest in YAML.")},
			[]string{"kind", "name", "yaml"}),

		def("patch_resource", "Apply a strategic-merge patch to a resource. Requires user approval before it runs.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "name": str("Resource name."), "patch": str("Strategic-merge patch as JSON, e.g. {\"spec\":{\"replicas\":3}}.")},
			[]string{"kind", "name", "patch"}),

		def("delete_resource", "Delete a resource. Requires user approval before it runs.",
			map[string]any{"kind": kindProp, "namespace": nsProp, "name": str("Resource name.")},
			[]string{"kind", "name"}),
	}
}
