# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./... -coverprofile cover.out

.PHONY: lint
lint: ## Run linting (go vet + go fmt check).
	@echo "🔍 Running linting checks..."
	@go vet ./...
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Code is not properly formatted. Run 'make fmt'"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "✅ Linting passed"

##@ CI/CD Local Testing

.PHONY: ci-test
ci-test: ## Run unit tests with coverage (matches CI).
	@echo "🧪 Running unit tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./internal/controller/ ./api/... ./cmd/...
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "📊 Test coverage: $${coverage}%"; \
	if [ "$$(echo "$${coverage} < 5" | bc -l)" -eq 1 ]; then \
		echo "❌ Test coverage is below 5%"; \
		exit 1; \
	fi; \
	echo "✅ Test coverage is acceptable (≥5%)"

.PHONY: ci-lint
ci-lint: ## Run linting checks (matches CI).
	@echo "🔍 Running CI linting checks..."
	@$(MAKE) lint

.PHONY: ci-security
ci-security: ## Run security scanning (matches CI).
	@echo "🔒 Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
		echo "Skipping security scan..."; \
	fi

.PHONY: ci-validate
ci-validate: ## Validate Kubernetes manifests (matches CI).
	@echo "📋 Validating Kubernetes manifests..."
	@if command -v helm >/dev/null 2>&1; then \
		echo "  • Validating Helm chart..."; \
		helm lint charts/parallax; \
		echo "  • Rendering Helm templates (offline)..."; \
		helm template test charts/parallax --dry-run >/dev/null 2>&1; \
		echo "✅ Helm validation passed"; \
	else \
		echo "⚠️  Helm not installed. Skipping Helm validation..."; \
	fi
	@if command -v kubectl >/dev/null 2>&1; then \
		echo "  • Validating CRDs..."; \
		KUBECONFIG=/dev/null kubectl --dry-run=client --validate=true apply -f config/crd/bases/ >/dev/null 2>&1; \
		echo "✅ CRD validation passed"; \
	else \
		echo "⚠️  kubectl not installed. Skipping CRD validation..."; \
	fi

.PHONY: ci-all
ci-all: ci-test ci-lint ci-security ci-validate ## Run all CI checks locally.
	@echo ""
	@echo "🎉 All CI checks passed! Your code is ready for PR."

.PHONY: ci-quick
ci-quick: ci-test ci-lint ## Run quick CI checks (test + lint only).
	@echo ""
	@echo "⚡ Quick CI checks passed!"

##@ E2E Testing

# E2E test cluster naming convention
E2E_CLUSTER_NAME ?= parallax-e2e-test

.PHONY: test-e2e
test-e2e: ## Run E2E tests (creates isolated Kind cluster).
	@echo "🚀 Running E2E tests with isolated cluster..."
	@$(MAKE) test-e2e-setup
	@trap '$(MAKE) test-e2e-cleanup' EXIT; \
	E2E_CLUSTER_NAME=$(E2E_CLUSTER_NAME) \
	KIND_CLUSTER=$(E2E_CLUSTER_NAME) \
	CERT_MANAGER_INSTALL_SKIP=true \
	KUBECONFIG="/tmp/$(E2E_CLUSTER_NAME)-kubeconfig" \
	go test ./test/e2e/ -timeout=30m -v || (echo "❌ E2E tests failed"; exit 1)
	@echo "✅ E2E tests completed successfully"

.PHONY: test-e2e-setup
test-e2e-setup: ## Set up isolated Kind cluster for E2E testing.
	@echo "📦 Setting up isolated E2E test cluster: $(E2E_CLUSTER_NAME)"
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "❌ Kind not installed. Install with: go install sigs.k8s.io/kind@latest"; \
		exit 1; \
	fi
	@if kind get clusters 2>/dev/null | grep -q "^$(E2E_CLUSTER_NAME)$$"; then \
		echo "♻️  Deleting existing test cluster: $(E2E_CLUSTER_NAME)"; \
		kind delete cluster --name $(E2E_CLUSTER_NAME); \
	fi
	@echo "🔧 Creating fresh test cluster: $(E2E_CLUSTER_NAME)"
	@kind create cluster --name $(E2E_CLUSTER_NAME) --wait 60s
	@echo "🎯 Setting KUBECONFIG for test cluster"
	@kind get kubeconfig --name $(E2E_CLUSTER_NAME) > /tmp/$(E2E_CLUSTER_NAME)-kubeconfig
	@echo "✅ Test cluster ready: $(E2E_CLUSTER_NAME)"

.PHONY: test-e2e-cleanup
test-e2e-cleanup: ## Clean up E2E test cluster.
	@echo "🧹 Cleaning up E2E test cluster: $(E2E_CLUSTER_NAME)"
	@if command -v kind >/dev/null 2>&1 && kind get clusters 2>/dev/null | grep -q "^$(E2E_CLUSTER_NAME)$$"; then \
		kind delete cluster --name $(E2E_CLUSTER_NAME); \
		echo "✅ Test cluster deleted: $(E2E_CLUSTER_NAME)"; \
	else \
		echo "ℹ️  No test cluster to clean up: $(E2E_CLUSTER_NAME)"; \
	fi
	@rm -f /tmp/$(E2E_CLUSTER_NAME)-kubeconfig
	@echo "✅ E2E cleanup complete"

.PHONY: test-e2e-connect
test-e2e-connect: ## Connect to the E2E test cluster (for debugging).
	@if ! kind get clusters 2>/dev/null | grep -q "^$(E2E_CLUSTER_NAME)$$"; then \
		echo "❌ Test cluster $(E2E_CLUSTER_NAME) not found. Run 'make test-e2e-setup' first."; \
		exit 1; \
	fi
	@echo "🔗 To connect to the test cluster, run:"
	@echo "export KUBECONFIG=$$(kind get kubeconfig --name $(E2E_CLUSTER_NAME))"
	@echo ""
	@echo "Or run commands with the test cluster:"
	@echo "kubectl --kubeconfig=$$(kind get kubeconfig --name $(E2E_CLUSTER_NAME)) get nodes"

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/dev-best-practices/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have a multi-arch builder. More info: https://docs.docker.com/build/building/multi-platform/
# - be able to push the image to your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
CONTROLLER_TOOLS_VERSION ?= v0.16.4
ENVTEST_VERSION ?= release-0.19

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - GOOS
# $4 - GOARCH

define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ; \
echo "Downloading $${package}" ; \
GOBIN=$(LOCALBIN) go install $${package} ; \
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ; \
}
endef
