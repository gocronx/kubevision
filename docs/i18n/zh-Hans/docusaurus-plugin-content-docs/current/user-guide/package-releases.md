---
title: Helm 软件包 Release
---

# Helm 软件包 Release

**软件包**工作区提供四个视图，用于管理所选集群中的 Helm 软件：

- **发布**：查看已安装 Release、Values、渲染资源和修订历史。
- **Chart 目录**：搜索 Artifact Hub、浏览托管 Helm 仓库和上传 `.tgz` Chart。
- **仓库**：由管理员管理 Helm 仓库和 OCI Registry。
- **自动升级**：管理受安全控制的 Chart 升级策略。

具备权限的用户可以安装 Chart、升级、回滚到早期修订或移除 Release。移除时必须
输入准确名称，同一 Release 的并发修改会被拒绝。

## 查找与检查 Chart

Artifact Hub 搜索同时支持 HTTPS Helm 仓库和公开 OCI Chart 引用。管理员可以浏览
已启用的托管 Helm 仓库。Editor 和管理员可以上传最大 50 MiB 的打包 Chart；上传
内容只在服务端受限内存中保留 30 分钟，绑定上传用户，并且不会持久化。

安装前可查看 Chart 的 README、默认 Values、模板、依赖、版本和摘要。KubeVision
不接受服务端本地路径或明文 HTTP Chart 来源。

## 仓库与凭据

管理员可以添加 HTTPS Helm 仓库和 OCI Registry、测试连接，并按需提供 Basic Auth
凭据。密码使用 KubeVision 配置的加密密钥进行静态加密，保存后 API 不会再次返回。

默认拒绝私网目标。管理员可以为单个托管仓库开启私网访问，该选项只应对可信的
内部服务启用。HTTPS 重定向跨域时会移除 Authorization 请求头。

托管仓库及其凭据仅限管理员使用。具有 `package-releases:install` 或
`package-releases:upgrade` 权限的角色仍可使用公开 OCI Chart 和本人上传的临时 Chart。

## 安装或升级

可以在**发布**视图选择**安装 Chart**，也可以安装检查过的目录结果；在 Release
详情页选择**升级**。KubeVision 支持公开 `oci://` 引用、HTTPS `.tgz` 直链、Chart
名称与 HTTPS Helm 仓库 URL 的组合、托管仓库以及临时上传。

以 JSON 对象填写覆盖值后选择**预览变更**。KubeVision 会执行服务端 Helm Dry Run，
展示渲染资源、Hook、风险和已脱敏清单。一次性确认信息十分钟后过期，并绑定用户、
集群、Release、Chart 来源、Values、操作类型和渲染摘要。执行前会再次 Dry Run，
输出发生变化时会拒绝操作。

### 检查更新

安装或升级成功后，KubeVision 会记录不含凭据的 Chart 来源。在 Release 详情页选择
**检查更新**，即可将当前 Chart 与索引 Helm 仓库中的最新稳定语义版本进行比较。
发现更新后，KubeVision 会自动填入 Chart 和目标版本并进入原有的受控升级流程。
覆盖值保持为空对象时，Helm 会复用 Release 已有 Values，包括长期密钥，不会把页面
中的脱敏值重新写回集群。

启用来源记录之前安装的 Release 只需填写一次 Chart 名称和仓库地址，KubeVision
验证仓库后会保存关联。OCI Chart、归档直链和临时上传不提供可移植的版本索引，仍
使用**手动升级**。检查更新不会跳过预览或高风险确认。

## 自动升级

管理员可以将已安装 Release 绑定到启用的托管 Helm 仓库，并配置语义化版本约束、
明确覆盖的 Values 和 15 分钟至 7 天的检查间隔。每次检查会选择符合约束的最新版
本，将新 Chart 默认值、当前 Release Values 和策略覆盖值依次合并，并使用与手动
变更相同的预览、摘要校验和原子升级流程。发现严重风险时会阻塞策略，等待人工审查。

自动升级目前只支持带索引的 Helm 仓库。OCI Chart 仍支持手动安装和升级，但暂未
启用 OCI 自动版本发现。

## 安全控制

严重风险包括 CRD、RBAC 角色或绑定、Admission Webhook、Namespace、特权或 root
容器、权限提升、增加 Linux Capability、主机端口、`hostPath` 和主机命名空间访问。
非管理员不能执行包含严重风险的预览，自动升级则始终阻塞此类变更。

除非管理员为托管仓库明确开启，否则下载会拒绝私网目标，同时拒绝不安全重定向、
过大的仓库索引和超过 50 MiB 的归档。使用 Helm `lookup` 函数的 Chart 会被拒绝，
因为使用仪表盘凭据渲染时可能暴露用户有效权限之外的资源。

Helm 操作使用 KubeVision 持有的集群凭据，因此应严格控制安装和升级权限，并在
每次操作前审查预览。Values 和 Secret 清单会在展示前脱敏，但敏感信息仍应通过
Kubernetes Secret 或外部密钥系统管理。
