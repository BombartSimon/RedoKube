# Image URL to use all building/pushing image targets
REGISTRY ?= your-registry
IMG ?= $(REGISTRY)/openapi-operator:latest

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'.
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -v

##@ Build

.PHONY: build
build: ## Build the operator binary.
	go build -o bin/openapi-operator ./cmd/

.PHONY: run
run: ## Run the operator from your host.
	go run ./cmd/

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

.PHONY: install-crd
install-crd: ## Install the CRD into the K8s cluster.
	kubectl apply -f config/crd.yaml

.PHONY: uninstall-crd
uninstall-crd: ## Uninstall the CRD from the K8s cluster.
	kubectl delete -f config/crd.yaml

.PHONY: deploy
deploy: ## Deploy the operator to the K8s cluster.
	kubectl apply -f deploy/rbac.yaml
	sed -e 's|${REGISTRY}|$(REGISTRY)|g' -e 's|${TAG}|$(shell echo $(IMG) | cut -d: -f2)|g' deploy/deployment.yaml | kubectl apply -f -
	kubectl apply -f deploy/service.yaml

.PHONY: undeploy
undeploy: ## Undeploy the operator from the K8s cluster.
	kubectl delete -f deploy/service.yaml
	kubectl delete -f deploy/deployment.yaml
	kubectl delete -f deploy/rbac.yaml

.PHONY: example
example: ## Deploy example OpenAPI spec.
	kubectl apply -f examples/example-openapi-spec.yaml

.PHONY: clean-example
clean-example: ## Remove example OpenAPI spec.
	kubectl delete -f examples/example-openapi-spec.yaml