---
sidebar_position: 1
title: 安装
---

# 安装

KubeVision 支持通过 Helm、Docker 或从源码进行部署。

## 前置条件

- Kubernetes 集群（v1.24+）
- 已配置好集群访问权限的 `kubectl`
- 开发环境：Go 1.26.6+、Node.js 22+、pnpm 10

## Helm（推荐）

```bash
helm repo add kubevision https://kubevision.github.io/charts
helm repo update

helm install kubevision gocronx/kubevision \
  --namespace kubevision \
  --create-namespace
```

### 自定义配置值

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

## 访问 KubeVision

### 本地或临时访问

本地体验或故障排查时可使用端口转发：

```bash
kubectl port-forward --namespace kubevision svc/kubevision 8080:8080
```

然后访问 `http://localhost:8080`。端口转发进程必须保持运行，因此不适合作为生产
环境的访问方式。

### 通过 Ingress 进行生产访问

生产环境应使用具有稳定域名和 TLS 的 Ingress 或 Gateway。集群中需要预先安装
Ingress Controller，并创建示例中引用的 TLS Secret：

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

将 DNS 指向 Ingress 入口后，访问 `https://kubevision.example.com`。使用通行密钥或
安全密钥时，公开域名还必须与配置的 RP ID 和来源保持一致。

### 通过 LoadBalancer 进行生产访问

如果集群支持外部负载均衡器，也可以直接暴露 Service：

```bash
helm upgrade --install kubevision gocronx/kubevision \
  --namespace kubevision \
  --create-namespace \
  --set service.type=LoadBalancer

kubectl get svc --namespace kubevision kubevision
```

应在负载均衡器或上游代理处终止 TLS，不要在不可信网络中通过纯 HTTP 暴露仪表盘。

## Docker

```bash
# 构建镜像
docker build -f deploy/Dockerfile -t kubevision:latest .

# 使用本地 kubeconfig 运行
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  kubevision:latest
```

## 从源码安装

```bash
git clone https://github.com/gocronx/kubevision.git
cd kubevision

# 后端
go mod tidy
make dev    # starts on :8080

# 前端（新终端窗口）
cd web
pnpm install
pnpm dev    # starts on :5173, proxies /api → :8080
```

## 验证安装

打开上面选择的访问地址：端口转发使用 `http://localhost:8080`，生产环境使用配置
好的 HTTPS 域名。

默认登录凭据：
- **用户名：** `admin`
- **密码：** `admin123`

:::warning
首次登录后请立即修改默认密码。生产部署还应使用 PostgreSQL、持久化备份、明确
配置的密钥，并对管理员账号强制启用双因素认证。
:::

## 下一步

- [快速开始](/docs/getting-started/quick-start) — 添加您的第一个集群
- [配置](/docs/getting-started/configuration) — 自定义设置
