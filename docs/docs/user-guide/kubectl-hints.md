---
sidebar_position: 10
title: kubectl Hints
---

# kubectl Hints

Every action you take in KubeVision shows the equivalent `kubectl` command. This lets you learn the Kubernetes CLI naturally while using the dashboard — and gives you an escape hatch to automate or script any operation.

## Where to Find Hints

A **kubectl** chip appears in the footer of every action:

- Resource list page footer — shows the `get` command for the current view
- Resource detail page header — shows the `get -o yaml` command
- After any create / update / delete — a toast notification shows the equivalent command
- YAML editor toolbar — shows `apply -f` or `delete -f` with a temp filename

Click the **Copy** icon next to any hint to copy it to your clipboard instantly.

## How It Works

Hints are generated entirely on the frontend from the current UI state — no extra API call is made. The frontend knows the cluster, namespace, resource type, and resource name, and assembles the command from a lookup table of `kubectl` verb mappings.

## Example Hints

### Listing Resources

```bash
# Viewing Deployments in namespace "default" on prod-cluster
kubectl get deployments -n default --context prod-cluster

# Viewing all Pods across namespaces
kubectl get pods -A --context prod-cluster
```

### Getting a Resource

```bash
kubectl get deployment nginx-deployment -n default -o yaml --context prod-cluster
```

### Creating / Applying

```bash
kubectl apply -f nginx-deployment.yaml --context prod-cluster
```

### Deleting

```bash
kubectl delete deployment nginx-deployment -n default --context prod-cluster
```

### Dry Run

```bash
kubectl apply -f nginx-deployment.yaml --dry-run=server --context prod-cluster
```

### Pod Operations

```bash
# Open a terminal
kubectl exec -it nginx-pod -n default -c app --context prod-cluster -- bash

# Stream logs
kubectl logs -f nginx-pod -n default -c app --context prod-cluster

# Previous container logs
kubectl logs nginx-pod -n default -c app --previous --context prod-cluster
```

### Scaling

```bash
kubectl scale deployment nginx-deployment --replicas=3 -n default --context prod-cluster
```

## Context Flag

The `--context` flag is always included, matching the cluster name registered in KubeVision. If the cluster name differs from the context name in your local kubeconfig, edit the cluster settings to align them.

:::tip
kubectl hints are an excellent learning tool for engineers who are new to Kubernetes. Pair them with the [Resource Topology](/docs/user-guide/resource-topology) view to build a mental model of the cluster while getting work done.
:::

## Related

- [Resource Management](/docs/user-guide/resource-crud) — Full CRUD operations
- [Pod Terminal](/docs/user-guide/pod-terminal) — In-browser shell access
- [Pod Logs](/docs/user-guide/pod-logs) — Real-time log streaming
