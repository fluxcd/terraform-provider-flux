.ONESHELL:
SHELL := /bin/bash

TEST_COUNT?=1
ACCTEST_PARALLELISM?=5
ACCTEST_TIMEOUT?=10m
EMBEDDED_MANIFESTS_TARGET=.manifests.done
FLUX_VERSION:=$(shell grep 'DefaultFluxVersion' internal/utils/flux.go | awk '{ print $$5 }' | tr -d '"')

rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$(d)/,$(2)) $(filter $(subst *,%,$(2)),$(d)))

all: test testacc build

$(EMBEDDED_MANIFESTS_TARGET): $(call rwildcard,manifests/,*.yaml)
	echo "Downloading manifests for Flux $(FLUX_VERSION)"
	rm -rf manifests && mkdir -p manifests
	curl -sLO https://github.com/fluxcd/flux2/releases/download/$(FLUX_VERSION)/manifests.tar.gz
	tar xzf manifests.tar.gz -C manifests
	rm -rf manifests.tar.gz
	touch $@

.PHONY: manifests
manifests: $(EMBEDDED_MANIFESTS_TARGET)

tidy:
	rm -f go.sum; go mod tidy -compat=1.22

fmt:
	go fmt ./...

vet: $(EMBEDDED_MANIFESTS_TARGET)
	go vet ./...

test: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	go test ./... -coverprofile cover.out

testacc: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	TF_ACC=1 go test ./... -v -count $(TEST_COUNT) -parallel $(ACCTEST_PARALLELISM) -timeout $(ACCTEST_TIMEOUT) -coverprofile cover.out

# Run acceptance tests on macOS with the gitea-flux instance
# Requires the following entry in /etc/hosts:
# 127.0.0.1 gitea-flux
testmacos: $(EMBEDDED_MANIFESTS_TARGET) tidy fmt vet
	TF_ACC=1 GITEA_HOSTNAME=gitea-flux go test ./... -v -parallel 1 -run TestAccBootstrapGit_Drift

build: $(EMBEDDED_MANIFESTS_TARGET)
	CGO_ENABLED=0 go build -o ./bin/terraform-provider-flux main.go

.PHONY: docs
docs: $(EMBEDDED_MANIFESTS_TARGET) tools
	tfplugindocs generate --ignore-deprecated true

tools:
	GO111MODULE=on go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

.SILENT:
lint:
	tflint --recursive --disable-rule=terraform_required_providers --disable-rule terraform_required_version --disable-rule=terraform_unused_declarations

terraformrc:
	cat .terraformrc.tmpl | envsubst > .terraformrc
