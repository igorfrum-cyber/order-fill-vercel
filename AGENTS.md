# AGENTS.md — Order Fill

## Branch workflow

- `artemch` is the canonical development branch. Create `feat/<slug>` from `artemch` before any edit.
- **Never commit to** `artemch`, `dev`, `main`, or `igorfrum` directly. Merge to `artemch` only when explicitly asked.
- `main` is a prototype sandbox (old browser-based Excel); do not merge it into `artemch`.
- `dev` is the deploy target; CI auto-deploys to self-hosted on push to `dev`.

## Setup (first time)

```bash
cp .env.example .env
npm ci --prefix frontend
make hooks          # enables pre-commit / pre-push git hooks
make up             # starts Compose stack (frontend, gateway, all services, Postgres, Redis, MinIO)
```

Web UI: `http://127.0.0.1:3200`. Gateway health: `http://127.0.0.1:8080/healthz`.

## Required commands

| Command | What it does |
|---|---|
| `make up` | Build and start full Compose stack |
| `make down` | Stop containers, keep named volumes |
| `make logs` | Follow all container logs |
| `make verify` | Deterministic local pre-commit gate |
| `make lint` | Toolchain + frontend lint + Go lint/security/tidy |
| `make test` | Load runner + frontend tests + all Go module tests |
| `make docs` | Sync and verify docs; fails if README tables drifted |
| `make docs-sync` | Regenerate env/RPC/HTTP tables in READMEs from code |
| `make contracts` | Verify HTTP routes match OpenAPI + Buf lint |
| `make security` | `govulncheck` all Go modules (needs network) |
| `make -C backend proto-gen` | Regenerate Go protobuf code from `.proto` sources |
| `make hooks` | Enable git hooks (run once per clone) |
| `make https` | Generate TLS certs + deploy production overlay with Caddy |
| `make lan-https` | Local HTTPS for passkey testing on phone |
| `make load-order-fill ARGS='…'` | Load test against public API |

**Always run `make verify` before committing.** Never use `--no-verify`. Pre-push runs it automatically too.

## Go workspace

- Go workspace is at `backend/go.work` — unifies `backend/pkg`, `backend/proto`, and all 11 services.
- Go version: **1.26.7** (pinned in go.work and toolchain scripts).
- When running tests in a single module, **must** set `GOWORK=off`:
  ```bash
  cd backend/services/gateway-service
  GOWORK=off go test ./...
  ```
- `scripts/verify-go.sh` sets `GOWORK=off` per module and handles toolchain auto-download.

## Frontend

- React 19 + Vite 7. UI components in `frontend/src/ui/` (`widgets.jsx`, `chrome.jsx`).
- Dev server: `npm run dev --prefix frontend` (Vite on `127.0.0.1:3200`).
- Test commands:
  - `npm run test --prefix frontend` — unit tests via `node --test`
  - `npm run test:component --prefix frontend` — Vitest + Testing Library
  - `npm run test:e2e --prefix frontend` — Playwright (install Chromium first: `npm run test:e2e:install --prefix frontend`)
  - `npm run test:ui --prefix frontend` — unit + component + Playwright. **Run after any UI change in the same agent session.** Playwright opens Chromium on screen locally.
  - `npm run test:e2e:real --prefix frontend` — Playwright against live stack with real Excel from `testdata/private`. Requires `REAL_E2E_OWNER_LOGIN` and `REAL_E2E_OWNER_PASSWORD` env vars.
- `npm run verify --prefix frontend` = lint + test + component test + build.

## Architecture invariants

- **Browser never parses Excel.** Document processing is backend-only.
- `gateway-service` is the **only** browser-facing backend. Keep it thin: validate HTTP, derive actor from session, call service owners.
- Internal communication is **protobuf/gRPC** or **Redis Stream** (`order-fill:jobs`). Never import another service's `internal/` packages.
- `file-service` is the **sole** object-storage boundary. Never bypass it.
- Matching, brand, and calculation services **never** read Excel or access object storage.
- Never execute macros from uploaded `.xlsm` files.
- `document-api` exists as an internal synchronous API but gateway does **not** currently call it.

