---
title: 扩展访问
---

# 扩展访问

本页介绍需要明确配置信任边界的出站访问和代理功能。

## 镜像标签查询

镜像标签查询是匿名只读操作，默认允许 Docker Hub。通过
`registry.allowed_registries` 和 `registry.allowed_auth_hosts` 添加准确的
Registry 和令牌服务主机，也可以使用 `KUBEVISION_REGISTRY_ALLOWED` 和
`KUBEVISION_REGISTRY_AUTH_HOSTS`。

私有地址和明文 HTTP 默认关闭，除非管理员设置 `registry.allow_private` 或
`registry.allow_http`。这些开关会扩大服务端网络访问范围，只应在可信网络使用。
该功能不存储 Registry 凭据。

## Kubernetes HTTP 访问

后端可以通过 Kubernetes proxy 子资源向指定 Pod 或 Service 转发有界的 `GET`
和 `HEAD` 请求。路由只接受结构化的集群、命名空间、类型、名称、端口和路径，
而不是任意目标 URL。

处理器检查 `pods:get` 或 `services:get` 权限，移除请求凭据和逐跳请求头，不跟随
重定向，并限制响应体大小。它不是开放式 HTTP 代理。浏览器直接导航无法安全附加
Bearer Token，因此没有未认证的新 Tab 跳转。

## 相关控制

- [认证提供商](/docs/admin-guide/authentication-providers)
- [Helm 软件包 Release](/docs/user-guide/package-releases)
- [RBAC](/docs/admin-guide/rbac)
