# Example 04: Scheduled Batch Processing

This example demonstrates how to run parallel batch jobs on a schedule using ListCronJob.

## Use Case

Your application needs to:
1. Run batch processing jobs at specific times or intervals
2. Process items from a data source (static, API, or database)
3. Control concurrency to prevent overlapping job runs
4. Maintain history of job executions

This is ideal for recurring workflows that need to run on a schedule.

## Real-World Scenarios

- Daily report generation at midnight
- Hourly data synchronization
- Weekly backup verification
- Monthly billing runs
- Periodic cleanup tasks

## Files in This Example

- `listsource.yaml` - Data source (reusable across runs)
- `listcronjob.yaml` - Scheduled job definition

## Architecture

```
┌────────────────┐
│  ListCronJob   │ → Runs on cron schedule
└────────┬───────┘
         │
         ▼
┌────────────────┐
│  ListJob (run) │ → Created for each schedule trigger
└────────┬───────┘
         │
         ▼
┌────────────────┐
│  ListSource    │ → Fetches latest data
└────────┬───────┘
         │
         ▼
┌────────────────────────┐
│ Job-0  Job-1  Job-2... │ → Process items in parallel
└────────────────────────┘
```

## Quick Start

### 1. Apply the ListSource

```bash
kubectl apply -f listsource.yaml
```

This creates a static list of reports to generate.

### 2. Apply the ListCronJob

```bash
kubectl apply -f listcronjob.yaml
```

### 3. Watch for Scheduled Runs

The ListCronJob is configured to run every 5 minutes. Wait for the next schedule:

```bash
# Watch for ListJobs being created
kubectl get listjob -w

# Watch for Jobs being created
kubectl get jobs -w
```

### 4. Check CronJob Status

```bash
kubectl get listcronjob report-generator -o yaml
```

Look for:
- `lastScheduleTime` - When it last ran
- `lastSuccessfulTime` - When it last succeeded
- `active` - Currently running ListJobs

### 5. View Job History

```bash
# List all ListJobs created by the CronJob
kubectl get listjob -l cronjob=report-generator

# Check logs from the latest run
kubectl logs -l job-name --tail=50
```

### 6. Manually Trigger a Run

You can manually create a ListJob to test immediately:

```bash
# Create a one-time ListJob
cat <<EOF | kubectl apply -f -
apiVersion: batchops.example.com/v1alpha1
kind: ListJob
metadata:
  name: report-generator-manual
  labels:
    manual-run: "true"
spec:
  listSourceRef:
    name: report-data
  parallelism: 2
  jobTemplate:
    spec:
      completionMode: Indexed
      ttlSecondsAfterFinished: 300
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: reporter
              image: alpine:3.19
              command:
                - sh
                - -c
                - |
                  echo "Generating report: \$REPORT_NAME"
                  sleep 3
                  echo "✓ Report generated: \$REPORT_NAME"
              env:
                - name: REPORT_NAME
                  value: "\$(ITEM_INDEX)"
EOF
```

### 7. Clean Up

```bash
kubectl delete -f .
```

## Cron Schedule Format

The `schedule` field uses standard cron syntax:

```
┌─────── minute (0 - 59)
│ ┌───── hour (0 - 23)
│ │ ┌─── day of month (1 - 31)
│ │ │ ┌─ month (1 - 12)
│ │ │ │ ┌ day of week (0 - 6) (Sunday=0)
│ │ │ │ │
* * * * *
```

### Common Schedules

| Schedule | Description | Example Use Case |
|----------|-------------|------------------|
| `*/5 * * * *` | Every 5 minutes | Frequent polling |
| `0 * * * *` | Every hour | Hourly sync |
| `0 */6 * * *` | Every 6 hours | Periodic checks |
| `0 0 * * *` | Daily at midnight | Daily reports |
| `0 2 * * 0` | Weekly on Sunday at 2 AM | Weekly cleanup |
| `0 0 1 * *` | Monthly on the 1st | Monthly billing |
| `30 4 * * 1-5` | Weekdays at 4:30 AM | Business day tasks |

### Testing Schedules

