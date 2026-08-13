# Build, preview, and publish KubeVision documentation to Cloudflare Pages.

PROJECT_NAME := kubevision-docs
OUTPUT_DIR := build
PACKAGE_MANAGER := pnpm
PAGES_BRANCH ?= master

.PHONY: help install dev build preview deploy deploy-only clean

help:
	@echo "Available commands:"
	@echo "  make install  - Install dependencies"
	@echo "  make dev      - Start the local documentation server"
	@echo "  make build    - Build the documentation site"
	@echo "  make preview  - Preview the production build locally"
	@echo "  make deploy   - Build and deploy to Cloudflare Pages"
	@echo "  make clean    - Remove build artifacts"

install:
	$(PACKAGE_MANAGER) install --frozen-lockfile

dev:
	$(PACKAGE_MANAGER) start

build:
	$(PACKAGE_MANAGER) build

preview: build
	$(PACKAGE_MANAGER) serve

deploy: build
	$(MAKE) deploy-only

deploy-only:
	npx wrangler pages deploy $(OUTPUT_DIR) --project-name $(PROJECT_NAME) --branch $(PAGES_BRANCH) --commit-dirty=true --no-bundle

clean:
	rm -rf $(OUTPUT_DIR)
