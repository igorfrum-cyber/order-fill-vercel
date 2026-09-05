# Backend v2 Microservices Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
> **For Cursor:** Execute tasks in order. Do not skip tests. Keep the current product behavior available until the gateway cutover task explicitly retires the old API path.

**Goal:** Rebuild the backend as a full microservice system with gateway, identity, twofa, passkey, job, file, document, matching, brand, calculation, and audit services.

**Architecture:** Follow `docs/plans/2026-09-04-microservice-architecture-v2-design.md`. The frontend talks only to `gateway-service` over HTTP/OpenAPI. Internal services communicate through gRPC contracts generated from `backend/proto`. Each service owns its own module, transport, storage, migrations, Dockerfile, README, health/readiness, and tests.

**Tech Stack:** Go, `go.work`, gRPC/protobuf, OpenAPI, PostgreSQL, Redis Streams, MinIO/S3-compatible object storage, Docker Compose, `slog`, table-driven Go tests.

---

## Non-negotiable rules

- Do not expand the current `services/api-service` with new domain code.
- Do not let one service import another service's `internal/...`.
- Do not let one service write another service's tables.
- Keep frontend-facing behavior compatible unless a task explicitly changes the OpenAPI contract.
- Write tests before moving behavior.
- Run the narrow test after each task and the wider verification after each phase.
- Prefer small commits after each completed task.
- Use local `GOCACHE` when sandboxed Go commands try to write outside the workspace:

```bash
GOCACHE="$PWD/.gocache" go test ./...
```

## Current behavior map

Use these files as the source for existing behavior while extracting services:

- Auth/session/admin/tenancy: `services/api-service/internal/domain/identity`, `services/api-service/internal/domain/authz`, `services/api-service/internal/app/usecase/auth.go`, `services/api-service/internal/app/usecase/admin.go`, `services/api-service/internal/adapter/outbound/postgres/identity.go`.
- Job workflow: `services/api-service/internal/domain/job`, `services/api-service/internal/app/usecase/create_job.go`, `services/api-service/internal/app/usecase/submit_edits.go`, `services/api-service/internal/app/usecase/read_job.go`.
- Public HTTP behavior: `services/api-service/internal/adapter/inbound/httpapi`.
- Queue contract: `services/api-service/internal/app/port/port.go`, `services/document-service/internal/app/port/port.go`.
- Document worker orchestration: `services/document-service/internal/app/usecase/process_job.go`.
- XLSX mechanics: `services/document-service/internal/adapter/outbound/xlsx`, `services/document-service/internal/domain/spreadsheet`, `services/document-service/internal/domain/preview`.
- Matching: `services/document-service/internal/domain/matching`, matching pieces in `services/document-service/internal/domain/orderfill/fill.go`.
- Brand policy: `services/document-service/internal/domain/brand`.
- Calculation: `services/document-service/internal/domain/orderfill/recalculate.go`, `frontend/src/features/north/northPlan.js`.
- API contract: `packages/contracts/openapi.yaml`.

## Phase 1: Backend workspace and shared infrastructure

### Task 1: Create backend workspace skeleton

**Files:**

- Create: `backend/go.work`
- Create: `backend/Makefile`
- Create: `backend/pkg/go.mod`
- Create: `backend/proto/go.mod`
- Create: `backend/pkg/README.md`
- Modify: `README.md`
- Modify: `docs/README.md`

**Step 1: Add workspace files**

Create `backend/go.work` with `pkg`, `proto`, and every planned service module listed. At this point service folders may be empty or contain only `go.mod`; the workspace should express the target shape.

**Step 2: Add backend Makefile**

Add targets:

```text
build
test
lint
fmt
tidy
fmt-check
tidy-check
vet
check
proto-gen
openapi-gen
compose-up
compose-down
compose-logs
```

The Makefile should iterate through `pkg`, `proto`, and `services/*` modules.

**Step 3: Verify**

Run:

```bash
cd backend && make fmt-check
cd backend && make tidy
```

Expected: commands complete without modifying unrelated files.

**Step 4: Commit**

