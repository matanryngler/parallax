#!/bin/bash
set -e

# Bump Chart Version Helper Script
# Usage: ./scripts/bump-chart-version.sh [patch|minor|major] [chart-name]

BUMP_TYPE=${1:-patch}
CHART_NAME=${2:-both}

function bump_version() {
    local current_version=$1
    local bump_type=$2
    
    # Parse version (assuming semantic versioning)
    if [[ $current_version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-.*)?$ ]]; then
        major=${BASH_REMATCH[1]}
        minor=${BASH_REMATCH[2]}
        patch=${BASH_REMATCH[3]}
        suffix=${BASH_REMATCH[4]}
    else
        echo "❌ Invalid version format: $current_version"
        exit 1
    fi
    
    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "❌ Invalid bump type: $bump_type"
            echo "Valid options: patch, minor, major"
            exit 1
            ;;
    esac
    
    echo "${major}.${minor}.${patch}${suffix}"
}

function update_chart() {
    local chart_path=$1
    local chart_name=$2
    
    if [[ ! -f "$chart_path/Chart.yaml" ]]; then
        echo "❌ Chart not found: $chart_path"
        exit 1
    fi
    
    current_version=$(grep '^version:' "$chart_path/Chart.yaml" | cut -d' ' -f2)
    new_version=$(bump_version "$current_version" "$BUMP_TYPE")
    
    echo "📦 Updating $chart_name chart: $current_version → $new_version"
    
    # Update version in Chart.yaml
    sed -i.bak "s/^version: .*/version: $new_version/" "$chart_path/Chart.yaml"
    rm "$chart_path/Chart.yaml.bak"
    
    echo "✅ Updated $chart_path/Chart.yaml"
}

echo "🚀 Bumping chart version(s) - type: $BUMP_TYPE"
echo ""

case $CHART_NAME in
    parallax)
        update_chart "charts/parallax" "parallax"
        ;;
    parallax-crds)
        update_chart "charts/parallax-crds" "parallax-crds"
        ;;
    both)
        update_chart "charts/parallax" "parallax"
        update_chart "charts/parallax-crds" "parallax-crds"
        ;;
    *)
        echo "❌ Invalid chart name: $CHART_NAME"
        echo "Valid options: parallax, parallax-crds, both"
        exit 1
        ;;
esac

echo ""
echo "🔄 Running sync to update charts with latest manifests..."
make sync-all

echo ""
echo "✅ Chart version bump complete!"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Commit the changes: git add . && git commit -m 'chore: bump chart version to include X'"
echo "3. Push to trigger chart release: git push"
echo ""
echo "📋 Release Policy:"
echo "• CI will create a release only when Chart.yaml versions are bumped"
echo "• Chart content changes without version bumps will NOT trigger releases"
echo "• This prevents overriding existing versions with different content"
echo "• Each version number can only be used once" 