PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
GOBINARY := $(shell which go)

all: build

all: getdeps test

getdeps:
	@echo "Installing golangci-lint" && go get github.com/golangci/golangci-lint/cmd/golangci-lint && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	@echo "Installing zbusc" && go install github.com/threefoldtech/zbus/zbusc
	go mod tidy

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golangci-lint run -c ../.golangci.yml

check: test
test: lint build
	sudo -E ${GOBINARY} test -vet=off -v $(shell go list ./... | grep -Ev "stubs|network" )

testrace: lint build
	sudo -E ${GOBINARY} test -vet=off -v $(shell go list ./... | grep -Ev "stubs|network" )

generate:
	@echo "Generating modules client stubs"
	go generate github.com/threefoldtech/zosbase/pkg

build:
	@CGO_ENABLED=0 go build -v ./...