```bash
git add backend/go.work backend/Makefile backend/pkg/go.mod backend/proto/go.mod backend/pkg/README.md README.md docs/README.md
git commit -m "chore: add backend v2 workspace"
```

### Task 2: Add shared infrastructure packages

**Files:**

- Create: `backend/pkg/apperr/apperr.go`
- Create: `backend/pkg/apperr/apperr_test.go`
- Create: `backend/pkg/healthz/healthz.go`
- Create: `backend/pkg/healthz/healthz_test.go`
- Create: `backend/pkg/logger/logger.go`
- Create: `backend/pkg/metrics/http.go`
- Create: `backend/pkg/grpcutil/server.go`
- Create: `backend/pkg/grpcutil/client.go`
- Create: `backend/pkg/grpcutil/request_id.go`
- Create: `backend/pkg/objectstore/objectstore.go`

**Step 1: Write package tests**

Cover:

- domain error to transport status mapping;
- health response for ready and not ready probes;
- logger default configuration;
- request id propagation helpers.

**Step 2: Implement minimal packages**

Keep packages infrastructure-only. Do not put `Job`, `User`, `Brand`, `ReportRow`, or other business types here.

**Step 3: Verify**

Run:

```bash
cd backend/pkg && GOCACHE="$PWD/../../.gocache" go test ./...
cd backend && make fmt-check
```

Expected: tests pass and formatting is clean.

**Step 4: Commit**

```bash
git add backend/pkg
git commit -m "chore: add backend shared infrastructure packages"
```

## Phase 2: Contracts first

### Task 3: Define internal protobuf contracts

**Files:**

- Create: `backend/proto/orderfill/identity/v1/identity.proto`
- Create: `backend/proto/orderfill/twofa/v1/twofa.proto`
- Create: `backend/proto/orderfill/passkey/v1/passkey.proto`
- Create: `backend/proto/orderfill/jobs/v1/jobs.proto`
- Create: `backend/proto/orderfill/files/v1/files.proto`
- Create: `backend/proto/orderfill/documents/v1/documents.proto`
- Create: `backend/proto/orderfill/matching/v1/matching.proto`
- Create: `backend/proto/orderfill/brand/v1/brand.proto`
- Create: `backend/proto/orderfill/calculation/v1/calculation.proto`
- Create: `backend/proto/orderfill/audit/v1/audit.proto`
- Create: `backend/proto/buf.yaml`
- Create: `backend/proto/buf.gen.yaml`

**Step 1: Define minimal RPCs**

Start with only the RPCs required by current product behavior.

Identity:

```text
Login
CompleteTwoFactorLogin
Logout
LogoutEverywhere
ValidateSession
GetMe
AcceptInvite
ChangePassword
CreateCompany
ListCompanies
UpdateCompany
DisableCompany
CreateUser
ListUsers
DisableUser
ResetUserAccess
```

TwoFA:

```text
Setup
Enable
Disable
IsEnabled
Verify
```

Passkey:

```text
BeginRegistration
FinishRegistration
ListCredentials
DeleteCredential
BeginLogin
FinishLogin
```

Jobs:

```text
CreateJob
GetJob
ListJobs
GetReport
ListFiles
SubmitEdits
UpdateProgress
CompleteJob
FailJob
```

Files:

```text
PutObject
GetObject
CreateUpload
FinalizeUpload
CreateArchive
```

Documents:

```text
AnalyzeInputs
BuildPreview
```

Matching:

```text
MatchRows
NormalizeArticle
NormalizeName
```

Brand:

```text
GetBrandPolicy
ListBrands
DetectBrand
```

Calculation:

```text
CalculateOrderRecommendations
CalculateAdjustedQuantity
CalculateNorthPlan
RecalculateNorthRow
ValidateManualEdits
```

Audit:

```text
Record
ListEvents
```

**Step 2: Include versioning fields**

Every request that can evolve should include:

```text
string request_id
string idempotency_key
string actor_user_id
string company_id
```

Use only where meaningful; do not force actor fields into anonymous login begin calls.

**Step 3: Generate code**

Run:

```bash
cd backend && make proto-gen
```

Expected: generated Go files appear under `backend/proto/gen/go/...`.

