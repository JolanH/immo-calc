# Every dev operation runs inside Docker. Host requirements: Docker and make.
SHELL := /bin/bash
.DEFAULT_GOAL := help

UID := $(shell id -u)
GID := $(shell id -g)

GO_IMAGE := immo-calc-go
NODE_IMAGE := immo-calc-node

DOCKER_RUN := docker run --rm -u $(UID):$(GID) -v "$(CURDIR)":/src -w /src \
	-e HOME=/src/.cache/home -e GOCACHE=/src/.cache/go-build -e GOMODCACHE=/src/.cache/gomod
GO_RUN := $(DOCKER_RUN) $(GO_IMAGE)
NODE_RUN := $(DOCKER_RUN) -w /src/web $(NODE_IMAGE)

.PHONY: help images test test-go test-web lint lint-go lint-web

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

test: test-go test-web ## Run all tests

test-go: images .cache ## Backend tests only
	$(GO_RUN) go test -race -cover ./...

test-web: web/node_modules/.installed ## Frontend tests only
	$(NODE_RUN) npx vitest run

lint: lint-go lint-web ## Run all linters

lint-go: images .cache
	@out="$$($(GO_RUN) gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(GO_RUN) go vet ./...

lint-web: web/node_modules/.installed
	$(NODE_RUN) npx eslint .
	$(NODE_RUN) npx prettier --check .
	$(NODE_RUN) npx vue-tsc --noEmit
