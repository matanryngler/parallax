# 🧪 E2E Testing Guide for Contributors

This guide explains how to run End-to-End (E2E) tests locally for Parallax development.

## 🚀 Quick Start

### Option 1: Full E2E Tests (Recommended)
Sets up everything automatically, runs comprehensive tests, and cleans up:
```bash
make test-e2e
```
**Perfect for:** Final testing before PR submission

### Option 2: Quick E2E Tests (Development)
Runs against your existing cluster for rapid iteration:
```bash
# Deploy operator to your cluster first
make deploy IMG=parallax:dev

# Run quick tests
make test-e2e-quick
```
**Perfect for:** Fast development feedback loop

## 📋 All Available E2E Commands

```bash
# Comprehensive tests (full setup + cleanup)
make test-e2e                    # Run all E2E tests
make test-e2e-functionality      # Run functionality tests only

# Quick tests (existing cluster)
make test-e2e-quick             # Quick functionality test
make test-e2e-golden            # Manifest validation tests

# Manual cluster management
make test-e2e-setup             # Create test cluster
make test-e2e-cleanup           # Delete test cluster
make test-e2e-connect           # Get connection info

# Legacy tests
make test-e2e-basic             # Basic operator deployment tests
```

## 💡 Development Workflow

### For New Features
1. **Write code** for your feature
2. **Quick test** during development:
   ```bash
   # Start development cluster
   kind create cluster
   make deploy IMG=parallax:dev
   
   # Rapid testing
   make test-e2e-quick
   ```
3. **Full test** before PR:
   ```bash
   make test-e2e
   ```

### For Bug Fixes
1. **Reproduce the bug** with quick tests
2. **Fix the code**
3. **Verify fix** with quick tests
4. **Final validation** with full E2E tests

## 🔧 What Each Test Does

### Full E2E Tests (`make test-e2e`)
- ✅ Creates isolated Kind cluster
- ✅ Builds and loads operator image
- ✅ Deploys operator with CRDs and RBAC
- ✅ Tests ListSource functionality (static lists, ConfigMaps)
- ✅ Tests ListJob functionality (static + ListSource references)
- ✅ Tests ListCronJob functionality (scheduling, CronJobs)
- ✅ Tests error handling (invalid references)
- ✅ Validates manifest generation (golden files)
- ✅ Cleans up everything automatically

### Quick E2E Tests (`make test-e2e-quick`)
- ⚡ Uses existing cluster (no setup/teardown)
- ⚡ Quick smoke tests for core functionality
- ⚡ Perfect for development iteration
- ⚡ Takes ~30 seconds vs 5-10 minutes

## 🛠️ Prerequisites

### Required Tools
```bash
# Install required tools
go install sigs.k8s.io/kind@latest          # Kind for local clusters
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"  # kubectl
```

### For Quick Tests Only
You need an existing Kubernetes cluster:
```bash
# Option 1: Kind cluster
kind create cluster

# Option 2: Minikube  
minikube start

# Option 3: Docker Desktop K8s
# Enable Kubernetes in Docker Desktop settings

# Deploy operator
make deploy IMG=parallax:dev
```

## 🐛 Troubleshooting

### Tests Fail with "cannot connect to cluster"
```bash
# Check cluster is running
kubectl cluster-info

# For Kind clusters
kind get clusters

# Recreate cluster if needed
kind delete cluster && kind create cluster
```

### Tests Fail with "operator not found" (quick tests)
```bash
# Deploy operator first
make deploy IMG=parallax:dev

# Check operator is running
kubectl get pods -n parallax-system
```

### Tests Hang or Timeout
```bash
# Check for resource issues
kubectl top nodes
kubectl get events --sort-by='.lastTimestamp' -A

# Clean up and retry
make test-e2e-cleanup
make test-e2e
```

### Image Build Issues
```bash
# Clean Docker cache
docker system prune -f

# Rebuild with verbose output
make docker-build IMG=parallax:dev VERBOSE=1
```

## 📊 Understanding Test Output

### Successful Run
```
🚀 Running comprehensive E2E tests with isolated cluster...
📦 Setting up isolated E2E test cluster: parallax-e2e-test
✅ Test cluster ready: parallax-e2e-test
✅ Operator image built and loaded
✅ Operator deployed and ready
✅ ListSource working - ConfigMap created
✅ ListJob working - Kubernetes Job created
✅ ListCronJob working - Kubernetes CronJob created
✅ All E2E functionality tests passed!
```

### Failed Run
Tests show exactly what failed with debug information:
```
❌ ListJob not working - Kubernetes Job not created
=== Operator Logs ===
[operator logs here]
=== Test Namespace Events ===
[recent events here]
```

## 🎯 Best Practices

### For Contributors
- **Always run `make test-e2e-quick`** during development
- **Run `make test-e2e`** before submitting PRs
- **Check both unit and E2E tests**: `make ci-quick && make test-e2e-quick`

### For Maintainers
- **Full E2E on main branch** ensures production readiness
- **Quick E2E on PRs** provides fast feedback
- **Golden file tests** catch manifest drift

## 🔍 Advanced Usage

### Debug Failed Tests
```bash
# Connect to test cluster for debugging
make test-e2e-setup
make test-e2e-connect

# Manual investigation
kubectl get all -n parallax-test
kubectl logs -n parallax-system -l control-plane=controller-manager

# Cleanup when done
make test-e2e-cleanup
```

### Custom Test Scenarios
```bash
# Run with custom namespace
TEST_NAMESPACE=my-test ./scripts/e2e-functionality.sh

# Run with custom image
OPERATOR_IMAGE=parallax:my-feature ./scripts/e2e-functionality.sh
```

### CI/CD Integration
E2E tests automatically run in GitHub Actions:
- **On PRs**: Quick validation (skips docs-only changes)
- **On main**: Comprehensive testing
- **On releases**: Full validation + upgrade tests

## 📚 Further Reading

- [Kubernetes Testing Guide](https://kubernetes.io/docs/reference/using-api/api-concepts/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)

---

**Questions?** Open an issue or ask in discussions! 🤝