**Step 4: Verify**

Run:

```bash
cd backend/proto && GOCACHE="$PWD/../../.gocache" go test ./...
```

Expected: generated package compiles.

**Step 5: Commit**

```bash
git add backend/proto
git commit -m "chore: define backend v2 protobuf contracts"
```

### Task 4: Move public OpenAPI source to gateway

**Files:**

- Create: `backend/services/gateway-service/api/openapi.yaml`
- Create: `backend/services/gateway-service/api/oapi-codegen.yaml`
- Create: `backend/services/gateway-service/api/oapi-codegen-spec.yaml`
- Modify: `packages/contracts/openapi.yaml`

**Step 1: Copy current public contract**

Move the current contract shape from `packages/contracts/openapi.yaml` into the gateway API folder. Keep `packages/contracts/openapi.yaml` either as a pointer document or generated copy, but make gateway the source of truth.

**Step 2: Add explicit tags**

Tag routes by domain:

```text
auth
companies
users
jobs
files
preview
audit
status
```

**Step 3: Verify**

Run whichever OpenAPI validation command exists after codegen tooling is added. Until then, verify YAML parses:

```bash
ruby -e 'require "yaml"; YAML.load_file("backend/services/gateway-service/api/openapi.yaml")'
```

Expected: no parse error.

**Step 4: Commit**

```bash
git add backend/services/gateway-service/api packages/contracts/openapi.yaml
git commit -m "chore: move public API contract to gateway"
```

## Phase 3: Service skeletons

### Task 5: Generate service skeletons

**Files:**

- Create under each `backend/services/<name>/`:
  - `go.mod`
  - `Dockerfile`
  - `README.md`
  - `cmd/<binary>/main.go`
  - `internal/bootstrap/bootstrap.go`
  - `internal/config/config.go`
  - `internal/domain/errors.go`
  - `internal/transport/grpcapi/server.go` for gRPC services
  - `internal/transport/httpapi/handler.go` for gateway or HTTP file routes

Services:

```text
gateway-service
identity-service
twofa-service
passkey-service
job-service
file-service
document-service
matching-service
brand-service
calculation-service
audit-service
```

**Step 1: Add health endpoints**

Every service must expose liveness and readiness. For gRPC services, add a `healthcheck` CLI command or a small HTTP metrics/health listener. For `gateway-service`, expose `/healthz` and `/readyz`.

**Step 2: Add config loading**

Each service uses a unique env prefix:

```text
GATEWAY_
IDENTITY_
TWOFA_
PASSKEY_
JOB_
FILE_
DOCUMENT_
MATCHING_
BRAND_
CALCULATION_
AUDIT_
```

**Step 3: Add compile tests**

Each service gets at least one package test that constructs config defaults and verifies the health handler.

**Step 4: Verify**

Run:

```bash
cd backend && make test
cd backend && make build
```

Expected: all skeleton modules compile.

**Step 5: Commit**

```bash
git add backend/services backend/go.work backend/Makefile
git commit -m "chore: scaffold backend v2 services"
```

## Phase 4: Identity, TwoFA, and Passkey

### Task 6: Extract identity domain and storage

**Files:**

- Create: `backend/services/identity-service/internal/domain/user.go`
- Create: `backend/services/identity-service/internal/domain/company.go`
- Create: `backend/services/identity-service/internal/domain/role.go`
- Create: `backend/services/identity-service/internal/domain/session.go`
- Create: `backend/services/identity-service/internal/password/password.go`
- Create: `backend/services/identity-service/internal/secret/token.go`
- Create: `backend/services/identity-service/internal/migrate/migrations/00001_init.sql`
- Create: `backend/services/identity-service/internal/storage/postgres/users.go`
- Create: `backend/services/identity-service/internal/storage/postgres/companies.go`
- Create: `backend/services/identity-service/internal/storage/postgres/roles.go`
- Create: `backend/services/identity-service/internal/session/redis.go`
- Read from: `services/api-service/internal/domain/identity`
- Read from: `services/api-service/internal/domain/authz`
- Read from: `services/api-service/internal/adapter/outbound/postgres/identity.go`

