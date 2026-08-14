---
sidebar_position: 3
title: Pod 终端
---

# Pod 终端

KubeVision 通过 [xterm.js](https://xtermjs.org/) 为任意运行中的 Pod 提供全功能的浏览器内终端。客户端无需安装 `kubectl`。

## 打开终端

1. 进入 **Workloads → Pods**，点击一个 Pod
2. 选择 **Terminal** 标签
3. 如果 Pod 包含多个容器，从 **Container** 下拉菜单中选择目标容器
4. 终端连接成功后，您将直接进入 Shell 环境

:::tip
使用的 Shell 由容器镜像决定。KubeVision 会优先尝试 `bash`，失败后回退到 `sh`。
:::

![KubeVision 中已连接的 Pod 终端](/img/screenshots/pod-terminal.png)

_Pod 详情页中已连接的终端，支持容器选择和重新连接。_

## 功能特性

| 功能 | 说明 |
|---------|---------|
| **完整 TTY** | 支持 ANSI 颜色、光标移动和控制序列的标准伪终端 |
| **自动调整大小** | 浏览器窗口变化时终端尺寸自动适配 |
| **剪贴板** | 使用 `Ctrl+Shift+V`（Linux/Windows）或 `Cmd+V`（macOS）粘贴内容 |
| **滚动缓冲** | 支持 10,000 行的滚动缓冲区 |

## 键盘快捷键

| 快捷键 | 操作 |
|----------|--------|
| `Ctrl+C` | 发送 SIGINT（中断当前进程） |
| `Ctrl+D` | 发送 EOF（关闭 Shell） |
| `Ctrl+L` | 清屏 |
| `Ctrl+Shift+C` | 复制选中内容到剪贴板 |

## 会话录制

所有终端会话均以 **asciinema v2** 格式自动录制，并存储在服务端。

### 回放会话（仅限管理员）

1. 进入 **Admin → Terminal Sessions**
2. 通过用户、Pod 或时间范围查找会话
3. 点击 **Replay** — 会话将以原始速度在浏览器中回放
4. 使用进度条跳转到会话中的任意时间点

```json
// asciinema v2 头部示例
{"version": 2, "width": 220, "height": 50, "timestamp": 1741132800, "title": "default/my-pod/app"}
```

:::info
会话录制文件存储在 KubeVision 数据库中。保留周期可通过 `TERMINAL_RECORDING_RETENTION_DAYS` 配置（默认值：30 天）。
:::

## 安全说明

- 每个会话均通过用户的 JWT 进行身份验证
- 需要具备 RBAC `exec` 权限（参见 [RBAC](/docs/admin-guide/rbac)）
- 所有按键操作均通过 TLS WebSocket（`wss://`）传输
- 会话录制仅管理员角色用户可访问

## 相关文档

- [Pod 日志](/docs/user-guide/pod-logs) — 流式传输并搜索容器日志
- [收藏夹](/docs/user-guide/favorites) — 收藏常用的 Pod
