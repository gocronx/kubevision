---
sidebar_position: 2
title: 快速开始
---

# 快速开始

本指南将引导您完成 KubeVision 的前 5 分钟上手体验。

## 1. 登录

访问 `http://localhost:8080`，使用以下凭据登录：
- 用户名：`admin`
- 密码：`admin123`

## 2. 添加集群

KubeVision 会自动检测 kubeconfig 中的集群。如需添加更多集群：

1. 前往 **设置 → 集群**
2. 点击 **添加集群**
3. 填写名称并上传 kubeconfig 文件
4. 点击 **保存**

## 3. 浏览资源

左侧边栏按类别展示所有可用的资源类型：

- **工作负载** — Deployments、StatefulSets、DaemonSets、Jobs、CronJobs、Pods
- **网络** — Services、Ingresses、NetworkPolicies
- **存储** — PersistentVolumes、PersistentVolumeClaims、StorageClasses
- **配置** — ConfigMaps、Secrets
- **集群** — Nodes、Namespaces、ServiceAccounts、Roles

## 4. 使用全局搜索

按下 `Cmd+K`（或 `Ctrl+K`）打开全局搜索。输入任意资源名称，即可跨所有集群和命名空间进行查找。

## 5. 打开终端

1. 导航至任意 Pod
2. 点击 **终端** 标签页
3. 选择容器（如存在多个容器）
4. 开始输入命令

## 6. 启用双因素认证（推荐）

1. 前往 **设置 → 安全**
2. 点击 **启用双因素认证**
3. 使用 Google Authenticator 或 Authy 扫描 QR 码
4. 输入验证码进行确认
5. 将恢复码保存在安全的地方

## 下一步

- [配置](/docs/getting-started/configuration) — 数据库、认证及 Kubernetes 相关设置
- [RBAC](/docs/admin-guide/rbac) — 配置角色和权限
- [资源管理](/docs/user-guide/resource-crud) — 增删改查操作
