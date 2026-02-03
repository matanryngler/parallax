# Parallax Troubleshooting Guide

This guide helps diagnose and resolve common issues with Parallax.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [ListSource Issues](#listsource-issues)
- [ListJob Issues](#listjob-issues)
- [ListCronJob Issues](#listcronjob-issues)
- [Operator Issues](#operator-issues)
- [Debugging Commands](#debugging-commands)
- [FAQ](#faq)

---

## Quick Diagnostics

### Check Operator Status

```bash
# Is the operator running?
kubectl get pods -n parallax-system

# Check operator logs
kubectl logs -n parallax-system deployment/parallax -f

# Check for recent errors
kubectl logs -n parallax-system deployment/parallax --tail=50 | grep -i error
```

### Check Resource Status

```bash
# List all Parallax resources
kubectl get listsource,listjob,listcronjob

# Get detailed status
kubectl describe listsource <name>
kubectl describe listjob <name>
kubectl describe listcronjob <name>
```

### Check Created Resources

```bash
# Check ConfigMaps (created by ListSource)
kubectl get configmap -l managed-by=parallax

# Check Jobs (created by ListJob)
kubectl get jobs -l managed-by=parallax

# Check Pods
kubectl get pods -l managed-by=parallax
```

---

## ListSource Issues

### ConfigMap Not Created

**Symptoms:**
- ListSource exists but no ConfigMap is created
- Status shows "Pending" indefinitely

**Diagnosis:**
```bash
kubectl describe listsource <name>
kubectl logs -n parallax-system deployment/parallax | grep <name>
```

**Common Causes:**

1. **Operator not running**
   ```bash
   kubectl get pods -n parallax-system
   # Should show parallax pod as Running
   ```
   **Solution:** Restart operator if crashed
   ```bash
   kubectl rollout restart deployment/parallax -n parallax-system
   ```

2. **RBAC permissions missing**
   ```bash
   kubectl auth can-i create configmaps --as=system:serviceaccount:parallax-system:parallax
   ```
   **Solution:** Reinstall CRDs and RBAC
   ```bash
   make install
   ```

3. **Invalid spec configuration**
   - Check that required fields are set based on `type`
   - For `static`: `staticList` must be non-empty
   - For `api`: `url` and `jsonPath` are required
   - For `postgresql`: `connectionString` and `query` are required

### ConfigMap Too Large

**Symptoms:**
- Error: "ConfigMap too large"
- Status shows error about size limit

**Diagnosis:**
```bash
kubectl get configmap <listsource-name>-data -o yaml | wc -c
# ConfigMaps are limited to 1MB
```

**Solution:**

1. **Reduce item count** (recommended):
   ```yaml
   # For API sources, limit results
   api:
     url: https://api.example.com/items?limit=1000

   # For PostgreSQL, add LIMIT clause
   postgres:
     query: "SELECT id FROM items WHERE status = 'pending' LIMIT 1000"
   ```

2. **Use pagination**:
   - Create multiple ListSources with different offsets/filters
   - Process in batches

3. **Store less data per item**:
   - Extract only IDs, not full objects
   - Use `jsonPath` to select minimal data

### API Rate Limiting

**Symptoms:**
- Status shows HTTP 429 errors
- "Too Many Requests" in error field

**Diagnosis:**
```bash
kubectl get listsource <name> -o jsonpath='{.status.error}'
```

**Solution:**

1. **Increase intervalSeconds**:
   ```yaml
   spec:
     intervalSeconds: 600  # 10 minutes instead of 1 minute
   ```

2. **Use caching headers** (if supported):
   ```yaml
   api:
     headers:
       Cache-Control: max-age=300
   ```

3. **Contact API provider** for higher rate limits

### Database Connection Failures

**Symptoms:**
- Error: "connection refused" or "timeout"
- PostgreSQL ListSource never becomes Ready

**Diagnosis:**
```bash
# Check database is accessible
kubectl run -it --rm debug --image=postgres:16-alpine --restart=Never -- \
  psql postgresql://user:pass@postgres:5432/db -c "SELECT 1"
```

**Common Causes:**

1. **Wrong connection string**:
   ```yaml
   # Format: postgresql://username:password@host:port/database?options
   connectionString: "postgresql://user:pass@postgres.default.svc.cluster.local:5432/mydb?sslmode=require"
   ```

2. **Database not accessible**:
   - Check database pod is running
   - Check network policies allow connections
   - Verify DNS resolution: `postgres.default.svc.cluster.local`

3. **SSL/TLS misconfiguration**:
   ```yaml
   # Try with SSL disabled for debugging
   postgres:
     sslMode: disable
   ```

4. **Authentication failure**:
   - Verify credentials in Secret
   - Check user has SELECT permissions
   ```sql
   GRANT SELECT ON table_name TO username;
   ```

### JSONPath Extraction Returns Empty

**Symptoms:**
- Status shows `itemCount: 0`
- API responds but no items extracted

**Diagnosis:**
```bash
# Fetch API response manually
curl -H "Authorization: Bearer token" https://api.example.com/items | jq

# Test JSONPath expression at https://jsonpath.com/
```

**Common Causes:**

1. **Incorrect JSONPath syntax**:
   ```yaml
   # Wrong
   jsonPath: "items[*].id"  # Missing $

   # Correct
   jsonPath: "$.items[*].id"
   ```

2. **Response structure different than expected**:
   ```bash
   # Check actual API response structure
   kubectl logs -n parallax-system deployment/parallax | grep -A 5 "API response"
   ```

3. **Items nested differently**:
   ```yaml
   # For {"data": {"results": [{"id": 1}]}}
   jsonPath: "$.data.results[*].id"

   # For [{"id": 1}, {"id": 2}]
   jsonPath: "$[*].id"
   ```

**Solution:** Test JSONPath at [jsonpath.com](https://jsonpath.com/) with actual API response

### ListSource Data is Stale

**Symptoms:**
- `lastUpdateTime` is old
- ConfigMap not updating despite `intervalSeconds` set

**Diagnosis:**
```bash
kubectl get listsource <name> -o yaml | grep -A 5 status
```

**Causes:**

1. **Operator not reconciling**:
   - Check operator logs for errors
   - Verify operator has CPU/memory available

2. **Data hasn't changed**:
   - ListSource only updates if data changes
   - Check API/database for new data

3. **intervalSeconds set to 0**:
   ```yaml
   intervalSeconds: 0  # Only fetches once
   ```
   **Solution:** Set appropriate interval:
   ```yaml
   intervalSeconds: 300  # Refresh every 5 minutes
   ```

---

## ListJob Issues

### Jobs Not Starting

**Symptoms:**
- ListJob created but no Kubernetes Jobs appear
- Status remains empty

**Diagnosis:**
```bash
kubectl describe listjob <name>
kubectl get events --field-selector involvedObject.name=<name>
```

**Common Causes:**

1. **ListSource not ready**:
   ```bash
   kubectl get listsource <source-name>
   # Check Ready status
   ```
   **Solution:** Fix ListSource issues first

2. **ListSource has no items**:
   ```bash
   kubectl get listsource <source-name> -o jsonpath='{.status.itemCount}'
   ```
   **Solution:** Add items to ListSource

3. **Invalid jobTemplate**:
   - Check that `image` is specified
   - Verify `parallelism` is > 0

4. **ResourceQuota exceeded**:
   ```bash
   kubectl get resourcequota
   ```
   **Solution:** Increase quota or reduce resource requests

### Jobs Failing

**Symptoms:**
- Jobs created but pods fail
- Status shows Failed condition

**Diagnosis:**
```bash
# Find failed pods
kubectl get pods -l job-name --field-selector status.phase=Failed

# Check pod logs
kubectl logs <pod-name>

# Check pod events
kubectl describe pod <pod-name>
```

**Common Causes:**

1. **Image pull errors**:
   - Error: "ImagePullBackOff" or "ErrImagePull"
   - **Solution:**
     - Check image name and tag
     - Add `imagePullSecrets` if private registry
     - Verify registry is accessible from cluster

2. **OOMKilled (Out of Memory)**:
   - Error: "OOMKilled" in pod status
   - **Solution:** Increase memory limits
     ```yaml
     resources:
       limits:
         memory: "512Mi"  # Increase from 256Mi
     ```

3. **Command failures**:
   - Container exits with non-zero code
   - **Solution:** Check logs for error messages
   ```bash
   kubectl logs <pod-name>
   ```

4. **Missing environment variables**:
   - Script expects variables that aren't set
   - **Solution:** Add to template:
     ```yaml
     env:
       - name: REQUIRED_VAR
         value: "value"
     ```

5. **Secrets not found**:
   - Error: "Secret 'xyz' not found"
   - **Solution:** Create secret or fix reference

### Items Not Passed to Containers

**Symptoms:**
- Jobs run but $ITEM is empty
- Pods don't see item values

**Diagnosis:**
```bash
# Check pod environment
kubectl exec <pod-name> -- env | grep ITEM
```

**Causes:**

1. **Wrong envName**:
   ```yaml
   # Template specifies ITEM but script uses different name
   template:
     envName: ITEM  # Should match variable used in script
     command:
       - sh
       - -c
       - echo $CUSTOMER_ID  # Mismatch!
   ```
   **Solution:** Use consistent variable name

2. **ConfigMap not created**:
   - Check ListSource ConfigMap exists
   ```bash
   kubectl get configmap <listsource-name>-data
   ```

3. **Init container failed**:
   - parallax-init container sets up environment
   ```bash
   kubectl logs <pod-name> -c parallax-init
   ```

### Parallelism Not Working as Expected

**Symptoms:**
- Only 1 job runs at a time despite higher parallelism
- All jobs run simultaneously despite parallelism limit

**Diagnosis:**
```bash
kubectl get jobs -w
kubectl get pods --watch
```

**Causes:**

1. **Resource constraints**:
   - Cluster doesn't have resources for more pods
   ```bash
   kubectl describe nodes | grep -A 5 "Allocated resources"
   ```
   **Solution:** Reduce resource requests or add nodes

2. **Node affinity/tolerations**:
   - Pods can't schedule on available nodes
   **Solution:** Check node selectors and tolerations

3. **PodDisruptionBudget**:
   - PDB prevents additional pods
   ```bash
   kubectl get pdb
   ```

### Jobs Not Cleaning Up

**Symptoms:**
- Old jobs and pods remain after completion
- Cluster filling with completed resources

**Diagnosis:**
```bash
kubectl get jobs --field-selector status.successful=1
kubectl get pods --field-selector status.phase=Succeeded
```

**Solution:**

1. **Set TTL**:
   ```yaml
   spec:
     ttlSecondsAfterFinished: 3600  # 1 hour
   ```

2. **Manual cleanup**:
   ```bash
   kubectl delete jobs --field-selector status.successful=1
   kubectl delete pods --field-selector status.phase=Succeeded
   ```

3. **Verify TTL controller is enabled**:
   - TTL controller is beta in K8s 1.23+
   - Check cluster feature gates

---

## ListCronJob Issues

### Schedule Not Triggering

**Symptoms:**
- ListCronJob exists but no ListJobs created
- `lastScheduleTime` is empty or old

**Diagnosis:**
```bash
kubectl describe listcronjob <name>
kubectl logs -n parallax-system deployment/parallax | grep CronJob
```

**Common Causes:**

1. **Suspended**:
   ```bash
   kubectl get listcronjob <name> -o jsonpath='{.spec.suspend}'
   ```
   **Solution:** Resume:
   ```bash
   kubectl patch listcronjob <name> -p '{"spec":{"suspend":false}}'
   ```

2. **Invalid cron syntax**:
   ```yaml
   # Wrong
   schedule: "* * * * * *"  # 6 fields (includes seconds)

   # Correct
   schedule: "* * * * *"  # 5 fields
   ```
   **Validation:** Use [crontab.guru](https://crontab.guru/)

3. **Missed startingDeadlineSeconds**:
   - Job couldn't start within deadline
   ```yaml
   startingDeadlineSeconds: 300  # Increase if too short
   ```

4. **Operator timezone issues**:
   - Cron schedules use operator's timezone (usually UTC)
   - **Solution:** Account for timezone in schedule
   ```yaml
   # For 9 AM EST (UTC-5):
   schedule: "0 14 * * *"  # 2 PM UTC = 9 AM EST
   ```

### Jobs Overlapping

**Symptoms:**
- Multiple ListJobs run simultaneously
- Previous job still running when next fires

**Diagnosis:**
```bash
kubectl get listjob -l cronjob=<name>
kubectl get listcronjob <name> -o jsonpath='{.status.active}'
```

**Solution:**

1. **Set concurrencyPolicy to Forbid**:
   ```yaml
   spec:
     concurrencyPolicy: Forbid  # Skip if job already active
   ```

2. **Increase schedule interval**:
   ```yaml
   schedule: "0 */2 * * *"  # Every 2 hours instead of every hour
   ```

3. **Add activeDeadlineSeconds**:
   ```yaml
   activeDeadlineSeconds: 3600  # Kill jobs after 1 hour
   ```

### History Cleanup Not Working

**Symptoms:**
- Old ListJobs accumulate
- History limit not being enforced

**Diagnosis:**
```bash
kubectl get listjob -l cronjob=<name>
```

**Causes:**

1. **History limits not set**:
   ```yaml
   spec:
     successfulJobsHistoryLimit: 3  # Explicitly set
     failedJobsHistoryLimit: 1
   ```

2. **Owner references missing**:
   - Check ListJobs have ownerReferences
   ```bash
   kubectl get listjob <name> -o jsonpath='{.metadata.ownerReferences}'
   ```

3. **Manual ListJobs**:
   - Manually created ListJobs aren't cleaned up
   - Only ListJobs created by CronJob are managed

### Schedule Running Late

**Symptoms:**
- Jobs start minutes after scheduled time
- `lastScheduleTime` shows delays

**Causes:**

1. **Operator under load**:
   - Check operator CPU/memory
   ```bash
   kubectl top pod -n parallax-system
   ```
   **Solution:** Increase operator resources

2. **Cluster time drift**:
   ```bash
   kubectl get nodes -o wide
   # Check system time
   ```

3. **Long reconciliation loops**:
   - Too many resources for operator to process
   - **Solution:** Distribute across multiple operators/clusters

---

## Operator Issues

### Operator Crashing

**Symptoms:**
- parallax pod in CrashLoopBackOff
- Operator restarts frequently

**Diagnosis:**
```bash
kubectl get pods -n parallax-system
kubectl logs -n parallax-system deployment/parallax --previous
```

**Common Causes:**

1. **OOMKilled**:
   - Increase memory limits
   ```bash
   kubectl edit deployment parallax -n parallax-system
   # Increase memory limits
   ```

2. **Panic/Fatal errors**:
   - Check logs for stack traces
   - File issue at https://github.com/yourusername/parallax/issues

3. **Invalid configuration**:
   - Check operator environment variables
   - Verify RBAC permissions

### High CPU/Memory Usage

**Symptoms:**
- Operator using excessive resources
- Slow reconciliation

**Diagnosis:**
```bash
kubectl top pod -n parallax-system
```

**Solutions:**

1. **Too many resources**:
   - Reduce number of ListSources with low intervalSeconds
   - Spread resources across namespaces

2. **Inefficient queries**:
   - For PostgreSQL sources, add indexes
   - For API sources, reduce polling frequency

3. **Increase operator resources**:
   ```bash
   kubectl edit deployment parallax -n parallax-system
   # Increase resource limits
   ```

### Operator Not Reconciling

**Symptoms:**
- Changes to resources not taking effect
- Status not updating

**Diagnosis:**
```bash
kubectl logs -n parallax-system deployment/parallax | grep "Reconciling"
```

**Solutions:**

1. **Force reconciliation**:
   ```bash
   # Add annotation to trigger reconcile
   kubectl annotate listsource <name> reconcile=true --overwrite
   ```

2. **Restart operator**:
   ```bash
   kubectl rollout restart deployment/parallax -n parallax-system
   ```

3. **Check for errors**:
   ```bash
   kubectl logs -n parallax-system deployment/parallax | grep -i error
   ```

---

## Debugging Commands

### View All Parallax Resources

```bash
kubectl api-resources --api-group=batchops.example.com
kubectl get listsource,listjob,listcronjob --all-namespaces
```

### Watch Resources in Real-Time

```bash
# Watch Parallax resources
kubectl get listsource,listjob,listcronjob --watch

# Watch Jobs and Pods
kubectl get jobs,pods --watch
```

### Check Resource Events

```bash
# Recent events for a resource
kubectl get events --field-selector involvedObject.name=<name>

# All recent events
kubectl get events --sort-by='.lastTimestamp'
```

### Debug Pod Issues

```bash
# Get pod logs
kubectl logs <pod-name>

# Get logs from all containers
kubectl logs <pod-name> --all-containers

# Get logs from previous instance
kubectl logs <pod-name> --previous

# Execute command in pod
kubectl exec -it <pod-name> -- sh

# Check pod resource usage
kubectl top pod <pod-name>
```

### Check ConfigMap Data

```bash
# View ConfigMap created by ListSource
kubectl get configmap <listsource-name>-data -o yaml

# Count items in ConfigMap
kubectl get configmap <listsource-name>-data -o jsonpath='{.data}' | jq 'keys | length'
```

### Check Operator Metrics

```bash
# Port-forward to metrics endpoint
kubectl port-forward -n parallax-system deployment/parallax 8080:8080

# Query metrics
curl http://localhost:8080/metrics | grep parallax
```

---

## FAQ

### Q: How large can my list be?

**A:** ConfigMaps are limited to 1MB. This typically allows for 10,000-50,000 items depending on item size. For larger datasets:
- Use pagination (multiple ListSources)
- Process in batches
- Consider external storage

### Q: Can I use multiple ListSources with one ListJob?

**A:** No, each ListJob references one ListSource. To process multiple sources:
- Create multiple ListJobs (recommended)
- Or create a "master" ListSource that aggregates data

### Q: How do I handle failed items?

**A:** Use `backoffLimit` for retries. For persistent failures:
- Check pod logs to identify root cause
- Implement idempotency in your processing logic
- Consider dead-letter queues for failed items
- Use monitoring/alerting to detect failures

### Q: Can I prioritize certain items?

**A:** Not directly, but you can:
- Create separate ListSources with different priorities
- Use Kubernetes PriorityClass for job pods
- Process high-priority items in a separate ListJob with higher parallelism

### Q: How do I monitor Parallax?

**A:** Several options:
- Prometheus metrics from operator (`:8080/metrics`)
- Kubernetes events (`kubectl get events`)
- Pod logs and status
- Custom monitoring scripts
- See [05-production-patterns](../examples/05-production-patterns/monitoring.yaml) for examples

### Q: Can I update a running ListJob?

**A:** No, ListJobs are immutable once created. To make changes:
- Delete the ListJob and create a new one
- For recurring jobs, use ListCronJob which creates fresh ListJobs

### Q: What happens if the operator restarts?

**A:** The operator is stateless:
- Existing Jobs continue running
- ConfigMaps persist
- Operator resumes reconciliation on restart
- No data loss occurs

### Q: How do I upgrade Parallax?

**A:**
```bash
# Update CRDs
make install

# Update operator
kubectl set image deployment/parallax parallax=newimage:tag -n parallax-system

# Or reinstall
make deploy IMG=newimage:tag
```

### Q: Can I use Parallax in multiple namespaces?

**A:** Yes, the operator watches all namespaces by default. Create resources in any namespace:
```bash
kubectl apply -f listsource.yaml -n my-namespace
```

### Q: How do I backup Parallax resources?

**A:**
```bash
# Backup all resources
kubectl get listsource,listjob,listcronjob -o yaml > parallax-backup.yaml

# Restore
kubectl apply -f parallax-backup.yaml
```

---

## Getting Help

If you're still experiencing issues:

1. **Check operator logs**: `kubectl logs -n parallax-system deployment/parallax`
2. **Review documentation**: [README.md](../README.md), [API Reference](./api-reference.md)
3. **Check examples**: [examples/](../examples/)
4. **File an issue**: [GitHub Issues](https://github.com/yourusername/parallax/issues)

When filing issues, please include:
- Parallax version
- Kubernetes version
- Resource YAML
- Operator logs
- Steps to reproduce
