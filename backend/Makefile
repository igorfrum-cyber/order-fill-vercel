.PHONY: build test lint fmt tidy fmt-check tidy-check vet check proto-gen openapi-gen compose-up compose-down compose-logs

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
GOBIN := $(shell go env GOPATH)/bin
export GOCACHE ?= $(ROOT)/.gocache

MODULES := $(shell find . -name go.mod -exec dirname {} \; | sort)

build:
	@for m in $(MODULES); do \
		if [ -n "$$(find $$m -name '*.go' -print -quit)" ]; then (cd $$m && go build ./...); fi; \
	done

test:
	@for m in $(MODULES); do \
		if [ -n "$$(find $$m -name '*.go' -print -quit)" ]; then (cd $$m && go test ./...); fi; \
	done

lint: fmt-check vet

fmt:
	@for m in $(MODULES); do (cd $$m && gofmt -w .); done

tidy:
	@for m in $(MODULES); do (cd $$m && go mod tidy); done

fmt-check:
	@unformatted=$$(for m in $(MODULES); do (cd $$m && gofmt -l .); done); \
	if [ -n "$$unformatted" ]; then printf '%s\n' "$$unformatted"; exit 1; fi

tidy-check: tidy
	@git diff --exit-code -- go.work pkg/go.mod pkg/go.sum proto/go.mod proto/go.sum

vet:
	@for m in $(MODULES); do \
		if [ -n "$$(find $$m -name '*.go' -print -quit)" ]; then (cd $$m && go vet ./...); fi; \
	done

check: fmt-check vet test

proto-gen:
	cd proto && PATH="$(GOBIN):$$PATH" buf generate

openapi-gen:
	@echo "openapi-gen: add oapi-codegen in the gateway OpenAPI task"

compose-up:
	docker compose $(if $(wildcard $(ROOT)/.env),--env-file $(ROOT)/.env) -f deploy/docker-compose.yml up --build

compose-down:
	docker compose $(if $(wildcard $(ROOT)/.env),--env-file $(ROOT)/.env) -f deploy/docker-compose.yml down --remove-orphans

compose-logs:
	docker compose $(if $(wildcard $(ROOT)/.env),--env-file $(ROOT)/.env) -f deploy/docker-compose.yml logs -f
