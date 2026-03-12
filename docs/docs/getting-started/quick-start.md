---
sidebar_position: 2
title: Quick Start
---

# Quick Start

This guide walks you through your first 5 minutes with KubeVision.

## 1. Login

Navigate to `http://localhost:8080` and login with:
- Username: `admin`
- Password: `admin123`

## 2. Add a Cluster

KubeVision auto-detects the cluster from your kubeconfig. For additional clusters:

1. Go to **Settings → Clusters**
2. Click **Add Cluster**
3. Provide a name and upload the kubeconfig file
4. Click **Save**

## 3. Explore Resources

The left sidebar shows all available resource types organized by category:

- **Workloads** — Deployments, StatefulSets, DaemonSets, Jobs, CronJobs, Pods
- **Networking** — Services, Ingresses, NetworkPolicies
- **Storage** — PersistentVolumes, PersistentVolumeClaims, StorageClasses
- **Configuration** — ConfigMaps, Secrets
- **Cluster** — Nodes, Namespaces, ServiceAccounts, Roles

## 4. Use Global Search

Press `Cmd+K` (or `Ctrl+K`) to open the global search. Type any resource name to find it across all clusters and namespaces.

## 5. Open a Terminal

1. Navigate to any Pod
2. Click the **Terminal** tab
3. Select a container (if multiple)
4. Start typing commands

## 6. Enable 2FA (Recommended)

1. Go to **Settings → Security**
2. Click **Enable 2FA**
3. Scan the QR code with Google Authenticator or Authy
4. Enter the verification code to confirm
5. Save your recovery codes in a safe place

## Next Steps

- [Configuration](/docs/getting-started/configuration) — Database, auth, and Kubernetes settings
- [RBAC](/docs/admin-guide/rbac) — Set up roles and permissions
- [Resource Management](/docs/user-guide/resource-crud) — CRUD operations
