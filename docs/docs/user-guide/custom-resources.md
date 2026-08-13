---
title: Custom Resources
---

# Custom Resources

KubeVision discovers CustomResourceDefinitions (CRDs) for each connected
cluster and exposes their custom resources through the same list, detail, YAML,
create, update, and delete workflows used for built-in resources.

Open **CRDs** to browse discovered definitions. Use refresh when a CRD was
installed after KubeVision connected and has not yet appeared; periodic
discovery is controlled by `kubernetes.crd_discovery_interval`.

RBAC applies to the custom resource's plural resource name and scope. CRD
discovery does not bypass Kubernetes authorization, and deleting a CRD can
delete all custom resources of that type. Review Kubernetes finalizers and
conversion webhooks before destructive changes.
