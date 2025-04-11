# Build stage
FROM --platform=$BUILDPLATFORM docker.io/golang:1.20 AS builder

# Set build arguments
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=unknown
ARG COMMIT=unknown
ARG DATE=unknown

# Set build environment
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS:-linux}
ENV GOARCH=${TARGETARCH}

# Set working directory
WORKDIR /workspace

# Copy dependency files
COPY go.mod go.mod
COPY go.sum go.sum

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build the binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -a \
    -ldflags "-X 'main.version=${VERSION}' \
              -X 'main.commit=${COMMIT}' \
              -X 'main.date=${DATE}'" \
    -o manager cmd/main.go

# Test stage
FROM builder AS test
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test ./...

# Production stage
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot

# Copy the binary from builder
COPY --from=builder /workspace/manager /manager

# Set non-root user
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/manager"]
