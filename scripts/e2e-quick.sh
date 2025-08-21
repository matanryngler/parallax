#!/bin/bash
set -euo pipefail

# Quick E2E Test Script
# This script runs E2E tests against an existing cluster
# Perfect for rapid development iteration

KUBECONFIG_FILE="${KUBECONFIG:-$HOME/.kube/config}"
TEST_NAMESPACE="parallax-test-quick"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

# Check if cluster is available
check_cluster() {
    log "Checking cluster connectivity..."
    
    if ! kubectl cluster-info >/dev/null 2>&1; then
        error "Cannot connect to Kubernetes cluster"
        error "Please ensure:"
        error "  1. A cluster is running (try: kind create cluster)"
        error "  2. KUBECONFIG is set correctly"
        error "  3. kubectl is working (try: kubectl get nodes)"
        exit 1
    fi
    
    CLUSTER_NAME=$(kubectl config current-context)
    success "Connected to cluster: $CLUSTER_NAME"
}

# Check if operator is running
check_operator() {
    log "Checking if Parallax operator is running..."
    
    if kubectl get deployment parallax-controller-manager -n parallax-system >/dev/null 2>&1; then
        if kubectl get deployment parallax-controller-manager -n parallax-system -o jsonpath='{.status.readyReplicas}' | grep -q "1"; then
            success "Parallax operator is running"
        else
            warn "Parallax operator deployment exists but may not be ready"
            kubectl get deployment parallax-controller-manager -n parallax-system
        fi
    else
        error "Parallax operator not found"
        error "Please deploy the operator first:"
        error "  make deploy IMG=parallax:dev"
        error "Or run full E2E tests:"
        error "  make test-e2e-functionality"
        exit 1
    fi
}

# Quick functionality test
quick_test() {
    log "Running quick functionality test..."
    
    # Create test namespace
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Test 1: Quick ListSource test
    log "Testing ListSource..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: quick-test
  namespace: $TEST_NAMESPACE
spec:
  type: static
  staticList:
    - test1
    - test2
  intervalSeconds: 30
EOF

    # Wait briefly and check
    sleep 8
    if kubectl get configmap quick-test-items -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "ListSource working - ConfigMap created"
    else
        error "ListSource not working - ConfigMap not created"
        return 1
    fi
    
    # Test 2: Quick ListJob test
    log "Testing ListJob..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: quick-job
  namespace: $TEST_NAMESPACE
spec:
  staticList:
    - quicktest
  parallelism: 1
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Quick test: \$ITEM"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
EOF

    sleep 10
    if kubectl get job -l listjob=quick-job -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "ListJob working - Kubernetes Job created"
    else
        error "ListJob not working - Kubernetes Job not created"
        return 1
    fi
    
    success "Quick functionality test passed!"
}

# Cleanup
cleanup() {
    log "Cleaning up quick test resources..."
    kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true --wait=false
    success "Cleanup complete"
}

# Main execution
main() {
    log "🚀 Running Parallax quick E2E test..."
    
    trap 'error "Quick test failed!"; cleanup; exit 1' ERR
    
    check_cluster
    check_operator
    quick_test
    cleanup
    
    success "🎉 Quick E2E test completed successfully!"
    log ""
    log "💡 For comprehensive testing, run: make test-e2e-functionality"
}

# Help function
show_help() {
    echo ""
    echo "Parallax Quick E2E Test"
    echo "======================="
    echo ""
    echo "This script runs quick E2E tests against an existing cluster."
    echo "Perfect for rapid development iteration."
    echo ""
    echo "Prerequisites:"
    echo "  • Kubernetes cluster running (kind, minikube, etc.)"
    echo "  • Parallax operator deployed"
    echo "  • kubectl configured"
    echo ""
    echo "Usage:"
    echo "  $0              Run quick tests"
    echo "  $0 --help       Show this help"
    echo ""
    echo "Examples:"
    echo "  # Start kind cluster and deploy operator"
    echo "  kind create cluster"
    echo "  make deploy IMG=parallax:dev"
    echo ""
    echo "  # Run quick tests"
    echo "  $0"
    echo ""
    echo "  # For comprehensive tests"
    echo "  make test-e2e-functionality"
    echo ""
}

# Parse arguments
case "${1:-}" in
    --help|-h)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac