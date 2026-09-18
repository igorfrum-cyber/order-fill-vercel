# Service Boundaries

This document defines the active backend v2 boundaries. Historical implementation
plans may mention the retired root `services/` tree; current runtime code lives
under `backend/`.

## Top-Level Ownership

- `frontend/` owns browser UI, state, rendering, and API calls through `frontend/src/api/`.
- `backend/services/gateway-service/` owns the public HTTP API, session gate, CSRF/CORS, request validation, and response mapping.
- `backend/services/identity-service/` owns users, companies, company order profiles, sessions, invites, password changes, and account authorization data.
- `backend/services/twofa-service/` owns TOTP secrets, verification, and TOTP rate limiting.
- `backend/services/passkey-service/` owns WebAuthn credentials and ceremony state.
- `backend/services/job-service/` owns job metadata, job authorization context, report state, and queue publishing.
- `backend/services/file-service/` owns object metadata and object storage.
- `backend/services/document-service/` owns Excel parsing/writing, supported supplier header-field mappings, preview artifacts, and document job execution. It maps workbook rows to matching items and applies returned identity decisions; it does not decide product identity itself.
- `backend/services/matching-service/` owns product matching decisions. It accepts structured items and returns canonical `ReportCategory` plus `MatchReasons`; it does not parse Excel.
- `backend/services/brand-service/` owns the read-only brand catalog and brand-specific rules; gateway may expose their projection to platform admins but does not own or edit them.
- `backend/services/calculation-service/` owns quantity calculations over normalized inputs, including budget-to-target planning: pricing (main discount), brand order rules, and the CHRISTINA PROFF set discount. The set discount applies on both the Tyumen and North flows — `document-service` attaches PROFF line metadata to North rows so the planner can complete sets there too. The browser sends raw report rows to `POST /api/v1/order/budget-plan`; it does not compute the plan.
- `backend/services/inbound-service/` owns the inbound 1С mail contour: CloudMailin webhook payloads, company receive addresses, message metadata, and attachment objects. It uses its own PostgreSQL instance and a dedicated S3 bucket and does not depend on business services.
- `backend/proto/` owns internal gRPC contracts.

## Public Boundary

Only `gateway-service` is a browser-facing backend service. Frontend requests use
same-origin `/api/v1/...` paths; the frontend nginx container proxies `/api/` to
`gateway-service:8080`.

Gateway handlers must not parse Excel workbooks, write object blobs directly, or
reach into other services' storage. They validate HTTP inputs once, derive user
identity from the session cookie, then call internal gRPC services. Budget
preview is session-gated `POST /api/v1/order/budget-plan`: gateway checks the
discount range and maps JSON to `PlanBudget`; it does not compute quantities.

## Internal Boundary

Internal services communicate through protobuf/gRPC. Shared transport behavior
lives in `backend/pkg/grpcutil`: request IDs, deadlines, message limits, server
shutdown, and optional TLS/mTLS.

Services must keep domain behavior independent of transport DTOs. gRPC handlers
map protobuf messages and status codes only; authorization and business state
belong in service packages or domain packages.

## Storage Boundary

Stateful services must use durable dependencies outside local development:

- identity, job, audit: PostgreSQL
- file: PostgreSQL metadata plus S3-compatible object storage
- twofa, passkey: PostgreSQL plus Redis-backed transient state
- document-worker: Redis queue plus gRPC access to job, file, brand, matching, and calculation
- inbound: dedicated PostgreSQL instance plus a dedicated S3-compatible bucket (separate credentials); `file-service` is not the boundary for inbound mail attachments

In production, config validation fails at startup instead of falling back to
in-memory stores.

## Forbidden Dependencies

- Frontend must not contain workbook parsing or order calculation rules.
- Gateway must not import `backend/services/*/internal/...` from other services.
- Document-service must not import gateway, identity, or job storage internals.
- Matching, brand, and calculation services must not read Excel files or object storage.
- Historical root `services/` modules must not be reintroduced.
