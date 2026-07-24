# Example 05: Production Patterns

This example demonstrates production-ready configurations for Parallax, including resource management, security hardening, monitoring, and reliability patterns.

## Overview

Production environments require additional considerations beyond basic functionality:

- **Resource Management** - Proper limits and requests
- **Security** - Least privilege, secrets management
- **Monitoring** - Metrics and observability
- **Reliability** - Retry logic, timeouts, and fault tolerance

## Files in This Example

- `resource-limits.yaml` - Proper resource configuration
- `security-context.yaml` - Security hardening patterns
- `monitoring.yaml` - Prometheus integration

## Resource Management

### Why Resource Limits Matter

Without proper resource limits:
- Jobs can consume all cluster resources
- Other workloads may be starved
- Costs can spiral out of control
- Cluster instability may occur

### Best Practices

**File**: `resource-limits.yaml`

```yaml
resources:
  requests:
    cpu: "500m"      # Guaranteed CPU
    memory: "256Mi"  # Guaranteed memory
  limits:
    cpu: "2000m"     # Maximum CPU (can burst to 2 cores)
    memory: "512Mi"  # Maximum memory (hard limit)
```

#### Sizing Guidelines

| Job Type | CPU Request | Memory Request | CPU Limit | Memory Limit |
|----------|-------------|----------------|-----------|--------------|
| Light (logging, simple transforms) | 100m | 128Mi | 500m | 256Mi |
| Medium (API calls, data processing) | 500m | 256Mi | 2000m | 512Mi |
| Heavy (ML, image processing) | 2000m | 1Gi | 4000m | 2Gi |

#### How to Size Your Jobs

1. **Start conservative**:
   ```yaml
   requests: {cpu: "100m", memory: "128Mi"}
   limits: {cpu: "500m", memory: "256Mi"}
   ```

2. **Monitor actual usage**:
   ```bash
   kubectl top pods -l job-name
   ```

3. **Adjust based on metrics**:
   - If pods are OOMKilled: Increase memory limit
   - If pods are throttled: Increase CPU limit
   - If resources are underutilized: Decrease requests

### Resource Quotas

Prevent runaway resource consumption:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: parallax-quota
  namespace: production
spec:
  hard:
    requests.cpu: "20"        # Max 20 cores requested
    requests.memory: "40Gi"   # Max 40GB memory requested
    limits.cpu: "40"          # Max 40 cores limit
    limits.memory: "80Gi"     # Max 80GB memory limit
    count/jobs.batch: "100"   # Max 100 jobs
```

### Pod Priority

Ensure critical jobs run first:

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority-jobs
value: 1000
globalDefault: false
description: "High priority for critical batch jobs"

---
# In your ListJob:
spec:
  jobTemplate:
    spec:
      template:
        spec:
          priorityClassName: high-priority-jobs
```

## Security Hardening

### Security Context

**File**: `security-context.yaml`

#### Pod-Level Security

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          # Run as non-root user
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            runAsGroup: 3000
            fsGroup: 2000
            seccompProfile:
              type: RuntimeDefault
```

#### Container-Level Security

```yaml
containers:
  - name: processor
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
          - ALL
```

### Secrets Management

**Never hardcode credentials**:

```yaml
# Create secret
kubectl create secret generic api-credentials \
  --from-literal=token="your-secure-token" \
  --from-literal=api-key="your-api-key"

# Reference in job
env:
  - name: API_TOKEN
    valueFrom:
      secretKeyRef:
        name: api-credentials
        key: token
```

### Network Policies

Restrict network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: parallax-jobs
spec:
  podSelector:
    matchLabels:
      app: parallax-job
  policyTypes:
    - Egress
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow specific API endpoints only
    - to:
        - podSelector:
            matchLabels:
              app: internal-api
      ports:
        - protocol: TCP
          port: 8080
```

### RBAC

Minimize operator permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: parallax-job-runner
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["create", "get", "list", "watch"]
  # Don't grant delete, update unless necessary
```

## Monitoring and Observability

### Prometheus Metrics

**File**: `monitoring.yaml`

The Parallax operator exposes metrics on `:8080/metrics`:

```
# Key metrics to monitor
parallax_listsource_items_total{name="api-source",type="api"}
parallax_listjob_active{name="processor"}
parallax_listjob_succeeded{name="processor"}
parallax_listjob_failed{name="processor"}
parallax_listcronjob_runs_total{name="report-gen"}
```

### ServiceMonitor

For Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: parallax-operator
  namespace: parallax-system
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
```

### Alerting Rules

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: parallax-alerts
spec:
  groups:
    - name: parallax
      interval: 30s
      rules:
        - alert: ParallaxJobsFailing
          expr: rate(parallax_listjob_failed[5m]) > 0.1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High job failure rate"
            description: "{{ $value }} jobs failing per second"

        - alert: ParallaxNoRecentRuns
          expr: time() - parallax_listcronjob_last_schedule_time > 3600
          for: 15m
          labels:
            severity: critical
          annotations:
            summary: "CronJob hasn't run recently"
```

### Logging

Structured logging for better observability:

```yaml
containers:
  - name: processor
    command:
      - sh
      - -c
      - |
        log_json() {
          echo "{\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"level\":\"$1\",\"message\":\"$2\",\"item\":\"$ITEM\"}"
        }

        log_json "INFO" "Starting processing"
        # ... processing logic ...
        log_json "INFO" "Processing complete"
