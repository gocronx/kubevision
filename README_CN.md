<div align="center">
  <img src="docs/assets/logo.svg" alt="KubeVision Logo" width="128" height="128">
  <h1>KubeVision</h1>
  <p>AI 原生 Kubernetes 仪表盘，通过自然语言理解、排障并安全操作集群。</p>
  <p>
    <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go"></a>
    <a href="https://react.dev"><img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React"></a>
    <a href="https://typescriptlang.org"><img src="https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License"></a>
  </p>
  <p><a href="README.md">English</a> · 简体中文</p>
</div>

<p align="center">
  <img src="assets/overview.png" alt="KubeVision 仪表盘" width="100%">
</p>

<table>
  <tr>
    <td width="50%" align="center"><b>Pod 终端与日志</b></td>
    <td width="50%" align="center"><b>资源拓扑</b></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-terminal.png" alt="Pod 终端" width="100%"></td>
    <td><img src="docs/assets/screenshot-topology.png" alt="资源拓扑" width="100%"></td>
  </tr>
  <tr>
    <td width="50%" align="center"><b>全局搜索</b></td>
    <td width="50%" align="center"><b>终端审计</b></td>
  </tr>
  <tr>
    <td><img src="docs/assets/screenshot-search.png" alt="全局搜索" width="100%"></td>
    <td><img src="docs/assets/screenshot-audit.png" alt="终端审计" width="100%"></td>
  </tr>
</table>

## AI 原生 Kubernetes 运维

KubeVision 以 AI 助手为核心，它使用实时集群上下文，而不只是回答通用的
Kubernetes 问题。你可以用自然语言让它检查资源、汇总 Pod 日志、解释异常工作
负载、查询 Prometheus，或准备资源变更。

AI 操作始终受控制台的安全机制约束：

- **上下文感知**：理解当前集群、命名空间、页面和资源
- **工具执行**：直接读取资源、日志、集群健康和 Prometheus 数据，而不是猜测
- **权限感知**：每次工具调用都按照当前用户的 RBAC 角色检查权限
- **人工确认**：创建、更新、Patch 和删除操作必须经过明确确认
- **完整审计**：记录已批准 AI 变更的用户、目标、工具和执行结果
- **模型灵活**：支持 OpenAI、OpenRouter、DeepSeek、通义千问等 OpenAI 兼容提供商

KubeVision 采用人机协同模式。AI 可以协助调查和执行，但不会绕过 RBAC，也不会
执行未经确认的集群变更。


## 📖 文档

完整文档请访问：**[kubevision-docs](https://kubevision-docs.pages.dev/)**

- 🚀 [快速开始](https://kubevision-docs.pages.dev/docs/getting-started/installation) - 安装、快速上手和配置
- 🤖 [AI 运维](https://kubevision-docs.pages.dev/docs/user-guide/ai-assistant) - 上下文感知调查和受控操作
- 🏛️ [架构设计](https://kubevision-docs.pages.dev/docs/architecture/overview) - 系统设计、数据流和组件交互
- 📘 [使用指南](https://kubevision-docs.pages.dev/docs/user-guide/cluster-management) - 功能介绍和使用说明
- 🔌 [API 参考](https://kubevision-docs.pages.dev/docs/api/overview) - REST 与 WebSocket API 文档

## ✨ 功能特性

- **AI 运维工作区**：通过自然语言进行上下文感知排障，并执行受 RBAC 控制的集群操作
- **实时同步**：基于 Informer 到 WebSocket 的推送架构，实现亚秒级更新和零轮询
- **多集群管理**：通过统一仪表盘高效管理所有集群
- **Pod 终端与日志**：集成终端模拟器，支持完整的会话录制与回放
- **Pod 实时指标**：通过 Metrics Server 展示当前 CPU、内存以及容器级请求值和限制值
- **资源拓扑**：通过可视化所有权关系图实时展示工作负载之间的关联
- **全局搜索**：使用 `Cmd+K` 在整个集群环境中进行模糊搜索
- **Deployment 操作**：支持扩缩容、重启、回滚，并可在应用变更前预览差异
- **Helm 工作区**：搜索 Artifact Hub、检查或上传 Chart、一键检查 Release 更新、管理加密的私有仓库，并执行受控安装、升级、回滚和自动更新策略
- **安全与访问控制**：5 种内置 RBAC 角色、TOTP 双因素认证和异步审计日志
- **高度可扩展**：内置 26 种以上资源，并可在运行时自动发现 CRD
- **现代化界面**：完整的国际化支持和原生暗色模式

## 🚀 快速开始

通过 Helm 可以最快地在集群中部署 KubeVision：

```bash
# 1. 安装 KubeVision
helm install kubevision deploy/helm/kubevision

# 2. 从本机临时访问仪表盘
kubectl port-forward svc/kubevision 8080:8080

# 3. 访问 Web 界面
# http://localhost:8080
```

> 默认登录账号：`admin` / `admin123`
>
> `kubectl port-forward` 仅适用于本地体验和故障排查。生产环境应通过启用 HTTPS
> 的 Ingress/Gateway 或 `LoadBalancer` Service 暴露 KubeVision。详情请参阅
> [安装指南](https://kubevision-docs.pages.dev/zh-Hans/docs/getting-started/installation#访问-kubevision)。
>
> 其他部署方式（Docker、二进制、本地开发）请参阅[安装指南](https://kubevision-docs.pages.dev/zh-Hans/docs/getting-started/installation)。

## 🔷 CLI 与 AI 助手

`kubevision` 二进制文件同时也是功能完整的管理 CLI。你可以直接管理账号，也可以在终端中与集群对话：

```bash
# 账号管理
kubevision reset-password --username admin
kubevision create-user --username dev --role editor --email dev@example.com

# 终端 AI 助手（使用设置页面中配置的 AI 提供商）
kubevision ai "default 命名空间中的 web pod 为什么崩溃？"   # 单次查询
kubevision ai                                                # 交互式 REPL
```

## 🤝 参与贡献

欢迎参与贡献！请先提交 Issue，讨论你希望进行的修改。

需要注意，Git Hook 会通过 [commitlint](https://github.com/conventional-changelog/commitlint)
校验提交信息，因此请使用交互式提交工具，而不是直接运行 `git commit`：

```bash
pnpm install      # 首次设置（在 web 目录中安装 Git Hook）
pnpm run commit   # 创建符合规范的提交
```

## 📄 开源协议

本项目采用 MIT License，详情请参阅 [LICENSE](LICENSE) 文件。
