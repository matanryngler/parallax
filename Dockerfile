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

# Test stage (runs BEFORE build)
FROM base AS test
# Install kubebuilder
RUN curl -L -o kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.14.1/kubebuilder_linux_amd64 && \
    chmod +x kubebuilder && \
    mv kubebuilder /usr/local/bin/

# Install PostgreSQL client
RUN apt-get update && \
    apt-get install -y postgresql-client && \
    rm -rf /var/lib/apt/lists/*

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v ./...

# Build stage
FROM base AS builder
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -v -o manager \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    cmd/main.go

# Final minimal image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
