# Backend v2 microservice architecture

Date: 2026-09-04
Status: decided

## Decision

Build the next backend as a full microservice system from the start. The goal is
not to maximize the number of containers, but to make each runtime own one
stable business responsibility, one public contract, and one operational
profile.

The target style follows the proven Rosneft backend pattern:

- one external gateway;
- internal service-to-service gRPC;
- OpenAPI only at the frontend-facing edge;
- one Go module per service;
- shared `proto/` and small shared `pkg/`;
- explicit database ownership;
- service-local migrations;
- health/readiness endpoints and bounded remote calls everywhere;
- one bounded context per package inside each service.

This replaces the current `api-service` shape, where auth, companies, users,
jobs, audit, status, storage orchestration, and queue publishing have grown into
one deployable unit.

## Repository layout

Target backend layout:

```text
backend/
  go.work
  pkg/
    apperr/
    grpcutil/
    healthz/
    logger/
    metrics/
    objectstore/
    requestid/
  proto/
    orderfill/audit/v1/audit.proto
    orderfill/brand/v1/brand.proto
    orderfill/calculation/v1/calculation.proto
    orderfill/documents/v1/documents.proto
    orderfill/files/v1/files.proto
    orderfill/identity/v1/identity.proto
    orderfill/jobs/v1/jobs.proto
    orderfill/matching/v1/matching.proto
  services/
    audit-service/
    brand-service/
    calculation-service/
    document-service/
    file-service/
    gateway-service/
    identity-service/
    job-service/
    matching-service/
    passkey-service/
    twofa-service/
```

Each service has its own `go.mod`, `Dockerfile`, README, config, transport,
storage, migrations when it owns durable data, and tests. `backend/pkg` is only
for infrastructure primitives that are boring and stable. No domain model,
business rule, DTO, or service-specific repository belongs in `pkg`.

## Service responsibilities

### gateway-service

The only externally reachable backend service.

Owns:

- public HTTP API;
- OpenAPI schema served to frontend and humans;
- cookies, CSRF, CORS, security headers;
- REST to gRPC mapping;
- authentication middleware;
- route-level permission gates;
- request IDs, logging, compression, error presentation;
- response aggregation for UI convenience.

Does not own users, jobs, files, Excel logic, matching, brand rules, or audit
storage. It may orchestrate several internal calls for one frontend request, but
it does not persist business state.

### identity-service

Owns the account and tenant core.

Owns:

- companies;
- users;
- roles and permissions;
- password login;
- sessions;
- invites and access reset;
- token validation;
- user lifecycle: active, disabled or later frozen/deleted.

Does not own TOTP secrets or WebAuthn credentials. It calls `twofa-service` and
`passkey-service` during login flows, and it is the only service allowed to mint
sessions.

### twofa-service

Owns TOTP as a separate security domain.

Owns:

- TOTP setup, enable, disable;
- encrypted TOTP secrets;
- recovery codes;
- login-time code verification;
- per-user verification lockout/rate-limit state.

It has no public HTTP surface. `gateway-service` calls it for user-facing
management. `identity-service` calls it during login verification.

### passkey-service

Owns WebAuthn/FIDO2 credentials and ceremonies.

Owns:

- passkey registration begin/finish;
- passkey login begin/finish;
- credential storage;
- challenge state;
- sign counter, backup eligibility, backup state, last used metadata.

It never creates sessions. A successful passkey assertion returns a verified
user id to `identity-service`; `identity-service` then mints the session.

### job-service

Owns workflow state.

Owns:

- job creation;
- job status and progress;
- export history;
- job ownership and access checks;
- report metadata;
- generated file metadata;
- manual edit submission;
- queue publishing for document work;
- accepting document worker progress, completion, and failure.

It does not read or write Excel files. It does not store file bytes. It does not
call matching directly. It treats document processing as asynchronous work.

### file-service

Owns binary file lifecycle.

Owns:

- upload sessions;
- object storage abstraction;
- input and output file metadata needed for access;
- download authorization hooks from `job-service`;
- archive creation or archive manifests;
- future resumable/chunked uploads.

At the start it may be thin over S3/MinIO, but it is a real service because file
handling, download policy, and upload limits will grow separately from jobs and
Excel processing.

### document-service

Owns workbook mechanics and document assembly.

Owns:

- reading `.xlsx` and `.xlsm`;
- writing output workbooks;
- preview snapshot creation;
- order-fill document assembly;
- north-merge document assembly;
- finalization after manual edits;
- worker process consuming document tasks.

