# Every dev operation runs inside Docker. Host requirements: Docker and make.
SHELL := /bin/bash
.DEFAULT_GOAL := help

UID := $(shell id -u)
GID := $(shell id -g)

GO_IMAGE := immo-calc-go

DOCKER_RUN := docker run --rm -u $(UID):$(GID) -v "$(CURDIR)":/src -w /src \
	-e HOME=/src/.cache/home -e GOCACHE=/src/.cache/go-build -e GOMODCACHE=/src/.cache/gomod
GO_RUN := $(DOCKER_RUN) $(GO_IMAGE)

.PHONY: help images test test-go lint lint-go

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

images: ## Build the Docker images
	@docker build -q -t $(GO_IMAGE) -f docker/go.Dockerfile docker >/dev/null

.cache:
	@mkdir -p .cache/home

test: test-go ## Run all tests

test-go: images .cache ## Backend tests only
	$(GO_RUN) go test -race -cover ./...

lint: lint-go ## Run all linters

lint-go: images .cache
	@out="$$($(GO_RUN) gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(GO_RUN) go vet ./...
