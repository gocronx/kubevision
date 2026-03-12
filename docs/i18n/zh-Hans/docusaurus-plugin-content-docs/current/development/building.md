---
sidebar_position: 2
title: 构建
---

# 构建

本页涵盖在本地构建 KubeVision 所需的全部内容——从单个层次的构建到生成发布二进制文件。

## 前置条件

| 依赖项 | 最低版本 | 安装方式 |
|-------------|-----------------|---------|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/) |
| pnpm | 8+ | `npm install -g pnpm` |
| Docker | 24+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Make | 任意版本 | 系统包管理器 |

验证你的环境：

```bash
go version    # go version go1.23.x ...
node -v       # v18.x.x or higher
pnpm -v       # 8.x.x or higher
docker --version
```

## 后端构建

```bash
# 下载依赖
go mod tidy

# 编译到 ./bin/kubevision
make build

# 直接运行（通过 Air 实现热重载）
make dev
```

`make build` 目标会在 `./bin/kubevision` 生成一个静态链接的二进制文件。服务器默认监听 `:8080` 端口。

## 前端构建

```bash
cd web

# 安装依赖
pnpm install

# 生产构建 → 输出到 web/dist/
pnpm build

# 带 HMR 的开发服务器（将 /api 代理到 :8080）
pnpm dev
```

生产构建将静态资源输出到 `web/dist/`。在嵌入模式下，这些资源会被内嵌到 Go 二进制文件中（见下文）。

## 嵌入模式（单一二进制）

KubeVision 使用 `go:embed` 将编译后的前端资源直接打包到后端二进制文件中，生成一个独立的可执行文件。

```go
//go:embed web/dist
var staticFS embed.FS
```

构建完整的嵌入二进制文件：

```bash
# 1. 先构建前端
cd web && pnpm install && pnpm build && cd ..

# 2. 构建嵌入式 Go 二进制
make build-embed
```

生成的 `./bin/kubevision` 同时提供 API 和静态 UI，无需任何外部文件依赖。

## Docker 构建

```bash
# 使用多阶段 Dockerfile 构建镜像
docker build -f deploy/Dockerfile -t kubevision:latest .

# 使用本地 kubeconfig 运行
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  kubevision:latest
```

`deploy/Dockerfile` 采用两阶段构建：Go 构建阶段编译二进制文件（含嵌入的前端），最终运行时层使用精简的 `distroless` 镜像。

## Makefile 目标

| 目标 | 描述 |
|--------|-------------|
| `make build` | 将后端二进制编译到 `./bin/kubevision` |
| `make build-embed` | 先构建前端，再编译嵌入式二进制 |
| `make dev` | 启动带实时重载的后端（Air） |
| `make lint` | 对所有包运行 `golangci-lint` |
| `make test` | 运行所有 Go 测试 |
| `make test-cover` | 运行测试并生成 HTML 覆盖率报告 |
| `make clean` | 删除 `./bin` 及构建产物 |
| `make docker` | 构建标记为 `kubevision:latest` 的 Docker 镜像 |

## 交叉编译

Go 的交叉编译支持使得针对任意平台的构建都非常简单：

```bash
# Linux amd64（常见服务器目标）
GOOS=linux GOARCH=amd64 make build

# Linux arm64（Raspberry Pi、AWS Graviton）
GOOS=linux GOARCH=arm64 make build

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 make build

# Windows amd64
GOOS=windows GOARCH=amd64 make build
```

:::tip
为发布版本进行交叉编译时，请始终构建嵌入式二进制（`make build-embed`），以确保前端资源包含在交叉编译的输出中。
:::

## 相关文档

- [贡献指南](/docs/development/contributing) — 代码风格与 PR 工作流
- [测试](/docs/development/testing) — 构建完成后运行测试套件