```

### Tracing

Add trace IDs for request correlation:

```yaml
env:
  - name: TRACE_ID
    valueFrom:
      fieldRef:
        fieldPath: metadata.uid
  - name: JOB_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.labels['job-name']
```

## Reliability Patterns

### Retry Logic

```yaml
spec:
  jobTemplate:
    spec:
      # Retry failed jobs
      backoffLimit: 3

      # Exponential backoff
      # First retry: immediate
      # Second retry: 10s delay
      # Third retry: 20s delay
      # Fourth retry: 40s delay (then give up)
      backoffLimitPerIndex: 1  # For indexed jobs
```

### Timeouts

Prevent jobs from running indefinitely:

```yaml
spec:
  jobTemplate:
    spec:
      # Kill the entire job after 1 hour
      activeDeadlineSeconds: 3600

      template:
        spec:
          containers:
            - name: processor
              # Container-level timeout
              command:
                - timeout
                - 300s  # 5 minutes
                - sh
                - -c
                - "your-command"
```

### Health Checks

```yaml
containers:
  - name: processor
    livenessProbe:
      exec:
        command:
          - sh
          - -c
          - "pgrep -f my-process"
      initialDelaySeconds: 30
      periodSeconds: 10
      timeoutSeconds: 5
      failureThreshold: 3
```

### Graceful Shutdown

Handle termination signals:

```yaml
containers:
  - name: processor
    command:
      - sh
      - -c
      - |
        # Trap SIGTERM for graceful shutdown
        trap 'echo "Received SIGTERM, cleaning up..."; cleanup; exit 0' TERM

        cleanup() {
          # Save state, close connections, etc.
          echo "Saving state..."
          # ... cleanup logic ...
        }

        # Main processing
        process_item "$ITEM"

        # Wait for signals
        wait

    # Give pod 30 seconds to gracefully shutdown
    terminationGracePeriodSeconds: 30
```

### Idempotency

Make jobs safe to retry:

```yaml
containers:
  - name: processor
    command:
      - sh
      - -c
      - |
        # Check if already processed
        if check_already_processed "$ITEM"; then
          echo "Item $ITEM already processed, skipping"
          exit 0
        fi

        # Process item
        process_item "$ITEM"

        # Mark as processed
        mark_as_processed "$ITEM"
```

## Cost Optimization

### Spot/Preemptible Instances

Use cheaper spot instances for fault-tolerant workloads:

```yaml
spec:
  jobTemplate:
    spec:
      template:
        spec:
          nodeSelector:
            # GKE
            cloud.google.com/gke-preemptible: "true"
            # EKS
            eks.amazonaws.com/capacityType: SPOT
            # AKS
            kubernetes.azure.com/scalesetpriority: spot

          tolerations:
            - key: cloud.google.com/gke-preemptible
              operator: Equal
              value: "true"
              effect: NoSchedule
```

### Auto-scaling

Scale based on queue depth or schedule:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: parallax-jobs
spec:
  scaleTargetRef:
    apiVersion: batchops.example.com/v1alpha1
    kind: ListJob
    name: processor
  minReplicas: 1
  maxReplicas: 10
  metrics:
    - type: External
      external:
        metric:
          name: queue_depth
        target:
          type: AverageValue
          averageValue: "30"
```

### Job Cleanup

Automatic cleanup of completed jobs:

```yaml
spec:
  jobTemplate:
    spec:
      # Clean up successful jobs after 1 hour
      ttlSecondsAfterFinished: 3600

      # For CronJobs, limit history
      successfulJobsHistoryLimit: 3
      failedJobsHistoryLimit: 1
```

## Multi-Environment Configuration

### Using Kustomize

```
├── base/
│   ├── kustomization.yaml
│   ├── listsource.yaml
│   └── listjob.yaml
├── overlays/
│   ├── dev/
│   │   ├── kustomization.yaml
│   │   └── resources.yaml  # Lower resources for dev
│   ├── staging/
│   │   └── kustomization.yaml
│   └── production/
│       ├── kustomization.yaml
│       ├── resources.yaml  # Higher resources
│       └── security.yaml   # Stricter security
```

Apply per environment:

```bash
kubectl apply -k overlays/production
```

## Disaster Recovery

### Backup

Critical resources to back up:

```bash
# Backup all Parallax resources
kubectl get listsource,listjob,listcronjob -o yaml > parallax-backup.yaml

# Backup secrets
kubectl get secret -o yaml > secrets-backup.yaml
```

### Restore

```bash
kubectl apply -f parallax-backup.yaml
```

### Testing Failures

Chaos engineering:

```yaml
# Randomly kill 10% of pods
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: parallax-chaos
spec:
  action: pod-kill
  mode: percentage
  value: "10"
  selector:
    labelSelectors:
      app: parallax-job
```

## Deployment Checklist

Before deploying to production:

- [ ] Resource requests and limits configured
- [ ] Security context with non-root user
- [ ] Secrets used for all credentials
- [ ] Monitoring and alerting configured
- [ ] Retry logic and timeouts set
- [ ] Logs structured and parseable
- [ ] Jobs are idempotent
- [ ] Network policies applied
- [ ] RBAC minimized
- [ ] Multi-environment configs tested
- [ ] Disaster recovery tested
- [ ] Cost optimization reviewed
- [ ] Documentation updated

## Next Steps

- Review all previous examples with these patterns in mind
- Implement monitoring for your specific use cases
- Test failure scenarios in staging
- Document your production runbooks
- Set up alerts and on-call rotation
