# Example 03: PostgreSQL ETL

This example demonstrates how to query a PostgreSQL database and process records in parallel using Parallax.

## Use Case

Your application needs to:
1. Query a PostgreSQL database periodically
2. Extract a list of IDs or values from the results
3. Process each record independently in parallel

This is ideal for ETL workflows, data migration, or processing database records that require external API calls or heavy computation.

## Real-World Scenarios

- Processing unprocessed orders from a database
- Running data transformations on specific records
- Migrating data between systems
- Enriching database records with external data
- Running maintenance tasks on specific rows

## Files in This Example

- `postgres-deployment.yaml` - Test PostgreSQL database
- `listsource-postgres.yaml` - Queries database for items
- `listjob-etl.yaml` - Processes database records in parallel

## Architecture

```
┌──────────────┐
│  PostgreSQL  │ ← Periodically queried every 60s
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ListSource  │ → Executes SQL query
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ConfigMap   │ → Stores extracted IDs
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   ListJob    │ → Creates parallel jobs
└──────┬───────┘
       │
       ▼
┌──────────────────────────┐
│ Job-0  Job-1  Job-2  ... │ → Process records
└──────────────────────────┘
```

## Quick Start

### 1. Deploy PostgreSQL Test Database

```bash
kubectl apply -f postgres-deployment.yaml
```

Wait for PostgreSQL to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=postgres --timeout=120s
```

### 2. Verify Database and Test Data

The deployment automatically creates a test database with sample data:

```bash
# Connect to the database
kubectl exec -it deployment/postgres -- psql -U postgres -d testdb

# View the orders table
testdb=# SELECT * FROM orders;
```

You should see:
```
 order_id |    customer     | status  |     created_at
----------+-----------------+---------+--------------------
        1 | alice@example   | pending | 2025-02-03 10:00:00
        2 | bob@example     | pending | 2025-02-03 10:05:00
        3 | charlie@example | pending | 2025-02-03 10:10:00
        4 | david@example   | pending | 2025-02-03 10:15:00
        5 | eve@example     | pending | 2025-02-03 10:20:00
```

Type `\q` to exit psql.

### 3. Apply the ListSource

```bash
kubectl apply -f listsource-postgres.yaml
```

Check that it queried the database:

```bash
kubectl get listsource postgres-orders -o yaml
```

The status should show:
- `Ready: True`
- `ItemCount: 5` (number of pending orders)

Check the ConfigMap:

```bash
kubectl get configmap postgres-orders-data -o yaml
```

You should see order IDs: `1`, `2`, `3`, `4`, `5`

### 4. Apply the ListJob

```bash
kubectl apply -f listjob-etl.yaml
```

### 5. Watch Processing

```bash
# Watch jobs being created
kubectl get jobs -l example=postgres-etl -w

# Check job logs
kubectl logs -l job-name --prefix=true | grep "Processing order"
```

Each job will:
1. Fetch the order details from the database
2. Process/transform the data
3. Update the order status to "processed"

### 6. Verify Results

Check that orders were processed:

```bash
kubectl exec -it deployment/postgres -- psql -U postgres -d testdb -c "SELECT * FROM orders;"
```

All orders should now have `status = 'processed'`.

### 7. Clean Up

```bash
kubectl delete -f .
```

## Database Configuration

### Connection String Format

The PostgreSQL connection string uses libpq format:

```
postgresql://username:password@host:port/database?options
```

Example:
```
postgresql://user:<password>@postgres.default.svc.cluster.local:5432/mydb?sslmode=disable
```

### Using Secrets for Credentials

**IMPORTANT**: Never hardcode database credentials in YAML files for production.

Create a secret:

```bash
kubectl create secret generic db-credentials \
  --from-literal=username=myuser \
  --from-literal=password=mypassword
```

Reference it in the ListSource:

```yaml
spec:
  type: postgresql
  postgresql:
    host: postgres.default.svc.cluster.local
    port: 5432
    database: mydb
    usernameSecret:
      name: db-credentials
      key: username
    passwordSecret:
      name: db-credentials
      key: password
    query: "SELECT id FROM orders WHERE status = 'pending'"
    idColumn: id
