---
sidebar_position: 1
title: Prometheus 集成
---

# Prometheus 集成

Prometheus 插件为已有 Prometheus 服务提供后端健康检查和有边界的即时 PromQL
查询。

打开**管理 > 插件**，填写 KubeVision 后端可访问的 `url`，启用插件并执行健康
检查。地址通常是内部服务，例如
`http://prometheus.monitoring.svc.cluster.local:9090`。

数据接口为：

```http
GET /api/v1/plugins/prometheus/query?query=<url-encoded-promql>
```

查询需要认证和插件权限。查询长度有限制，且会拒绝不受限的全指标名称扫描。当前
前端不提供自定义指标面板或 Alertmanager 告警视图，客户端可以直接使用该 API。
