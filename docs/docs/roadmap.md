---
title: Roadmap
---

# Roadmap

KubeVision already includes multi-cluster resource management, English and
Simplified Chinese UI, AI assistance, OAuth/OIDC, directory login, passkeys,
CRD discovery, guarded Helm package management, and read-only Prometheus,
Grafana, and Argo CD integration APIs.

The roadmap tracks work that is not yet delivered. Priorities may change based
on maintenance capacity and community feedback.

## Near Term

- Helm revision-to-revision Values diff and broader OCI update discovery
- Kubernetes schema-aware YAML validation in the editor
- Read-only Pod filesystem browsing and controlled downloads
- Broader end-to-end coverage for authentication and destructive workflows
- API contract generation to keep route documentation synchronized with code

## Platform Integrations

- Resource-level Argo CD and Flux status, links, and explicitly authorized sync
- Grafana dashboard views and resource-to-dashboard associations
- NetworkPolicy traffic visualization and policy impact analysis
- OpenCost-backed namespace and workload cost views

## Accessibility and Localization

English and Simplified Chinese are shipped from `web/src/i18n/`. Additional
locales, accessibility audits, and keyboard workflow improvements remain open
for contribution.

## Proposing Work

Start with a [GitHub Discussion](https://github.com/gocronx/kubevision/discussions)
that explains the operational problem and security boundary. Accepted work can
then be tracked as an issue and implemented using the
[contribution guide](/docs/development/contributing).
