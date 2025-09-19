# Makefile for TwigBush
SHELL := /bin/bash
GO    ?= go

.PHONY: all tidy deps fmt test build run clean ci

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

run:
	$(GO) run ./cmd/as

build-demo:
	$(GO) build -o bin/demo ./cmd/demo

run-demo:
	$(GO) run ./cmd/demo

clean:
	$(GO) clean -modcache
	rm -rf bin/

ci: tidy fmt test build
	@echo "CI checks passed"

