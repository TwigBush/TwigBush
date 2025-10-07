SHELL := /bin/bash
GO    ?= go

# Detect version from git tags, fallback to "dev"
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || \
                   git describe --tags --always --dirty 2>/dev/null || \
                   echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION ?= $(shell go version | awk '{print $$3}')

# Build flags to inject version info
LDFLAGS := -ldflags "\
    -X github.com/TwigBush/gnap-go/internal/version.Version=$(VERSION) \
    -X github.com/TwigBush/gnap-go/internal/version.GitCommit=$(GIT_COMMIT) \
    -X github.com/TwigBush/gnap-go/internal/version.BuildDate=$(BUILD_DATE) \
    -X github.com/TwigBush/gnap-go/internal/version.GoVersion=$(GO_VERSION)"

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: version
version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

.PHONY: build
build: ## Build all binaries
	@echo "Building version $(VERSION)..."
	@mkdir -p dist
	go build $(LDFLAGS) -o dist/twigbush ./cmd/twigbush
	go build $(LDFLAGS) -o dist/as ./cmd/as
	go build $(LDFLAGS) -o dist/playground ./cmd/playground
	@echo "✓ Build complete"

.PHONY: build-all
build-all: ## Build binaries for all platforms
	@echo "Building version $(VERSION) for all platforms..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/twigbush-$(VERSION)-linux-amd64 ./cmd/twigbush
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/twigbush-$(VERSION)-linux-arm64 ./cmd/twigbush
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/twigbush-$(VERSION)-darwin-amd64 ./cmd/twigbush
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/twigbush-$(VERSION)-darwin-arm64 ./cmd/twigbush
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/twigbush-$(VERSION)-windows-amd64.exe ./cmd/twigbush
	@echo "✓ Cross-platform build complete"

.PHONY: install
install: ## Install binaries to $GOPATH/bin
	go install $(LDFLAGS) ./cmd/twigbush
	go install $(LDFLAGS) ./cmd/as
	go install $(LDFLAGS) ./cmd/dev

.PHONY: lint
lint: ## Run linter (requires golangci-lint)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: clean
clean: ## Clean build artifacts
	$(GO) clean -modcache
	rm -rf dist/
	rm -f coverage.out coverage.html

.PHONY: release-check
release-check: ## Verify repository is ready for release
	@echo "Checking release readiness..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "✗ Error: Working directory is not clean"; \
		git status --short; \
		exit 1; \
	fi
	@echo "✓ Working directory is clean"
	@if ! git describe --tags --exact-match HEAD 2>/dev/null; then \
		echo "✗ Error: HEAD is not tagged"; \
		exit 1; \
	fi
	@echo "✓ HEAD is tagged: $$(git describe --tags --exact-match HEAD)"
	@echo "✓ Ready for release"

.PHONY: tag
tag: ## Create a new version tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory is not clean"; \
		exit 1; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "✓ Created tag $(VERSION)"
	@echo "Push with: git push origin $(VERSION)"


.PHONEY: test
test:
	$(GO) test ./...




