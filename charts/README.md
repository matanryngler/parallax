# Parallax Helm Charts

This directory contains Helm charts for installing the Parallax operator on Kubernetes.

## Charts

### 📦 `parallax` - Operator Installation

The main chart that installs the Parallax operator including:
- RBAC permissions
- Controller deployment
- Service account

**Note**: This chart does NOT include CRDs. You must install the `parallax-crds` chart first.

**Quick Start:**
```bash
# Step 1: Install CRDs first
helm install parallax-crds https://github.com/matanryngler/parallax/releases/download/v0.1.0/parallax-crds-0.1.0.tgz

# Step 2: Install operator
helm install parallax https://github.com/matanryngler/parallax/releases/download/v0.1.0/parallax-0.1.0.tgz

# Or install from source
helm install parallax-crds ./charts/parallax-crds
helm install parallax ./charts/parallax
```

### 🔧 `parallax-crds` - CRDs Only

A standalone chart that only installs the Custom Resource Definitions. Useful for:
- Cluster administrators who want to install CRDs separately
- Multi-tenant environments where CRDs are managed centrally
- Upgrade scenarios where you want to update CRDs independently

**Usage:**
```bash
# Install CRDs first
helm install parallax-crds ./charts/parallax-crds

# Then install operator
helm install parallax ./charts/parallax
```

## Installation Options

### Option 1: Local Charts (Recommended)
```bash
# Step 1: Install CRDs
helm install parallax-crds ./charts/parallax-crds

# Step 2: Install operator
helm install parallax ./charts/parallax
```

### Option 2: GitHub Releases
```bash
# Step 1: Install CRDs
helm install parallax-crds https://github.com/matanryngler/parallax/releases/download/v0.1.0/parallax-crds-0.1.0.tgz

# Step 2: Install operator
helm install parallax https://github.com/matanryngler/parallax/releases/download/v0.1.0/parallax-0.1.0.tgz
```


## Configuration

The main `parallax` chart supports extensive configuration. See [values.yaml](parallax/values.yaml) for all options.

**Common configurations:**

```yaml
# Resource limits
resources:
  limits:
    cpu: 1000m
    memory: 256Mi
  requests:
    cpu: 200m
    memory: 128Mi

# Operator configuration
operator:
  logLevel: debug
  leaderElection: true
```

## Upgrading

### Upgrading the Full Installation
```bash
helm upgrade parallax ./charts/parallax
```

### Upgrading with Separate CRDs
```bash
# Upgrade CRDs first (if needed)
helm upgrade parallax-crds ./charts/parallax-crds

# Then upgrade operator
helm upgrade parallax ./charts/parallax
```

## Uninstalling

### Full Installation
```bash
helm uninstall parallax
```

### Separate CRDs Installation
```bash
# Remove operator first
helm uninstall parallax

# Then remove CRDs (this will delete all custom resources!)
helm uninstall parallax-crds
```

⚠️ **Warning:** Uninstalling the CRDs will delete all ListSource, ListJob, and ListCronJob resources in your cluster!

## Development

### Auto-sync from Generated Manifests

The charts are automatically synchronized from the operator's generated manifests:

```bash
# Sync all manifests to charts
make sync-all

# Sync only CRDs
make sync-crds

# Sync only RBAC
make sync-rbac

# Check if sync is needed (used by CI)
make check-sync
```

This ensures the charts always reflect the latest operator permissions and CRD definitions.

### Releasing New Chart Versions

Chart versions are managed manually and tracked in git:

```bash
# Bump patch version for both charts
make bump-chart-version

# Bump minor version for both charts  
make bump-chart-version BUMP=minor

# Bump only the main chart
make bump-chart-version CHART=parallax

# Or use the script directly
./scripts/bump-chart-version.sh patch both
```

The CI will automatically detect chart version bumps and create releases when you push version updates.

**Important**: The CI will only create releases when chart versions are actually bumped in Chart.yaml. If you modify chart content without bumping the version, no release will be created to prevent overriding existing versions.

### Testing Charts Locally

```bash
# Lint charts
helm lint charts/parallax
helm lint charts/parallax-crds

# Test templating
helm template test charts/parallax --dry-run
helm template test charts/parallax-crds --dry-run

# Test without CRDs
helm template test charts/parallax --set installCRDs=false --dry-run
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Parallax Deployment                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │ parallax-crds   │    │         parallax               │ │
│  │                 │    │                                 │ │
│  │ • ListSource    │    │ • Controller Deployment        │ │
│  │ • ListJob       │    │ • RBAC (ClusterRole/Binding)   │ │
│  │ • ListCronJob   │    │ • ServiceAccount                │ │
│  │                 │    │                                 │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Support

- 📖 [Main Documentation](../../README.md)
- 🐛 [Issues](https://github.com/matanryngler/parallax/issues)
- 💬 [Discussions](https://github.com/matanryngler/parallax/discussions) 