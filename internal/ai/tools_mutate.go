package ai

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

func (e *executor) createResource(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	raw := argString(args, "yaml")
	if kind == "" || raw == "" {
		return "", fmt.Errorf("kind and yaml are required")
	}
	obj, err := parseManifest(raw)
	if err != nil {
		return "", err
	}
	if err := e.validateManifestKind(kind, obj); err != nil {
		return "", err
	}
	namespace := argString(args, "namespace")
	if err := validateManifestTarget("", namespace, obj); err != nil {
		return "", err
	}
	if namespace == "" {
		namespace = manifestNamespace(obj)
	}
	res, err := e.k8s.Create(ctx, e.clusterName, kind, namespace, obj)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created %s %s/%s.", res.Kind, res.Namespace, res.Name), nil
}

func (e *executor) updateResource(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	name := argString(args, "name")
	raw := argString(args, "yaml")
	if kind == "" || name == "" || raw == "" {
		return "", fmt.Errorf("kind, name and yaml are required")
	}
	obj, err := parseManifest(raw)
	if err != nil {
		return "", err
	}
	if err := e.validateManifestKind(kind, obj); err != nil {
		return "", err
	}
	namespace := argString(args, "namespace")
	if err := validateManifestTarget(name, namespace, obj); err != nil {
		return "", err
	}
	if namespace == "" {
		namespace = manifestNamespace(obj)
	}
	res, err := e.k8s.Update(ctx, e.clusterName, kind, namespace, name, obj)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Updated %s %s/%s.", res.Kind, res.Namespace, res.Name), nil
}

func validateManifestTarget(name, namespace string, obj map[string]any) error {
	meta, _ := obj["metadata"].(map[string]any)
	manifestName, _ := meta["name"].(string)
	manifestNamespace, _ := meta["namespace"].(string)
	if manifestName == "" {
		return fmt.Errorf("manifest metadata.name is required")
	}
	if name != "" && manifestName != name {
		return fmt.Errorf("manifest name %q does not match requested name %q", manifestName, name)
	}
	if namespace != "" && manifestNamespace != "" && manifestNamespace != namespace {
		return fmt.Errorf("manifest namespace %q does not match requested namespace %q", manifestNamespace, namespace)
	}
	return nil
}

func (e *executor) validateManifestKind(resourceName string, obj map[string]any) error {
	meta, ok := e.registry.Get(strings.ToLower(resourceName))
	if !ok {
		return fmt.Errorf("unknown resource kind: %s", resourceName)
	}
	manifestKind, _ := obj["kind"].(string)
	if manifestKind == "" || !strings.EqualFold(manifestKind, meta.GVK.Kind) {
		return fmt.Errorf("manifest kind %q does not match requested resource %q", manifestKind, resourceName)
	}
	return nil
}

func (e *executor) patchResource(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	name := argString(args, "name")
	patch := argString(args, "patch")
	if kind == "" || name == "" || patch == "" {
		return "", fmt.Errorf("kind, name and patch are required")
	}
	res, err := e.k8s.Patch(ctx, e.clusterName, kind, argString(args, "namespace"), name, []byte(patch))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Patched %s %s/%s.", res.Kind, res.Namespace, res.Name), nil
}

func (e *executor) deleteResource(ctx context.Context, args map[string]any) (string, error) {
	kind := argString(args, "kind")
	name := argString(args, "name")
	if kind == "" || name == "" {
		return "", fmt.Errorf("kind and name are required")
	}
	namespace := argString(args, "namespace")
	if err := e.k8s.Delete(ctx, e.clusterName, kind, namespace, name); err != nil {
		return "", err
	}
	return fmt.Sprintf("Deleted %s %s/%s.", kind, namespace, name), nil
}

// parseManifest decodes a YAML (or JSON) manifest into an unstructured object.
func parseManifest(raw string) (map[string]any, error) {
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("empty manifest")
	}
	return obj, nil
}

// manifestNamespace reads .metadata.namespace from a parsed manifest.
func manifestNamespace(obj map[string]any) string {
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		return ""
	}
	ns, _ := meta["namespace"].(string)
	return ns
}