**Step 1: Write domain tests**

Port the existing tests for password hashing, sessions, tokens, slugs, logo parsing, roles, and authorization decisions.

**Step 2: Implement domain**

Move behavior, do not redesign it unless required by the new service boundary.

**Step 3: Add migrations**

Identity owns companies, users, roles, permissions, invites, and durable session metadata if used. Sessions may live in Redis, but user/session indexes needed for revocation must be owned here.

**Step 4: Verify**

Run:

```bash
cd backend/services/identity-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: identity tests pass.

**Step 5: Commit**

```bash
git add backend/services/identity-service
git commit -m "feat: add identity service domain and storage"
```

### Task 7: Implement identity gRPC service

**Files:**

- Create: `backend/services/identity-service/internal/service/auth/auth.go`
- Create: `backend/services/identity-service/internal/service/auth/login.go`
- Create: `backend/services/identity-service/internal/service/auth/logout.go`
- Create: `backend/services/identity-service/internal/service/auth/validate_session.go`
- Create: `backend/services/identity-service/internal/service/users/users.go`
- Create: `backend/services/identity-service/internal/service/users/create.go`
- Create: `backend/services/identity-service/internal/service/companies/companies.go`
- Create: `backend/services/identity-service/internal/clients/twofa/client.go`
- Create: `backend/services/identity-service/internal/clients/passkey/client.go`
- Create: `backend/services/identity-service/internal/transport/grpcapi/server.go`
- Create one file per RPC under `internal/transport/grpcapi/`

**Step 1: Write service tests**

Cover:

- missing user and wrong password return the same auth failure;
- disabled user cannot log in;
- invite is one-time;
- reset access clears password and sessions;
- only identity mints sessions;
- TOTP-enabled user returns a challenge until code is verified;
- passkey finish returns session only through identity.

**Step 2: Implement services**

Use narrow interfaces in package root files. Keep `ctx context.Context` first. Keep transport mapping separate from business logic.

**Step 3: Verify**

Run:

```bash
cd backend/services/identity-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: service and transport tests pass.

**Step 4: Commit**

```bash
git add backend/services/identity-service
git commit -m "feat: expose identity service over grpc"
```

### Task 8: Extract twofa-service

**Files:**

- Create: `backend/services/twofa-service/internal/domain/credential.go`
- Create: `backend/services/twofa-service/internal/secret/aesgcm.go`
- Create: `backend/services/twofa-service/internal/totp/totp.go`
- Create: `backend/services/twofa-service/internal/ratelimit/redis.go`
- Create: `backend/services/twofa-service/internal/storage/postgres/credentials.go`
- Create: `backend/services/twofa-service/internal/service/twofa/twofa.go`
- Create one file per RPC under `backend/services/twofa-service/internal/transport/grpcapi/`
- Read from: `services/api-service/internal/domain/identity/totp.go`
- Read from: `services/api-service/internal/app/usecase/totp.go`

**Step 1: Write tests**

Cover setup, enable, disable, recovery code consumption, wrong code lockout, and secret encryption/decryption.

**Step 2: Implement service**

TOTP secrets are encrypted at rest. Recovery codes are hashed and single-use. Login-time verification is callable by `identity-service`.

**Step 3: Verify**

Run:

```bash
cd backend/services/twofa-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/twofa-service
git commit -m "feat: add twofa service"
```

### Task 9: Extract passkey-service

**Files:**

- Create: `backend/services/passkey-service/internal/domain/credential.go`
- Create: `backend/services/passkey-service/internal/ceremony/redis.go`
- Create: `backend/services/passkey-service/internal/webauthn/webauthn.go`
- Create: `backend/services/passkey-service/internal/storage/postgres/credentials.go`
- Create: `backend/services/passkey-service/internal/service/passkey/passkey.go`
- Create one file per RPC under `backend/services/passkey-service/internal/transport/grpcapi/`
- Read from: `services/api-service/internal/domain/identity/passkey.go`
- Read from: `services/api-service/internal/app/usecase/passkey.go`
- Read from: `services/api-service/internal/adapter/outbound/passkeys`

**Step 1: Write tests**

