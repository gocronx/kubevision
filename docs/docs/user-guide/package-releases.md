---
title: Helm Package Releases
---

# Helm Package Releases

The **Packages** workspace provides four views for managing Helm software in the
selected cluster:

- **Releases** lists installed releases, values, rendered resources, and revision history.
- **Chart catalog** searches Artifact Hub, browses managed Helm repositories, and accepts packaged `.tgz` charts.
- **Repositories** lets administrators manage Helm repositories and OCI registries.
- **Automation** manages guarded chart upgrade policies.

Authorized users can install charts, upgrade releases, roll back to an earlier
revision, or remove a release. Removal requires typing the exact release name,
and concurrent changes to the same release are rejected.

## Find and inspect charts

Artifact Hub search supports both HTTPS Helm repositories and public OCI chart
references. Administrators can browse enabled managed Helm repositories. Editors
and administrators can upload a packaged chart up to 50 MiB; uploads remain in
bounded server memory for 30 minutes, are tied to the uploading user, and are not
persisted.

Before installation, the chart inspector shows its README, default Values,
templates, dependencies, version, and digest. KubeVision does not accept server
local paths or plain HTTP chart sources.

## Repositories and credentials

Administrators can add HTTPS Helm repositories and OCI registries, test
connections, and optionally provide Basic Auth credentials. Passwords are
encrypted at rest with KubeVision's configured encryption key and are never
returned by the API after saving.

Private network destinations are denied by default. An administrator may enable
private network access for an individual managed repository, which should only
be done for a trusted internal service. Authorization headers are removed when
an HTTPS redirect crosses origins.

Managed repositories and their credentials are administrator-only. Roles with
`package-releases:install` or `package-releases:upgrade` can still use public OCI
charts and their own temporary uploads.

## Install or upgrade

Select **Install chart** from the Releases view or install an inspected catalog
result. Select **Upgrade** from a release detail page. KubeVision accepts public
`oci://` references, direct HTTPS `.tgz` URLs, chart names paired with an HTTPS
Helm repository URL, managed repositories, and temporary uploads.

Enter overrides as a JSON object and select **Preview changes**. KubeVision runs
a server-side Helm dry run and shows rendered resources, hooks, risks, and a
redacted manifest. The one-time confirmation expires after ten minutes and is
bound to the user, cluster, release, chart source, Values, operation, and rendered
digest. Execution repeats the dry run and rejects changed output.

### Check for updates

KubeVision records the non-secret chart source after a successful install or
upgrade. On the release detail page, select **Check for updates** to compare the
installed chart with the newest stable semantic version in its indexed Helm
repository. When an update exists, KubeVision opens the normal guarded upgrade
flow with the chart and target version already filled in. An empty override
object reuses the release's existing Helm Values, including stable secrets,
without sending redacted values back to the cluster.

Releases installed before source tracking ask for their chart name and repository
URL once. KubeVision verifies the repository before associating it. OCI charts,
direct archives, and temporary uploads do not provide portable version indexes,
so they continue to use **Manual upgrade**. Update checks never skip preview or
critical-risk confirmation.

## Automatic upgrades

Administrators can bind an installed release to an enabled managed Helm
repository, a semantic-version constraint, retained Values, and an interval from
15 minutes to 7 days. Each check selects the newest matching version, merges the
new chart defaults with the current release Values and explicit policy overrides,
and uses the same preview, digest, and atomic upgrade path as a manual change.
Critical risks block the policy for manual review.

Automatic upgrades currently support indexed Helm repositories only. OCI charts
remain available for manual install and upgrade, but OCI automatic version
discovery is not enabled.

## Safety controls

Critical findings include CRDs, RBAC roles or bindings, admission webhooks,
Namespaces, privileged or root containers, privilege escalation, added Linux
capabilities, host ports, `hostPath`, and host namespace access. Non-administrators
cannot execute a preview with critical findings, and automation always blocks it.

Downloads reject private network targets unless explicitly allowed for a managed
repository, unsafe redirects, oversized indexes, and archives larger than 50 MiB.
Charts whose templates use Helm's `lookup` function are rejected because rendering
them with dashboard credentials could expose resources outside the user's effective
access.

Helm operations use KubeVision's cluster credentials. Keep install and upgrade
permissions narrowly assigned and review every preview. Values and Secret
manifests are redacted before display, but sensitive values should still be
managed through Kubernetes Secrets or an external secret store.
