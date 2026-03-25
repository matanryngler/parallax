#!/usr/bin/env bash

# Orchestration script for Parallax Integration Tests
# This script sets up a pristine Kind cluster, builds and loads images,
# installs Helm charts, and runs integration tests.

set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME="parallax-integration"
NAMESPACE="parallax-system"

# Cleanup function to be called on exit
cleanup() {
  echo "🧹 Cleaning up..."
  if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "🗑️ Deleting Kind cluster: ${CLUSTER_NAME}"
    kind delete cluster --name "${CLUSTER_NAME}"
  fi
}

# Trap EXIT to ensure cleanup happens even on failure
trap cleanup EXIT

echo "🚀 Starting Integration Test Orchestration"

# 1. Build Operator image
echo "📦 Building Operator image..."
docker build -t parallax:integration .

# 2. Build Mock API image
echo "📦 Building Mock API image..."
docker build -t parallax-test-api:integration test/local/testdata/api-server/

# 3. Create Kind cluster
echo "🔧 Creating Kind cluster: ${CLUSTER_NAME}"
kind create cluster --name "${CLUSTER_NAME}" --wait 5m

# 4. Load docker images into Kind
echo "🚚 Loading images into Kind..."
kind load docker-image parallax:integration --name "${CLUSTER_NAME}"
kind load docker-image parallax-test-api:integration --name "${CLUSTER_NAME}"

# 5. Create namespace
echo "🏗️ Creating namespace: ${NAMESPACE}"
kubectl create namespace "${NAMESPACE}"

# 6. Install CRDs
echo "📜 Installing Parallax CRDs..."
helm install parallax-crds ./charts/parallax-crds

# 7. Install Parallax Operator
echo "🚀 Installing Parallax Operator..."
helm install parallax ./charts/parallax \
  --namespace "${NAMESPACE}" \
  --set image.repository=parallax \
  --set image.tag=integration \
  --wait

# 8. Create ConfigMap for Postgres Init
echo "💾 Creating postgres-init ConfigMap..."
kubectl create configmap postgres-init --from-file=01-init.sql=test/local/testdata/postgres/init.sql

# 9. Install Test Infrastructure
echo "🧪 Installing Test Infrastructure..."
helm install test-infra ./charts/test-infra \
  --set mockApi.image.repository=parallax-test-api \
  --set mockApi.image.tag=integration \
  --wait

# 10. Run Integration Tests
echo "🧪 Running Integration Tests..."
if [ -d "./test/integration" ]; then
  go test -v ./test/integration/...
else
  echo "⚠️ Warning: ./test/integration directory not found. Skipping test execution."
  echo "Once tests are implemented, they will be executed here."
fi

echo "✅ Integration Test Orchestration completed successfully!"