Cover registration begin/finish, login begin/finish, credential listing, deletion, unknown credential, expired challenge, and sign-counter update.

**Step 2: Implement service**

Passkey service returns verified user ids. It never mints sessions and never validates passwords.

**Step 3: Verify**

Run:

```bash
cd backend/services/passkey-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/passkey-service
git commit -m "feat: add passkey service"
```

## Phase 5: Job and file services

### Task 10: Extract job-service

**Files:**

- Create: `backend/services/job-service/internal/domain/job.go`
- Create: `backend/services/job-service/internal/domain/report.go`
- Create: `backend/services/job-service/internal/domain/authz.go`
- Create: `backend/services/job-service/internal/migrate/migrations/00001_init.sql`
- Create: `backend/services/job-service/internal/storage/postgres/jobs.go`
- Create: `backend/services/job-service/internal/storage/postgres/reports.go`
- Create: `backend/services/job-service/internal/queue/redis.go`
- Create: `backend/services/job-service/internal/service/jobs/jobs.go`
- Create: `backend/services/job-service/internal/service/jobs/create.go`
- Create: `backend/services/job-service/internal/service/jobs/submit_edits.go`
- Create: `backend/services/job-service/internal/service/jobs/complete.go`
- Create: `backend/services/job-service/internal/transport/grpcapi/server.go`
- Read from: `services/api-service/internal/domain/job`
- Read from: `services/api-service/internal/domain/authz/job.go`
- Read from: `services/api-service/internal/app/usecase/create_job.go`
- Read from: `services/api-service/internal/app/usecase/submit_edits.go`

**Step 1: Write tests**

Cover:

- upload validation rules for `order_fill` and `north_merge`;
- job owner is required;
- purchaser only sees own jobs;
- company admin sees company jobs;
- platform role behavior if preserved;
- completed jobs can accept edits;
- job create publishes exactly one versioned queue message;
- worker completion records report and files.

**Step 2: Implement service**

The service owns job state and queue publishing. Do not read Excel bytes. Do not call `document-service` synchronously for processing.

**Step 3: Verify**

Run:

```bash
cd backend/services/job-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/job-service
git commit -m "feat: add job service"
```

### Task 11: Extract file-service

**Files:**

- Create: `backend/services/file-service/internal/domain/object.go`
- Create: `backend/services/file-service/internal/domain/upload.go`
- Create: `backend/services/file-service/internal/storage/objectstore/s3.go`
- Create: `backend/services/file-service/internal/storage/postgres/files.go`
- Create: `backend/services/file-service/internal/service/files/files.go`
- Create: `backend/services/file-service/internal/service/files/put.go`
- Create: `backend/services/file-service/internal/service/files/get.go`
- Create: `backend/services/file-service/internal/service/files/archive.go`
- Create: `backend/services/file-service/internal/transport/grpcapi/server.go`
- Read from: `services/api-service/internal/adapter/outbound/objectstore`
- Read from: `services/document-service/internal/adapter/outbound/objectstore`

**Step 1: Write tests**

Cover object put/get, missing object, safe file names, archive generation, content type preservation, and idempotent upload finalize.

**Step 2: Implement service**

At first this can be thin over MinIO/S3. Keep the service boundary real so resumable uploads and download authorization can grow here later.

**Step 3: Verify**

Run:

```bash
cd backend/services/file-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/file-service
git commit -m "feat: add file service"
```

## Phase 6: Brand, matching, and calculation services

### Task 12: Extract brand-service

**Files:**

- Create: `backend/services/brand-service/internal/domain/brand.go`
- Create: `backend/services/brand-service/internal/domain/policy.go`
- Create: `backend/services/brand-service/internal/storage/static/rules.go`
- Create: `backend/services/brand-service/internal/service/brands/brands.go`
- Create: `backend/services/brand-service/internal/service/brands/get_policy.go`
- Create: `backend/services/brand-service/internal/service/brands/detect.go`
- Create: `backend/services/brand-service/internal/transport/grpcapi/server.go`
- Read from: `services/document-service/internal/domain/brand/rules.go`

**Step 1: Write tests**

