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
helm repo add kubevision https://kubevision.github.io/charts
helm repo update

helm install kubevision gocronx/kubevision \
  --namespace kubevision \
  --create-namespace
```

### Custom Values

```yaml
# values.yaml
replicaCount: 1
image:
  repository: gocronx/kubevision
  tag: latest

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
helm install kubevision gocronx/kubevision -f values.yaml
```

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
helm upgrade --install kubevision gocronx/kubevision \
  --namespace kubevision \
  --create-namespace \
  -f production-values.yaml
```

Open `https://kubevision.example.com` after DNS points to the Ingress endpoint.
For passkeys/security keys, the public hostname must also match the configured
RP ID and origin.

### Production Access with LoadBalancer

If the cluster provides external load balancers, expose the Service directly:

```bash
helm upgrade --install kubevision gocronx/kubevision \
  --namespace kubevision \
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
