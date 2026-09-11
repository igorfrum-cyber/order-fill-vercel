# CLAUDE.md

Instructions for Claude Code, Cursor, and other coding agents working in this
repository.

## Read this first

Order Fill is an active React + Go microservice application. The browser does
not parse or rewrite Excel files; document processing belongs to the backend.

Use sources in this order when they disagree:

1. Runtime code, `backend/deploy/docker-compose.yml`, canonical OpenAPI, and
   protobuf source files.
2. The README inside the affected service.
3. `README.md`, `docs/ARCHITECTURE.md`, and `docs/service-boundaries.md`.
4. Files under `docs/plans/`, which are design and implementation history.

Active Go services live only under `backend/services/`. Do not restore or
reference the retired root-level `services/` implementation.

## Current runtime

```text
browser
  |
  v
frontend/nginx --HTTP /api--> gateway-service
                                  |
                                  +--gRPC--> identity / twofa / passkey
                                  +--gRPC--> jobs / files / audit
                                                   |       |
                                                   v       v
                                             Redis Stream  S3/MinIO
                                                   |
                                                   v
                                            document-worker
                                              |   |   |
                                              v   v   v
                                          matching brand calculation

stateful services ---------------------------> PostgreSQL
```

- `gateway-service` is the only browser-facing backend.
- Internal business APIs use protobuf/gRPC.
- `job-service` publishes `order-fill:jobs`; `document-worker` consumes it.
- `file-service` is the object-storage boundary.
- `document-api` exists as an internal synchronous API, but gateway does not
  currently call it.
- Local Compose provides PostgreSQL, Redis, and MinIO.

## Ownership

| Area | Owner |
| --- | --- |
| Browser UI, routing, state, rendering | `frontend/` |
| Public HTTP, cookies, CORS/CSRF, request mapping | `gateway-service` |
| Users, companies, roles, invites, sessions | `identity-service` |
| TOTP credentials and rate state | `twofa-service` |
| WebAuthn credentials and ceremonies | `passkey-service` |
| Jobs, status, report state, queue publishing | `job-service` |
| Binary objects and file metadata | `file-service` |
| Excel/OOXML parsing, writing, preview, worker pipeline | `document-service` |
| Product matching decisions and match reasons | `matching-service` |
| Brand catalog, detection, and policies | `brand-service` |
| Quantity calculations and North planning | `calculation-service` |
| Audit event persistence | `audit-service` |
| Internal gRPC contracts | `backend/proto/` |
| Shared transport/infrastructure helpers | `backend/pkg/` |

Read the relevant service README before changing it. Each README documents its
actual RPCs, environment variables, dependencies, and known limitations.

## Architectural invariants

- Keep workbook parsing, formulas, styles, and file generation out of the
  browser and gateway handlers.
- Keep gateway thin: validate HTTP once, derive the actor from the session, call
  service owners, and map the response.
- Do not import another service's `internal/` packages. Cross-service behavior
  goes through protobuf/gRPC or the Redis job contract.
- Do not let matching, brand, or calculation services read Excel or access
  object storage.
- Do not bypass `file-service` for persistent object operations.
- Keep calculations deterministic and explainable unless an explicit task asks
  for an experiment.
- Preserve current brand-specific behavior unless the change explicitly alters
  the product specification.
- Never execute macros from uploaded `.xlsm` files.

## Before editing

1. Load `.cursor/skills/order-fill-work/SKILL.md`. If HEAD is `artemch`, `dev`,
   `main`, or `igorfrum`, create `feat/<slug>` from `artemch` first. Do not
   commit those four branches. Merge into `artemch` only when asked.
2. Inspect `git status` and preserve unrelated changes.
3. Read the service README, domain code, transport, config, tests, and contract
   involved in the change. Load `order-fill-docs` and the Go/UI skills the
   files require. Friend's `main` is product intent, not a merge source.
4. Confirm which component owns the behavior; do not patch around an ownership
   boundary in gateway or frontend. A new microservice or routed screen needs
   an explicit yes before you create it.
5. Prefer a focused test that demonstrates the requested behavior before or
   with the implementation.

Do not edit generated protobuf files directly. Do not rewrite applied migration
files; add a new migration.

## Change checklists

### Public HTTP API

Update together:

- `backend/services/gateway-service/internal/transport/httpapi/`;
- `backend/services/gateway-service/api/openapi.yaml`;
- gateway tests;
- `frontend/src/api/` and its tests when the browser contract changes;
- gateway README when routes, limits, configuration, or behavior change.

`packages/contracts/openapi.yaml` is only a compatibility pointer. Do not place
the full contract there. `node scripts/verify-contracts.mjs` requires runtime
routes and OpenAPI operations to match exactly.

### Internal gRPC API

1. Change the source under `backend/proto/orderfill/`.
2. Run `make -C backend proto-gen`.
3. Commit generated changes under `backend/proto/gen/go/`.
4. Update every producer and consumer in the same change.
5. Run Buf lint and all affected module tests.

Avoid breaking field renumbering or reuse. Breaking changes require an explicit
migration plan; do not silence the CI check casually.

### Configuration

When adding or changing an environment variable, update:

- the service's `internal/config` package and tests;
- `backend/deploy/docker-compose.yml` when Compose uses it;
- `.env.example` when operators set it;
- the service README and root README when it affects deployment.

Keep production fail-fast behavior. Local in-memory fallbacks are development
tools, not production defaults.