Port all brand rule tests. Cover current brands: `angiopharm`, `christina`, `levissime`, `sothys`, `novacutan`, `skin_synergy`, `klapp`.

**Step 2: Implement static policy storage**

Start with static rules in code. Keep data model versionable so later rules can move to Postgres without changing callers.

**Step 3: Verify**

Run:

```bash
cd backend/services/brand-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/brand-service
git commit -m "feat: add brand service"
```

### Task 13: Extract matching-service

**Files:**

- Create: `backend/services/matching-service/internal/domain/item.go`
- Create: `backend/services/matching-service/internal/domain/match.go`
- Create: `backend/services/matching-service/internal/normalize/article.go`
- Create: `backend/services/matching-service/internal/normalize/name.go`
- Create: `backend/services/matching-service/internal/service/matching/matching.go`
- Create: `backend/services/matching-service/internal/service/matching/match_rows.go`
- Create: `backend/services/matching-service/internal/service/matching/duplicates.go`
- Create: `backend/services/matching-service/internal/transport/grpcapi/server.go`
- Read from: `services/document-service/internal/domain/matching`
- Read from: `services/document-service/internal/domain/normalize`
- Read from: matching code in `services/document-service/internal/domain/orderfill/fill.go`

**Step 1: Write tests**

Port matching tests and add service-level tests for:

- article exact match;
- article alias match;
- name fallback;
- suspicious name difference;
- no-article source rows;
- duplicate source articles;
- duplicate blank articles;
- stable candidate ordering.

**Step 2: Implement service**

Inputs are structured rows, not workbooks. Outputs include status, score, selected row, candidate list, duplicate info, and reason code.

**Step 3: Verify**

Run:

```bash
cd backend/services/matching-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/matching-service
git commit -m "feat: add matching service"
```

### Task 14: Extract calculation-service

**Files:**

- Create: `backend/services/calculation-service/internal/domain/order.go`
- Create: `backend/services/calculation-service/internal/domain/north.go`
- Create: `backend/services/calculation-service/internal/service/calculation/calculation.go`
- Create: `backend/services/calculation-service/internal/service/calculation/recommendations.go`
- Create: `backend/services/calculation-service/internal/service/calculation/adjust_quantity.go`
- Create: `backend/services/calculation-service/internal/service/calculation/north_plan.go`
- Create: `backend/services/calculation-service/internal/service/calculation/manual_edits.go`
- Create: `backend/services/calculation-service/internal/transport/grpcapi/server.go`
- Read from: `services/document-service/internal/domain/orderfill/recalculate.go`
- Read from: `services/document-service/internal/domain/brand/rules.go`
- Read from: `frontend/src/features/north/northPlan.js`
- Read from: `frontend/src/features/order/quantityPresentation.js`

**Step 1: Write recommendation tests**

Cover:

- ABC revenue ranking;
- category thresholds A+, A, B, C;
- target stock formula;
- delivery coefficient formula;
- novelty detection for one to three recent months;
- stable-history average demand behavior;
- stock and in-transit subtraction;
- non-negative recommendation.

**Step 2: Write adjusted quantity tests**

Cover:

- no adjustment;
- multiple adjustment;
- nearest multiple adjustment;
- box/minimum adjustment;
- small positive orders below threshold;
- ordered-fact override behavior.

**Step 3: Write North tests**

Port frontend North tests and add cases for:

- Tyumen free stock allocation order;
- supplier need conversion into supplier units;
- KLAPP nearest multiple;
- NOVACUTAN minimum;
- comment/source part generation;
- recalculation after manual city edits.

**Step 4: Implement service**

Do not import workbook packages. Do not parse Excel cell strings beyond normalized numeric/string inputs already supplied by `document-service`.

**Step 5: Verify**

Run:

