# CLAUDE.md

This file provides guidance to Claude Code and Cursor when working in this repository.

## Overview

Order-fill processes supplier Excel workbooks. The browser never parses Excel; that stays in `document-service`.

```text
frontend  -->  gateway-service  -->  identity / job / file / audit / 2fa / passkey
                         |          redis (job queue)
                         |          minio (workbooks)
                         v
                  document-worker
```

- `frontend/` — Vite browser app: upload UI, report, manual edits, polling, download links. Talks to the API only through `frontend/src/api/`.
- `backend/services/gateway-service/` — public HTTP API, session gate, request validation, service orchestration.
- `backend/services/job-service/` — job metadata, queue publishing, report state.
- `backend/services/file-service/` — object-storage boundary and file metadata.
- `backend/services/document-service/` — Excel read/write, job processing, reports; runs as `document-api` and `document-worker`.
- `backend/proto/` — internal protobuf/gRPC contracts.
- `backend/services/gateway-service/api/openapi.yaml` — public HTTP contract between frontend and gateway.
- `deploy/docker-compose.yml` — local runtime; includes `backend/deploy/docker-compose.yml`.

Do not put workbook rules in HTTP handlers or in the browser. Match existing brand-specific behavior before generalizing. Do not execute macros from uploaded `.xlsm` files.

## Setup

```bash
cp .env.example .env
npm ci --prefix frontend
```

Go modules vendor themselves on first `go test` / `go build` in each service.

## Common commands

```bash
make up                          # docker compose stack
make down
make logs

npm run dev --prefix frontend    # UI only, http://127.0.0.1:3200, API still from the stack
```

Service health:

- frontend: http://127.0.0.1:3200
- gateway-service: http://127.0.0.1:8080/healthz
- document-api: http://127.0.0.1:8087/healthz inside compose network
- document-worker: http://127.0.0.1:8092/healthz inside compose network

From a service directory:

```bash
cd backend/services/gateway-service && go test ./...
cd backend/services/document-service && go test ./internal/domain/orderfill -run TestFill
```

## Pre-commit gate — mandatory before every commit

Run the full local suite from the repository root and commit only if it passes. It is the same script CI runs in `.github/workflows/verify.yml`. Do not push and wait for GitHub to find a failure that `scripts/verify.sh` would have caught.

```bash
make verify
```

That is `bash scripts/verify.sh`, which runs:

```bash
bash scripts/verify-toolchain.sh          # Node/Go pins: engines, go.mod, Docker, CI
npm run verify --prefix frontend          # ESLint + syntax, node:test, vite build
bash scripts/verify-go.sh backend/pkg
bash scripts/verify-go.sh backend/proto
bash scripts/verify-go.sh backend/services/*
docker compose -f backend/deploy/docker-compose.yml config
```

Each Go module is checked with:

```bash
gofmt -l .              # must print nothing; fix with gofmt -w .
go vet ./...
golangci-lint run       # unused, errcheck, staticcheck, misspell, …
gosec ./...             # security
go mod tidy             # go.mod / go.sum must stay unchanged
go build ./...
go test ./...
```

Local Node must be ≥ `engines.node` (24). Local Go must be ≥ the `go` line in `backend/**/go.mod`, and those lines must match. `golangci-lint` v2 and `gosec` are installed on first run if missing.

For vulnerability reachability checks, install and run the official Go scanner:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
cd backend/services/gateway-service && govulncheck ./...
cd backend/services/document-service && govulncheck ./...
```

Use the installed scanner in this repo; verification pins `GOTOOLCHAIN=local` so the gate does not silently download a different Go toolchain.

`npm run precommit` is an alias for the same gate.

`make lint` is the same checks without tests and Vite build.

If `frontend/package.json` or `frontend/package-lock.json` changed:

```bash
npm ci --prefix frontend
make verify
```

Commit the lockfile together with `frontend/package.json`.

If `gofmt -l`, ESLint, golangci-lint, or gosec report issues, fix them and create a **new** commit after a failed hook — do not `--no-verify`.

## Working rules

- Keep workbook calculations deterministic and explainable unless the user explicitly asks for an ML experiment.
- Keep the frontend thin: UI, state, API orchestration, and rendering stay in focused modules under `frontend/src/`.
- Keep business rules out of `httpapi` handlers and out of the browser.
- Add or update Go tests when changing matching, rounding, period validation, ЧЗ merge, or order-quantity rules.
- Keep generated output out of git: `dist/`, `test-output/`, `.vercel/`, `node_modules/`, `testdata/private/`.
