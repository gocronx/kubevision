---
title: Helm 软件包 Release
---

# Helm 软件包 Release

**软件包**工作区提供五个视图，用于管理所选集群中的 Helm 软件：

- **发布**：查看已安装 Release、Values、渲染资源和修订历史。
- **Chart 目录**：搜索 Artifact Hub、浏览托管 Helm 仓库和上传 `.tgz` Chart。
- **仓库**：由管理员管理 Helm 仓库和 OCI Registry。
- **自动升级**：管理受安全控制的 Chart 升级策略。
- **快捷部署**：为 PostgreSQL、Redis 和 gocron 提供受维护的参数预设。

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

Chart 的默认标量 Values 可以在**基础配置**模式中按嵌套路径搜索和编辑。布尔值、
数值、文本和敏感字段使用对应的控件，敏感字段可在浏览器中生成加密安全的随机值。
数组、对象或完整 Values 文档可切换到 **JSON** 模式编辑；两种模式修改的是同一个
JSON 对象。

安装前选择**预览变更**。KubeVision 会执行服务端 Helm Dry Run，展示渲染资源数量、
Hook、风险和已脱敏清单。高级选项可以调整操作超时、等待资源就绪、失败时原子回滚
以及创建命名空间。一次性确认信息十分钟后过期，并绑定用户、集群、Release、Chart
来源、Values、操作类型和渲染摘要。修改任何内容后都必须重新预览；执行前还会再次
Dry Run，输出发生变化时会拒绝操作。

安装成功后，KubeVision 会打开 Release 详情页，展示 Helm Release 状态，并将支持的
渲染资源链接到 Kubernetes 资源详情页。Helm 状态为 `deployed` 只表示 Helm 已完成
该 Release，不代表所有 Pod 或应用依赖都已健康；对外提供服务前仍应检查工作负载和
事件。操作失败时会返回脱敏后的原因，并根据情况提示检查 Pod、事件、Hook、连接或
超时，不会在错误中返回凭据。

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

## 后台操作

确认后，安装、升级、回滚、移除和已批准的 AI 变更会作为持久化后台操作执行。
可打开**应用包 > 操作记录**查看进度、失败建议和请求 ID。刷新页面、重新登录或
网络中断不会丢失执行状态。只有可安全重复的操作才提供重试；AI 变更必须先重新
检查当前状态，再发起新的操作。

包括 Values 在内的操作输入会使用 KubeVision 加密密钥加密，不会由 API 返回，
也不会写入进度事件。已完成的操作记录保留 30 天。

## 快捷部署模板

PostgreSQL 和 Redis 模板使用公开的 Bitnami OCI Chart。凭据在浏览器中生成，并
保留在确认后的 Helm Values 中。gocron 模板使用公开的 gocron Chart，自动生成
应用密钥，并要求填写已有 PostgreSQL 服务的密码。模板只是可信 Chart 来源的便捷
参数预设，不会复制第三方 Chart。确认前仍需检查渲染后的资源。
