APP_NAME := kubevision
BUILD_DIR := bin
MAIN_PKG := ./cmd/kubevision
DEV_PORT ?= 18082

GO := go
GOFLAGS := -v
LDFLAGS := -s -w

.PHONY: dev dev-frontend build build-frontend build-all test lint clean tidy tidy-check mod-verify verify fmt fmt-check vet \
	frontend-install frontend-lint frontend-typecheck frontend-test frontend-build \
	e2e-install e2e-list e2e-test helm-validate release-chart docs docs-dev docs-deploy

## dev: Run with air hot-reload (requires: go install github.com/air-verse/air@latest)
dev:
	KUBEVISION_SERVER_PORT=$(DEV_PORT) air -c .air.toml || \
	KUBEVISION_SERVER_PORT=$(DEV_PORT) $(GO) run $(MAIN_PKG)

## dev-frontend: Run the frontend with a proxy to the development backend
dev-frontend:
	cd web && VITE_API_PROXY_TARGET=http://127.0.0.1:$(DEV_PORT) pnpm dev --host 127.0.0.1 --port 5178 --strictPort

## build-frontend: Build the React frontend
build-frontend:
	cd web && pnpm install --frozen-lockfile && pnpm build

## build-all: Build frontend + backend into a single binary
build-all: build-frontend build

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)

## test: Run all tests with race detector
test:
	$(GO) test -race -count=1 ./...

## tidy-check: Fail when go.mod or go.sum are not tidy
tidy-check:
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

## mod-verify: Verify module dependencies have expected content
mod-verify:
	$(GO) mod verify

## verify: Run local backend quality checks
verify: fmt-check tidy-check mod-verify vet test

## lint: Run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run ./...

## tidy: Clean up go.mod and go.sum
tidy:
	$(GO) mod tidy

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...

## fmt-check: Fail when Go source is not gofmt formatted
fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

## vet: Run Go static analysis
vet:
	$(GO) vet ./...

## frontend-install: Install frontend dependencies from the lockfile
frontend-install:
	cd web && pnpm install --frozen-lockfile

## frontend-lint: Lint frontend source
frontend-lint:
	cd web && pnpm lint

## frontend-typecheck: Type-check frontend source
frontend-typecheck:
	cd web && pnpm typecheck

## frontend-test: Run frontend unit tests once
frontend-test:
	cd web && pnpm test

## frontend-build: Create the frontend production bundle
frontend-build:
	cd web && pnpm build

## e2e-install: Install the Playwright Chromium browser
e2e-install:
	cd web && pnpm exec playwright install --with-deps chromium

## e2e-list: Discover browser smoke tests without starting services
e2e-list:
	cd web && pnpm test:e2e:list

## e2e-test: Run browser smoke tests (set KUBEVISION_E2E_BASE_URL for an existing app)
e2e-test:
	cd web && pnpm test:e2e

## helm-validate: Lint and render all production baseline fixtures
helm-validate:
	helm lint deploy/helm/kubevision
	@grep -q '^USER 65532:65532$$' deploy/Dockerfile
	@for values in deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		deploy/helm/kubevision/tests/fixtures/helm-values-existing-secret.yaml \
		deploy/helm/kubevision/tests/fixtures/helm-values-external-database.yaml \
		deploy/helm/kubevision/tests/fixtures/helm-values-persistent-sqlite.yaml; do \
		helm template kubevision deploy/helm/kubevision --values "$$values" >/dev/null || exit 1; \
	done
	@for values in deploy/helm/kubevision/tests/fixtures/helm-values-invalid-*.yaml; do \
		! helm template kubevision deploy/helm/kubevision --values "$$values" >/dev/null 2>&1 || exit 1; \
	done
	@! helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-existing-secret.yaml \
		--show-only templates/configmap.yaml | grep -q 'dsn:'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-existing-secret.yaml \
		--show-only templates/deployment.yaml | grep -q 'name: "kubevision-runtime-secrets"'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/deployment.yaml | grep -q 'path: /readyz'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/deployment.yaml | grep -q 'readOnlyRootFilesystem: true'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/deployment.yaml | grep -q 'allowPrivilegeEscalation: false'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/deployment.yaml | grep -q 'runAsUser: 65532'
	@helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/deployment.yaml | grep -q 'value: /data/.kubevision-secrets.yaml'
	@for resource in deployments/scale replicasets/scale statefulsets/scale; do \
		helm template kubevision deploy/helm/kubevision \
			--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
			--show-only templates/clusterrole.yaml | grep -q "$$resource" || exit 1; \
	done
	@! helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-default.yaml \
		--show-only templates/clusterrole.yaml | grep -q '\*'
	@test "$$(helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-external-database.yaml \
		--show-only templates/deployment.yaml | grep -c 'emptyDir: {}')" -eq 2
	@! helm template kubevision deploy/helm/kubevision \
		--values deploy/helm/kubevision/tests/fixtures/helm-values-external-database.yaml | grep -q 'kind: PersistentVolumeClaim'

## release-chart: Package a versioned chart (VERSION=x.y.z OUTPUT_DIR=dist)
release-chart:
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	mkdir -p "$(or $(OUTPUT_DIR),dist)"
	helm package deploy/helm/kubevision --version "$(VERSION)" --app-version "$(VERSION)" --destination "$(or $(OUTPUT_DIR),dist)"

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f kubevision.db

## docs: Build documentation site
docs:
	$(MAKE) -C docs build

## docs-dev: Start documentation dev server
docs-dev:
	$(MAKE) -C docs dev

## docs-deploy: Build and deploy docs to Cloudflare Pages
docs-deploy: docs
	$(MAKE) -C docs deploy-only

## help: Show this help message
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