The service should contain two binaries:

```text
document-service/
  cmd/document-api/
  cmd/document-worker/
```

`document-api` exposes internal gRPC for synchronous document metadata or
controlled operations. `document-worker` consumes queued work and calls
`job-service` to publish progress and results.

`document-service` does not decide whether two product rows are the same item.
It converts workbooks into structured rows, calls `matching-service`, applies
brand policy from `brand-service`, then writes files.

### matching-service

Owns product identity decisions.

Owns:

- article normalization;
- product name normalization;
- exact article matching;
- fuzzy name matching;
- duplicate detection;
- confidence scoring;
- candidate ranking;
- suspicious match reasons;
- future company dictionaries and synonym overrides;
- future learning from approved manual corrections;
- future ML or embedding-backed matching.

It accepts structured source items and blank items, not Excel workbooks. It may
receive brand and company context, but it must not import or depend on workbook
code.

### brand-service

Owns supplier and brand policy.

Owns:

- brand catalog;
- blank variants such as HOME and PROFF;
- quantity rounding rules;
- minimum quantity policy;
- column detection policy;
- brand-specific validation;
- future company-specific brand overrides;
- versioning of rule changes.

`document-service` asks for policy by brand and job context. `matching-service`
may receive normalized policy hints, but must not create a dependency cycle with
`brand-service`.

### calculation-service

Owns business math and replenishment decisions.

Owns:

- ABC analysis;
- category assignment such as A+, A, B, C, and New;
- average demand calculation;
- delivery coefficients;
- target stock;
- recommended order quantity;
- city-specific calculation rules such as Urengoy;
- North allocation between Tyumen stock, transfers, and supplier order;
- supplier unit conversion;
- recalculation after manual edits;
- future forecasting, seasonality, anomaly detection, and ML recommendations.

It accepts normalized product, stock, sales, city, and brand-policy inputs. It
does not read Excel cells and does not write workbooks. Excel formula
preservation remains in `document-service`; business formulas move here.

### audit-service

Owns the audit trail.

Owns:

- login/logout/security events;
- user, company, role, and permission changes;
- job view/download/admin events;
- file download events;
- append-only audit storage;
- later database-trigger capture and checkpoint digests if required.

Core product flows should not synchronously fail because audit is temporarily
unavailable. Events should go through a durable queue or outbox so they can be
replayed. Security-critical mutations may require stronger audit guarantees once
compliance requirements are explicit.

## Future services

These are not in the first backend v2 runtime set, but the architecture must not
block them.

```text
integration-1c-service
  imports and synchronizes 1C data into normalized source models
```

`document-service`, `matching-service`, and `calculation-service` must work with
normalized item models so Excel and 1C can become interchangeable data sources.

## Contracts

Frontend-facing contract:

```text
backend/services/gateway-service/api/openapi.yaml
```

Internal contracts:

```text
backend/proto/orderfill/*/v1/*.proto
```

Rules:

- frontend never calls internal services;
- internal services do not expose host ports;
- internal DTOs are generated from protobuf;
- public REST schemas are generated or validated from OpenAPI;
- queue messages are versioned and documented;
- breaking protocol changes require expand/migrate/contract rollout;
- no service imports another service's `internal/...`;
- no service writes another service's owned tables.

## Data ownership

One physical Postgres instance is acceptable for local development and early
production, but ownership is logical and strict.

```text
identity-service owns:
  companies
  users
  roles
  permissions
  sessions metadata if durable
  invites

twofa-service owns:
  twofa_credentials
  twofa_recovery_codes

passkey-service owns:
  passkey_credentials
  passkey_challenges if durable

job-service owns:
  jobs
  job_events
  job_reports
  job_file_refs

file-service owns:
  upload_sessions
  file_objects metadata
  archive_manifests

matching-service owns:
  match_dictionaries
  match_overrides
  approved_match_feedback

brand-service owns:
  brands
  brand_rule_versions
  brand_column_policies
  brand_quantity_policies

calculation-service owns:
  calculation_rule_versions later
  calculation_runs later
  forecast_models later
  forecast_results later

audit-service owns:
  audit_events
  audit_checkpoints later
```

Shared database access does not mean shared ownership. A service can read another
service's data only through that service's gRPC API or through a deliberately
designed read model.

## Main flows

### Password login with optional TOTP

