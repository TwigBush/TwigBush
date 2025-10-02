# Makefile for TwigBush
SHELL := /bin/bash
GO    ?= go

# Ports and storage
AS_ADDR ?= :8089
UI_ADDR ?= :8088
STORE   ?= fs
DATA_DIR ?= $(HOME)/.twigbush/data

.PHONY: all tidy deps fmt test build run clean ci \
        build-as build-playground build-dev \
        run-as run-playground run-dev run-dev-fs run-dev-mem \
        run-dev-both-fs run-dev-both-mem run-as-fs run-playground-fs help

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

# Existing run targets
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

# ---------- New: explicit builds ----------
build-as:
	$(GO) build -o bin/as ./cmd/as

build-playground:
	$(GO) build -o bin/playground ./cmd/playground

build-dev:
	$(GO) build -o bin/dev ./cmd/dev

# ---------- New: prod-like single servers ----------
run-as:
	$(GO) run ./cmd/as

run-playground:
	$(GO) run ./cmd/playground

# File store variants (respect DATA_DIR)
run-as-fs:
	TWIGBUSH_DATA_DIR="$(DATA_DIR)" $(GO) run ./cmd/as

run-playground-fs:
	TWIGBUSH_DATA_DIR="$(DATA_DIR)" $(GO) run ./cmd/playground

# ---------- New: dev runner that serves both ports ----------
# Uses cmd/dev which starts both servers in one process
run-dev:
	TWIGBUSH_ENV=local TWIGBUSH_DATA_DIR="$(DATA_DIR)" $(GO) run ./cmd/dev -store=$(STORE) -as=$(AS_ADDR) -ui=$(UI_ADDR)

# Shorthand for common modes
run-dev-fs:
	TWIGBUSH_ENV=local TWIGBUSH_DATA_DIR="$(DATA_DIR)" $(GO) run ./cmd/dev -store=fs -as=$(AS_ADDR) -ui=$(UI_ADDR)

# Explicit both with file or mem store
run-dev-both-fs:
	TWIGBUSH_ENV=local TWIGBUSH_DATA_DIR="$(DATA_DIR)" $(GO) run ./cmd/dev -store=fs -as=$(AS_ADDR) -ui=$(UI_ADDR)

run-dev-both-mem:
	$(GO) run ./cmd/dev -store=mem -as=$(AS_ADDR) -ui=$(UI_ADDR)

# ---------- New: help ----------
help:
	@echo "Targets:"
	@echo "  run-dev-fs           Run dev runner on $(AS_ADDR) and $(UI_ADDR) using file store"
	@echo "  run-as-fs            Run AS only with file store at DATA_DIR=$(DATA_DIR)"
	@echo "  run-playground-fs    Run Playground only with file store at DATA_DIR=$(DATA_DIR)"
	@echo "Vars: AS_ADDR=$(AS_ADDR) UI_ADDR=$(UI_ADDR) STORE=$(STORE) DATA_DIR=$(DATA_DIR)"
