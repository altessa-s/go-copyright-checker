SHELL = bash

PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

APP_VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
APP_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

BIN_OUTPUT_DIR ?= ./bin

.PHONY: all
all: help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n\033[36m\033[0m"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Remove temporary files
	@rm -rf ${BIN_OUTPUT_DIR}

.PHONY: fmt
fmt: tidy  ## Run go fmt on all go files
	@go install github.com/daixiang0/gci@latest
	@gci write \
		-s standard \
		-s default \
		-s "prefix(github.com/altessa-s/go-copyright-checker)" \
		-s blank -s alias \
	 $$(go list -f {{.Dir}} ./...) \

.PHONY: lint
lint: tidy fmt ## Run linter
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	@go mod tidy

.PHONY: test
test: ## Run tests
	@go test -race -v ./...

.PHONY: build
build: clean tidy ## Build project
	@mkdir -p ${BIN_OUTPUT_DIR}
	@echo "Building ${APP_VERSION} (${APP_COMMIT})..."
	@go build \
		-ldflags "-s -w -X main.Version=${APP_VERSION} -X main.Commit=${APP_COMMIT}" \
		-o ${BIN_OUTPUT_DIR}/go-copyright-checker ${PROJECT_ROOT}

.PHONY: install
install: ## Install to GOPATH/bin
	@go install ./...
