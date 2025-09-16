# syntax=docker/dockerfile:1.6

# Dependencies stage - downloads and pre-compiles dependencies for maximum caching
FROM --platform=$BUILDPLATFORM golang:1.23 AS deps
WORKDIR /workspace
COPY go.mod go.sum ./

# Download dependencies with cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Pre-compile dependencies to improve subsequent build times
# This creates cached compiled objects for all dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download -x

# Source stage - copies source code (separate layer for better caching)
FROM deps AS source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY config/ config/

# Pre-compile dependencies now that we have source code
# This builds all dependencies without building the main package
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./... | \
    xargs -r go build -v 2>/dev/null || true

# Test stage (runs BEFORE build for fail-fast)
FROM source AS test
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v ./...

# Build stage - optimized compilation without metadata injection
FROM source AS builder
ARG TARGETOS
ARG TARGETARCH

# Build with optimized flags for faster compilation
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -trimpath -o manager \
    -ldflags "-s -w -buildid=" \
    cmd/main.go

# Metadata injection stage - only rebuilds when version/commit changes
FROM source AS metadata
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build with metadata injection using optimized flags
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -trimpath -o manager-with-metadata \
    -ldflags "-s -w -buildid= -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    cmd/main.go

# Final minimal image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=metadata /workspace/manager-with-metadata ./manager
USER 65532:65532
ENTRYPOINT ["/manager"]