```bash
cd backend/services/calculation-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 6: Commit**

```bash
git add backend/services/calculation-service
git commit -m "feat: add calculation service"
```

## Phase 7: Document service rewrite

### Task 15: Split document-service into API and worker

**Files:**

- Create: `backend/services/document-service/cmd/document-api/main.go`
- Create: `backend/services/document-service/cmd/document-worker/main.go`
- Create: `backend/services/document-service/internal/domain/workbook.go`
- Create: `backend/services/document-service/internal/xlsx/...`
- Create: `backend/services/document-service/internal/preview/...`
- Create: `backend/services/document-service/internal/service/orderfill/orderfill.go`
- Create: `backend/services/document-service/internal/service/north/north.go`
- Create: `backend/services/document-service/internal/clients/brand/client.go`
- Create: `backend/services/document-service/internal/clients/matching/client.go`
- Create: `backend/services/document-service/internal/clients/calculation/client.go`
- Create: `backend/services/document-service/internal/clients/file/client.go`
- Create: `backend/services/document-service/internal/clients/jobs/client.go`
- Read from: `services/document-service`

**Step 1: Port XLSX mechanics tests**

Move tests for parsing, writing, shared formulas, preview snapshots, large files, calcChain removal, and forced recalculation.

**Step 2: Port order-fill behavior tests**

Order-fill tests should use fake clients for brand, matching, and calculation. The document service should prove it orchestrates correctly and writes expected workbook cells.

**Step 3: Add North processor tests**

North must be a first-class processor. It should no longer be a frontend-only calculation path.

**Step 4: Implement worker dispatcher**

Dispatch by job type:

```text
order_fill -> orderfill processor
north_merge -> north processor
```

Unsupported job types are permanent invalid input failures.

**Step 5: Verify**

Run:

```bash
cd backend/services/document-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 6: Commit**

```bash
git add backend/services/document-service
git commit -m "feat: rewrite document service around processors"
```

## Phase 8: Audit service

### Task 16: Add audit-service

**Files:**

- Create: `backend/services/audit-service/internal/domain/event.go`
- Create: `backend/services/audit-service/internal/migrate/migrations/00001_init.sql`
- Create: `backend/services/audit-service/internal/storage/postgres/events.go`
- Create: `backend/services/audit-service/internal/service/audit/audit.go`
- Create: `backend/services/audit-service/internal/service/audit/record.go`
- Create: `backend/services/audit-service/internal/service/audit/list.go`
- Create: `backend/services/audit-service/internal/transport/grpcapi/server.go`

**Step 1: Write tests**

Cover recording login, logout, access reset, user disabled, company disabled, job viewed, file downloaded, and archive downloaded.

**Step 2: Implement service**

Start with explicit event recording. Leave database triggers/checkpoint digests for a later hardening phase.

**Step 3: Verify**

Run:

```bash
cd backend/services/audit-service && GOCACHE="$PWD/../../../.gocache" go test ./...
```

Expected: tests pass.

**Step 4: Commit**

```bash
git add backend/services/audit-service
git commit -m "feat: add audit service"
```

## Phase 9: Gateway and frontend cutover

### Task 17: Implement gateway-service

**Files:**

