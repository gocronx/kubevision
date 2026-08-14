---
sidebar_position: 8
title: Pod Metrics
---

# Pod Metrics

KubeVision displays current CPU and memory usage in **Workloads → Pods**. The
list shows totals for each Pod and refreshes every 15 seconds while it is open.

Open a Pod and use the **Overview** tab to see:

- Total CPU and memory usage
- Configured requests and limits
- Usage as a percentage of the configured limit
- CPU and memory usage for each regular container

## Requirements

The cluster must expose the Kubernetes Metrics API through Metrics Server at
`metrics.k8s.io/v1beta1`. The credentials used to connect the cluster need
`get` and `list` access to `pods` in that API group. The KubeVision Helm chart
grants only those read operations to its in-cluster service account.

If Metrics Server is starting, a new Pod may briefly show that metrics are
pending. If the API is unavailable or access is denied, the Pods page remains
usable and shows an unavailable state instead of failing the resource request.

Metrics Server does not report container filesystem or PVC usage. Those values
require a monitoring source such as kubelet/cAdvisor or Prometheus and are not
estimated from the node filesystem shown by `df` inside a container.
