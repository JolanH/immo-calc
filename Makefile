# Every dev operation runs inside Docker. Host requirements: Docker and make.
SHELL := /bin/bash
.DEFAULT_GOAL := help

UID := $(shell id -u)
GID := $(shell id -g)

# Our own Go packages. Not ./..., which would also walk web/node_modules.
GO_PKGS := ./cmd/... ./internal/... ./web
GO_IMAGE := immo-calc-go
NODE_IMAGE := immo-calc-node

DOCKER_RUN := docker run --rm -u $(UID):$(GID) -v "$(CURDIR)":/src -w /src \
	-e HOME=/src/.cache/home -e GOCACHE=/src/.cache/go-build -e GOMODCACHE=/src/.cache/gomod
# Interactive targets share the host network so the app can keep binding
# 127.0.0.1 only (a published port can't reach a container's loopback).
DOCKER_RUN_HOST := $(DOCKER_RUN) --init --network host
GO_RUN := $(DOCKER_RUN) $(GO_IMAGE)
NODE_RUN := $(DOCKER_RUN) -w /src/web $(NODE_IMAGE)

.PHONY: help images test test-go test-web lint lint-go lint-web run build

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

images: ## Build the Docker images
	@docker build -q -t $(GO_IMAGE) -f docker/go.Dockerfile docker >/dev/null
	@docker build -q -t $(NODE_IMAGE) -f docker/node.Dockerfile docker >/dev/null

.cache:
	@mkdir -p .cache/home

# Reinstall node_modules whenever the lockfile changes.
web/node_modules/.installed: web/package.json web/package-lock.json | images .cache
	$(NODE_RUN) npm ci --no-audit --no-fund
	@touch $@

# The Go packages embed web/dist, so it must be built before compiling them.
WEB_SOURCES := $(shell find web/src web/public -type f 2>/dev/null) web/index.html web/vite.config.ts
web/dist/index.html: web/node_modules/.installed $(WEB_SOURCES)
	$(NODE_RUN) npx vite build --logLevel warn

test: test-go test-web ## Run all tests

test-go: web/dist/index.html ## Backend tests only
	$(GO_RUN) go test -race -cover $(GO_PKGS)

test-web: web/node_modules/.installed ## Frontend tests only
	$(NODE_RUN) npx vitest run

lint: lint-go lint-web ## Run all linters

lint-go: web/dist/index.html
	@out="$$($(GO_RUN) gofmt -l cmd internal web/embed.go)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(GO_RUN) go vet $(GO_PKGS)

lint-web: web/node_modules/.installed
	$(NODE_RUN) npx eslint .
	$(NODE_RUN) npx prettier --check .
	$(NODE_RUN) npx vue-tsc --noEmit

run: web/dist/index.html ## Build and run the app → http://localhost:8421
	$(DOCKER_RUN_HOST) $(GO_IMAGE) go run ./cmd/immo-calc

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GO_BUILD := env CGO_ENABLED=0 GOARCH=amd64 go build -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

build: web/dist/index.html ## Cross-compile Linux and Windows binaries into dist/
	$(GO_RUN) $(GO_BUILD) -ldflags "$(LDFLAGS)" -o dist/immo-calc-linux-amd64 ./cmd/immo-calc
	$(GO_RUN) env GOOS=windows $(GO_BUILD) -ldflags "$(LDFLAGS) -H windowsgui" \
		-o dist/immo-calc-windows-amd64.exe ./cmd/immo-calc
