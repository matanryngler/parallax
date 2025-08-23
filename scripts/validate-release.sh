#!/bin/bash

# 🔒 Parallax Release Validation Script
# Run this script before creating a release to validate version availability

set -euo pipefail

VERSION=${1:-}
if [[ -z "$VERSION" ]]; then
    echo "❌ Usage: $0 <version>"
    echo "   Example: $0 v0.1.0"
    exit 1
fi

# Remove 'v' prefix if present for consistency
VERSION_NUMBER=${VERSION#v}
VERSION_TAG="v${VERSION_NUMBER}"

echo "🔍 Validating release version: $VERSION_TAG"
echo "==============================================="

# Check GitHub release
echo "1️⃣ Checking GitHub release..."
if gh release view "$VERSION_TAG" >/dev/null 2>&1; then
    echo "❌ GitHub release $VERSION_TAG already exists!"
    exit 1
else
    echo "✅ GitHub release $VERSION_TAG is available"
fi

# Check container image
echo ""
echo "2️⃣ Checking container image..."
IMAGE_REPO="ghcr.io/matanryngler/parallax"
if docker manifest inspect "${IMAGE_REPO}:${VERSION_TAG}" >/dev/null 2>&1; then
    echo "❌ Container image ${IMAGE_REPO}:${VERSION_TAG} already exists!"
    exit 1
else
    echo "✅ Container image ${IMAGE_REPO}:${VERSION_TAG} is available"
fi

# Check chart versions
echo ""
echo "3️⃣ Checking Helm chart versions..."

# Get current chart versions
PARALLAX_CHART_VERSION=$(grep '^version:' charts/parallax/Chart.yaml | cut -d' ' -f2)
CRDS_CHART_VERSION=$(grep '^version:' charts/parallax-crds/Chart.yaml | cut -d' ' -f2)

echo "Current chart versions:"
echo "  parallax: $PARALLAX_CHART_VERSION"
echo "  parallax-crds: $CRDS_CHART_VERSION"

# Check if chart releases exist
if gh release view "charts-parallax-v${PARALLAX_CHART_VERSION}" >/dev/null 2>&1; then
    echo "❌ Parallax chart version v${PARALLAX_CHART_VERSION} already exists!"
    exit 1
else
    echo "✅ Parallax chart version v${PARALLAX_CHART_VERSION} is available"
fi

if gh release view "charts-crds-v${CRDS_CHART_VERSION}" >/dev/null 2>&1; then
    echo "❌ Parallax CRDs chart version v${CRDS_CHART_VERSION} already exists!"
    exit 1
else
    echo "✅ Parallax CRDs chart version v${CRDS_CHART_VERSION} is available"
fi

# Validate version format
echo ""
echo "4️⃣ Validating version format..."
if [[ ! "$VERSION_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+(.[0-9]+)?)?$ ]]; then
    echo "❌ Invalid version format: $VERSION_TAG"
    echo "Expected format: v1.2.3 or v1.2.3-alpha.1"
    exit 1
else
    echo "✅ Version format is valid: $VERSION_TAG"
fi

# Check if we're on the right branch
echo ""
echo "5️⃣ Checking current branch..."
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo "⚠️  Warning: You're on branch '$CURRENT_BRANCH', not 'main'"
    echo "   Releases are typically created from 'main' branch"
else
    echo "✅ On main branch"
fi

# Check if working directory is clean
echo ""
echo "6️⃣ Checking working directory..."
if [[ -n $(git status --porcelain) ]]; then
    echo "⚠️  Warning: Working directory has uncommitted changes"
    echo "   Consider committing or stashing changes before release"
else
    echo "✅ Working directory is clean"
fi

echo ""
echo "🎉 All validations passed!"
echo ""
echo "📋 Release Summary:"
echo "   Version: $VERSION_TAG"
echo "   Container: ${IMAGE_REPO}:${VERSION_TAG}"
echo "   Charts: parallax-${PARALLAX_CHART_VERSION}, parallax-crds-${CRDS_CHART_VERSION}"
echo ""
echo "🚀 Ready to create release:"
echo "   git tag $VERSION_TAG && git push origin $VERSION_TAG"