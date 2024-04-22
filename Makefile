# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
CONTROLLER_GEN = go run ${PROJECT_DIR}/vendor/sigs.k8s.io/controller-tools/cmd/controller-gen
ENVTEST = go run ${PROJECT_DIR}/vendor/sigs.k8s.io/controller-runtime/tools/setup-envtest
GINKGO = go run ${PROJECT_DIR}/vendor/github.com/onsi/ginkgo/v2/ginkgo
GOLANGCI_LINT = go run ${PROJECT_DIR}/vendor/github.com/golangci/golangci-lint/cmd/golangci-lint

VERSION     ?= $(shell git describe --always --abbrev=7)
MUTABLE_TAG ?= latest
IMAGE       ?= cluster-control-plane-machine-set-operator
BUILD_IMAGE ?= registry.ci.openshift.org/openshift/release:golang-1.21


.PHONY: all
all: check build test

NO_DOCKER ?= 0

ifeq ($(shell command -v podman > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=podman
else ifeq ($(shell command -v docker > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=docker
else
	NO_DOCKER=1
endif

USE_DOCKER ?= 0
ifeq ($(USE_DOCKER), 1)
	ENGINE=docker
endif

ifeq ($(NO_DOCKER), 1)
	DOCKER_CMD =
	IMAGE_BUILD_CMD = imagebuilder
else
	DOCKER_CMD := $(ENGINE) run --env GO111MODULE=$(GO111MODULE) --env GOFLAGS=$(GOFLAGS) --rm -v "$(PWD)":/go/src/github.com/openshift-splat-team/vsphere-capacity-manager:Z  -w /go/src/github.com/openshift-splat-team/vsphere-capacity-manager $(BUILD_IMAGE)
	# The command below is for building/testing with the actual image that Openshift uses. Uncomment/comment out to use instead of above command. CI registry pull secret is required to use this image.
	# DOCKER_CMD := $(ENGINE) run --env GO111MODULE=$(GO111MODULE) --env GOFLAGS=$(GOFLAGS) --rm -v "$(PWD)":/go/src/github.com/openshift-splat-team/vsphere-capacity-manager:Z -w /go/src/github.com/openshift-splat-team/vsphere-capacity-manager registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.19-openshift-4.11
	IMAGE_BUILD_CMD = $(ENGINE) build
endif

.PHONY: build
build: capacity-manager ## Build binaries

.PHONY: capacity-manager
capacity-manager:
	$(DOCKER_CMD) ./hack/build.sh 

.PHONY: test
test:
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(PROJECT_DIR)/bin)" ./hack/test.sh	

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) paths=./... crd rbac:roleName=lease-perms output:crd:artifacts:config=config/crd/bases
	go generate ./...
	
# Use podman to build the image.
.PHONY: image
image:
	hack/build-image
