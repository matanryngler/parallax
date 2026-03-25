# Pristine Integration Test Suite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal**: Refactor the integration testing workflow to use Helm and Kind for a "pristine" contributor experience, removing `docker-compose`.

**Architecture**: A single `make test-integration` command orchestrates the build, setup of a Kind cluster, deployment of the Operator and a `test-infra` Helm chart, and execution of a Go-based integration suite.

**Tech Stack**: Go (Ginkgo/Gomega), Kubernetes (Kind), Helm, Docker.

---

### Task 1: Initialize `test-infra` Helm Chart
**Files**:
- Create: `charts/test-infra/Chart.yaml`
- Create: `charts/test-infra/values.yaml`
- Create: `charts/test-infra/templates/postgres.yaml`
- Create: `charts/test-infra/templates/mock-api.yaml`
- Create: `charts/test-infra/templates/secrets.yaml`

- [ ] **Step 1: Create Chart.yaml**
```yaml
apiVersion: v2
name: test-infra
description: External dependencies for Parallax integration testing
type: application
version: 0.1.0
appVersion: "1.0.0"
```

- [ ] **Step 2: Create values.yaml**
```yaml
postgres:
  image: postgres:16-alpine
  password: postgres
  database: testdb

mockApi:
  image:
    repository: parallax-test-api
    tag: integration
```

- [ ] **Step 3: Create Postgres templates**
Create a Deployment and Service named `postgres`. Mount the `init.sql` from a ConfigMap to `/docker-entrypoint-initdb.d`.

- [ ] **Step 4: Create Mock API templates**
Create a Deployment and Service named `mock-api` using the `parallax-test-api` image.

- [ ] **Step 5: Create test secrets**
Create secrets for database and API authentication as used in the examples.

- [ ] **Step 6: Commit**
`git add charts/test-infra && git commit -m "test: add test-infra helm chart"`

---

### Task 2: Orchestration Script & Makefile
**Files**:
- Create: `scripts/test-integration.sh`
- Modify: `Makefile`

- [ ] **Step 1: Create `scripts/test-integration.sh`**
Implement the cluster lifecycle: `kind create` -> `kind load` -> `helm install` -> `go test`. Use a `trap` to ensure cluster deletion.

- [ ] **Step 2: Add `test-integration` to `Makefile`**
Add the target and ensure it depends on `docker-build`.

- [ ] **Step 3: Commit**
`git add scripts/test-integration.sh Makefile && git commit -m "test: add integration test orchestration"`

---

### Task 3: Integration Test Suite Setup
**Files**:
- Modify: `test/utils/utils.go`
- Create: `test/integration/integration_suite_test.go`

- [ ] **Step 1: Add Port-Forwarding helper to `test/utils/utils.go`**
Implement a helper to port-forward services from the Kind cluster to the host.

- [ ] **Step 2: Create `test/integration/integration_suite_test.go`**
Initialize the Ginkgo suite. Add `BeforeSuite` to wait for resources and `AfterSuite` for cleanup.

- [ ] **Step 3: Commit**
`git add test/utils/utils.go test/integration && git commit -m "test: bootstrap integration test suite"`

---

### Task 4: Implement Example Scenarios (01-05)
**Files**:
- Create: `test/integration/static_list_test.go`
- Create: `test/integration/api_test.go`
- Create: `test/integration/postgres_test.go`
- Create: `test/integration/scheduled_batch_test.go`
- Create: `test/integration/production_patterns_test.go`

- [ ] **Step 1: Implement Scenario 01 (Static List)**
Verify `ListSource` -> `ConfigMap` -> `ListJob` -> `Job` flow.

- [ ] **Step 2: Implement Scenario 02 & 03 (API & Postgres)**
Verify external data source fetching and processing.

- [ ] **Step 3: Implement Scenario 04 & 05 (Cron & Patterns)**
Verify scheduling and security/resource configurations.

- [ ] **Step 4: Commit each test file individually**
`git add test/integration/<file> && git commit -m "test: verify scenario <X>"`

---

### Task 5: Final Cleanup
**Files**:
- Delete: `test/local`
- Modify: `README.md`, `Makefile`

- [ ] **Step 1: Remove `test/local`**
`git rm -rf test/local`

- [ ] **Step 2: Remove redundant mentions of docker-compose**
Update `README.md` and `Makefile` to reflect the new integration suite.

- [ ] **Step 3: Commit**
`git commit -m "cleanup: remove redundant local test infra"`
