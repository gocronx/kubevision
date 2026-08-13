---
title: Batch Actions
---

# Batch Actions

Resource lists support selecting multiple items for batch deletion and eligible
workloads for batch restart. The operation stays within the selected cluster;
namespace and resource identity are carried per item.

KubeVision authorizes and executes each item independently and returns an item
result instead of treating the whole batch as one transaction. Review partial
failures and retry only the failed items after correcting permissions,
conflicts, or Kubernetes validation errors.

Batch deletion is irreversible. Stateful workloads may retain or remove data
depending on their storage and retention policy, so inspect PVC ownership and
finalizers before confirming.