Use [crontab.guru](https://crontab.guru/) to validate and understand cron expressions.

## Concurrency Policies

Control how overlapping runs are handled:

### Allow (Default)

```yaml
spec:
  concurrencyPolicy: Allow
```

- Multiple ListJobs can run simultaneously
- New jobs start even if previous ones are still running
- **Use when**: Jobs are independent and can overlap

### Forbid

```yaml
spec:
  concurrencyPolicy: Forbid
```

- Only one ListJob runs at a time
- Skips new runs if previous run is still active
- **Use when**: Jobs modify shared state or resources

### Replace

```yaml
spec:
  concurrencyPolicy: Replace
```

- Cancels the currently running job
- Starts the new job immediately
- **Use when**: Only the latest run matters

## Job History Management

Control how many completed and failed runs to keep:

```yaml
spec:
  successfulJobsHistoryLimit: 3  # Keep last 3 successful runs
  failedJobsHistoryLimit: 1      # Keep last 1 failed run
```

This automatically cleans up old ListJobs to prevent cluster clutter.

## Suspend and Resume

Temporarily pause scheduled runs:

```yaml
spec:
  suspend: true  # Set to true to pause scheduling
```

Or using kubectl:

```bash
# Suspend the CronJob
kubectl patch listcronjob report-generator -p '{"spec":{"suspend":true}}'

# Resume the CronJob
kubectl patch listcronjob report-generator -p '{"spec":{"suspend":false}}'
```

## Time Zones

**IMPORTANT**: CronJobs use the cluster's time zone (typically UTC).

To run at specific local times:

```yaml
# Run at 9 AM Eastern Time (UTC-5)
schedule: "0 14 * * *"  # 14:00 UTC = 9:00 AM EST

# Run at 6 PM Pacific Time (UTC-8)
schedule: "0 2 * * *"   # 02:00 UTC = 6:00 PM PST
```

Use [Time Zone Converter](https://www.worldtimebuddy.com/) for conversions.

## Advanced Patterns

### Pattern 1: Dynamic Data Source with Scheduling

Combine API polling with scheduled processing:

```yaml
apiVersion: batchops.example.com/v1alpha1
kind: ListSource
metadata:
  name: daily-orders
spec:
  type: api
  api:
    url: https://api.example.com/orders/today
    jsonPath: "$.orders[*].id"
  intervalSeconds: 0  # Fetch once per job run

---
apiVersion: batchops.example.com/v1alpha1
kind: ListCronJob
metadata:
  name: daily-order-processor
spec:
  schedule: "0 0 * * *"  # Daily at midnight
  listSourceRef:
    name: daily-orders
  # ... rest of config
```

### Pattern 2: Sequential Processing

Ensure jobs don't overlap:

```yaml
spec:
  concurrencyPolicy: Forbid
  startingDeadlineSeconds: 300  # Skip if can't start within 5 min
  successfulJobsHistoryLimit: 7  # Keep one week of history
```

### Pattern 3: Long-Running Batch with Timeout

```yaml
spec:
  jobTemplate:
    spec:
      activeDeadlineSeconds: 3600  # Kill jobs after 1 hour
      backoffLimit: 2              # Retry failed jobs twice
```

## Monitoring and Alerts

### Check for Missed Schedules

```bash
kubectl get listcronjob report-generator -o jsonpath='{.status.lastScheduleTime}'
```

If `lastScheduleTime` is much older than expected, investigate:

1. Check if CronJob is suspended
2. Check operator logs for errors
3. Verify concurrency policy isn't blocking runs

### Monitor Job Success Rate

```bash
# Count successful vs failed jobs
kubectl get listjob -l cronjob=report-generator \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.succeeded}{"\t"}{.status.failed}{"\n"}{end}'
```

### Set Up Alerts

Create a monitoring job that checks CronJob health:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cronjob-monitor
spec:
  schedule: "*/15 * * * *"  # Check every 15 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: monitor
              image: bitnami/kubectl:latest
              command:
                - sh
                - -c
                - |
                  LAST_SCHEDULE=$(kubectl get listcronjob report-generator -o jsonpath='{.status.lastScheduleTime}')
                  if [ -z "$LAST_SCHEDULE" ]; then
                    echo "WARNING: No schedule time found"
                    exit 1
                  fi
                  # Add more checks...
          restartPolicy: OnFailure
```

## Troubleshooting

### Jobs Not Being Created

1. Check CronJob status:
   ```bash
   kubectl describe listcronjob report-generator
   ```

2. Verify schedule syntax:
   ```bash
   kubectl get listcronjob report-generator -o jsonpath='{.spec.schedule}'
   ```

3. Check if suspended:
   ```bash
   kubectl get listcronjob report-generator -o jsonpath='{.spec.suspend}'
   ```

4. Check operator logs:
   ```bash
   kubectl logs -n parallax-system deployment/parallax -f | grep ListCronJob
   ```

### Jobs Starting Late

- Check `startingDeadlineSeconds` - jobs may be skipped if deadline passed
- Verify cluster time is correct: `kubectl get nodes -o wide`
- Check operator resource limits - CPU throttling can delay scheduling

### Jobs Overlapping Unexpectedly

- Set `concurrencyPolicy: Forbid`
- Increase `startingDeadlineSeconds` to skip late runs
- Adjust schedule to allow more time between runs

## Best Practices

1. **Start with infrequent schedules** - Use hourly or daily initially, then optimize
2. **Set appropriate history limits** - Balance between debugging and resource usage
3. **Use Forbid for stateful operations** - Prevent data corruption from concurrent runs
4. **Monitor missed schedules** - Set up alerts for production CronJobs
5. **Test schedule changes** - Use `suspend: true` while testing new schedules
6. **Document time zones** - Always note expected local time in comments

## Next Steps

- See [05-production-patterns](../05-production-patterns/) for production configurations
- Combine with [02-api-integration](../02-api-integration/) for scheduled API polling
- Combine with [03-postgres-etl](../03-postgres-etl/) for scheduled ETL jobs