## Microservices (all under `backend/services/`)

| Service | Port | Responsibility |
|---|---|---|
| `gateway-service` | HTTP 8080, gRPC clients | Public HTTP API, CORS/CSRF, session edge |
| `identity-service` | gRPC 9091 | Users, companies, roles, invites, sessions |
| `twofa-service` | gRPC 9092 | TOTP, backup codes, rate state |
| `passkey-service` | gRPC 9093 | WebAuthn credentials and ceremonies |
| `job-service` | gRPC 9094 | Jobs, status, reports, Redis publishing |
| `file-service` | gRPC 9095 | File metadata, S3/MinIO boundary |
| `document-service` | gRPC 9096 | Excel parsing, preview, document-worker |
| `matching-service` | gRPC 9097 | Item normalization, matching decisions |
| `brand-service` | gRPC 9098 | Brand catalog, detection rules |
| `calculation-service` | gRPC 9099 | Order quantities, edit validation, North planning |
| `audit-service` | gRPC 9100 | Audit event persistence |

## Protobuf / gRPC contracts

- Source `.proto` files: `backend/proto/orderfill/`.
- Generated Go code: `backend/proto/gen/go/` — **committed but must be regenerated**, never hand-edited.
- After changing `.proto`: `make -C backend proto-gen`, then update all producer/consumer services, commit generated code, run Buf lint and affected module tests.
- Avoid breaking field renumbering or reuse. Breaking changes need an explicit migration plan.

## Environment & secrets

- `.env` is **gitignored**. Always `cp .env.example .env` to get started.
- Local Compose defaults: `APP_ENV=local`, `GRPC_TLS_MODE=insecure`, `SESSION_COOKIE_SECURE=false`.
- **Production requires**: `APP_ENV=production`, `GRPC_TLS_MODE=mtls`, `SESSION_COOKIE_SECURE=true`, `WEBAUTHN_RP_ID` set, `TWOFA_MASTER_KEY` ≥32 bytes non-default, `WORKER_TOKEN` ≥16 bytes non-default, explicit HTTPS `API_ALLOWED_ORIGINS`.
- Never commit `.env`, passwords, TOTP master keys, WebAuthn ceremony data, TLS private keys, or `testdata/private/` contents.

## Testing quirks

- Go tests use `GOWORK=off` per module. CI runs with `-race -shuffle=on`.
- Backend services use **in-memory adapters in local mode** — this is dev-only, not production config.
- Readiness endpoints (`/readyz`) are intentionally shallow — they don't prove all downstream dependencies are available.
- Frontend Playwright opens Chromium on screen locally. Run outside the sandbox so the window is visible.
- `testdata/private/` is gitignored — contains real commercial fixtures.

## Pre-commit / Pre-push hooks

- `.githooks/pre-commit`: runs `node scripts/sync-docs.mjs --write`, adds updated `backend/services/*/README.md`. On `artemch` branch, also runs `make verify`.
- `.githooks/pre-push`: runs `make verify` on every push.
- `make hooks` must be run once per clone to enable them.

## Documentation sync

- `node scripts/sync-docs.mjs --write` regenerates env/RPC/HTTP tables in service READMEs from `config.go`, proto, and router.
- `node scripts/verify-docs.mjs` checks local links and required README sections.
- `make verify` fails if generated README tables weren't committed.

## Source-of-truth order (when docs conflict)

1. Runtime code, `backend/deploy/docker-compose.yml`, canonical OpenAPI, protobuf sources
2. README inside the affected service
3. `README.md`, `docs/ARCHITECTURE.md`, `docs/service-boundaries.md`
4. `docs/plans/` (design/history only)

## Skill references

- `.cursor/skills/order-fill-work/SKILL.md` — mandatory for any code change
- `.cursor/skills/order-fill-docs/SKILL.md` — for documentation changes
- `order-fill-docs` skill — load before reading service READMEs, proto, OpenAPI
- `use-modern-go` + `golang-how-to` — load for any Go file changes
- `systematic-debugging` — load for unexpected failures
- `finishing-a-development-branch` — use when user asks to land, base `artemch`
