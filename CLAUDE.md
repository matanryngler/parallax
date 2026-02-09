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
# Unit tests
make test                       # Run unit tests with coverage
make ci-quick                   # Fast: unit tests + linting
make ci-all                     # Complete: all CI checks

# Run specific test
go test -v ./internal/controller/ -run TestListSourceController

# Run specific package tests
go test -v ./api/v1alpha1/

# E2E tests
make test-e2e                   # Full E2E: setup cluster + test + cleanup
make test-e2e-quick            # Quick E2E: test against existing cluster
make test-e2e-functionality    # Full functionality tests with cluster setup
make test-e2e-golden           # Manifest validation tests

# Pre-commit validation
./scripts/pre-commit.sh         # Same as ci-all
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

**Data Sources**: ListSource supports three types (`api/v1alpha1/listsource_types.go`):
- `static`: Hardcoded list of items
- `api`: REST API with JSONPath extraction
  - Authentication: `basic` (username/password) or `bearer` (token)
  - Configurable timeout (1-300 seconds, default 30)
  - Custom headers support
- `postgresql`: Database queries with connection pooling
  - **Security**: Uses parameterized queries with `$1`, `$2` placeholders
  - SSL modes: disable, allow, prefer, require, verify-ca, verify-full (default: require)
  - Password stored in Kubernetes Secrets

**Job Processing**: Each list item becomes an environment variable in a separate Job pod, enabling true parallel processing with configurable parallelism.

**Scheduling**: ListCronJob provides standard cron scheduling with built-in concurrency policies (Allow, Forbid, Replace).

## Project Structure

```
├── api/v1alpha1/           # CRD definitions and types
├── internal/controller/    # Controller implementations
├── internal/metrics/       # Prometheus metrics definitions
├── cmd/main.go            # Main entry point
├── config/                # Kubernetes manifests and Kustomize configs
├── charts/                # Helm charts (auto-synced from config/)
├── test/e2e/             # End-to-end tests
└── scripts/              # Build and utility scripts
```

### Useful Scripts
- `scripts/pre-commit.sh`: Complete validation (same as `make ci-all`)
- `scripts/e2e-functionality.sh`: Full E2E test suite with cluster management
- `scripts/e2e-quick.sh`: Quick E2E tests against existing cluster
- `scripts/bump-chart-version.sh`: Bump Helm chart versions (called by `make bump-chart-version`)
- `scripts/validate-release.sh`: Validate release artifacts

## Development Workflow

1. **Make changes** to CRD types in `api/v1alpha1/`
2. **Generate code**: `make generate` (creates deepcopy methods)
3. **Generate manifests**: `make manifests` (creates CRDs and RBAC)
4. **Sync to Helm**: `make sync-all` (updates charts automatically)
5. **Test**: `make ci-quick` for fast feedback or `make ci-all` for complete validation
6. **E2E testing**: `make test-e2e` (creates isolated Kind cluster)

**CRITICAL**: After modifying CRD types, you MUST run `make generate && make manifests && make sync-all` or your build will fail. The `generate` target automatically runs `sync-all`, so `make generate` is usually sufficient.

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
- `NAMESPACE`: Watch specific namespace (default: all namespaces)

### Resource Requirements
- **Minimum**: CPU 100m, Memory 64Mi (1-10 resources)
- **Recommended**: CPU 500m, Memory 128Mi (10-50 resources, production default)
- **Large Scale**: CPU 500m, Memory 192Mi (50-100+ resources)

Memory scales sub-linearly: base ~24MB + 150-400KB per resource depending on complexity.

## Helm Charts

Two charts in `charts/`:
- `parallax`: Full operator with optional CRDs
- `parallax-crds`: CRDs only (for separate lifecycle management)

Charts are automatically synchronized from `config/` using `make sync-all`. Never edit chart templates directly.

## Metrics

The operator exposes Prometheus metrics on port 8080 (configurable via `METRICS_BIND_ADDRESS`):

**ListCronJob Metrics** (`internal/metrics/metrics.go`):
- `parallax_cronjob_cycles_started_total`: Total cycles started
- `parallax_cronjob_cycles_skipped_total`: Total cycles skipped (with reason label)
- `parallax_cronjob_cycle_duration_seconds`: Duration of last completed cycle
- `parallax_cronjob_active_pods`: Current number of active pods processing items

**Controller-Runtime Metrics**: Standard controller metrics automatically exposed (reconciliation rates, queue depths, etc.)

Access metrics:
```bash
# Port-forward to access locally
kubectl port-forward -n parallax-system deployment/parallax 8080:8080
curl http://localhost:8080/metrics
```

## Security

- Uses RBAC with minimal required permissions
- Supports secure metrics endpoint with TLS
- Regular security scanning with gosec
- Container images are signed with cosign
- Secrets handled securely for API and database authentication

**PostgreSQL Security**: Always use parameterized queries with `$1`, `$2` placeholders and the `queryParams` field to prevent SQL injection. Never concatenate user input into queries.

## Debugging

### Local Development
```bash
# Run operator locally with verbose logging
LOG_LEVEL=debug make run

# Connect to E2E test cluster for debugging
make test-e2e-setup
export KUBECONFIG=/tmp/parallax-e2e-test-kubeconfig
kubectl get all -A

# Clean up when done
make test-e2e-cleanup
```

### Controller Logs
```bash
# View operator logs in cluster
kubectl logs -n parallax-system deployment/parallax -f

# Check controller manager events
kubectl get events -n parallax-system --sort-by='.lastTimestamp'
```

### Performance Profiling
```bash
# Enable profiling (disabled by default)
# In Helm values: operator.profiling.enabled=true

# Port-forward to profiling endpoint
kubectl port-forward -n parallax-system deployment/parallax 6060:6060

# Analyze memory usage
go tool pprof http://localhost:6060/debug/pprof/heap

# Analyze CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile

# View goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Common Issues
- **Build fails after CRD changes**: Run `make generate && make manifests && make sync-all`
- **Tests fail with "cannot find module"**: Run `go mod tidy`
- **E2E tests hang**: Clean up stale Kind cluster with `make test-e2e-cleanup`
- **Helm validation fails**: Ensure you ran `make sync-all` after manifest changes

## Important Notes

- **No Attribution**: Never add Claude Code attribution or credit to any commits, code, or documentation in this project
- All code contributions should appear as if written entirely by the project maintainer