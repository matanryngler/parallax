# Parallax Examples

This directory contains real-world examples demonstrating how to use Parallax for parallel batch processing in Kubernetes.

## Quick Start

Choose an example based on your use case:

| Example | Use Case | Complexity |
|---------|----------|------------|
| [01-basic-static-list](./01-basic-static-list/) | Process a fixed list of items in parallel | Beginner |
| [02-api-integration](./02-api-integration/) | Fetch data from REST API and process items | Intermediate |
| [03-postgres-etl](./03-postgres-etl/) | Query PostgreSQL and process records | Intermediate |
| [04-scheduled-batch](./04-scheduled-batch/) | Schedule recurring batch jobs with cron | Intermediate |
| [05-production-patterns](./05-production-patterns/) | Production-ready configurations | Advanced |

## Prerequisites

Before running these examples, ensure you have:

1. **Kubernetes cluster** with kubectl configured
2. **Parallax operator installed**:
   ```bash
   # From the repository root
   make deploy
   ```
3. **Permissions** to create Jobs, ConfigMaps, and custom resources

## Example Structure

Each example directory contains:

- `README.md` - Detailed explanation of the use case
- `*.yaml` - Kubernetes manifests ready to apply
- `expected-output.md` - What you should see when running the example

## Running an Example

```bash
# Navigate to an example directory
cd 01-basic-static-list

# Apply all manifests
kubectl apply -f .

# Watch resources being created
kubectl get listsource,listjob,jobs,pods -w

# Check logs from a job pod
kubectl logs job/<job-name>

# Clean up
kubectl delete -f .
```

## Learning Path

We recommend following the examples in order:

1. **Start with 01-basic-static-list** to understand core concepts
2. **Try 02-api-integration** to see dynamic data sources
3. **Explore 03-postgres-etl** for database integration
4. **Use 04-scheduled-batch** to learn scheduling
5. **Study 05-production-patterns** before deploying to production

## Troubleshooting

If you encounter issues:

1. Check that the Parallax operator is running:
   ```bash
   kubectl get pods -n parallax-system
   ```

2. View operator logs:
   ```bash
   kubectl logs -n parallax-system deployment/parallax -f
   ```

3. Describe resources for detailed status:
   ```bash
   kubectl describe listsource <name>
   kubectl describe listjob <name>
   ```

For more help, see the [Troubleshooting Guide](../docs/troubleshooting.md).

## Contributing Examples

Have a useful example? Contributions are welcome! Please ensure your example:

- Includes clear documentation
- Uses realistic scenarios
- Can be run with minimal setup
- Includes cleanup instructions
