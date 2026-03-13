---
sidebar_position: 1
title: 贡献指南
---

# 贡献指南

KubeVision 欢迎各种形式的贡献——问题报告、文档改进、功能实现和代码审查。所有贡献均受 [Apache 2.0 许可证](https://www.apache.org/licenses/LICENSE-2.0)约束。

## Fork → 分支 → PR 工作流

```bash
# 1. 在 GitHub 上 Fork 仓库，然后克隆你的 Fork
git clone https://github.com/<your-username>/kubevision.git
cd kubevision

# 2. 添加上游远程仓库
git remote add upstream https://github.com/gocronx/kubevision.git

# 3. 从 main 创建功能分支
git checkout -b feat/my-feature

# 4. 修改代码，提交并推送
git push origin feat/my-feature

# 5. 在 GitHub 上向 main 分支发起 Pull Request
```

:::tip
保持分支专注于单一目标。较小的 PR 审查速度更快，如有必要也更容易回滚。
:::

## 代码风格

### Go（后端）

| 工具 | 用途 |
|------|-------|
| `gofmt` | 规范格式化——每次提交前运行 |
| `go vet` | 检测常见错误 |
| `golangci-lint` | 完整的 lint 套件（通过 `make lint` 运行） |

```bash
# 格式化并检查所有包
gofmt -w ./...
go vet ./...

# 完整 lint（需要安装 golangci-lint）
make lint
```

### 前端（React + TypeScript）

| 工具 | 配置文件 |
|------|-------------|
| ESLint | `web/.eslintrc.cjs` |
| Prettier | `web/.prettierrc` |

```bash
cd web
pnpm lint        # ESLint 检查
pnpm lint:fix    # 自动修复 ESLint 问题
pnpm format      # Prettier 格式化
```

:::warning
存在 lint 错误的 PR 不会被合并。请在推送前在本地运行 `make lint` 和 `pnpm lint`。
:::

## 提交信息规范

KubeVision 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <short summary>

[optional body]

[optional footer]
```

| 类型 | 使用场景 |
|------|-------------|
| `feat` | 新功能 |
| `fix` | 问题修复 |
| `docs` | 仅文档变更 |
| `refactor` | 既非修复也非功能的代码重构 |
| `test` | 添加或更新测试 |
| `chore` | 构建、工具链、依赖项更新 |

**示例：**

```
feat(rbac): add custom role permission editor
fix(websocket): reconnect on 1006 close code
docs(contributing): add commit message guide
```

## Issue 模板

提交报告时请使用对应的 GitHub Issue 模板：

- **Bug 报告** — 复现步骤、预期行为与实际行为、KubeVision 版本及 Kubernetes 版本。
- **功能请求** — 使用场景、建议方案及已考虑的替代方案。
- **安全漏洞** — **请勿**开公开 Issue。请发送邮件至 `security@kubevision.io`。

## 代码审查流程

1. 维护者将在 PR 提交后 **2 个工作日**内完成分配。
2. 对所有审查意见进行代码修改或给出书面说明。
3. 获得一位维护者批准后，由第二位维护者合并。
4. 默认采用 Squash Merge 策略，以保持 `main` 历史的整洁。

:::note
首次贡献者在 PR 被合并之前，必须通过 GitHub 自动化机器人签署 CLA（贡献者许可协议）。
:::

## 许可证

向 KubeVision 贡献代码即表示你同意，你的贡献将依据 **Apache License, Version 2.0** 授权。完整条款请参阅 [LICENSE](https://github.com/gocronx/kubevision/blob/main/LICENSE)。

## 相关文档

- [构建](/docs/development/building) — 搭建本地构建环境
- [测试](/docs/development/testing) — 运行和编写测试
