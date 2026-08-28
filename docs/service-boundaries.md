# Service Boundaries

This document defines the intended module boundaries for the service rewrite. The first service baseline is intentionally thin; these rules keep later implementation from re-creating the browser monolith inside one package.

## Top-Level Ownership

- `frontend/` owns the browser app image and static asset packaging.
- `services/api-service/` owns public HTTP API, request validation, job metadata, storage orchestration, and queue publishing.
- `services/document-service/` owns Excel reading/writing, workbook domain logic, report generation, and worker execution.
- `packages/contracts/` owns API contracts shared by services and frontend.
- `deploy/` owns local and production-like composition.

## API Service

Allowed dependencies:

- `cmd/api` may import `internal/http`.
- `internal/http` may import `internal/jobs` and `internal/storage` ports.
- `internal/jobs` may define job models, repository ports, queue publisher ports, and service orchestration.
- `internal/storage` may define object storage ports and implementations.

Forbidden dependencies:

- `api-service` must not import `document-service/internal/...`.
- `api-service` must not parse or mutate Excel workbook contents.
- HTTP handlers must not depend on concrete database, queue, or object storage clients directly; use ports owned by `internal/jobs` and `internal/storage`.

## Document Service

Allowed dependencies:

- `cmd/worker` may import worker/job orchestration packages.
- `internal/jobs` may import `internal/orderfill`, `internal/north`, `internal/reports`, and storage/queue ports.
- `internal/orderfill` and `internal/north` may import `internal/domain`, `internal/brands`, `internal/matching`, `internal/excel`, and `internal/reports`.
- `internal/excel` must stay infrastructure-focused: workbook zip/XML parsing and writing only.
- `internal/domain` must stay pure domain data and validation.
- `internal/brands` owns brand-specific quantity and detection rules.
- `internal/matching` owns article/name matching and duplicate detection.
- `internal/reports` owns JSON report DTOs and output-file metadata.

Forbidden dependencies:

- `document-service` must not import `api-service/internal/...`.
- `internal/domain` must not import `internal/excel`, queue clients, storage clients, or HTTP packages.
- `internal/excel` must not import brand, matching, order-fill, or north packages.
- Worker code must not write user-facing HTTP responses.

## Contracts

`packages/contracts/openapi.yaml` is the source of truth for public job API shape. Frontend and API implementation should conform to the contract rather than inventing local DTO variants.
