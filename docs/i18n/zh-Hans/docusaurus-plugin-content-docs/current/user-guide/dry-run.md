---
sidebar_position: 8
title: Dry-Run 预览
---

# Dry-Run 预览

Dry-Run 预览让您在将变更应用到集群之前进行验证和预览。Kubernetes API Server 会执行完整的准入和校验流程——因此您看到的结果与实际执行完全一致——但不会真正写入任何内容。

## 使用方式

### 在 YAML 编辑器中使用

1. 打开任意资源，点击 **Edit YAML**（或点击 **Create** 创建新资源）
2. 在 Monaco 编辑器中进行修改
3. 点击 **Dry Run**，而非 **Save**
4. 并排差异视图随即打开：左侧为**当前配置**，右侧为**拟应用配置**
5. 确认差异无误后，点击 **Apply** 真正保存，或点击 **Cancel** 继续编辑

### 差异视图说明

| 面板 | 内容 |
|-------|---------|
| 左侧（当前） | 集群中现有的资源 YAML |
| 右侧（拟应用） | 经 API Server 规范化处理后的编辑结果（包含注入的默认值等） |

变更预览仅高亮显示有实际意义的内容；渲染前会移除 `managedFields` 和服务端
分配的元数据。

## API 参考

```http
POST /api/v1/clusters/:cluster/namespaces/:namespace/resources/:resource/:name/dry-run
Content-Type: application/json

{
  "manifest": { /* 完整资源对象 */ }
}
```

响应：

```json
{
  "current":  { /* 现有资源 */ },
  "proposed": { /* API Server 规范化后的结果 */ },
  "valid":    true,
  "warnings": []
}
```

如果 API Server 拒绝该变更，`valid` 将为 `false`，错误信息会直接在界面中展示。

## 准入 Webhook 校验

由于 dry run 使用 `dryRun: All` 参数通过真实的 API Server 执行，所有准入 Webhook（OPA/Gatekeeper、Kyverno 等）都会对您的变更进行检验。在任何内容写入之前，策略违规情况会被提前报告。

```yaml
# 示例：Kyverno 策略违规信息出现在 warnings 中
- "Resource 'api-server' violates policy 'require-resource-limits': container 'app' has no CPU limit"
```

:::info
Dry-run 支持 KubeVision 管理的所有 26 种以上资源类型，包括 CRD。
:::

## 使用场景

- **安全的生产环境变更** — 在提交前验证 Deployment 镜像版本更新不会被策略拒绝
- **模板开发** — 在首次应用前确认新的资源清单格式正确
- **学习用途** — 查看 API Server 默认注入了哪些字段（例如 Volume 上的 `defaultMode`）

## 相关文档

- [资源管理](/docs/user-guide/resource-crud) — 完整的 CRUD 工作流
- [kubectl 提示](/docs/user-guide/kubectl-hints) — 等效的 `kubectl apply --dry-run=server` 命令
