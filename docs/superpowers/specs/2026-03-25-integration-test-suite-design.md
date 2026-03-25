# Spec: Pristine Helm-Based Integration Test Suite

**Date**: 2026-03-25
**Status**: Draft
**Topic**: Refactoring the integration testing workflow to use Helm and Kind for a "pristine" contributor experience, removing `docker-compose`.

## 1. Overview
The current integration testing relies on a mix of `docker-compose` for external services (Postgres, Mock API) and `Kind` for the operator. This creates a fragmented environment that requires manual network bridging (`docker network connect`) and is difficult to replicate reliably. 

This spec defines a unified, Helm-native workflow where **all** dependencies are deployed into a single Kind cluster using Helm charts, orchestrated via `make` commands.

## 2. Goals
- **Unified Environment**: All test dependencies (Postgres, Mock API, Operator) run inside Kind.
- **Pristine DevEx**: A single command (`make test-integration`) handles the entire lifecycle (setup, execution, teardown).
- **Helm-First**: Use the actual Helm charts (`charts/parallax`) for deployment to verify the "real" installation path.
- **No Side-Effects**: Clean cluster creation and destruction for every test run (Option A).

## 3. Architecture & Orchestration

### 3.1 Orchestration Workflow (`make test-integration`)
The workflow will be managed by a shell script (`scripts/test-integration.sh`) triggered by the Makefile.

1.  **Build Phase**:
    - Build Operator image (`parallax:integration`).
    - Build Mock API server image (`parallax-test-api:integration`).
2.  **Setup Phase**:
    - `kind create cluster --name parallax-integration`.
    - `kind load docker-image` for both images.
3.  **Deployment Phase (Helm)**:
    - `helm install parallax-crds ./charts/parallax-crds`.
    - `helm install parallax ./charts/parallax --set image.tag=integration`.
    - `helm install test-infra ./charts/test-infra`. (New internal chart).
4.  **Execution Phase**:
    - `go test -v ./test/integration/...`.
5.  **Teardown Phase**:
    - `kind delete cluster --name parallax-integration` (via `trap` for reliability).

### 3.2 The `test-infra` Helm Chart
A new internal Helm chart located at `charts/test-infra/` will manage:
- **PostgreSQL**: Deployment + Service + ConfigMap (for `init.sql`).
- **Mock API**: Deployment + Service (using the locally built `api-server`).

### 3.3 Integration Test Suite (`test/integration/`)
A new Go test package focusing on end-to-end scenarios:
- **`integration_suite_test.go`**: Suite setup, wait for operator/infra readiness.
- **`postgres_test.go`**: Verifies PostgreSQL ListSource -> ListJob flow.
- **`api_test.go`**: Verifies API ListSource -> ListJob flow.
- **`listcronjob_test.go`**: Verifies scheduling and concurrency policies.

## 4. Implementation Details

### 4.1 Networking
By moving all services into the cluster, we use standard Kubernetes DNS (e.g., `http://postgres.default.svc.cluster.local`) instead of host-mapped ports or Docker network bridges.

### 4.2 Image Caching
To speed up local iteration, the `Makefile` will use standard Docker layer caching for image builds before loading them into Kind.

## 5. Success Criteria
1.  `make test-integration` passes on a clean machine with only `docker`, `kind`, `helm`, and `go` installed.
2.  All five example scenarios from the `examples/` directory are programmatically verified.
3.  `docker-compose` and associated network bridging scripts are removed from the repository.

## 6. Security Considerations
- Test secrets (DB passwords) are managed via Kubernetes Secrets within the Kind cluster.
- No sensitive data is committed; templates use placeholders or default "test" credentials.
