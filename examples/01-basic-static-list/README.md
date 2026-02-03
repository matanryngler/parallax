# Example 01: Basic Static List

This example demonstrates the simplest use case for Parallax: processing a fixed list of items in parallel.

## Use Case

You have a predefined list of items (e.g., customer IDs, file names, configuration names) that need to be processed independently. Instead of processing them sequentially, Parallax creates parallel Kubernetes Jobs to process multiple items simultaneously.

## Real-World Scenarios

- Batch processing customer reports
- Running health checks on multiple services
- Processing a list of files from a manifest
- Sending notifications to specific users
- Running cleanup tasks on specific resources

## Files in This Example

- `listsource.yaml` - Defines a static list of items to process
- `listjob.yaml` - Creates parallel jobs to process each item
- `expected-output.md` - Shows what to expect when running

## Step-by-Step Guide

### 1. Apply the ListSource

The ListSource defines a static list of three items:

```bash
kubectl apply -f listsource.yaml
```

Check that the ConfigMap was created:

```bash
kubectl get configmap basic-list-data -o yaml
```

You should see a ConfigMap containing:
```
ITEM_0: alice
ITEM_1: bob
ITEM_2: charlie
```

### 2. Apply the ListJob

The ListJob creates parallel jobs that process each item:

```bash
kubectl apply -f listjob.yaml
```

### 3. Watch Jobs Being Created

```bash
kubectl get jobs -w
```

You should see three jobs created (one for each item):
- `basic-processor-0`
- `basic-processor-1`
- `basic-processor-2`

### 4. Check Job Logs

View the output from each job:

```bash
# Check logs from the first job
kubectl logs job/basic-processor-0

# Or check all jobs at once
kubectl logs -l job-name --prefix=true
```

Each job will print:
```
Processing item: <item-name>
Item processed successfully!
```

### 5. Check Status

View the ListJob status:

```bash
kubectl get listjob basic-processor -o yaml
```

The status should show:
- `conditions` indicating success or failure
- `active`, `succeeded`, and `failed` job counts

### 6. Clean Up

Delete all resources:

```bash
kubectl delete -f .
```

## Key Concepts

### ListSource
- **Type**: `static` - uses a hardcoded list
- **Items**: Array of strings to process
- **Output**: Creates a ConfigMap with items as environment variables

### ListJob
- **ListSourceRef**: References the ListSource by name
- **Parallelism**: How many jobs run simultaneously (2 in this example)
- **CompletionMode**: Uses `Indexed` mode for parallel processing
- **JobTemplate**: Defines the container that processes each item

### Environment Variables

Each job pod receives:
- `ITEM` - The item to process (from `$(ITEM_INDEX)`)
- `JOB_COMPLETION_INDEX` - The index of this job (0, 1, 2, etc.)

## Customization

### Change the List

Edit `listsource.yaml` to add or remove items:

```yaml
spec:
  static:
    items:
      - alice
      - bob
      - charlie
      - david  # Add more items
```

### Adjust Parallelism

Edit `listjob.yaml` to control how many jobs run at once:

```yaml
spec:
  parallelism: 5  # Run up to 5 jobs simultaneously
```

### Customize Processing

Replace the container image and command in `listjob.yaml`:

```yaml
containers:
  - name: processor
    image: your-image:tag
    command: ["your-command"]
    env:
      - name: ITEM
        value: "$(ITEM_INDEX)"
```

## Next Steps

- Try [02-api-integration](../02-api-integration/) to learn about dynamic data sources
- Learn about [04-scheduled-batch](../04-scheduled-batch/) for recurring jobs
- See [05-production-patterns](../05-production-patterns/) for production configurations
