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
# Install kubebuilder and its dependencies
RUN curl -L -o kubebuilder.tar.gz https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.14.1/kubebuilder_linux_amd64.tar.gz && \
    tar -xzf kubebuilder.tar.gz && \
    mv kubebuilder_linux_amd64 /usr/local/kubebuilder && \
    rm kubebuilder.tar.gz && \
    export PATH=$PATH:/usr/local/kubebuilder/bin

# Install PostgreSQL client
RUN apt-get update && \
    apt-get install -y postgresql-client && \
    rm -rf /var/lib/apt/lists/*

# Set environment variables for tests
ENV KUBEBUILDER_ASSETS=/usr/local/kubebuilder/bin
ENV PATH=$PATH:/usr/local/kubebuilder/bin

# Run tests with verbose output
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v -count=1 ./...

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
