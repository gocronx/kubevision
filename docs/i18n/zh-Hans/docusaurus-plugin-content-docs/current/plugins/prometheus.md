---
sidebar_position: 1
title: Prometheus 集成
---

# Prometheus 集成

KubeVision 包含一个可选的 Prometheus 插件，可将 CPU、内存和网络指标直接嵌入资源详情视图。该插件需要一个可从 KubeVision 后端访问到的已有 Prometheus 实例。

## 配置

### 1. 启用插件

在 `values.yaml`（Helm）或 `config.yaml`（源码部署）中添加以下配置块：

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus.monitoring.svc.cluster.local:9090"
    timeout: 10s
```

### 2. 重启后端

```bash
# Helm
helm upgrade kubevision kubevision/kubevision -f values.yaml

# From source
make dev
```

:::tip
KubeVision 从后端而非浏览器连接 Prometheus。该 endpoint 必须在集群内部可达，而非从你的本地机器访问。
:::

## 显示的指标

| 视图 | 指标 |
|------|---------|
| 节点详情 | CPU 使用率 %、内存使用率 %、网络入/出流量 bytes/s |
| Pod 详情 | 容器 CPU/内存用量与限制对比 |
| Deployment 列表 | 所有 Pod 的 CPU 和内存聚合值 |
| 命名空间概览 | 资源消耗量排名前 N 的对象 |

## 自定义查询

你可以定义额外的 PromQL 查询，这些查询将以附加面板的形式出现在资源详情页面上。

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    customQueries:
      - name: "Request Rate"
        query: 'rate(http_requests_total{pod=~"{{pod}}"}[5m])'
        unit: "req/s"
      - name: "Error Rate"
        query: 'rate(http_errors_total{pod=~"{{pod}}"}[5m])'
        unit: "err/s"
```

`{{pod}}`、`{{namespace}}` 和 `{{cluster}}` 模板变量在查询执行时会被替换为当前正在查看的资源名称。

## 告警可视化

配置 Prometheus Alertmanager 后，活跃告警会以内联方式显示在资源卡片上。触发中的告警显示为红色徽标，待处理告警显示为黄色。

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    alertmanager:
      endpoint: "http://alertmanager.monitoring.svc.cluster.local:9093"
```

:::warning
每次页面加载时都会拉取告警数据。在存在大量活跃告警的大型集群中，请设置 `alertmanager.maxAlerts` 以限制返回的告警数量。
:::

## 相关文档

- [Grafana 集成](/docs/plugins/grafana) — 在指标旁嵌入完整的 Grafana 仪表板
- [架构概览](/docs/architecture/overview) — 插件如何挂载到后端插件注册表
