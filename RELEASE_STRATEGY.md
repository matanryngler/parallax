# 🚀 Parallax Release Strategy

This document outlines the comprehensive release strategy for Parallax, designed to provide enterprise-grade reliability and clear artifact management.

## 📋 Overview

Our release strategy follows industry best practices with:
- **Semantic Versioning** (SemVer) for all releases
- **Multi-platform container images** (linux/amd64, linux/arm64)
- **Signed containers** with Cosign for security
- **Comprehensive artifacts** including SBOM and vulnerability reports
- **All artifacts hosted on GitHub Container Registry (GHCR)**

## 🎯 Release Types

### 1. 🏷️ **Version Releases** (`v1.2.3`)

**Trigger**: Git tags matching `v*` pattern
**Artifacts**: Complete release package
**Workflow**: `.github/workflows/release.yml`

**What gets released:**
- ✅ Multi-platform container images (`ghcr.io/matanryngler/parallax:v1.2.3`)
- ✅ Helm charts (parallax and parallax-crds)
- ✅ SBOM (Software Bill of Materials)
- ✅ Signed container images with Cosign
- ✅ Comprehensive release notes
- ✅ Security scan results

**Container image tags created:**
```bash
ghcr.io/matanryngler/parallax:v1.2.3     # Exact version
ghcr.io/matanryngler/parallax:v1.2       # Minor version (latest patch)
ghcr.io/matanryngler/parallax:v1         # Major version (latest minor.patch)
ghcr.io/matanryngler/parallax:latest     # Only for stable releases (not pre-releases)
```

### 2. 📦 **Chart Releases** (`charts-v1.0.5`)

**Trigger**: Chart version changes in `charts/*/Chart.yaml`
**Artifacts**: Helm charts only
**Workflow**: `.github/workflows/chart-release.yml`

**What gets released:**
- ✅ Updated Helm charts
- ✅ Chart-specific release notes
- ✅ Installation validation

### 3. 🔄 **Development Builds** (`main`)

**Trigger**: Pushes to `main` branch
**Artifacts**: Development container images
**Workflow**: `.github/workflows/ci.yml`

**Container image tags created:**
```bash
ghcr.io/matanryngler/parallax:main        # Latest main branch
ghcr.io/matanryngler/parallax:latest      # Same as main for dev
ghcr.io/matanryngler/parallax:main-abc123 # Specific commit
```

## 🔒 Security & Compliance

### Container Image Security

Every released container image includes:

1. **🔐 Cosign Signatures**: All images are signed with keyless signing
2. **📋 SBOM**: Software Bill of Materials in SPDX format
3. **🛡️ Vulnerability Scanning**: Trivy security reports
4. **🏗️ Minimal Base**: Distroless images for reduced attack surface

### Verification Commands

```bash
# Verify image signature
cosign verify ghcr.io/matanryngler/parallax:v1.2.3 \
  --certificate-identity "https://github.com/matanryngler/parallax/.github/workflows/release.yml@refs/tags/v1.2.3" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"

# Download and inspect SBOM
curl -L https://github.com/matanryngler/parallax/releases/download/v1.2.3/sbom-v1.2.3.spdx.json
```

## 📝 Versioning Strategy

### Semantic Versioning (SemVer)

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (`v2.0.0`): Breaking changes, API incompatibility
- **MINOR** (`v1.1.0`): New features, backward compatible
- **PATCH** (`v1.0.1`): Bug fixes, backward compatible
- **PRE-RELEASE** (`v1.0.0-alpha.1`): Development versions

### Pre-release Versions

Pre-releases are marked appropriately:
- `v1.0.0-alpha.1` - Early development
- `v1.0.0-beta.1` - Feature complete, testing
- `v1.0.0-rc.1` - Release candidate

Pre-releases:
- ❌ Do NOT get the `latest` tag
- ✅ Are marked as "Pre-release" on GitHub
- ✅ Can be used for testing new features

### Chart Versioning

Helm charts use **independent versioning**:
- Chart versions are separate from app versions
- Charts can be updated without new app releases
- Both main chart and CRDs chart have separate versions

## 🚀 Release Process

### Creating a New Release

#### Option 1: Git Tag (Recommended)
```bash
# Create and push a tag
git tag v1.2.3
git push origin v1.2.3

# Release workflow automatically triggers
```

#### Option 2: Manual Dispatch
```bash
# Go to GitHub Actions -> Release Pipeline -> Run workflow
# Enter version: v1.2.3
```