### Database and queue state

- Add migrations; do not mutate an existing migration that may have run.
- Keep tenant and actor authorization explicit at service boundaries.
- Treat job state plus Redis publication as non-atomic unless an outbox is
  implemented.
- Remember that the current worker acknowledges a Redis message after handler
  errors; do not describe this path as automatic retry.
- Readiness endpoints are intentionally shallow in several services. Do not
  assume `/readyz` proves every downstream dependency is available.

### Documentation

Before writing or updating documentation, load the `order-fill-docs` skill
(`.cursor/skills/order-fill-docs/SKILL.md`). For Go godoc, also follow
`samber/cc-skills-golang@golang-documentation`.

Update documentation in the same change as behavior. At minimum, keep the root
README, affected service README, contracts, `.env.example`, and architecture
documents consistent. `node scripts/sync-docs.mjs --write` regenerates env/RPC/HTTP
lists in service READMEs from `config.go`, protobuf and the gateway router.
`node scripts/verify-docs.mjs` checks local links and the required README sections.
`make verify` runs the sync and fails if the generated lists were not committed.

After a frontend UI or user-behavior change, run `npm run test:ui --prefix frontend`
in the same agent session before claiming the work is done. Locally Playwright
opens Chromium on screen; run the command outside the sandbox so the window is
visible. Read the output. If tests fail, fix the root cause and re-run until
green. Do not replace that suite with exploratory Playwright MCP clicks.

## Setup and commands

```bash
cp .env.example .env
npm ci --prefix frontend
make up
```

- Web UI: `http://127.0.0.1:3200`
- Gateway health: `http://127.0.0.1:8080/healthz`

Common commands:

```bash
make up                 # build and start the Compose stack
make logs               # follow container logs
make down               # stop containers; keep named volumes
make docs               # validate documentation and local links
make contracts          # compare HTTP/OpenAPI and run Buf lint
make test               # frontend and all Go module tests
npm run test:ui --prefix frontend         # unit + component + Playwright after a UI change
npm run test:component --prefix frontend  # Vitest + Testing Library
npm run test:e2e --prefix frontend        # Playwright UI tests
make lint               # frontend + Go lint/security/tidy checks
make verify             # deterministic local pre-commit gate
make security           # govulncheck for every active Go module
```

Go dependencies are downloaded through Go modules. This repository does not
use a `vendor/` directory. `scripts/verify-go.sh` sets `GOWORK=off` per module
and defaults `GOTOOLCHAIN=auto`, so the pinned Go toolchain may be downloaded
when the locally installed Go version is older.

Run a focused module test from its directory:

```bash
cd backend/services/gateway-service
GOWORK=off go test ./...
```

## Required verification

Before every commit, run:

```bash
make verify
```

It checks toolchain pins, documentation, HTTP/OpenAPI drift, protobuf lint,
frontend lint/tests/build, every Go module, security linting, module tidiness,
and both Compose entrypoints.

CI adds two expensive checks to the same scripts and module list:

- Go tests with `-race -shuffle=on`;
- `govulncheck ./...` for every active Go module.

Run `make security` locally before dependency, authentication, authorization,
networking, file-processing, or release-related changes. It queries the Go
vulnerability database and therefore requires network access.

Do not bypass a failed gate with `--no-verify`. Fix the problem or document a
genuine external blocker.

## CI structure

`.github/workflows/verify.yml` has independent layers:

- `Documentation and contracts` — toolchain pins, docs, OpenAPI drift, Buf;
- `Frontend` — install, load-runner tests, lint, unit tests, component tests, build;
- `Go quality` — vet, lint, SAST, and module tidiness for all modules;
- `Go test · <module>` — independent race/shuffled tests per module;
- `Go vulnerability scan` — reachable vulnerability checks for all modules;
- `Docker Compose` — backend and root configuration validation;
- `Required checks` — one aggregate result for branch protection.
- `Deploy self-hosted` — only on push to `dev` after required checks; pulls
  and rebuilds Compose on the laptop runner. Never runs on pull requests.

The workflow uses least-privilege read permissions, cancels obsolete runs for
the same ref, and keeps matrix failures independent. The self-hosted runner
must not pick up fork pull requests. Local git hooks run `make verify` on
`artemch` commits and on every push.

## Security and data handling

- Never commit `.env`, passwords, session tokens, TOTP master keys, WebAuthn
  ceremony data, TLS private keys, or object-storage credentials.
- Never commit real commercial workbooks. Private fixtures belong under
  `testdata/private/`, which is ignored.
- Treat internal gRPC as a trusted boundary only when the deployment provides
  network isolation and TLS/mTLS. Several RPCs trust actor IDs supplied by the
  caller.
- Production CORS origins must be explicit HTTPS origins; wildcard is local
  only.
- Production cookies must be secure, passkeys require the correct RP ID, and
  `TWOFA_MASTER_KEY` must be stable, non-default, and at least 32 bytes.
- Do not log file contents, credentials, raw recovery codes, or private keys.

## Generated and ignored content

Do not commit or review as source:

- `node_modules/`, `frontend/node_modules/`;
- `dist/`, `frontend/dist/`, `.vercel/`;
- `.cache/`, `.gocache/`, `test-output/`;
- `testdata/private/`;
- `.worktrees/`.

Generated protobuf code under `backend/proto/gen/go/` is committed, but must be
regenerated from `.proto` sources rather than hand-edited.
