# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Parallax is a Kubernetes operator for parallel batch processing that enables processing lists of items concurrently using various data sources. Built with Go 1.23 using the kubebuilder/controller-runtime framework.

## Key Commands

### Development
```bash
# Build the operator
make build

# Run locally (requires kubeconfig)
make run

# Generate manifests and code
make generate
make manifests
```

### Testing
```bash
# Run unit tests with coverage
make test

# Run E2E tests (creates isolated Kind cluster named 'parallax-e2e-test')
make test-e2e

# Quick CI checks (test + lint)
make ci-quick

# Full CI checks (matches GitHub Actions)
make ci-all

# Pre-commit script (same as ci-all)
./scripts/pre-commit.sh
```

### Linting and Validation
```bash
# Code formatting and linting
make fmt
make lint

# Security scanning (requires gosec)
make ci-security

# Validate Kubernetes manifests
make ci-validate
```

### Helm and Deployment
```bash
# Sync generated manifests to Helm charts
make sync-all

# Bump chart versions
make bump-chart-version BUMP=patch CHART=both

# Deploy to cluster
make deploy

# Install/uninstall CRDs
make install
make uninstall
```

### Docker
```bash
# Build and push images
make docker-build IMG=my-registry/parallax:tag
make docker-push IMG=my-registry/parallax:tag
```

## Architecture

### Core Components

**Three Custom Resource Definitions:**
- **ListSource** (`api/v1alpha1/listsource_types.go`): Fetches lists from various sources
  - Static lists, REST APIs, PostgreSQL databases
  - Configurable refresh intervals
  - Supports authentication for APIs and databases
- **ListJob** (`api/v1alpha1/listjob_types.go`): Creates parallel Kubernetes Jobs
  - References ListSource or uses static lists
  - Configurable parallelism and job templates
- **ListCronJob** (`api/v1alpha1/listcronjob_types.go`): Schedules ListJobs on cron
  - Standard cron scheduling with concurrency policies
  - Job history limits and cleanup

**Controllers** (`internal/controller/`):
- Each CRD has a dedicated controller using controller-runtime reconciliation
- Controllers handle the lifecycle and status updates of resources
- Main entry point: `cmd/main.go` sets up all three controllers

### Key Features

**Data Sources**: ListSource supports three types:
- `static`: Hardcoded list of items
- `api`: REST API with JSONPath extraction
- `postgresql`: Database queries with connection pooling

**Job Processing**: Each list item becomes an environment variable in a separate Job pod, enabling true parallel processing with configurable parallelism.

**Scheduling**: ListCronJob provides standard cron scheduling with built-in concurrency policies (Allow, Forbid, Replace).

## Project Structure

```
├── api/v1alpha1/           # CRD definitions and types
├── internal/controller/    # Controller implementations
├── cmd/main.go            # Main entry point
├── config/                # Kubernetes manifests and Kustomize configs
├── charts/                # Helm charts (auto-synced from config/)
├── test/e2e/             # End-to-end tests
└── scripts/              # Build and utility scripts
```

## Development Workflow

1. **Make changes** to CRD types in `api/v1alpha1/`
2. **Generate code**: `make generate` (creates deepcopy methods)
3. **Generate manifests**: `make manifests` (creates CRDs and RBAC)
4. **Sync to Helm**: `make sync-all` (updates charts automatically)
5. **Test**: `make ci-quick` for fast feedback or `make ci-all` for complete validation
6. **E2E testing**: `make test-e2e` (creates isolated Kind cluster)

## Testing Philosophy

- **Isolated E2E Testing**: Creates dedicated Kind clusters (`parallax-e2e-test`) that are automatically cleaned up
- **No Production Impact**: Unit tests run offline, E2E tests use isolated clusters
- **CI Matching**: Local `make ci-all` matches GitHub Actions exactly
- **Coverage Requirements**: Minimum 5% test coverage enforced

## Configuration

### Environment Variables
- `METRICS_BIND_ADDRESS`: Metrics server address (default: `:8080`)
- `LEADER_ELECT`: Enable leader election (default: `false`)
- `LOG_LEVEL`: Log level - debug, info, warn, error (default: `info`)

### Resource Requirements
- **Minimum**: CPU 100m, Memory 128Mi
- **Recommended**: CPU 500m, Memory 256Mi

## Helm Charts

Two charts in `charts/`:
- `parallax`: Full operator with optional CRDs
- `parallax-crds`: CRDs only (for separate lifecycle management)

Charts are automatically synchronized from `config/` using `make sync-all`. Never edit chart templates directly.

## Security

- Uses RBAC with minimal required permissions
- Supports secure metrics endpoint with TLS
- Regular security scanning with gosec
- Container images are signed with cosign
- Secrets handled securely for API and database authentication