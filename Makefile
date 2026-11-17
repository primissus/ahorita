.DEFAULT_GOAL := help

BINARY := ahorita

.PHONY: help build build-web install test vet clean

help: ## Show available make targets
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the ahorita CLI binary
	go build -o $(BINARY) ./cmd/ahorita

build-web: ## Build the WASM module and web UI
	cd web && npm run build

install: ## Install the ahorita CLI binary via go install
	go install ./cmd/ahorita

test: ## Run Go and web tests
	go test ./...
	cd web && npm test

vet: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -f $(BINARY) web/public/ahorita.wasm web/public/wasm_exec.js
	rm -rf web/dist
