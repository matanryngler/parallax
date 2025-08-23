#!/bin/bash
set -euo pipefail

# E2E Functionality Test Script
# This script runs comprehensive functionality tests for Parallax operator
# It assumes a Kind cluster is already set up and KUBECONFIG is configured

KUBECONFIG_FILE="/tmp/parallax-e2e-test-kubeconfig"
TEST_NAMESPACE="parallax-test"
OPERATOR_IMAGE="parallax:e2e-test"

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

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v kubectl >/dev/null 2>&1; then
        error "kubectl is required but not installed"
        exit 1
    fi
    
    if ! command -v kind >/dev/null 2>&1; then
        error "kind is required but not installed"
        exit 1
    fi
    
    if [[ ! -f "$KUBECONFIG_FILE" ]]; then
        error "KUBECONFIG file not found: $KUBECONFIG_FILE"
        error "Please run 'make test-e2e-setup' first"
        exit 1
    fi
    
    export KUBECONFIG="$KUBECONFIG_FILE"
    success "Prerequisites check passed"
}

# Build and load operator image
build_and_load_operator() {
    log "Building and loading operator image..."
    
    log "Building operator image with local caching: $OPERATOR_IMAGE"
    
    # Use Docker buildx for better local caching
    if command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1; then
        log "Using Docker buildx with local caching..."
        docker buildx build \
            --platform linux/amd64 \
            --load \
            --tag "$OPERATOR_IMAGE" \
            --cache-from type=local,src=/tmp/.buildx-cache \
            --cache-to type=local,dest=/tmp/.buildx-cache-new,mode=max \
            --build-arg VERSION=e2e-test \
            --build-arg COMMIT=$(git rev-parse HEAD) \
            --build-arg DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
            .
        
        # Move cache to avoid growing cache indefinitely
        rm -rf /tmp/.buildx-cache
        mv /tmp/.buildx-cache-new /tmp/.buildx-cache 2>/dev/null || true
    else
        log "Docker buildx not available, falling back to regular build..."
        make docker-build IMG="$OPERATOR_IMAGE"
    fi
    
    log "Loading image into Kind cluster..."
    kind load docker-image "$OPERATOR_IMAGE" --name parallax-e2e-test
    
    success "Operator image built and loaded"
}

# Deploy operator
deploy_operator() {
    log "Deploying Parallax operator..."
    
    log "Installing CRDs..."
    make install
    
    log "Deploying operator with image: $OPERATOR_IMAGE"
    make deploy IMG="$OPERATOR_IMAGE"
    
    log "Waiting for operator to be ready..."
    kubectl wait --for=condition=Available deployment/parallax-controller-manager \
        -n parallax-system --timeout=300s
    
    success "Operator deployed and ready"
}

# Create test namespace
setup_test_namespace() {
    log "Setting up test namespace: $TEST_NAMESPACE"
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    success "Test namespace ready"
}

# Test ListSource functionality
test_listsource() {
    log "Testing ListSource functionality..."
    
    # Create static ListSource
    log "Creating static ListSource..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListSource
metadata:
  name: test-fruits
  namespace: $TEST_NAMESPACE
spec:
  type: static
  staticList:
    - apple
    - banana
    - orange
  intervalSeconds: 60
EOF

    # Wait for ListSource to be processed
    log "Waiting for ListSource to be processed..."
    sleep 15
    
    # Check if ConfigMap was created
    if kubectl get configmap test-fruits -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "ListSource ConfigMap created successfully"
        
        # Verify content
        ITEMS=$(kubectl get configmap test-fruits -n "$TEST_NAMESPACE" -o jsonpath='{.data.items}')
        if echo "$ITEMS" | grep -q "apple" && echo "$ITEMS" | grep -q "banana" && echo "$ITEMS" | grep -q "orange"; then
            success "ListSource items correctly stored in ConfigMap"
        else
            error "ListSource items not correctly stored in ConfigMap"
            echo "Found items: $ITEMS"
            return 1
        fi
    else
        error "ListSource ConfigMap not created"
        kubectl get events -n "$TEST_NAMESPACE" --sort-by='.lastTimestamp' | tail -10
        return 1
    fi
    
    # Check ListSource status
    ITEM_COUNT=$(kubectl get listsource test-fruits -n "$TEST_NAMESPACE" -o jsonpath='{.status.itemCount}' 2>/dev/null || echo "0")
    if [[ "$ITEM_COUNT" == "3" ]]; then
        success "ListSource status correctly shows 3 items"
    else
        warn "ListSource status shows $ITEM_COUNT items, expected 3"
    fi
}

