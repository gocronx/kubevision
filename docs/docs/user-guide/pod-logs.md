---
sidebar_position: 4
title: Pod Logs
---

# Pod Logs

KubeVision streams container logs in real time directly to the browser. No log aggregation pipeline is required — it talks to the Kubernetes API Server's log endpoint over WebSocket.

## Opening Logs

1. Navigate to **Workloads → Pods** and click a Pod
2. Select the **Logs** tab
3. Logs begin streaming immediately from the selected container

## Container Selection

For multi-container Pods, use the **Container** dropdown at the top of the Logs tab to switch between containers. The stream reconnects automatically on selection change.

## Features

| Feature | How to Use |
|---------|-----------|
| **Real-time streaming** | Logs tail continuously via WebSocket — no manual refresh needed |
| **Search / Filter** | Type in the search box to highlight matching lines (case-insensitive) |
| **Timestamp display** | Toggle the **Timestamps** switch to show/hide RFC3339 timestamps |
| **Previous container** | Enable **Previous** to view logs from the last terminated container instance |
| **Download** | Click **Download** to save the current log buffer as a `.log` file |
| **Pause / Resume** | Click **Pause** to freeze the stream and scroll freely, then **Resume** to catch up |

## Log Search

The search bar performs a live client-side filter on the buffered lines:

```
# Examples of search patterns
Error
OOMKilled
"connection refused"
```

Matched lines are highlighted in yellow. Non-matching lines are dimmed but not hidden, preserving context.

:::tip
To filter by log level, search for `ERROR`, `WARN`, or `INFO`. Most structured logging frameworks emit these tokens consistently.
:::

## Controlling Log Volume

| Option | Default | Description |
|--------|---------|-------------|
| **Tail lines** | 100 | Number of historical lines loaded on open |
| **Max buffer** | 5,000 lines | Lines kept in memory; oldest are evicted when full |

Adjust **Tail lines** in the dropdown next to the Container selector.

## Previous Container Logs

If a container has restarted (e.g., after an OOMKill or crash), enable the **Previous** toggle to view the logs of the terminated instance. This is equivalent to:

```bash
kubectl logs <pod> -c <container> --previous
```

## Related

- [Pod Terminal](/docs/user-guide/pod-terminal) — Execute commands inside a running container
- [kubectl Hints](/docs/user-guide/kubectl-hints) — Copy the equivalent `kubectl logs` command
