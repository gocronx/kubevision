APP_NAME := kubevision
BUILD_DIR := bin
MAIN_PKG := ./cmd/kubevision

GO := go
GOFLAGS := -v
LDFLAGS := -s -w

.PHONY: dev build build-frontend build-all test lint clean tidy fmt docs docs-dev docs-deploy

## dev: Run with air hot-reload (requires: go install github.com/air-verse/air@latest)
dev:
	GIN_MODE=debug air -c .air.toml || GIN_MODE=debug $(GO) run $(MAIN_PKG)

## build-frontend: Build the React frontend
build-frontend:
	cd web && pnpm install && pnpm build

## build-all: Build frontend + backend into a single binary
build-all: build-frontend build

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)

## test: Run all tests with race detector
test:
	$(GO) test -race -count=1 ./...

## lint: Run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run ./...

## tidy: Clean up go.mod and go.sum
tidy:
	$(GO) mod tidy

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f kubevision.db

## docs: Build documentation site
docs:
	cd docs && pnpm install && pnpm build

## docs-dev: Start documentation dev server
docs-dev:
	cd docs && pnpm install && pnpm start

## docs-deploy: Build and deploy docs to Cloudflare Pages
docs-deploy: docs
	cd docs && wrangler pages deploy build --project-name kubevision-docs

## help: Show this help message
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
