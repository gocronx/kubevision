# syntax=docker/dockerfile:1.7

# ==============================================================================
# Stage 1: Build frontend
# ==============================================================================
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

# Install pnpm
RUN corepack enable && corepack prepare pnpm@10 --activate

# Copy frontend dependency manifests first for layer caching.
COPY web/package.json web/pnpm-lock.yaml* ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# Copy the rest of the frontend source and build.
COPY web/ ./
RUN pnpm build

# ==============================================================================
# Stage 2: Build Go backend
# ==============================================================================
FROM golang:1.26-alpine AS backend-builder

ARG VERSION=dev
ARG REVISION=unknown

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency manifests first for layer caching.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code.
COPY . .

# Copy frontend build output into the embed directory.
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build the binary with CGO enabled (required for SQLite).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${REVISION}" -o /app/bin/kubevision ./cmd/kubevision

# ==============================================================================
# Stage 3: Final minimal image
# ==============================================================================
FROM alpine:3.24

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="KubeVision" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/gocronx/kubevision"

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN addgroup -S kubevision && adduser -S kubevision -G kubevision

WORKDIR /app

# Copy the compiled binary from the builder stage.
COPY --from=backend-builder /app/bin/kubevision .

# Switch to non-root user.
USER kubevision

EXPOSE 8080

ENTRYPOINT ["./kubevision"]
CMD ["--config", "/app/config.yaml"]
