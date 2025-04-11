# Parallax Operator

A Kubernetes operator for managing batch processing of lists of items with support for multiple data sources and scheduling.

## Overview

Parallax is a Kubernetes operator that provides a flexible way to process lists of items in parallel. It supports multiple data sources (static lists, APIs, and PostgreSQL) and can be scheduled as either one-time jobs or recurring cron jobs.

## Features

- **Multiple Data Sources**:
  - Static lists
  - API endpoints with JSONPath support
  - PostgreSQL database queries
- **Flexible Processing**:
  - Process items in parallel with configurable parallelism
  - Each item processed in a separate pod
  - Configurable resource limits and requests
- **Scheduling Options**:
  - One-time jobs (`ListJob`)
  - Recurring cron jobs (`ListCronJob`)
- **Source Management**:
  - Automatic updates from data sources
  - Configurable update intervals
  - Support for authentication (Basic, Bearer token)
- **Monitoring and Observability**:
  - Status tracking
  - Error reporting
  - Event recording

## Installation

### Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured to communicate with your cluster
- cert-manager installed (for webhook certificates)

### Installation

The operator can be installed using kustomize:

```bash
# Install CRDs
kubectl apply -k config/crd

# Install RBAC
kubectl apply -k config/rbac

# Install manager
kubectl apply -k config/manager
```

Alternatively, you can use the Makefile:

```bash
# Install everything
make install

# Deploy the operator
make deploy
```

## Usage

### ListSource

A `ListSource` manages a list of items from various sources and makes them available to `ListJob` and `ListCronJob` resources.

#### Example: API Source

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: fruit-list
spec:
  type: api
  intervalSeconds: 60  # Update every minute
  api:
    url: "http://api.example.com/fruits.json"
    jsonPath: "$.fruits[*]"  # Extract items from the fruits array
    headers:
      Accept: "application/json"
    auth:
      type: bearer
      secretRef:
        name: api-token
        key: token
```

#### Example: PostgreSQL Source

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: user-list
spec:
  type: postgresql
  intervalSeconds: 300  # Update every 5 minutes
  postgres:
    connectionString: "host=postgres.default.svc.cluster.local port=5432 dbname=users"
    query: "SELECT username FROM active_users"
    auth:
      secretRef:
        name: postgres-credentials
        key: password
```

### ListJob

A `ListJob` processes a list of items once, either from a `ListSource` or a static list.

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: process-fruits
spec:
  listSourceRef: "fruit-list"  # Reference to a ListSource
  parallelism: 2  # Process 2 items at a time
  template:
    image: "busybox"
    command: ["echo", "Processing fruit: $ITEM"]
    envName: "ITEM"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "200m"
        memory: "256Mi"
  ttlSecondsAfterFinished: 3600  # Delete job after 1 hour
```

### ListCronJob

A `ListCronJob` processes a list of items on a schedule, similar to Kubernetes CronJob.

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: daily-fruit-processor
spec:
  listSourceRef: "fruit-list"
  parallelism: 2
  template:
    image: "busybox"
    command: ["echo", "Processing fruit: $ITEM"]
    envName: "ITEM"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "200m"
        memory: "256Mi"
  schedule: "0 0 * * *"  # Run daily at midnight
  concurrencyPolicy: "Forbid"  # Don't allow concurrent runs
  startingDeadlineSeconds: 300  # Allow 5 minutes of delay
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  suspend: false
```

## Development

### Prerequisites

- Go 1.20+
- Docker
- kubectl
- kind (for local testing)

### Building

```bash
# Build the operator image
make docker-build

# Push the operator image
make docker-push
```

### Testing

```bash
# Run unit tests
make test

# Run e2e tests
make test-e2e
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

# Contributors
Matan Ryngler