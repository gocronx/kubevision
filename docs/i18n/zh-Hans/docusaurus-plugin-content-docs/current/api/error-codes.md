---
sidebar_position: 5
title: 错误码
---

# 错误码

所有 API 响应均使用 HTTP 200。响应包装结构中的 `code` 字段决定了请求的结果。`0` 始终表示成功。每个非零错误码代表客户端应当处理的特定失败情况。

## 错误码参考

| 错误码 | 类别 | 含义 |
|------|----------|---------|
| `0` | 成功 | 请求成功 |
| **4000x — 参数错误** | | |
| `40001` | 参数 | 缺少必填参数 |
| `40002` | 参数 | 参数格式无效（类型错误、格式错误） |
| **4010x — 认证错误** | | |
| `40100` | 认证 | 未登录——未提供令牌 |
| `40101` | 认证 | 访问令牌已过期——请使用刷新端点 |
| `40102` | 认证 | 需要双因素认证——请提供 TOTP 码 |
| `40103` | 认证 | 双因素认证失败——TOTP 码不正确或已过期 |
| **4030x — 权限错误** | | |
| `40300` | 权限 | 用户无权执行此操作 |
| `40301` | 权限 | 用户无权访问所请求的集群 |
| `40302` | 权限 | 用户无权访问所请求的命名空间 |
| **4040x — 资源未找到错误** | | |
| `40400` | 未找到 | 集群中未找到 Kubernetes 资源 |
| `40401` | 未找到 | 未找到 KubeVision 用户 |
| `40402` | 未找到 | 未找到集群配置 |
| **4090x — 冲突错误** | | |
| `40900` | 冲突 | 资源已存在（创建冲突） |
| `40901` | 冲突 | 资源版本冲突——请重新获取后再重试更新 |
| **4220x — 验证错误** | | |
| `42200` | 验证 | YAML 语法无效或无法解析 |
| `42201` | 验证 | 预演请求被 Kubernetes API Server 拒绝 |
| **5000x / 5020x — 服务端错误** | | |
| `50000` | 服务端 | 服务器内部错误——请使用 `requestId` 检查后端日志 |
| `50200` | 服务端 | 无法访问所请求集群的 Kubernetes API Server |

## 在前端处理错误

推荐的做法是使用一个统一的 Axios 响应拦截器，将 `code` 值映射到面向用户的操作。以下示例取自 KubeVision 前端源码：

```typescript
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import { toast } from "@/components/ui/use-toast";

axios.interceptors.response.use(
  (response) => {
    const { code, message, data } = response.data;

    if (code === 0) return response;

    // 令牌过期——尝试静默刷新，然后重试
    if (code === 40101) {
      return useAuthStore.getState().refreshAndRetry(response.config);
    }

    // 需要双因素认证——跳转到双因素认证页面
    if (code === 40102) {
      window.location.href = "/auth/2fa";
      return Promise.reject(new Error(message));
    }

    // 未登录——跳转到登录页
    if (code === 40100) {
      useAuthStore.getState().logout();
      return Promise.reject(new Error(message));
    }

    // 其他所有错误——显示 toast 提示并拒绝
    toast({ variant: "destructive", title: `Error ${code}`, description: message });
    return Promise.reject(new Error(message));
  }
);
```

:::tip
报告 `50000` 错误时，请始终附上 `meta.requestId` 中的值。它能让后端团队立即定位到对应的日志条目。
:::

:::warning
切勿将原始的 `50000` 错误详情暴露给最终用户。服务端错误的 `message` 字段是有意设计为通用描述的。完整的错误上下文仅在后端结构化日志中可见。
:::

## 相关文档

- [API 概览](/docs/api/overview) — 统一响应格式与 `meta` 对象
- [认证](/docs/api/authentication) — `401xx` 认证流程详情
- [RBAC](/docs/admin-guide/rbac) — `403xx` 权限错误与角色定义的对应关系
