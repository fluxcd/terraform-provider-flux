TEST_COUNT?=1
ACCTEST_PARALLELISM?=4
ACCTEST_TIMEOUT?=10m
NAME=terraform-provider-flux
PLUGIN_PATH=$(HOME)/.terraform.d/plugins

all: test testacc build

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ./... -coverprofile cover.out

testacc: tidy fmt vet
	TF_ACC=1 go test ./pkg/provider -v -count $(TEST_COUNT) -parallel $(ACCTEST_PARALLELISM) -timeout $(ACCTEST_TIMEOUT)

install: build
	install -d $(PLUGIN_PATH)
	install -m 775 ./bin/$(NAME) $(PLUGIN_PATH)/

build:
	CGO_ENABLED=0 go build -o ./bin/$(NAME) main.go

.PHONY: docs
docs:
	tfplugindocs generate
