# CLAUDE.md

This file provides guidance to Claude Code and Cursor when working in this repository.

## Overview

Order-fill processes supplier Excel workbooks. The browser never parses Excel; that stays in `document-service`.

```text
frontend  -->  api-service  -->  postgres
                    |               redis (job queue)
                    |               minio (workbooks)
                    v
            document-service
```

- `frontend/` — Vite browser app: upload UI, report, manual edits, polling, download links. Talks to the API only through `frontend/src/api/`.
- `services/api-service/` — public HTTP API, job metadata, uploads, queue publish, object-storage boundary.
- `services/document-service/` — Excel read/write, brand rules, matching, job processing, reports.
- `packages/contracts/openapi.yaml` — HTTP contract between frontend and api-service.
- `deploy/docker-compose.yml` — local runtime for all three apps plus Postgres, Redis, MinIO.

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
- api-service: http://127.0.0.1:8080/healthz
- document-service: http://127.0.0.1:8081/healthz

From a service directory:

```bash
cd services/api-service && go test ./...
cd services/document-service && go test ./internal/domain/orderfill -run TestFill
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
bash scripts/verify-go.sh services/api-service
bash scripts/verify-go.sh services/document-service
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

Local Node must be ≥ `engines.node` (24). Local Go must be ≥ the `go` line in both `go.mod` files, and those lines must match. `golangci-lint` v2 and `gosec` are installed on first run if missing.

For vulnerability reachability checks, install and run the official Go scanner:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
cd services/api-service && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
cd services/document-service && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Prefer the `go run ...@latest` form in this repo so the scanner follows the module toolchain when `go.mod` requires a newer Go version than the locally installed default.

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
