---
sidebar_position: 3
title: 测试
---

# 测试

KubeVision 在 Go 后端和 React 前端均维护测试覆盖率。接口驱动设计是保持测试快速、无外部依赖的核心模式。

## 后端测试（Go）

### 运行测试

```bash
# 运行所有测试
go test ./...

# 输出详细信息
go test -v ./...

# 运行特定包
go test ./internal/service/...

# 按名称运行特定测试
go test -run TestResourceService_List ./internal/service/...

# 通过 Makefile 运行
make test
```

### 测试覆盖率

```bash
# 生成覆盖率文件
go test -coverprofile=coverage.out ./...

# 在终端查看摘要
go tool cover -func=coverage.out

# 在浏览器中打开交互式 HTML 报告
go tool cover -html=coverage.out

# Makefile 快捷方式（生成并打开 HTML 报告）
make test-cover
```

### 接口 Mock

每个外部依赖——Kubernetes API、数据库、缓存——都通过 Go 接口访问。测试时注入 mock 实现，而非真实依赖。

```go
// 真实接口（internal/repo/k8s.go）
type K8sResourceRepo interface {
    List(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error)
    Get(ctx context.Context, cluster, ns, resource, name string) (*unstructured.Unstructured, error)
    Create(ctx context.Context, cluster, ns, resource string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
    Delete(ctx context.Context, cluster, ns, resource, name string) error
}

// 测试中使用的 mock 实现
type mockK8sRepo struct {
    listFn func(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error)
}

func (m *mockK8sRepo) List(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error) {
    return m.listFn(ctx, cluster, ns, resource)
}

// 使用 mock 的测试
func TestResourceService_List_ReturnsStaleWarning(t *testing.T) {
    svc := NewResourceService(&mockK8sRepo{
        listFn: func(_ context.Context, _, _, _ string) ([]unstructured.Unstructured, bool, error) {
            return []unstructured.Unstructured{}, true /* stale */, nil
        },
    })

    result, err := svc.ListResources(context.Background(), "prod", "default", "pods")
    require.NoError(t, err)
    assert.True(t, result.Stale)
}
```

:::tip
由于每一层只依赖接口而非具体结构体，你可以在不触及网络或文件系统的情况下 mock 任何依赖。测试可在毫秒级别完成。
:::

### 测试组织结构

```
internal/
  service/
    resource_service_test.go   # 服务层单元测试
    cluster_service_test.go
  handler/
    resource_handler_test.go   # 使用 httptest 的 Handler 测试
  repo/
    k8s_repo_integration_test.go  # 集成测试（构建标签：integration）
```

集成测试通过构建标签保护，不会在默认的 `go test ./...` 中运行：

```bash
# 运行集成测试（需要 KUBECONFIG 中有真实集群）
go test -tags=integration ./...
```

## 前端测试（React）

### 运行测试

```bash
cd web

# 运行所有测试（监听模式）
pnpm test

# 单次运行并生成覆盖率（CI 模式）
pnpm test -- --coverage --watchAll=false
```

### 测试方式

前端测试使用 **React Testing Library** 将组件渲染到 mock API 层上。测试以用户的方式与 DOM 交互——通过可访问性角色和标签查找元素。

```tsx
// 示例：ResourceTable 从 API 数据渲染行
import { render, screen, waitFor } from '@testing-library/react';
import { ResourceTable } from '@/components/ResourceTable';
import { server } from '@/mocks/server'; // msw mock server
import { rest } from 'msw';

test('displays pod rows returned by the API', async () => {
  server.use(
    rest.get('/api/v1/clusters/prod/resources/pods', (req, res, ctx) =>
      res(ctx.json({ items: [{ name: 'web-abc123', namespace: 'default' }] }))
    )
  );

  render(<ResourceTable cluster="prod" resource="pods" />);

  await waitFor(() => {
    expect(screen.getByText('web-abc123')).toBeInTheDocument();
  });
});
```

### Mock Service Worker（MSW）

API 调用在网络层被 [MSW](https://mswjs.io/) 拦截。Handler 位于 `web/src/mocks/` 目录下，在测试和 `pnpm dev` 开发服务器之间共享。

```
web/src/mocks/
  handlers.ts   # MSW 请求处理器
  server.ts     # 用于 jest/vitest 的 Node 服务器
  browser.ts    # 用于 pnpm dev 的浏览器 Service Worker
```

## 集成测试

端到端集成测试在真实（或 `kind`）集群上运行，验证从 HTTP Handler 到 Kubernetes API 的完整请求链路。

```bash
# 启动本地 kind 集群
kind create cluster --name kubevision-test

# 运行集成测试
go test -tags=integration -v ./...

# 清理
kind delete cluster --name kubevision-test
```

:::warning
集成测试不在每个 PR 的标准 CI 流水线中运行。它们由人工触发或在发布分支上触发，以避免每次提交都需要一个实时集群。
:::

## 为什么接口驱动设计让测试更简单

| 没有接口 | 使用接口 |
|--------------------|-----------------|
| 测试必须连接真实的 K8s 集群 | Mock repo——无需集群 |
| 测试必须连接真实的数据库 | Mock DB repo——无需 Postgres |
| 测试套件运行缓慢（每次测试都有网络 I/O） | 毫秒级单元测试 |
| 因集群状态导致的不稳定测试 | 确定性、隔离的测试 |

KubeVision 后端中的每个构造函数都以接口参数的形式接收其依赖项，使得注入测试替身变得非常简单，无需任何额外的 mock 框架。

## 相关文档

- [构建](/docs/development/building) — 在测试前编译项目
- [贡献指南](/docs/development/contributing) — 包含测试覆盖率要求的 PR 规范
