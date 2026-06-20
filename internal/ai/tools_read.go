package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocronx/kubevision/internal/kubernetes/exec"
	"github.com/gocronx/kubevision/internal/repository"
	"sigs.k8s.io/yaml"
)

// maxLogBytes caps how much log output is fed back to the model to protect the
// token budget.
const maxLogBytes = 16 * 1024

func (e *executor) getResource(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	name := argString(args, "name")
	if kind == "" || name == "" {
		return "", fmt.Errorf("kind and name are required")
	}
	res, err := e.k8s.Get(ctx, e.clusterName, kind, argString(args, "namespace"), name)
	if err != nil {
		return "", err
	}
	out, err := yaml.Marshal(res.Raw)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (e *executor) listResources(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	if kind == "" {
		return "", fmt.Errorf("kind is required")
	}
	opts := repository.ListOptions{LabelSelector: argString(args, "labelSelector"), Limit: 500}
	list, err := e.k8s.List(ctx, e.clusterName, kind, argString(args, "namespace"), opts)
	if err != nil {
		return "", err
	}
	if len(list.Items) == 0 {
		return fmt.Sprintf("No %s found.", kind), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d %s:\n", len(list.Items), kind)
	for _, item := range list.Items {
		if item.Namespace != "" {
			fmt.Fprintf(&b, "- %s/%s", item.Namespace, item.Name)
		} else {
			fmt.Fprintf(&b, "- %s", item.Name)
		}
		if status := summarizeStatus(item.Raw); status != "" {
			fmt.Fprintf(&b, " [%s]", status)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (e *executor) getPodLogs(ctx context.Context, args map[string]any) (string, error) {
	namespace := argString(args, "namespace")
	name := argString(args, "name")
	if namespace == "" || name == "" {
		return "", fmt.Errorf("namespace and name are required")
	}
	restCfg, err := e.clusterMgr.RESTConfig(e.clusterName)
	if err != nil {
		return "", err
	}
	clientset, err := exec.NewClientset(restCfg)
	if err != nil {
		return "", err
	}

	tail := int64(argInt(args, "tailLines", 200))
	var buf bytes.Buffer
	logErr := exec.StreamLogs(ctx, clientset, exec.LogOptions{
		Namespace: namespace,
		PodName:   name,
		Container: argString(args, "container"),
		Previous:  argBool(args, "previous"),
		TailLines: &tail,
	}, &buf)
	if logErr != nil {
		return "", logErr
	}

	out := buf.Bytes()
	if len(out) > maxLogBytes {
		out = out[len(out)-maxLogBytes:]
		return "...(truncated)...\n" + string(out), nil
	}
	if len(out) == 0 {
		return "(no log output)", nil
	}
	return string(out), nil
}

func (e *executor) getClusterOverview(ctx context.Context) (string, error) {
	count := func(kind string) (int, string) {
		list, err := e.k8s.List(ctx, e.clusterName, kind, "", repository.ListOptions{Limit: 1000})
		if err != nil {
			return 0, err.Error()
		}
		return len(list.Items), ""
	}

	var b strings.Builder
	b.WriteString("Cluster overview:\n")
	for _, kind := range []string{"nodes", "namespaces", "pods", "deployments", "services"} {
		n, errMsg := count(kind)
		if errMsg != "" {
			fmt.Fprintf(&b, "- %s: unavailable (%s)\n", kind, errMsg)
			continue
		}
		fmt.Fprintf(&b, "- %s: %d\n", kind, n)
	}

	// Surface not-running pods so the model can drill in.
	pods, err := e.k8s.List(ctx, e.clusterName, "pods", "", repository.ListOptions{Limit: 1000})
	if err == nil {
		var unhealthy []string
		for _, p := range pods.Items {
			if phase := podPhase(p.Raw); phase != "" && phase != "Running" && phase != "Succeeded" {
				unhealthy = append(unhealthy, fmt.Sprintf("%s/%s (%s)", p.Namespace, p.Name, phase))
			}
		}
		if len(unhealthy) > 0 {
			fmt.Fprintf(&b, "Pods not Running/Succeeded (%d):\n", len(unhealthy))
			for _, u := range unhealthy {
				fmt.Fprintf(&b, "- %s\n", u)
			}
		}
	}
	return b.String(), nil
}

func (e *executor) queryPrometheus(ctx context.Context, args map[string]any) (string, error) {
	query := argString(args, "query")
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	if e.prom == nil {
		return "", fmt.Errorf("prometheus is not configured")
	}
	p, ok := e.prom()
	if !ok {
		return "", fmt.Errorf("prometheus plugin is not available")
	}
	result, err := p.Query(ctx, query)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// summarizeStatus extracts a short status hint from a resource's raw object.
func summarizeStatus(raw map[string]any) string {
	if phase := podPhase(raw); phase != "" {
		return phase
	}
	return ""
}

// podPhase reads .status.phase from an unstructured object, if present.
func podPhase(raw map[string]any) string {
	status, ok := raw["status"].(map[string]any)
	if !ok {
		return ""
	}
	if phase, ok := status["phase"].(string); ok {
		return phase
	}
	return ""
}

func argInt(args map[string]any, key string, def int) int {
	if args == nil {
		return def
	}
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return def
	}
}

func argBool(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	b, _ := args[key].(bool)
	return b
}
