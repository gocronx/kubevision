---
sidebar_position: 3
title: Testing
---

# Testing

KubeVision maintains test coverage across both the Go backend and the React frontend. Interface-driven design is the core pattern that keeps tests fast and free of external dependencies.

## Backend Tests (Go)

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific package
go test ./internal/service/...

# Run a specific test by name
go test -run TestResourceService_List ./internal/service/...

# Equivalent via Makefile
make test
```

### Test Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View summary in terminal
go tool cover -func=coverage.out

# Open interactive HTML report in browser
go tool cover -html=coverage.out

# Makefile shortcut (generates + opens HTML)
make test-cover
```

### Interface Mocking

Every external dependency — Kubernetes API, database, cache — is accessed through a Go interface. Tests inject a mock implementation instead of a real dependency.

```go
// The real interface (internal/repo/k8s.go)
type K8sResourceRepo interface {
    List(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error)
    Get(ctx context.Context, cluster, ns, resource, name string) (*unstructured.Unstructured, error)
    Create(ctx context.Context, cluster, ns, resource string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
    Delete(ctx context.Context, cluster, ns, resource, name string) error
}

// Mock implementation used in tests
type mockK8sRepo struct {
    listFn func(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error)
}

func (m *mockK8sRepo) List(ctx context.Context, cluster, ns, resource string) ([]unstructured.Unstructured, bool, error) {
    return m.listFn(ctx, cluster, ns, resource)
}

// Test using the mock
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
Because every layer depends only on interfaces — never on concrete structs — you can mock any dependency without touching network or filesystem. Tests run in milliseconds.
:::

### Test Organization

```
internal/
  service/
    resource_service_test.go   # unit tests for service layer
    cluster_service_test.go
  handler/
    resource_handler_test.go   # handler tests using httptest
  repo/
    k8s_repo_integration_test.go  # integration tests (build tag: integration)
```

Integration tests are guarded by a build tag so they do not run in the default `go test ./...` sweep:

```bash
# Run integration tests (requires a real cluster in KUBECONFIG)
go test -tags=integration ./...
```

## Frontend Tests (React)

### Running Tests

```bash
cd web

# Run all tests (watch mode)
pnpm test

# Single run with coverage (CI mode)
pnpm test -- --coverage --watchAll=false
```

### Testing Approach

Frontend tests use **React Testing Library** to render components against a mock API layer. Tests interact with the DOM the way a user would — by finding elements through accessible roles and labels.

```tsx
// Example: ResourceTable renders rows from API data
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

### Mock Service Worker (MSW)

API calls are intercepted by [MSW](https://mswjs.io/) at the network layer. Handlers live in `web/src/mocks/` and are shared between tests and the `pnpm dev` development server.

```
web/src/mocks/
  handlers.ts   # MSW request handlers
  server.ts     # Node server for jest/vitest
  browser.ts    # Browser service worker for pnpm dev
```

## Integration Tests

End-to-end integration tests run against a real (or `kind`) cluster and verify the full request path from HTTP handler through to the Kubernetes API.

```bash
# Start a local kind cluster
kind create cluster --name kubevision-test

# Run integration tests
go test -tags=integration -v ./...

# Teardown
kind delete cluster --name kubevision-test
```

:::warning
Integration tests are not run in the standard CI pipeline on every PR. They are triggered manually or on release branches to avoid requiring a live cluster for every commit.
:::

## Why Interface-Driven Design Makes Testing Easy

| Without interfaces | With interfaces |
|--------------------|-----------------|
| Tests must connect to a real K8s cluster | Mock the repo — no cluster needed |
| Tests must connect to a real database | Mock the DB repo — no Postgres needed |
| Slow test suite (network I/O on every test) | Millisecond unit tests |
| Flaky tests due to cluster state | Deterministic, isolated tests |

Every constructor in the KubeVision backend accepts its dependencies as interface parameters, making injection of test doubles trivial with no additional mocking framework required.

## Related

- [Building](/docs/development/building) — Compile the project before testing
- [Contributing](/docs/development/contributing) — PR requirements including test coverage
