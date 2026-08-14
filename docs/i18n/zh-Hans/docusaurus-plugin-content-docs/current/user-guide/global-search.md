---
sidebar_position: 5
title: 全局搜索
---

# 全局搜索

全局搜索让您无需浏览侧边栏，即可在当前选中集群的各个 Namespace 和资源类型中
查找 Kubernetes 资源。

## 打开搜索对话框

| 平台 | 快捷键 |
|----------|---------|
| macOS | `Cmd+K` |
| Windows / Linux | `Ctrl+K` |

也可以点击顶部导航栏中的**搜索**图标打开搜索对话框。

![全局资源搜索对话框](/img/screenshots/global-search.png)

_全局搜索覆盖在当前工作区上方，并支持键盘优先的导航方式。_

## 工作原理

输入经过短暂防抖后显示结果。前端调用当前集群的
`/api/v1/clusters/:id/search` 接口，后端在该用户可访问的资源数据中搜索。结果按
资源类型分组，并在浏览器中短暂缓存。

```
输入 "nginx" → 匹配：
  Deployment  nginx-deployment      default      prod-cluster
  Pod         nginx-deployment-xyz  default      prod-cluster
  Service     nginx-svc             default      prod-cluster
  Ingress     nginx-ingress         kube-system  prod-cluster
```

## 匹配方式

搜索不区分大小写，并可在资源身份字段中匹配，不要求输入完整名称：

| 查询 | 匹配结果 |
|-------|---------|
| `nginx` | 可搜索字段中包含 `nginx` 的资源 |
| `configmap` | `ConfigMap` 类型资源 |
| `kube sys` | `kube-system` Namespace 下的资源 |

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

全局搜索限定在当前选中集群和当前用户的 RBAC 权限范围内。它覆盖后端支持的资源
类型，并在可用时包含已发现的自定义资源。搜索其他集群前请先切换集群。

:::tip
全局搜索是在 KubeVision 中导航的最快方式。资深用户几乎不需要使用侧边栏。
:::

## 相关文档

- [集群管理](/docs/user-guide/cluster-management) — 选择要搜索的集群
- [收藏夹](/docs/user-guide/favorites) — 固定最常访问的资源
