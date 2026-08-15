---
sidebar_position: 1
title: Installation
---

# Installation

KubeVision can be deployed via Helm, Docker, or from source.

## Prerequisites

- Kubernetes cluster (v1.24+)
- `kubectl` configured with cluster access
- For development: Go 1.26.6+, Node.js 22+, pnpm 10

## Helm (Recommended)

```bash
helm install kubevision oci://ghcr.io/gocronx/charts/kubevision \
  -n kubevision \
  --create-namespace
```

The chart and application image are published by the GitHub Release workflow.
Helm installs the latest stable chart by default. In production, add
`--version <version>` to pin a reviewed release.

### Custom Values

```yaml
# values.yaml
replicaCount: 1

service:
  type: ClusterIP
  port: 8080

database:
  driver: sqlite     # sqlite | postgres
  dsn: kubevision.db

auth:
  jwtSecret: ""      # auto-generated if empty
```

```bash
helm install kubevision oci://ghcr.io/gocronx/charts/kubevision \
  -n kubevision \
  --create-namespace \
  -f values.yaml
```

The chart runs the container as numeric UID/GID `65532` with a read-only root
filesystem, dropped Linux capabilities, and the runtime-default seccomp
profile. Auto-generated application keys are written to
`/data/.kubevision-secrets.yaml`; the default SQLite PVC preserves this file.
When persistence is disabled, `/data` is an ephemeral volume, so PostgreSQL
installations should use `existingSecret` to preserve JWT and encryption keys
across Pod replacements. The ClusterRole lists the built-in resource types used
by the dashboard instead of using wildcard resources. Add narrowly scoped
`rbac.extraRules` entries when custom resources or optional controllers require
more Kubernetes API permissions.

## Accessing KubeVision

### Local or Temporary Access

Use port forwarding for local evaluation or troubleshooting:

```bash
kubectl port-forward --namespace kubevision svc/kubevision 8080:8080
```

Then open `http://localhost:8080`. The forwarding process must remain running;
this is not a production access method.

### Production Access with Ingress

For production, use an Ingress or Gateway with a stable DNS name and TLS. The
cluster must already have an Ingress controller and the referenced TLS Secret:

```yaml
# production-values.yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: kubevision.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kubevision-tls
      hosts:
        - kubevision.example.com
```

```bash
helm upgrade --install kubevision oci://ghcr.io/gocronx/charts/kubevision \
  -n kubevision \
  --create-namespace \
  -f production-values.yaml
```

Open `https://kubevision.example.com` after DNS points to the Ingress endpoint.
For passkeys/security keys, the public hostname must also match the configured
RP ID and origin.

### Production Access with LoadBalancer

If the cluster provides external load balancers, expose the Service directly:

```bash
helm upgrade --install kubevision oci://ghcr.io/gocronx/charts/kubevision \
  -n kubevision \
  --create-namespace \
  --set service.type=LoadBalancer

kubectl get svc --namespace kubevision kubevision
```

Terminate TLS at the load balancer or an upstream proxy. Do not expose the
dashboard over plain HTTP on an untrusted network.

## Docker

```bash
# Build
docker build -f deploy/Dockerfile -t kubevision:latest .

# Run with local kubeconfig
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  kubevision:latest
```

## From Source

```bash
git clone https://github.com/gocronx/kubevision.git
cd kubevision

# Backend
go mod tidy
make dev    # starts on :8080

# Frontend (new terminal)
cd web
pnpm install
pnpm dev    # starts on :5173, proxies /api → :8080
```

## Verify Installation

Open the URL selected above: `http://localhost:8080` for port forwarding, or
the configured HTTPS hostname for production.

Default credentials:
- **Username:** `admin`
- **Password:** `admin123`

:::warning
Change the default password immediately after first login. Production
deployments should also use PostgreSQL, persistent backups, explicit secrets,
and enforced 2FA for administrator accounts.
:::

## Next Steps

- [Quick Start](/docs/getting-started/quick-start) — Add your first cluster
- [Configuration](/docs/getting-started/configuration) — Customize settings
