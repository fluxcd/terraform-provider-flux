.ONESHELL:
SHELL := /bin/bash

TEST_COUNT?=1
ACCTEST_PARALLELISM?=5
ACCTEST_TIMEOUT?=10m

all: test testacc build

tidy:
	rm -f go.sum; go mod tidy -compat=1.20

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ./... -coverprofile cover.out

testacc: tidy fmt vet
	TF_ACC=1 go test ./... -v -count $(TEST_COUNT) -parallel $(ACCTEST_PARALLELISM) -timeout $(ACCTEST_TIMEOUT)

build:
	CGO_ENABLED=0 go build -o ./bin/terraform-provider-flux main.go

.PHONY: docs
docs: tools
	tfplugindocs generate

tools:
	GO111MODULE=on go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

.SILENT:
lint:
	tflint --recursive --disable-rule=terraform_required_providers --disable-rule terraform_required_version --disable-rule=terraform_unused_declarations

terraformrc:
	cat .terraformrc.tmpl | envsubst > .terraformrc
