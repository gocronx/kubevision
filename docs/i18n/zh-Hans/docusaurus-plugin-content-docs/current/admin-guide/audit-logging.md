---
sidebar_position: 3
title: 审计日志
---

# 审计日志

KubeVision 中的每次写操作都会异步记录到结构化审计日志中。日志存储在数据库内，并在 90 天后自动清除。

## 记录范围

所有改变系统状态的操作均会被捕获，包括：

- 创建、更新或删除 Kubernetes 资源
- 添加或移除集群
- 用户登录、登出及认证失败尝试
- RBAC 分配变更
- 2FA 启用/禁用事件
- API Key 创建与吊销
- Webhook 配置变更
- 终端会话开始与结束

只读操作（list、get、watch）默认不记录，以控制日志量。

## 日志条目字段

| 字段 | 类型 | 说明 |
|-------|------|-------------|
| `id` | UUID | 日志条目唯一标识符 |
| `user` | string | 执行操作的用户名 |
| `action` | string | 动作类型：`create`、`update`、`delete`、`login` 等 |
| `resource` | string | 资源类型：`Deployment`、`Secret`、`User` 等 |
| `resource_name` | string | 受影响资源的名称 |
| `cluster` | string | 目标集群（系统操作时为空） |
| `namespace` | string | 目标命名空间（集群级别资源时为空） |
| `status_code` | int | 响应的 HTTP 状态码 |
| `detail` | JSON | 变更前后差异或请求载荷摘要 |
| `ip` | string | 客户端 IP 地址 |
| `timestamp` | RFC3339 | 操作发生时间 |

## 异步批量写入

审计条目以批量方式写入，避免对用户请求增加延迟：

```
Request → Handler → (返回响应) → channel ← 批量工作器 → DB（每 500ms 或每 100 条）
```

若应用在批次刷盘前崩溃，该批次中的条目可能丢失。这是为写入性能所做的可接受权衡。对于需要保证交付的合规环境，请配置 PostgreSQL 并在配置文件中通过 `audit.sync: true` 启用同步写入。

:::warning
设置 `audit.sync: true` 会在每次变更请求中增加一次同步数据库写入。在生产环境启用前，请先对数据库进行基准测试。
:::

## 保留策略

审计日志在 **90 天**后自动删除。后台任务每晚 02:00 UTC 运行，清除超出保留窗口的条目。

如需修改保留周期：

```yaml
# config.yaml
audit:
  retention_days: 90   # 设为 0 可禁用自动清理
```

## 查看审计日志

在管理面板中进入 **Settings → Audit Logs**。

### 筛选条件

| 筛选项 | 说明 |
|--------|-------------|
| **User** | 按特定用户名筛选 |
| **Action** | 按动作类型筛选（create、update、delete、login……） |
| **Resource** | 按资源类型筛选 |
| **Cluster** | 限定到特定集群 |
| **Time Range** | 开始/结束日期时间选择器 |
| **Status** | 成功（2xx）或失败（4xx/5xx） |

结果按每页 50 条分页显示，可通过工具栏导出为 CSV。

:::tip
组合使用 **User** 和 **Resource: Secret** 筛选条件，可审计特定团队成员对敏感资源的所有访问记录。
:::

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — 控制哪些用户可以执行哪些操作
- [双因素认证](/docs/admin-guide/two-factor-auth) — 认证事件会记录在审计日志中
- [API Keys](/docs/admin-guide/api-keys) — API Key 的使用记录归属于其所有者用户
