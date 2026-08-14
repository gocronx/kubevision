---
title: 路线图
---

# 路线图

KubeVision 已经提供多集群资源管理、中英文界面、AI 助手、OAuth/OIDC、目录
登录、通行密钥、CRD 发现、受控 Helm 软件包管理，以及只读的 Prometheus、
Grafana 和 Argo CD 集成 API。

路线图仅记录尚未交付的工作，优先级可能随维护能力和社区反馈调整。

## 近期

- Helm 修订版本间的 Values 差异，以及更广泛的 OCI 更新发现
- YAML 编辑器中的 Kubernetes Schema 感知校验
- 只读 Pod 文件浏览和受控下载
- 扩大认证与破坏性操作的端到端测试覆盖
- 生成 API 契约，让路由文档与代码保持同步

## 平台集成

- 资源级 Argo CD 和 Flux 状态、链接以及显式授权的同步操作
- Grafana Dashboard 视图和资源关联
- NetworkPolicy 流量可视化和策略影响分析
- 基于 OpenCost 的命名空间与工作负载成本视图

## 可访问性与本地化

英文和简体中文翻译位于 `web/src/i18n/`。其他语言、可访问性审计和键盘操作
改进仍欢迎社区贡献。

## 提议功能

请先在 [GitHub Discussions](https://github.com/gocronx/kubevision/discussions)
说明实际运维问题和安全边界。确认后的工作可以创建 Issue，并按照
[贡献指南](/docs/development/contributing)实现。
