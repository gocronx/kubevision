---
sidebar_position: 5
title: 全局搜索
---

# 全局搜索

全局搜索让您无需浏览侧边栏，即可即时在所有集群、所有 Namespace 和所有资源类型中查找任意 Kubernetes 资源。

## 打开搜索对话框

| 平台 | 快捷键 |
|----------|---------|
| macOS | `Cmd+K` |
| Windows / Linux | `Ctrl+K` |

也可以点击顶部导航栏中的**搜索**图标打开搜索对话框。

## 工作原理

输入时结果即时出现。搜索在客户端对预先获取的索引（包含所有资源名称、类型、Namespace 和集群）执行，每次按键无需额外的 API 请求。

```
输入 "nginx" → 匹配：
  Deployment  nginx-deployment      default      prod-cluster
  Pod         nginx-deployment-xyz  default      prod-cluster
  Service     nginx-svc             default      prod-cluster
  Ingress     nginx-ingress         kube-system  staging-cluster
```

## 模糊匹配

搜索支持模糊匹配，无需输入精确名称：

| 查询 | 匹配结果 |
|-------|---------|
| `ngxdep` | `nginx-deployment` |
| `cfgmap` | `ConfigMap` 类型资源 |
| `kube sys` | `kube-system` Namespace 下的资源 |
| `prod nginx` | `prod-cluster` 上的 nginx 相关资源 |

## 键盘导航

对话框打开后，可完全通过键盘操作：

| 按键 | 操作 |
|-----|--------|
| `Arrow Up / Down` | 在结果列表中上下移动 |
| `Enter` | 跳转到选中的资源 |
| `Escape` | 关闭对话框 |

## 结果布局

每条结果行展示四项信息：

```
[类型图标]  资源名称                Namespace        集群
            nginx-deployment       default          prod-cluster
```

点击结果或按 `Enter` 可直接跳转到该资源的详情页。

## 搜索范围

全局搜索涵盖 KubeVision 已加载的所有资源类型——包括缓存类型（Pods、Deployments、Services 等）以及通过 CRD 自动发现的按需类型。索引随 Informer 缓存的更新自动刷新。

:::tip
全局搜索是在 KubeVision 中导航的最快方式。资深用户几乎不需要使用侧边栏。
:::

## 相关文档

- [集群管理](/docs/user-guide/cluster-management) — 添加或移除集群以纳入搜索范围
- [收藏夹](/docs/user-guide/favorites) — 固定最常访问的资源
