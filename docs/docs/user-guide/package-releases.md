---
title: Helm Package Releases
---

# Helm Package Releases

The **Packages** workspace shows Helm releases discovered in the selected
cluster, including namespace, chart, application version, status, revision,
values, and revision history.

Open a release to inspect its redacted values and history. Authorized users can
roll back to an earlier revision or remove the release. Removal requires typing
the exact release name, and concurrent changes to the same release are rejected.

KubeVision currently manages existing releases only. Installing new charts,
upgrading chart packages, and registering repositories are not provided by this
workspace. Values returned by Helm are redacted before display, but secrets
should still be managed through Kubernetes Secrets or an external secret store.
