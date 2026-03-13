---
title: Roadmap
---

# Roadmap

KubeVision follows a phased release plan. Each phase builds on the previous one — core stability first, advanced integrations second, intelligent automation third.

:::note
This roadmap reflects current intentions and is subject to change based on community feedback and contributor availability. Track progress and vote on features in [GitHub Discussions](https://github.com/gocronx/kubevision/discussions).
:::

---

## Phase 1 — v0.2 (Short-term)

Focus: **Developer experience polish and common daily-use features.**

### Helm Chart Management

Manage Helm releases directly from the dashboard — no need to switch to the terminal for routine chart operations.

| Capability | Detail |
|-----------|--------|
| List releases | All releases across namespaces, with chart name + version |
| Upgrade release | Upload or select a new chart version, edit values inline |
| Rollback release | Roll back to any previous revision |
| Uninstall release | With optional `--purge` for CRD cleanup |
| Release history | Revision timeline with diff between values |

### YAML Schema Validation

The built-in YAML editor will validate resource manifests against the official Kubernetes JSON Schema before submission, providing inline red underlines for unknown fields, missing required fields, and type mismatches — without requiring an API Server round-trip.

```yaml
# Example: typo caught before apply
spec:
  replicas: "3"   # Error: must be integer, not string
  tempalte:       # Error: unknown field (did you mean 'template'?)
```

### Pod File Browser

Browse the filesystem of a running container without opening a full terminal session. The file browser provides read access and allows downloading individual files — useful for inspecting config files, logs written to disk, or generated artifacts.

```
/etc/nginx/
  ├── nginx.conf        [download]
  ├── conf.d/
  │   └── default.conf  [download]
/var/log/nginx/
  ├── access.log        [download]
  └── error.log         [download]
```

---

## Phase 2 — v0.3 (Mid-term)

Focus: **GitOps and observability integration for platform engineering teams.**

### GitOps Integration (ArgoCD / Flux)

Surface ArgoCD and Flux sync status directly on resource cards — no need to switch to a separate GitOps UI.

| Integration | Features |
|------------|---------|
| ArgoCD | App sync status, health, diff vs Git, manual sync trigger |
| Flux | Kustomization and HelmRelease status, reconcile trigger |
| Both | Out-of-sync badge on Deployments and Services, link to source Git commit |

```yaml
# config.yaml
plugins:
  argocd:
    enabled: true
    endpoint: "http://argocd-server.argocd.svc.cluster.local"
    token: "<argocd-api-token>"
```

### Network Policy Visualization

Render an interactive graph showing which pods can communicate with which other pods based on active `NetworkPolicy` resources. Quickly identify overly permissive policies or unexpected isolation.

- Node-link diagram with namespace swimlanes
- Click a pod to highlight all allowed ingress/egress paths
- Diff view: compare policy graph before and after a proposed `NetworkPolicy` change (uses dry-run diff under the hood)

### Cost Analysis (OpenCost)

Integrate with [OpenCost](https://www.opencost.io/) to surface per-namespace and per-workload cost estimates alongside resource utilization metrics.

| View | Cost data added |
|------|----------------|
| Namespace list | Monthly cost estimate per namespace |
| Deployment detail | Cost per replica, projected monthly cost |
| Node detail | Hourly node cost vs utilization efficiency |
| Global overview | Top-10 most expensive workloads |

:::tip
OpenCost must be installed separately in your cluster. KubeVision reads from its API — it does not replicate cost data into its own database.
:::

---

## Phase 3 — v0.4+ (Long-term)

Focus: **Intelligent automation and broader accessibility.**

### AI Assistant

A natural-language assistant embedded in the dashboard to help operators diagnose and resolve issues without leaving the UI.

Planned capabilities:

- **Explain resource** — "What does this CrashLoopBackOff mean and how do I fix it?"
- **Generate YAML** — "Write a Deployment for a 3-replica nginx with a readiness probe on /healthz"
- **Analyze logs** — Summarize the last 500 log lines and highlight anomalies
- **Suggest action** — Given a failing pod, suggest the most likely remediation steps

The assistant runs queries against a configurable LLM endpoint (self-hosted Ollama or OpenAI-compatible API). No cluster data is sent to external services unless you configure an external endpoint.

### Multi-language Expansion

The UI currently ships in English. Planned additional locales:

| Locale | Status |
|--------|--------|
| English (`en`) | Shipped |
| Simplified Chinese (`zh-CN`) | Planned |
| Japanese (`ja`) | Planned |
| Spanish (`es`) | Planned |

Translations are managed via `i18next` and community-contributed JSON files under `web/src/locales/`. Contributors are welcome.

### OAuth / OIDC via Dex

Replace the current local-only login with a full federated identity flow using [Dex](https://dexidp.io/) as the OIDC broker.

| Provider | Via Dex |
|---------|---------|
| GitHub | Yes |
| Google | Yes |
| Microsoft Azure AD | Yes |
| LDAP / Active Directory | Yes |
| SAML 2.0 | Yes (Dex connector) |

Existing local accounts and 2FA will remain supported alongside OIDC for environments that cannot federate identity.

---

## Contributing to the Roadmap

Have a feature idea not listed here? The best way to influence the roadmap is to:

1. Open a [GitHub Discussion](https://github.com/gocronx/kubevision/discussions) and describe the use case.
2. If there is community interest, a maintainer will create a tracking issue and add it to the roadmap.
3. If you want to implement it yourself, see [Contributing](/docs/development/contributing) to get started.

## Related

- [Introduction](/docs/intro) — What KubeVision does today
- [Comparison](/docs/comparison) — How KubeVision compares to alternatives
- [Contributing](/docs/development/contributing) — Help build these features
