---
title: Helm 软件包 Release
---

# Helm 软件包 Release

**软件包**工作区显示所选集群中发现的 Helm Release，包括命名空间、Chart、
应用版本、状态、修订版本、Values 和修订历史。

打开 Release 可以查看已脱敏的 Values 和历史。具备权限的用户可以回滚到早期
修订或移除 Release。移除时必须输入准确名称，同一 Release 的并发修改会被拒绝。

KubeVision 当前仅管理已有 Release，不提供新 Chart 安装、Chart 包升级或仓库
注册。虽然 Helm Values 会在显示前脱敏，敏感信息仍应通过 Kubernetes Secret
或外部密钥系统管理。
