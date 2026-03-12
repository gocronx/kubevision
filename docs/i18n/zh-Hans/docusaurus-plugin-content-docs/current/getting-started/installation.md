---
sidebar_position: 1
title: 安装
---

# 安装

KubeVision 支持通过 Helm、Docker 或从源码进行部署。

## 前置条件

- Kubernetes 集群（v1.24+）
- 已配置好集群访问权限的 `kubectl`
- 开发环境：Go 1.23+、Node.js 18+、pnpm

## Helm（推荐）

```bash
helm repo add kubevision https://kubevision.github.io/charts
helm repo update

helm install kubevision kubevision/kubevision \
  --namespace kubevision \
  --create-namespace
```

### 自定义配置值

```yaml
# values.yaml
replicaCount: 1
image:
  repository: kubevision/kubevision
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
helm install kubevision kubevision/kubevision -f values.yaml
```

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
git clone https://github.com/kubevision/kubevision.git
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

在浏览器中打开 `http://localhost:8080`。

默认登录凭据：
- **用户名：** `admin`
- **密码：** `admin123`

:::warning
首次登录后请立即修改默认密码。
:::

## 下一步

- [快速开始](/docs/getting-started/quick-start) — 添加您的第一个集群
- [配置](/docs/getting-started/configuration) — 自定义设置