### Release Checklist

Before creating a release:

- [ ] **Tests pass**: All CI checks are green
- [ ] **Version bumped**: Update version numbers if needed
- [ ] **Changelog**: Review what's changed since last release
- [ ] **Security**: Latest security scans are clean
- [ ] **Documentation**: README and docs are up to date

## 📊 Artifact Locations

All release artifacts are stored in standardized locations:

### Container Images
```
Registry: ghcr.io/matanryngler/parallax
Tags: latest, v1.2.3, v1.2, v1, main, main-<sha>
Platforms: linux/amd64, linux/arm64
```

### Helm Charts
```
Location: GitHub Releases
Files: parallax-<version>.tgz, parallax-crds-<version>.tgz
Installation: helm install parallax <chart-url>
```

### Security Artifacts
```
SBOM: sbom-<version>.spdx.json
Signatures: Stored in Sigstore transparency log
Vulnerability Reports: GitHub Security tab
```

## 🎛️ Installation Methods

### 1. Latest Stable Release
```bash
# Helm (recommended - requires both charts)
helm install parallax-crds https://github.com/matanryngler/parallax/releases/latest/download/parallax-crds-0.1.0.tgz
helm install parallax https://github.com/matanryngler/parallax/releases/latest/download/parallax-0.1.0.tgz

# Direct container
docker pull ghcr.io/matanryngler/parallax:latest
```

### 2. Specific Version
```bash
# Specific Helm chart version (install CRDs first)
helm install parallax-crds https://github.com/matanryngler/parallax/releases/download/v1.2.3/parallax-crds-1.0.5.tgz
helm install parallax https://github.com/matanryngler/parallax/releases/download/v1.2.3/parallax-1.0.5.tgz

# Specific container version
docker pull ghcr.io/matanryngler/parallax:v1.2.3
```

### 3. Development Version
```bash
# Latest development build
docker pull ghcr.io/matanryngler/parallax:main
```

## 🔄 Automated Workflows

### Release Workflow (`./.github/workflows/release.yml`)

**Triggers**: Git tags `v*`, manual dispatch
**Jobs**:
1. **Validate Release** - Check version format and availability
2. **Build & Push** - Multi-platform container images  
3. **Sign Images** - Cosign signatures
4. **Package Charts** - Helm chart packaging
5. **Security Scan** - Vulnerability assessment
6. **Create Release** - GitHub release with artifacts

### Chart Release Workflow (`./.github/workflows/chart-release.yml`)

**Triggers**: Chart version changes, manual dispatch
**Jobs**:
1. **Detect Changes** - Identify which charts changed
2. **Validate Versions** - Ensure versions are new
3. **Test Charts** - Install charts in Kind cluster
4. **Package Charts** - Create chart packages
5. **Create Releases** - Separate releases for each chart

### CI Workflow (`./.github/workflows/ci.yml`)

**Triggers**: All pushes and PRs
**Jobs**:
1. **Test & Lint** - Code quality and tests
2. **Security Scan** - Container vulnerability scanning
3. **E2E Tests** - End-to-end validation
4. **Build Dev** - Development images (main branch only)

## 📈 Migration from Old Strategy

The new release strategy provides:

✅ **Clear separation** between development builds and stable releases
✅ **Comprehensive security** with signed images and SBOM
✅ **Multi-platform support** for broader deployment options
✅ **Independent chart releases** for faster chart updates
✅ **Enterprise compliance** with proper artifact management

### Breaking Changes
- Chart releases now have separate tags (`charts-parallax-v1.0.5`)
- Development builds only get `main` and `latest` tags
- All container images moved to GHCR exclusively

## 🆘 Troubleshooting

### Common Issues

**Q: Release failed with "version already exists"**
A: Check if the version tag already exists in releases. Use a new version number.

**Q: Container image not found**
A: Ensure you're using the correct registry: `ghcr.io/matanryngler/parallax`

**Q: Helm chart installation fails**
A: Check if CRDs are installed. Use separate CRDs chart if needed.

### Getting Help

- 🐛 [Create an issue](https://github.com/matanryngler/parallax/issues) for release problems
- 💬 [Start a discussion](https://github.com/matanryngler/parallax/discussions) for questions
- 📖 Check the [troubleshooting guide](https://github.com/matanryngler/parallax/wiki/Troubleshooting)

---

This release strategy ensures Parallax delivers enterprise-grade reliability with clear, secure, and automated artifact management.