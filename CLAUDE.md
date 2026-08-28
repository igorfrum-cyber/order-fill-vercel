# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Overview

Order-fill service system for processing supplier Excel workbooks.

- `src/` contains the thin browser frontend. It owns upload UI, report rendering, manual edits, polling, and download links.
- `src/api/` contains frontend API clients. UI code should depend on these clients instead of calling service endpoints ad hoc.
- `services/api-service/` owns HTTP endpoints, job metadata, upload handling, queue publication, and object-storage boundaries.
- `services/document-service/` owns Excel parsing/writing, brand rules, matching, document job orchestration, and reports.
- `packages/contracts/openapi.yaml` is the API contract.
- `deploy/docker-compose.yml` is the local service runtime.

The frontend must not process Excel workbooks locally. Excel logic belongs in `document-service`.

## Common Commands

Install dependencies first on a fresh checkout:

```bash
npm ci
```

Run frontend locally:

```bash
npm run dev
```

Run the full local stack:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Build and test:

```bash
npm run verify
```

`npm run verify` is the local precommit gate for this repository.

## Pre-Commit Gate

### Mandatory before every commit

Run the full local gate from the repository root and commit only if it passes:

```bash
npm run verify
```

This currently expands to:

```bash
npm run build
npm run verify:api
npm run verify:document
```

### When dependencies changed

If `package.json` or `package-lock.json` changed, run a clean install before the gate:

```bash
npm ci
npm run verify
```

Commit `package-lock.json` together with `package.json` when dependency versions change.

## Working Rules

- Keep workbook calculations deterministic and explainable unless the user explicitly asks for an ML experiment.
- Keep frontend thin: UI, state, API orchestration, and rendering must be separated into focused modules.
- Keep business rules out of HTTP handlers and browser code.
- Do not execute macros from uploaded `.xlsm` files.
- Keep generated output out of git: `dist/`, `test-output/`, `.vercel/`, `node_modules/`, and `testdata/private/` are ignored.
- Match existing brand-specific behavior before generalizing shared logic.
- Add or update Go regression coverage when changing matching, rounding, period validation, or order quantity calculations.