# Test ListJob functionality
test_listjob() {
    log "Testing ListJob functionality..."
    
    # Test with static list
    log "Creating ListJob with static list..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: test-processor
  namespace: $TEST_NAMESPACE
spec:
  staticList:
    - item1
    - item2
    - item3
  parallelism: 2
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Processing: \$ITEM && sleep 5"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
EOF

    # Wait for Job to be created
    log "Waiting for Kubernetes Job to be created..."
    sleep 20
    
    # Check if Job was created
    if kubectl get job -l listjob=test-processor -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "ListJob created Kubernetes Job successfully"
        
        # Verify Job configuration
        PARALLELISM=$(kubectl get job -l listjob=test-processor -n "$TEST_NAMESPACE" -o jsonpath='{.items[0].spec.parallelism}')
        COMPLETIONS=$(kubectl get job -l listjob=test-processor -n "$TEST_NAMESPACE" -o jsonpath='{.items[0].spec.completions}')
        
        if [[ "$PARALLELISM" == "2" ]]; then
            success "Job parallelism correctly set to 2"
        else
            warn "Job parallelism is $PARALLELISM, expected 2"
        fi
        
        if [[ "$COMPLETIONS" == "3" ]]; then
            success "Job completions correctly set to 3 (number of items)"
        else
            warn "Job completions is $COMPLETIONS, expected 3"
        fi
    else
        error "ListJob did not create Kubernetes Job"
        kubectl get events -n "$TEST_NAMESPACE" --sort-by='.lastTimestamp' | tail -10
        return 1
    fi
    
    # Test ListJob with ListSource reference
    log "Creating ListJob with ListSource reference..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: test-source-processor
  namespace: $TEST_NAMESPACE
spec:
  listSourceRef: test-fruits
  parallelism: 1
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Processing from source: \$ITEM && sleep 3"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
EOF

    sleep 15
    
    if kubectl get job -l listjob=test-source-processor -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        COMPLETIONS=$(kubectl get job -l listjob=test-source-processor -n "$TEST_NAMESPACE" -o jsonpath='{.items[0].spec.completions}')
        if [[ "$COMPLETIONS" == "3" ]]; then
            success "ListJob with ListSource reference created successfully (3 completions from ListSource)"
        else
            warn "ListJob with ListSource reference shows $COMPLETIONS completions, expected 3"
        fi
    else
        error "ListJob with ListSource reference failed to create Job"
        return 1
    fi
}

# Test ListCronJob functionality
test_listcronjob() {
    log "Testing ListCronJob functionality..."
    
    log "Creating ListCronJob..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListCronJob
metadata:
  name: test-scheduled
  namespace: $TEST_NAMESPACE
spec:
  schedule: "0 */1 * * *"  # Every hour
  staticList:
    - scheduled-item1
    - scheduled-item2
  parallelism: 1
  template:
    image: busybox:latest
    command: ["/bin/sh", "-c", "echo Scheduled: \$ITEM"]
    envName: ITEM
    resources:
      requests:
        cpu: "50m"
        memory: "32Mi"
  successfulJobsHistoryLimit: 2
  failedJobsHistoryLimit: 1
EOF

    # Wait for CronJob to be created
    log "Waiting for Kubernetes CronJob to be created..."
    sleep 15
    
    if kubectl get cronjob -l listcronjob=test-scheduled -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "ListCronJob created Kubernetes CronJob successfully"
        
        # Verify CronJob configuration
        SCHEDULE=$(kubectl get cronjob -l listcronjob=test-scheduled -n "$TEST_NAMESPACE" -o jsonpath='{.items[0].spec.schedule}')
        if [[ "$SCHEDULE" == "0 */1 * * *" ]]; then
            success "CronJob schedule correctly set"
        else
            warn "CronJob schedule is '$SCHEDULE', expected '0 */1 * * *'"
        fi
        
        SUCCESS_LIMIT=$(kubectl get cronjob -l listcronjob=test-scheduled -n "$TEST_NAMESPACE" -o jsonpath='{.items[0].spec.successfulJobsHistoryLimit}')
        if [[ "$SUCCESS_LIMIT" == "2" ]]; then
            success "CronJob success history limit correctly set"
        else
            warn "CronJob success history limit is $SUCCESS_LIMIT, expected 2"
        fi
    else
        error "ListCronJob did not create Kubernetes CronJob"
        kubectl get events -n "$TEST_NAMESPACE" --sort-by='.lastTimestamp' | tail -10
        return 1
    fi
}

# Test error handling
test_error_handling() {
    log "Testing error handling..."
    
    log "Creating ListJob with non-existent ListSource reference..."
    kubectl apply -f - <<EOF
apiVersion: batchops.io/v1alpha1
kind: ListJob
metadata:
  name: test-invalid-ref
  namespace: $TEST_NAMESPACE
spec:
  listSourceRef: non-existent-source
  parallelism: 1
  template:
    image: busybox:latest
    command: ["echo", "test"]
    envName: ITEM
EOF

    sleep 10
    
    # Should not create a Job when ListSource doesn't exist
    if ! kubectl get job -l listjob=test-invalid-ref -n "$TEST_NAMESPACE" >/dev/null 2>&1; then
        success "Error handling: No Job created for invalid ListSource reference"
    else
        warn "Error handling: Job was created despite invalid ListSource reference"
    fi
}

# Cleanup test resources
cleanup_test_resources() {
    log "Cleaning up test resources..."
    kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true --wait=false
    success "Test resources cleanup initiated"
}

# Collect debug information
collect_debug_info() {
    error "Test failed. Collecting debug information..."
    
    echo "=== Operator Logs ==="
    kubectl logs -l control-plane=controller-manager -n parallax-system --tail=50 || true
    
    echo "=== Test Namespace Events ==="
    kubectl get events -n "$TEST_NAMESPACE" --sort-by='.lastTimestamp' || true
    
    echo "=== Test Resources ==="
    kubectl get all,listsources,listjobs,listcronjobs -n "$TEST_NAMESPACE" -o wide || true
}

# Main execution
main() {
    log "Starting Parallax E2E functionality tests..."
    
    # Set up error handling
    trap 'collect_debug_info; cleanup_test_resources; exit 1' ERR
    
    check_prerequisites
    build_and_load_operator
    deploy_operator
    setup_test_namespace
    
    log "Running functionality tests..."
    test_listsource
    test_listjob
    test_listcronjob
    test_error_handling
    
    success "All E2E functionality tests passed!"
    cleanup_test_resources
}

# Run main function
main "$@"