```text
frontend
  -> gateway-service
  -> identity-service Login
       -> twofa-service IsEnabled
       -> twofa-service Verify when challenge is completed
       -> identity-service mints session
       -> audit-service via event/outbox
```

`identity-service` owns failed password throttling. `twofa-service` owns failed
code throttling. Session tokens are opaque and stored server-side.

### Passkey login

```text
frontend
  -> gateway-service
  -> identity-service PasskeyLoginBegin
       -> passkey-service BeginLogin
  -> gateway-service
  -> identity-service PasskeyLoginFinish
       -> passkey-service FinishLogin returns verified user id
       -> identity-service mints session
```

Only `identity-service` can mint sessions.

### Order-fill job

```text
frontend
  -> gateway-service
  -> file-service upload source and blanks
  -> job-service create order_fill job
  -> queue
  -> document-worker
       -> file-service reads input bytes
       -> document-service parses workbooks into structured rows
       -> brand-service returns brand policy
       -> matching-service matches source rows to blank rows
       -> calculation-service calculates recommendations and inserted quantities
       -> document-service writes draft outputs and preview snapshots
       -> file-service stores outputs
       -> job-service records needs_review, report, files
```

Manual edits:

```text
frontend
  -> gateway-service
  -> job-service submit edits
  -> queue
  -> document-worker finalize
  -> job-service records completed or failed
```

### North merge

```text
frontend
  -> gateway-service
  -> file-service upload city blanks and optional Tyumen source
  -> job-service create north_merge job
  -> queue
  -> document-worker
       -> document-service parses city needs
       -> brand-service returns North policy
       -> matching-service deduplicates/matches shared items
       -> calculation-service allocates Tyumen stock, transfers, and supplier need
       -> document-service builds plan, transfer files, supplier order files
       -> job-service records needs_review or completed state
```

North is a first-class processor, not a special case hidden inside order-fill.

## Resilience rules

Every remote call needs an explicit deadline budget. The gateway propagates the
request context and derives shorter sub-budgets for internal calls.

Rules:

- no unbounded retries;
- retries only where the operation has stable idempotency identity;
- async document work uses queue retry ownership, not gateway retry loops;
- job creation uses an idempotency key;
- file upload finalize is content-addressed or idempotent by upload id;
- audit events are durable through queue/outbox;
- health checks are cheap;
- liveness means process is alive;
- readiness means dependencies needed for useful work are available;
- services reject overload early instead of building unbounded queues in memory;
- background workers have explicit concurrency limits.

Interactive HTTP requests should stay short. Excel processing, archive building,
matching over large inputs, and integration sync run asynchronously.

## Code style

Every service follows the same internal shape:

```text
cmd/<service>/
internal/
  bootstrap/
  config/
  domain/
  migrate/
  service/
  storage/
  transport/
```

For packages with multiple operations:

```text
internal/service/jobs/
  jobs.go
  create.go
  get.go
  list.go
  submit_edits.go
  complete.go
```

Rules:

- package root file owns interfaces, constructor, and type wiring;
- one operation per file;
- interfaces live with the consumer;
- central error mapper per transport;
- domain packages do not import transport, storage, queue, or config;
- storage packages do not contain business decisions;
- `cmd` and `bootstrap` own process wiring only;
- no catch-all `util`, `common`, `models`, or `interfaces` packages;
- public behavior, config keys, protobuf fields, OpenAPI schemas, queue
  messages, and database migrations are compatibility surfaces.

## Migration stance

The implementation can happen in phases, but the target architecture is not a
temporary halfway design. New backend work should move toward this layout rather
than expanding the current `api-service`.

High-level sequencing:

1. Create backend workspace, shared infrastructure packages, proto module, and
   service skeletons.
2. Move frontend-facing API into `gateway-service`.
3. Build identity, twofa, and passkey services.
4. Build job and file services.
5. Build matching, brand, and calculation services.
6. Rework document-service into document API plus worker.
7. Move audit to a dedicated service and durable event path.
8. Retire the old `api-service` once gateway calls the new services.

Detailed implementation plans will be written separately per phase.

## Invariants

- Only `gateway-service` is externally reachable.
- Only `identity-service` mints sessions.
- Only `job-service` owns job state.
- Only `file-service` owns binary file lifecycle.
- Only `document-service` reads and writes workbook formats.
- Only `matching-service` decides item identity.
- Only `brand-service` owns supplier policy.
- Only `calculation-service` owns business formulas and replenishment math.
- Only `audit-service` owns audit storage.
- Services communicate through contracts, not package imports or table writes.