- Create: `backend/services/gateway-service/internal/clients/identity/client.go`
- Create: `backend/services/gateway-service/internal/clients/twofa/client.go`
- Create: `backend/services/gateway-service/internal/clients/passkey/client.go`
- Create: `backend/services/gateway-service/internal/clients/jobs/client.go`
- Create: `backend/services/gateway-service/internal/clients/files/client.go`
- Create: `backend/services/gateway-service/internal/clients/audit/client.go`
- Create: `backend/services/gateway-service/internal/transport/httpapi/router.go`
- Create: `backend/services/gateway-service/internal/transport/authhttp/...`
- Create: `backend/services/gateway-service/internal/service/gateway/gateway.go`
- Modify: `frontend/src/api/client.js`
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/api/jobs.js`
- Read from: `services/api-service/internal/adapter/inbound/httpapi`

**Step 1: Write route tests**

Cover public login routes, protected routes, CSRF, cookie behavior, job creation, report read, file download, preview, company admin, and status.

**Step 2: Implement handlers**

Preserve current frontend route paths unless the OpenAPI task changed them intentionally.

**Step 3: Add frontend API compatibility tests**

Update tests under `frontend/src/api/*.test.js` only after gateway behavior is implemented.

**Step 4: Verify**

Run:

```bash
cd backend/services/gateway-service && GOCACHE="$PWD/../../../.gocache" go test ./...
npm test --prefix frontend
```

Expected: gateway and frontend API tests pass.

**Step 5: Commit**

```bash
git add backend/services/gateway-service frontend/src/api
git commit -m "feat: route frontend api through gateway service"
```

### Task 18: Compose full backend v2 stack

**Files:**

- Create: `backend/deploy/docker-compose.yml`
- Modify: `deploy/docker-compose.yml`
- Modify: `README.md`
- Modify: `.env.example`

**Step 1: Add services**

Compose should include:

```text
gateway-service
identity-service
twofa-service
passkey-service
job-service
file-service
document-api
document-worker
matching-service
brand-service
calculation-service
audit-service
postgres
redis
minio
```

Only gateway is published to localhost. Other services use `expose` or internal network only.

**Step 2: Add health dependencies**

Use readiness where startup requires dependencies. Avoid fragile global startup ordering. Clients should tolerate services becoming ready after process start.

**Step 3: Verify**

Run:

```bash
cd backend && make compose-up
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
```

Expected: gateway health and readiness return success after dependencies are ready.

**Step 4: Commit**

```bash
git add backend/deploy deploy/docker-compose.yml README.md .env.example
git commit -m "chore: compose backend v2 stack"
```

## Phase 10: Behavior parity and retirement

### Task 19: Add end-to-end parity tests

**Files:**

- Create: `backend/tests/order_fill_e2e_test.go`
- Create: `backend/tests/north_merge_e2e_test.go`
- Create: `backend/tests/auth_e2e_test.go`
- Modify: `scripts/verify.sh`
- Modify: `Makefile`

**Step 1: Define parity scenarios**

Cover:

- bootstrap admin invite/login;
- create company;
- create user;
- login as purchaser;
- upload order-fill files;
- job reaches `needs_review`;
- report matches current expected statuses;
- submit edits;
- job reaches `completed`;
- download files;
- North merge path reaches review/completed;
- unauthorized users receive 404/401 as appropriate.

**Step 2: Implement tests**

Use existing `testdata/` and add only missing fixtures.

**Step 3: Verify**

Run:

```bash
make verify
cd backend && make check
```

Expected: full existing and backend v2 checks pass.

**Step 4: Commit**

```bash
git add backend/tests scripts/verify.sh Makefile testdata
git commit -m "test: cover backend v2 behavior parity"
```

### Task 20: Retire old api-service path

**Files:**

- Modify: `deploy/docker-compose.yml`
- Modify: `README.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/service-boundaries.md`
- Modify: `packages/contracts/openapi.yaml`
- Delete only after parity: `services/api-service`
- Move or delete after parity: old duplicated document-service code under `services/document-service` if superseded by `backend/services/document-service`

**Step 1: Confirm no frontend calls old API**

Run:

```bash
rg "8080|api-service|services/api-service" frontend backend deploy docs packages
```

Expected: no runtime dependency on old `services/api-service` remains.

**Step 2: Remove old compose services**

Remove old `api-service` from the active compose path only after gateway stack passes parity tests.

**Step 3: Verify**

Run:

```bash
make verify
cd backend && make check
```

Expected: full verification passes without old API runtime.

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: retire legacy api service"
```

## Final acceptance criteria

- Frontend talks only to `gateway-service`.
- Only gateway publishes a host port.
- Internal services communicate through generated gRPC clients.
- `identity-service` is the only session issuer.
- `twofa-service` owns TOTP secrets and verification lockout.
- `passkey-service` owns WebAuthn credentials and ceremonies.
- `job-service` owns job status, history, reports, and file references.
- `file-service` owns binary object lifecycle.
- `document-service` owns XLSX parsing/writing and processors.
- `matching-service` owns item identity decisions.
- `brand-service` owns supplier policy.
- `calculation-service` owns business formulas and replenishment math.
- `audit-service` owns audit storage.
- `document-service` does not write job tables directly.
- North merge is processed server-side.
- Existing order-fill behavior is covered by parity tests.
- `make verify` and `cd backend && make check` pass before the legacy path is retired.
