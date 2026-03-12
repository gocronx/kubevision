---
sidebar_position: 8
title: Dry-Run Preview
---

# Dry-Run Preview

Dry-Run Preview lets you validate and preview a change before it is applied to the cluster. The Kubernetes API Server performs the full admission and validation pass — so what you see is exactly what would happen — without actually writing anything.

## How to Use

### From the YAML Editor

1. Open any resource and click **Edit YAML** (or click **Create** for a new resource)
2. Make your changes in the Monaco editor
3. Click **Dry Run** instead of **Save**
4. A side-by-side diff opens: **current spec** on the left, **proposed spec** on the right
5. Review the diff, then either click **Apply** to save for real or **Cancel** to keep editing

### What the Diff Shows

| Panel | Content |
|-------|---------|
| Left (current) | The resource YAML as it exists in the cluster right now |
| Right (proposed) | Your edited YAML after the API Server normalizes it (defaults injected, etc.) |

The diff highlights only meaningful changes — `managedFields` and server-assigned metadata are stripped from the comparison.

## API Reference

```http
POST /api/v1/clusters/:cluster/namespaces/:namespace/resources/:resource/:name/dry-run
Content-Type: application/json

{
  "manifest": { /* full resource object */ }
}
```

Response:

```json
{
  "current":  { /* existing resource */ },
  "proposed": { /* API Server normalized result */ },
  "valid":    true,
  "warnings": []
}
```

If the API Server rejects the change, `valid` is `false` and the error message is surfaced directly in the UI.

## Admission Webhook Validation

Because the dry run goes through the real API Server with `dryRun: All`, all admission webhooks (OPA/Gatekeeper, Kyverno, etc.) run against your change. Policy violations are reported before anything is written.

```yaml
# Example: Kyverno policy violation returned in warnings
- "Resource 'api-server' violates policy 'require-resource-limits': container 'app' has no CPU limit"
```

:::info
Dry-run is supported for all 26+ resource types KubeVision manages, including CRDs.
:::

## Use Cases

- **Safe production edits** — Verify a Deployment image bump won't be rejected by policy
- **Template development** — Confirm a new manifest is well-formed before first apply
- **Education** — See exactly what fields the API Server adds by default (e.g., `defaultMode` on volumes)

## Related

- [Resource Management](/docs/user-guide/resource-crud) — Full CRUD workflow
- [Cross-Cluster Diff](/docs/user-guide/cross-cluster-diff) — Compare the same resource across environments
- [kubectl Hints](/docs/user-guide/kubectl-hints) — Equivalent `kubectl apply --dry-run=server` command
