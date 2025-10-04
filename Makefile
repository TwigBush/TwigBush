SHELL := /bin/bash
GO    ?= go

# Ports and storage
AS_ADDR ?= :8089
UI_ADDR ?= :8088
STORE   ?= fs


.PHONY: all tidy deps fmt test build run clean ci \
        build-twigbush

# Default target
all: build

tidy:
	$(GO) mod tidy

deps: tidy
	$(GO) mod download

fmt:
	$(GO) fmt ./...
	gofmt -w -s .

test:
	$(GO) test ./...

build:
	$(GO) build ./...

clean:
	$(GO) clean -modcache
	rm -rf bin/

ci: tidy fmt test build
	@echo "CI checks passed"

build-twigbush:
	$(GO) build -o bin/as ./cmd/as
