# syntax=docker/dockerfile:1.6

FROM --platform=$BUILDPLATFORM golang:1.23 AS base
WORKDIR /workspace
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY config/ config/

# Test stage (runs BEFORE build)
FROM base AS test
# Run unit tests
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v ./...

# Build stage (cacheable - no changing build args)
FROM base AS builder
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -o manager cmd/main.go

# Metadata injection stage (only this layer rebuilds when args change)
FROM builder AS metadata
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -o manager-with-metadata \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    cmd/main.go

# Final minimal image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=metadata /workspace/manager-with-metadata ./manager
USER 65532:65532
ENTRYPOINT ["/manager"]
