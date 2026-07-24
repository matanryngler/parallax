# Parallax API Reference

This document provides comprehensive documentation for all Parallax Custom Resource Definitions (CRDs).

## Table of Contents

- [ListSource](#listsource)
  - [Spec Fields](#listsource-spec)
  - [Status Fields](#listsource-status)
  - [Examples](#listsource-examples)
- [ListJob](#listjob)
  - [Spec Fields](#listjob-spec)
  - [Status Fields](#listjob-status)
  - [Examples](#listjob-examples)
- [ListCronJob](#listcronjob)
  - [Spec Fields](#listcronjob-spec)
  - [Status Fields](#listcronjob-status)
  - [Examples](#listcronjob-examples)

---

## ListSource

ListSource is a Kubernetes custom resource that fetches and maintains a list of items from various data sources.

### API Group and Version

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
```

### ListSource Spec

The `spec` field defines the desired state of the ListSource.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Data source type. Valid values: `static`, `api`, `postgresql` |
| `intervalSeconds` | integer | No | 0 | Refresh interval in seconds. Set to 0 to fetch once. Minimum: 0 |
| `staticList` | []string | Conditional | - | Static list of items. Required when `type` is `static` |
| `api` | [APIConfig](#apiconfig) | Conditional | - | API configuration. Required when `type` is `api` |
| `postgres` | [PostgresConfig](#postgresconfig) | Conditional | - | PostgreSQL configuration. Required when `type` is `postgresql` |

#### APIConfig

Configuration for REST API data sources.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | Yes | - | REST API endpoint URL. Must be HTTP/HTTPS |
| `method` | string | No | GET | HTTP method. Valid values: `GET`, `POST`, `PUT`, `DELETE` |
| `headers` | map[string]string | No | - | Custom HTTP headers |
| `body` | string | No | - | Request body (for POST/PUT) |
| `auth` | [APIAuth](#apiauth) | No | - | Authentication configuration |
| `jsonPath` | string | Yes | - | JSONPath expression to extract items from response |
| `timeoutSeconds` | integer | No | 30 | HTTP request timeout. Range: 1-300 seconds |

#### APIAuth

Authentication configuration for API requests.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Auth type. Valid values: `basic`, `bearer` |
| `secretRef` | [SecretRef](#secretref) | Yes | - | Reference to Kubernetes Secret containing credentials |
| `usernameKey` | string | Conditional | - | Secret key for username (required for `basic` auth) |
| `passwordKey` | string | Conditional | - | Secret key for password/token (required for `basic` auth) |

#### PostgresConfig

Configuration for PostgreSQL data sources.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `connectionString` | string | Yes | - | PostgreSQL connection URL. Format: `postgresql://user:pass@host:port/db?options` |
| `query` | string | Yes | - | SQL SELECT query. Use `$1`, `$2`, etc. for parameters |
| `queryParams` | []string | No | - | Parameters for parameterized queries |
| `auth` | [PostgresAuth](#postgresauth) | No | - | Authentication configuration |
| `sslMode` | string | No | require | SSL/TLS mode. Valid values: `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full` |

#### PostgresAuth

Authentication configuration for PostgreSQL.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `secretRef` | [SecretRef](#secretref) | Yes | - | Reference to Kubernetes Secret containing password |
| `passwordKey` | string | Yes | - | Secret key for database password |

#### SecretRef

Reference to a Kubernetes Secret.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Name of the Secret |
| `namespace` | string | No | Same as ListSource | Namespace of the Secret |
| `key` | string | Yes | - | Key within the Secret's data field |

### ListSource Status

The `status` field represents the observed state of the ListSource.

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) | Latest observations. Condition types: `Ready`, `Error` |
| `lastUpdateTime` | [Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta) | Timestamp of last successful fetch |
| `itemCount` | integer | Number of items in the list |
| `error` | string | Error message from last fetch attempt |
| `state` | string | Operational state. Values: `Active`, `Pending`, `Error`, `Stale` |

### ListSource Examples

<details>
<summary>Static List</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: customer-list
spec:
  type: static
  staticList:
    - customer-001
    - customer-002
    - customer-003
  intervalSeconds: 0
```
</details>

<details>
<summary>REST API with Bearer Token</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: api-items
spec:
  type: api
  intervalSeconds: 300
  api:
    url: https://api.example.com/v1/items
    method: GET
    headers:
      Accept: application/json
    auth:
      type: bearer
      secretRef:
        name: api-credentials
        key: token
    jsonPath: "$.items[*].id"
    timeoutSeconds: 30
```
</details>

<details>
<summary>PostgreSQL with Parameterized Query</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: pending-orders
spec:
  type: postgresql
  intervalSeconds: 60
  postgres:
    connectionString: "postgresql://postgres@postgres:5432/mydb?sslmode=require"
    query: "SELECT id FROM orders WHERE status = $1 LIMIT 1000"
    queryParams:
      - "pending"
    auth:
      secretRef:
        name: db-credentials
        key: password
    sslMode: require
```
</details>

---

## ListJob

ListJob is a Kubernetes custom resource that processes items from a ListSource in parallel.

### API Group and Version

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListJob
```

### ListJob Spec

The `spec` field defines the desired state of the ListJob.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `listSourceRef` | string | Conditional | - | Name of ListSource. Mutually exclusive with `staticList` |
| `staticList` | []string | Conditional | - | Static list of items. Mutually exclusive with `listSourceRef` |
| `parallelism` | integer | Yes | - | Maximum concurrent jobs. Minimum: 1 |
| `template` | [JobTemplateSpec](#jobtemplatespec) | Yes | - | Job pod template |
| `ttlSecondsAfterFinished` | integer | No | - | Time to keep completed Jobs (seconds) |
| `deleteAfter` | [Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta) | No | - | Alternative to `ttlSecondsAfterFinished` using Duration format |
| `backoffLimit` | integer | No | 6 | Number of retries before marking Job as failed |
| `activeDeadlineSeconds` | integer | No | - | Maximum Job duration (seconds) |

#### JobTemplateSpec

Template for Job pods.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `image` | string | Yes | - | Container image (include registry and tag) |
| `command` | []string | No | - | Container command (overrides image entrypoint) |
| `envName` | string | Yes | ITEM | Environment variable name for the item value |
| `env` | [][EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core) | No | - | Additional environment variables |
| `envFrom` | [][EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core) | No | - | Environment variable sources (ConfigMap/Secret) |
| `resources` | [ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | No | - | CPU and memory requests/limits |
| `serviceAccountName` | string | No | default | Service account for pod |
| `imagePullPolicy` | string | No | IfNotPresent | Image pull policy |
| `imagePullSecrets` | [][LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core) | No | - | Secrets for pulling images |
| `tolerations` | [][Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core) | No | - | Node tolerations |
| `affinity` | [Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core) | No | - | Pod scheduling constraints |
| `labels` | map[string]string | No | - | Additional pod labels |
| `volumes` | [][Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core) | No | - | Pod volumes |
| `volumeMounts` | [][VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core) | No | - | Container volume mounts |
| `ports` | [][ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#containerport-v1-core) | No | - | Container ports |
| `initContainers` | [][Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core) | No | - | Init containers |
| `initImage` | string | No | busybox:1.36 | Image for internal parallax-init container |
| `initResources` | [ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core) | No | Minimal | Resources for parallax-init container |

### ListJob Status

The `status` field represents the observed state of the ListJob.

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) | Latest observations. Condition types: `Complete`, `Failed`, `Running` |
| `jobName` | string | Name of the created Kubernetes Job |

### ListJob Examples

<details>
<summary>Basic ListJob with ListSource</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: process-customers
spec:
  listSourceRef: customer-list
  parallelism: 3
  template:
    image: alpine:3.19
    command:
      - sh
      - -c
      - echo "Processing customer $CUSTOMER_ID"
    envName: CUSTOMER_ID
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "500m"
        memory: "256Mi"
  ttlSecondsAfterFinished: 3600
  backoffLimit: 3
```
</details>

<details>
<summary>ListJob with Static List</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: one-time-task
spec:
  staticList:
    - task-alpha
    - task-beta
    - task-gamma
  parallelism: 2
  template:
    image: busybox:1.36
    command: ["echo", "Processing", "$TASK"]
    envName: TASK
  ttlSecondsAfterFinished: 300
```
</details>

<details>
<summary>Production ListJob with Full Configuration</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: production-processor
spec:
  listSourceRef: api-items
  parallelism: 10
  template:
    image: myregistry/processor:v1.2.3
    imagePullPolicy: IfNotPresent
    imagePullSecrets:
      - name: registry-credentials
    command: ["/app/processor"]
    envName: ITEM_ID
    env:
      - name: LOG_LEVEL
        value: info
      - name: API_KEY
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: key
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
      limits:
        cpu: "2000m"
        memory: "1Gi"
    serviceAccountName: processor-sa
    labels:
      team: platform
      environment: production
    volumes:
      - name: cache
        emptyDir:
          sizeLimit: 1Gi
    volumeMounts:
      - name: cache
        mountPath: /cache
  ttlSecondsAfterFinished: 7200
  backoffLimit: 2
  activeDeadlineSeconds: 3600
```
</details>

---

## ListCronJob

ListCronJob is a Kubernetes custom resource that schedules ListJobs to run on a cron schedule.

### API Group and Version

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
```

### ListCronJob Spec

The `spec` field defines the desired state of the ListCronJob.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `schedule` | string | Yes | - | Cron schedule expression. Format: `minute hour day month dayOfWeek` |
| `listSourceRef` | string | Conditional | - | Name of ListSource. Mutually exclusive with `staticList` |
| `staticList` | []string | Conditional | - | Static list of items. Mutually exclusive with `listSourceRef` |
| `parallelism` | integer | Yes | - | Maximum concurrent jobs per run. Minimum: 1 |
| `template` | [JobTemplateSpec](#jobtemplatespec) | Yes | - | Job pod template |
| `concurrencyPolicy` | string | No | Forbid | How to handle concurrent runs. Valid values: `Allow`, `Forbid`, `Replace` |
| `startingDeadlineSeconds` | integer | No | - | Deadline for starting missed jobs (seconds) |
| `successfulJobsHistoryLimit` | integer | No | 3 | Number of successful ListJobs to retain |
| `failedJobsHistoryLimit` | integer | No | 1 | Number of failed ListJobs to retain |
| `suspend` | boolean | No | false | Suspend subsequent runs |
| `ttlSecondsAfterFinished` | integer | No | - | Time to keep completed Jobs (seconds) |
| `backoffLimit` | integer | No | 6 | Number of retries per Job |
| `activeDeadlineSeconds` | integer | No | - | Maximum Job duration (seconds) |

### ListCronJob Status

The `status` field represents the observed state of the ListCronJob.

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta) | Latest observations. Condition types: `Scheduled`, `Suspended` |
| `active` | [][ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectreference-v1-core) | Currently running ListJobs |
| `lastScheduleTime` | [Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta) | Timestamp of last scheduled run |
| `lastSkipEventUID` | string | Internal: tracks last skip event (for ConcurrencyPolicy) |

### ListCronJob Examples

<details>
<summary>Hourly Processing</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: hourly-sync
spec:
  schedule: "0 * * * *"  # Every hour
  listSourceRef: api-items
  parallelism: 5
  concurrencyPolicy: Forbid
  startingDeadlineSeconds: 300
  template:
    image: alpine:3.19
    command: ["sh", "-c", "echo Processing $ITEM"]
    envName: ITEM
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
  ttlSecondsAfterFinished: 3600
```
</details>

<details>
<summary>Daily Report Generation</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: daily-reports
spec:
  schedule: "0 0 * * *"  # Daily at midnight UTC
  staticList:
    - sales-report
    - inventory-report
    - customer-report
  parallelism: 2
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 7
  template:
    image: report-generator:v1.0.0
    command: ["/app/generate", "--report=$REPORT"]
    envName: REPORT
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
  ttlSecondsAfterFinished: 86400
  activeDeadlineSeconds: 7200
```
</details>

<details>
<summary>Weekly Maintenance</summary>

```yaml
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: weekly-maintenance
spec:
  schedule: "0 2 * * 0"  # Weekly on Sunday at 2 AM UTC
  listSourceRef: maintenance-targets
  parallelism: 1
  concurrencyPolicy: Forbid
  startingDeadlineSeconds: 3600
  successfulJobsHistoryLimit: 4
  failedJobsHistoryLimit: 2
  template:
    image: maintenance:latest
    command: ["/app/maintain", "--target=$TARGET"]
    envName: TARGET
    serviceAccountName: maintenance-sa
    resources:
      requests:
        cpu: "1000m"
        memory: "1Gi"
  ttlSecondsAfterFinished: 604800
  backoffLimit: 0
  activeDeadlineSeconds: 10800
```
</details>

---

## Common Patterns

### Using Secrets for Credentials

**Create Secret**:
```bash
kubectl create secret generic api-credentials \
  --from-literal=token="your-token-here"
```

**Reference in ListSource**:
```yaml
spec:
  api:
    auth:
      type: bearer
      secretRef:
        name: api-credentials
        key: token
```

**Reference in ListJob**:
```yaml
spec:
  template:
    env:
      - name: API_TOKEN
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: token
```

### Resource Management

Always specify resource requests and limits in production:

```yaml
resources:
  requests:
    cpu: "500m"      # Guaranteed
    memory: "256Mi"  # Guaranteed
  limits:
    cpu: "2000m"     # Maximum
    memory: "512Mi"  # Maximum (hard limit)
```

### Job Cleanup

Prevent cluster clutter with TTL:

```yaml
# Automatic cleanup after 1 hour
ttlSecondsAfterFinished: 3600

# For CronJobs, also set history limits
successfulJobsHistoryLimit: 3
failedJobsHistoryLimit: 1
```

### Error Handling

Configure retries and timeouts:

```yaml
# Retry failed jobs up to 3 times
backoffLimit: 3

# Kill jobs running longer than 30 minutes
activeDeadlineSeconds: 1800
```

---

## JSONPath Reference

JSONPath expressions are used to extract items from API responses.

### Basic Syntax

| Expression | Description | Example Input | Result |
|------------|-------------|---------------|--------|
| `$[*]` | Root array elements | `["a", "b", "c"]` | `["a", "b", "c"]` |
| `$.items[*]` | All items in `items` array | `{"items": ["x", "y"]}` | `["x", "y"]` |
| `$.items[*].id` | `id` field from each item | `{"items": [{"id": 1}, {"id": 2}]}` | `[1, 2]` |
| `$.data.users[*].name` | Nested field extraction | `{"data": {"users": [{"name": "alice"}]}}` | `["alice"]` |

### Testing JSONPath

Use [JSONPath Online Evaluator](https://jsonpath.com/) to test expressions against your API responses.

---

## Kubernetes API Links

- [Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#condition-v1-meta)
- [Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#time-v1-meta)
- [Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#duration-v1-meta)
- [EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envvar-v1-core)
- [EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#envfromsource-v1-core)
- [ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#resourcerequirements-v1-core)
- [LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#localobjectreference-v1-core)
- [Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#toleration-v1-core)
- [Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#affinity-v1-core)
- [Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volume-v1-core)
- [VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#volumemount-v1-core)
- [ContainerPort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#containerport-v1-core)
- [Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#container-v1-core)
- [ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectreference-v1-core)
