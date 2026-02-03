# Example 02: API Integration

This example demonstrates how to fetch data from a REST API and process the results in parallel using Parallax.

## Use Case

Your application needs to:
1. Periodically fetch a list from a REST API
2. Extract specific items using JSONPath
3. Process each item independently in parallel

This is ideal for workflows where the data source changes frequently and you need to react to new items.

## Real-World Scenarios

- Processing pending orders from an e-commerce API
- Handling webhook events from a queue
- Processing new files from a storage service API
- Syncing data from external systems
- Processing items from a work queue API

## Files in This Example

- `mock-server.yaml` - Optional: Local API server for testing
- `listsource-api.yaml` - Fetches data from a REST API
- `listjob-api.yaml` - Processes API items in parallel

## Architecture

```
┌─────────────┐
│  REST API   │ ← Periodically polled every 30s
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ ListSource  │ → Extracts items using JSONPath
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  ConfigMap  │ → Stores extracted items
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  ListJob    │ → Creates parallel jobs
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│ Job-0  Job-1  Job-2 ... │ → Process items
└─────────────────────────┘
```

## Quick Start with Mock Server

### 1. Deploy the Mock API Server

This creates a simple API that returns JSON data:

```bash
kubectl apply -f mock-server.yaml
```

Wait for it to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=mock-api --timeout=60s
```

### 2. Test the API

```bash
# Port-forward to access the API
kubectl port-forward service/mock-api 8080:80 &

# Test the endpoint
curl http://localhost:8080/api/items
```

Expected response:
```json
{
  "status": "success",
  "items": [
    {"id": "task-001", "priority": "high"},
    {"id": "task-002", "priority": "medium"},
    {"id": "task-003", "priority": "low"}
  ]
}
```

### 3. Apply the ListSource

```bash
kubectl apply -f listsource-api.yaml
```

Check that it fetched the data:

```bash
kubectl get listsource api-items -o yaml
```

The status should show:
- `Ready: True`
- `ItemCount: 3`

### 4. Apply the ListJob

```bash
kubectl apply -f listjob-api.yaml
```

### 5. Watch Processing

```bash
# Watch jobs being created
kubectl get jobs -w

# Check job logs
kubectl logs -l job-name --prefix=true
```

### 6. Clean Up

```bash
kubectl delete -f .
```

## Using with External APIs

To use with a real external API, modify `listsource-api.yaml`:

### Example: GitHub API

```yaml
spec:
  type: api
  api:
    url: https://api.github.com/repos/kubernetes/kubernetes/pulls
    method: GET
    headers:
      - name: Accept
        value: application/vnd.github.v3+json
    jsonPath: "$[*].number"  # Extract PR numbers
  intervalSeconds: 300  # Check every 5 minutes
```

### Example: REST API with Authentication

```yaml
spec:
  type: api
  api:
    url: https://api.example.com/v1/tasks
    method: GET
    headers:
      - name: Authorization
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: token
    jsonPath: "$.data.tasks[*].id"
  intervalSeconds: 60
```

First create the secret:

```bash
kubectl create secret generic api-credentials \
  --from-literal=token="Bearer your-api-token"
```

### Example: POST Request with Body

```yaml
spec:
  type: api
  api:
    url: https://api.example.com/v1/query
    method: POST
    headers:
      - name: Content-Type
        value: application/json
    body: |
      {
        "query": "status:pending",
        "limit": 100
      }
    jsonPath: "$.results[*].id"
  intervalSeconds: 120
```

## JSONPath Examples

The `jsonPath` field extracts items from the API response:

| API Response | JSONPath | Extracted Items |
|-------------|----------|-----------------|
| `{"items": ["a", "b"]}` | `$.items[*]` | `["a", "b"]` |
| `[{"id": 1}, {"id": 2}]` | `$[*].id` | `[1, 2]` |
| `{"data": {"users": [{"name": "alice"}]}}` | `$.data.users[*].name` | `["alice"]` |
| `{"list": [{"val": "x"}, {"val": "y"}]}` | `$.list[*].val` | `["x", "y"]` |

Test your JSONPath expressions at [JSONPath Online Evaluator](https://jsonpath.com/).

## Refresh Behavior

The ListSource automatically refreshes based on `intervalSeconds`:

- **30 seconds** - Frequent updates for real-time processing
- **300 seconds (5 min)** - Moderate polling for semi-real-time
- **3600 seconds (1 hour)** - Periodic batch processing
- **0** - No refresh (fetch once)

Each refresh:
1. Fetches data from the API
2. Extracts items using JSONPath
3. Updates the ConfigMap
4. The ListJob automatically picks up changes

## Error Handling

If the API request fails:

- The ListSource status will show `Ready: False`
- The error message appears in status conditions
- Previous data remains in the ConfigMap
- Retries occur on the next interval

Check for errors:

```bash
kubectl describe listsource api-items
```

## Rate Limiting

To avoid hitting API rate limits:

1. **Increase intervalSeconds**:
   ```yaml
   intervalSeconds: 600  # 10 minutes
   ```

2. **Use caching headers** (if supported by API):
   ```yaml
   headers:
     - name: Cache-Control
       value: max-age=300
   ```

3. **Monitor API usage** in your job logs

## Security Best Practices

1. **Always use secrets** for API credentials:
   ```yaml
   headers:
     - name: Authorization
       valueFrom:
         secretKeyRef:
           name: api-credentials
           key: token
   ```

2. **Use HTTPS** for external APIs (never HTTP for production)

3. **Limit API token permissions** to read-only access

4. **Rotate credentials regularly**

## Next Steps

- Try [03-postgres-etl](../03-postgres-etl/) for database integration
- Learn about [04-scheduled-batch](../04-scheduled-batch/) for scheduled API polling
- See [05-production-patterns](../05-production-patterns/) for production configurations
