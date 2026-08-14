---
sidebar_position: 6
title: 终端录制
---

# 终端录制

通过 KubeVision 打开的每一个交互式终端会话——无论是连接到 Pod 容器还是节点 Shell——都会被自动录制。录制内容可在浏览器中回放，用于合规审查、培训或故障排查。

## 工作原理

```
Browser WebSocket  →  KubeVision backend  →  Kubernetes exec API
        ↑                    |
        |              tee 至录制器
        |                    ↓
   实时输出         terminal_sessions 表
                   （asciinema v2 格式）
```

所有按键输入和输出内容均在服务端通过将 WebSocket 流分流至录制器来捕获，客户端对录制过程无感知。

## 存储格式

会话存储在 `terminal_sessions` 数据库表中，采用 **asciinema v2** 格式——这是一种开放的行分隔 JSON 标准，被众多回放工具广泛支持。

| 列名 | 类型 | 说明 |
|--------|------|-------------|
| `id` | UUID | 会话标识符 |
| `user` | string | 打开会话的用户名 |
| `cluster` | string | 目标集群 |
| `namespace` | string | 目标命名空间 |
| `pod` | string | Pod 名称 |
| `container` | string | 容器名称 |
| `cast` | text | 完整的 asciinema v2 cast 数据 |
| `started_at` | timestamp | 会话开始时间 |
| `ended_at` | timestamp | 会话结束时间（会话仍活跃时为 null） |
| `expires_at` | timestamp | 自动删除日期 |

## 保留策略

默认保留周期为 **30 天**。超过 `expires_at` 的会话将由每晚运行的清理任务删除。

如需修改默认值：

```yaml
# config.yaml
terminal:
  retention_days: 30   # 设为 0 可永久保留录制内容
```

:::warning
将 `retention_days` 设为 `0` 将使数据库无限增长。禁用自动清理时请监控磁盘用量。
:::

## 网页播放器

点击会话列表中的任意会话行即可打开内置播放器：

- **播放 / 暂停** — 空格键或播放按钮
- **速度控制** — 0.5×、1×、2×、4× 回放速度
- **跳转** — 点击进度条上的任意位置
- **下载** — 导出 `.cast` 文件，可使用 `asciinema play` 在本地播放

播放器完全在浏览器端渲染，回放过程中无需服务端流式传输。

![终端会话回放与审计列表](/img/screenshots/terminal-recording.png)

_管理员可以在审计列表中定位终端录制，并直接在浏览器中回放。_

## 访问控制

| 角色 | 可查看内容 |
|------|-------------------|
| `admin` | 所有用户和集群的全部会话 |
| `ops` | 已分配集群内的所有会话 |
| `dev`、`readonly` | 仅限自己的会话 |

管理员可在 **Settings → Terminal Sessions** 中访问完整会话列表。其他用户在 **Profile → Terminal Sessions** 下查看自己的会话。

:::tip
在事故复盘时，使用会话列表筛选器（用户、集群、Pod、时间范围）可快速定位特定会话。列表中会显示会话时长，便于跳过时长极短的误操作会话。
:::

## 导出至外部工具

从播放器下载 `.cast` 文件后，可在任何地方回放：

```bash
# 安装 asciinema CLI
pip install asciinema

# 本地播放
asciinema play session-abc123.cast

# 上传至 asciinema.org（可选）
asciinema upload session-abc123.cast
```

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — 控制哪些用户可以打开终端会话
- [审计日志](/docs/admin-guide/audit-logging) — 会话开始/结束事件记录在审计日志中
