---
sidebar_position: 2
title: Grafana 集成
---

# Grafana 集成

Grafana 插件连接已有 Grafana API 并列出 Dashboard。打开**管理 > 插件**，配置
`url` 和 API/服务账户 `token`，启用插件并执行健康检查。

当前数据接口为：

```http
GET /api/v1/plugins/grafana/dashboards
```

请求由 KubeVision 后端发出，因此 Grafana 地址必须能从 KubeVision 服务访问。
应使用只具备 Dashboard 列表/搜索权限的只读 Token。

当前前端只管理插件配置和健康状态，不嵌入 Grafana iframe、不保存资源到面板的
关联，也不会把用户 Bearer Token 传给 Grafana。