```

### Connection Pooling

The ListSource controller uses connection pooling automatically:
- **Max connections**: 5 per ListSource
- **Max idle**: 2
- **Connection lifetime**: 5 minutes

For high-frequency polling, ensure your database can handle concurrent connections.

## Query Best Practices

### 1. Use Indexed Columns

Ensure your query uses indexed columns for performance:

```sql
CREATE INDEX idx_orders_status ON orders(status);
```

### 2. Limit Results

For large datasets, use `LIMIT`:

```yaml
query: "SELECT id FROM orders WHERE status = 'pending' LIMIT 1000"
```

### 3. Use Specific Columns

Only select the ID column you need:

```yaml
# Good: Only fetches IDs
query: "SELECT id FROM orders WHERE status = 'pending'"
idColumn: id

# Avoid: Fetches unnecessary data
query: "SELECT * FROM orders WHERE status = 'pending'"
```

### 4. Filter Efficiently

Use WHERE clauses to limit rows:

```yaml
# Good: Filters at database level
query: "SELECT id FROM orders WHERE status = 'pending' AND created_at > NOW() - INTERVAL '24 hours'"

# Avoid: Fetches all rows
query: "SELECT id FROM orders"
```

## Processing Patterns

### Pattern 1: ETL with Status Updates

```yaml
# Query for unprocessed items
query: "SELECT id FROM items WHERE processed = false"

# Job updates status after processing
command:
  - sh
  - -c
  - |
    # Process item
    echo "Processing item $ORDER_ID..."

    # Update status
    PGPASSWORD=$DB_PASSWORD psql -h postgres -U postgres -d testdb \
      -c "UPDATE items SET processed = true WHERE id = $ORDER_ID"
```

### Pattern 2: Batch Processing with Chunks

```yaml
# Process in chunks of 100
query: "SELECT id FROM large_table WHERE status = 'pending' LIMIT 100"
intervalSeconds: 300  # Process next batch every 5 minutes
```

### Pattern 3: Time-Based Processing

```yaml
# Process records from the last hour
query: |
  SELECT id FROM events
  WHERE processed = false
    AND created_at > NOW() - INTERVAL '1 hour'
```

## Error Handling

### Database Connection Failures

If the database is unreachable:

1. Check ListSource status:
   ```bash
   kubectl describe listsource postgres-orders
   ```

2. Verify database is running:
   ```bash
   kubectl get pods -l app=postgres
   ```

3. Test connectivity from a pod:
   ```bash
   kubectl run -it --rm debug --image=postgres:16-alpine --restart=Never -- \
     psql postgresql://testuser:testpass@postgres:5432/testdb -c "SELECT 1"
   ```

### Query Errors

If the query fails:

- Check SQL syntax in the status conditions
- Verify table and column names exist
- Ensure the user has SELECT permissions

### Job Processing Failures

If jobs fail during processing:

1. Check job logs:
   ```bash
   kubectl logs job/order-processor-0
   ```

2. Verify database permissions for updates:
   ```bash
   kubectl exec -it deployment/postgres -- psql -U postgres -d testdb \
     -c "GRANT UPDATE ON orders TO postgres;"
   ```

## Performance Tuning

### For Large Result Sets

```yaml
# Increase parallelism for faster processing
parallelism: 10

# Add resource limits
resources:
  requests:
    cpu: "500m"
    memory: "256Mi"
  limits:
    cpu: "1000m"
    memory: "512Mi"
```

### For Frequent Polling

```yaml
# Reduce polling frequency to lower database load
intervalSeconds: 300  # 5 minutes instead of 60 seconds
```

### For Slow Queries

1. **Add indexes** to queried columns
2. **Use EXPLAIN** to analyze query performance
3. **Consider materialized views** for complex queries

## Security Best Practices

1. **Use SSL/TLS** for database connections:
   ```yaml
   connectionString: "postgresql://user:<password>@host:5432/db?sslmode=require"
   ```

2. **Rotate credentials** regularly

3. **Use read-only users** when possible:
   ```sql
   CREATE USER readonly WITH PASSWORD 'secure_password';
   GRANT CONNECT ON DATABASE mydb TO readonly;
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly;
   ```

4. **Limit query results** to prevent memory issues:
   ```yaml
   query: "SELECT id FROM orders WHERE status = 'pending' LIMIT 10000"
   ```

## Next Steps

- Try [04-scheduled-batch](../04-scheduled-batch/) for scheduled database processing
- See [05-production-patterns](../05-production-patterns/) for production configurations
- Learn about [02-api-integration](../02-api-integration/) for combining API and database sources
