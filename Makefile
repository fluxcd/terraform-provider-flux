TEST_COUNT?=1
ACCTEST_PARALLELISM?=4
ACCTEST_TIMEOUT?=10m

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

build:
	CGO_ENABLED=0 go build -o ./bin/flux main.go

.PHONY: docs
docs:
	tfplugindocs generate
