# Dev-since-Sept13 Follow-up Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the Critical/Important gaps from the 2026-09-13…09-17 `dev` review without expanding product scope: drop the committed inbound binary, make inbound migrations and ACL match the shipped UI/docs, stop CloudMailin retry storms on bad payloads, restore CLAUDE.md ownership, and delete leftover over-engineering.

**Architecture:** Keep the existing owners. `inbound-service` already landed as an isolated mail contour behind `gateway-service`; do not add another microservice or routed screen. Fix the contour in place (gateway ACL + inbound gRPC + migrations + docs). Brand/identity/document changes stay in those services. Work only on `feat/<slug>` from `artemch`.

**Tech Stack:** Go 1.26.7 (`GOWORK=off` per module), gateway OpenAPI, protobuf already generated (do not hand-edit `backend/proto/gen/go/`), React UI in `frontend/src/ui/`, Compose, `make verify`.

---

## Context for the implementer

Review window: `BASE_SHA=1e6865f12aa75a4ba8363d49415e021cae85241d` … `HEAD_SHA=fd6ca08d1c26a19ada0b6762624d76b2e7c26dfa` (33 commits on `dev`, 19 non-merge). Verdict: **mixed** — one new service (`inbound-service`) plus incremental patches to identity/brand/document/gateway/frontend.

Do **not**:
- invent a new microservice or screen
- merge `main` or copy `src/app.js` / workbook parsing into the browser
- commit to `artemch` / `dev` / `main` / `igorfrum`
- silently add webhook rate-limit or `audit-service` inbound events (design leftovers — Task 12 is a stop/decision)

Skills: `@order-fill-work`, `@order-fill-docs`, `@use-modern-go`, `@samber/cc-skills-golang@golang-error-handling`, `@samber/cc-skills-golang@golang-security`, `@samber/cc-skills-golang@golang-grpc`, `@samber/cc-skills-golang@golang-testing`, `@ponytail`.

---

### Task 1: Drop the committed inbound binary and accidental agent config

**Files:**
- Delete: `backend/services/inbound-service/inbound` (28 MB ELF, committed in `70e12cf`)
- Delete: `agent_config.txt` (machine-local Copilot notes, not product)
- Modify: `.gitignore`

**Step 1: Ignore binaries next to cmd packages**

Append to `.gitignore`:

```
backend/services/*/inbound
!backend/services/inbound-service/
agent_config.txt
```

Safer and clearer — add these exact lines:

```
# Accidental `go build` outputs next to sources
backend/services/inbound-service/inbound
agent_config.txt
```

**Step 2: Remove from the index without leaving the file if present**

```bash
git rm -f --ignore-unmatch backend/services/inbound-service/inbound agent_config.txt
```

**Step 3: Confirm git no longer tracks the blob**

```bash
git ls-files backend/services/inbound-service/inbound agent_config.txt
```

Expected: empty.

**Step 4: Commit**

```bash
git add .gitignore
git commit -m "$(cat <<'EOF'
chore: drop accidental inbound binary and agent_config

A 28MB go-build artifact and local agent notes do not belong in the repo.
EOF
)"
```

---

### Task 2: Renumber the colliding inbound `00004` migrations

**Files:**
- Rename: `backend/services/inbound-service/internal/migrate/migrations/00004_inbound_message_body.sql` → `00005_inbound_message_body.sql`
- Keep: `00004_allow_shared_receive_address.sql`
- Test: `backend/services/inbound-service/internal/migrate/migrate.go` (no version table today — still rename so the next person does not treat the prefix as a version)

Current migrator re-executes **every** `*.sql` on each boot (`migrate.Up`). Both `00004_*` files happen to be idempotent (`DROP CONSTRAINT IF EXISTS`, `ADD COLUMN IF NOT EXISTS`), so production that already started will survive a rename. The collision is still a landmine if anyone later switches to golang-migrate.

**Step 1: Write a failing check that filenames have unique numeric prefixes**

Create `backend/services/inbound-service/internal/migrate/migrate_test.go` if missing, otherwise add:

```go
func TestMigrationPrefixesAreUnique(t *testing.T) {
	t.Parallel()
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range entries {
		prefix := strings.SplitN(e.Name(), "_", 2)[0]
		if other, ok := seen[prefix]; ok {
			t.Fatalf("duplicate migration prefix %s: %s and %s", prefix, other, e.Name())
		}
		seen[prefix] = e.Name()
	}
}
```

**Step 2: Run test — expect FAIL on two `00004_` files**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/migrate/ -count=1
```

Expected: FAIL `duplicate migration prefix 00004`.

**Step 3: Rename the body-text file to `00005_inbound_message_body.sql`**

```bash
git mv backend/services/inbound-service/internal/migrate/migrations/00004_inbound_message_body.sql \
  backend/services/inbound-service/internal/migrate/migrations/00005_inbound_message_body.sql
```

Do not edit SQL bodies.

**Step 4: Re-run test — expect PASS**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/migrate/ -count=1
```

**Step 5: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix: give inbound message-body migration a unique prefix

Two 00004_ files would break any versioned migrator and confuse operators.
EOF
)"
```

---

### Task 3: Align company_admin inbound ACL with the UI

**Bug:** Gateway `inboundCompanyReadAllowed` lets `company_admin` read company inbound settings (`inbound.go` around the helper). gRPC `GetCompanyInbound` uses `requireOwner`, which allows only `company_owner` / `platform_admin`. Company admins hitting `/inbound` get a gRPC permission error after the HTTP gate already passed. README already promises owner **or** admin for `GetCompanyInbound`.

**Files:**
- Modify: `backend/services/inbound-service/internal/transport/grpcapi/server.go`
- Test: `backend/services/inbound-service/internal/transport/grpcapi/` (add `server_test.go` if none covers this RPC)
- Gateway already correct — do not add a second check

**Step 1: Failing test**

```go
func TestGetCompanyInboundAllowsCompanyAdmin(t *testing.T) {
	t.Parallel()
	// seed a fake service/store with company "co-1"
	ctx := grpcutil.WithActorRole(t.Context(), "company_admin")
	req := &inboundv1.GetCompanyInboundRequest{
		Meta:      &commonv1.RequestMeta{CompanyId: "co-1"},
		CompanyId: "co-1",
	}
	if _, err := srv.GetCompanyInbound(ctx, req); err != nil {
		t.Fatalf("company_admin of co-1 must read inbound settings: %v", err)
	}
}

func TestGetCompanyInboundRejectsForeignCompany(t *testing.T) {
	t.Parallel()
	ctx := grpcutil.WithActorRole(t.Context(), "company_admin")
	req := &inboundv1.GetCompanyInboundRequest{
		Meta:      &commonv1.RequestMeta{CompanyId: "co-1"},
		CompanyId: "co-2",
	}
	if status.Code(errFrom(srv.GetCompanyInbound(ctx, req))) != codes.PermissionDenied {
		t.Fatal("foreign company_id must be denied")
	}
}
```

Wire `srv` the same way existing grpc tests in this package (or identity-service) do. Use `t.Context()` (modern Go `testing_t_context`).

**Step 2: Run — expect FAIL**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/transport/grpcapi/ -count=1 -run 'TestGetCompanyInbound'
```

**Step 3: Minimal fix**

In `GetCompanyInbound`, replace `requireOwner` with `requireCompanyRead` (already used by `ListMessages` / `GetMessage`). Keep `UpdateCompanyInbound` on `requireOwner` + explicit `company_owner` (platform_admin updating a company address is out of the design table — do not expand it).

**Step 4: Run tests PASS, then module tests**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
```

**Step 5: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix: allow company_admin to read inbound company settings

The UI and README already expose the screen; gRPC still required owner.
EOF
)"
```

---

### Task 4: Stop CloudMailin retry storms on unparseable webhooks

**Bug:** `IngestWebhook` maps every service error to `codes.Internal` (`server.go`). JSON parse failures and `ErrPayloadTooLarge` then become HTTP 5xx from gateway → CloudMailin retries forever. Unknown-address / no-attachments already return `nil` via `saveMinimalMessage` (200) — keep that.

**Files:**
- Modify: `backend/services/inbound-service/internal/transport/grpcapi/server.go`
- Modify: `backend/services/inbound-service/internal/domain/errors.go` if a sentinel for parse errors is missing
- Test: `backend/services/inbound-service/internal/transport/grpcapi/` and/or `internal/service/inbound/inbound_test.go`
- Gateway: `writeGRPCError` already maps InvalidArgument → 400 — confirm, do not reimplement

**Step 1: Failing test on the gRPC mapper**

```go
func TestIngestWebhookInvalidJSONIsInvalidArgument(t *testing.T) {
	t.Parallel()
	ctx := grpcutil.WithWorkerToken(t.Context(), workerToken)
	_, err := srv.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{RawPayload: []byte("{")})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument so CloudMailin stops retrying poison payloads", err)
	}
}
```

**Step 2: Run — FAIL (currently Internal)**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/transport/grpcapi/ -count=1 -run TestIngestWebhookInvalidJSON
```

**Step 3: Map sentinels with `errors.Is` / `errors.AsType` (Go 1.26)**

```go
if err := s.svc.IngestWebhook(ctx, req.GetRawPayload()); err != nil {
    switch {
    case errors.Is(err, domain.ErrPayloadTooLarge):
        return nil, status.Error(codes.InvalidArgument, "payload too large")
    case isParseError(err): // unwrap fmt.Errorf("parse inbound webhook: %w", err)
        return nil, status.Error(codes.InvalidArgument, "invalid payload")
    default:
        return nil, status.Error(codes.Internal, "ingest failed")
    }
}
```

Prefer a domain sentinel `ErrInvalidPayload` from `parseWebhookPayload` instead of string matching.

Do **not** return 200 for parse failures — 400 is enough for CloudMailin to stop. Store nothing.

**Step 4: Module tests PASS**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
cd backend/services/gateway-service && GOWORK=off go test ./internal/transport/httpapi/ -count=1 -run Inbound
```

**Step 5: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix: reject unparseable inbound webhooks as invalid argument

Internal errors made CloudMailin retry poison JSON forever.
EOF
)"
```

---

### Task 5: Do not persist letter bodies on error-status ingest

**Bug:** README (`inbound-service/README.md` «содержимое не хранится») and the design doc say unknown-address / mismatch / too-large / no-attachments keep metadata only. `saveMinimalMessage` now writes `BodyText` / `BodyHTML`. That leaks 1С letter content into rows that platform_admin deliveries might later grow into, and contradicts the shipped docs.

**Files:**
- Modify: `backend/services/inbound-service/internal/service/inbound/inbound.go` (`saveMinimalMessage`)
- Test: `backend/services/inbound-service/internal/service/inbound/inbound_test.go`

**Step 1: Extend the existing unknown-sender test**

Assert `saved.BodyText == "" && saved.BodyHTML == ""` (and attachments nil). If the current test only checks status, add those two assertions — it should FAIL.

**Step 2: Run FAIL**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/service/inbound/ -count=1 -run Unknown
```

**Step 3: Stop copying body fields in `saveMinimalMessage`**

Leave `saveMinimalMessage` signature if callers pass bodies; just do not assign them. Successful `StatusReceived` path keeps bodies.

**Step 4: PASS + commit**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
git commit -m "$(cat <<'EOF'
fix: omit inbound letter bodies on error ingest

Error rows are diagnostic metadata; storing HTML/text contradicted the README.
EOF
)"
```

---

### Task 6: Strip subject/from from platform_admin delivery feed

**Bug:** Design: platform_admin sees counters and status, not subject/sender. `ListDeliveries` currently returns full `protoMessageSummary` minus bodies, so `subject` and `envelope_from` still go out. Gateway `presentInboundMessage` forwards them. `InboundScreen` PlatformPanel table likely renders them.

**Files:**
- Modify: `backend/services/inbound-service/internal/transport/grpcapi/server.go` (`ListDeliveries`)
- Modify: `backend/services/gateway-service/internal/transport/httpapi/inbound.go` if the JSON still has the fields (zero them)
- Test: inbound grpc test + `frontend/src/ui/admin/InboundScreen.ui.test.jsx` if the platform table currently expects from/subject — update the assertion to statuses only
- OpenAPI: if the deliveries schema documents those fields as required, make them optional / omit

**Step 1: Failing Go test** that `ListDeliveries` summaries have empty `Subject` and `EnvelopeFrom`.

**Step 2: Clear those fields in `ListDeliveries` after `protoMessageSummary` (same pattern as bodies).**

**Step 3: Gateway/OpenAPI/UI follow the smaller DTO — do not add a new route.**

**Step 4:**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
cd backend/services/gateway-service && GOWORK=off go test ./internal/transport/httpapi/ -count=1 -run Inbound
npm run test:ui --prefix frontend
```

Playwright locally needs a visible Chromium — run outside the sandbox.

**Step 5: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix: hide inbound subject and sender from platform delivery feed

Platform admins diagnose delivery, they do not read tenant mail headers.
EOF
)"
```

---

### Task 7: Fail-fast inbound production config and refuse a nil store

**Bugs:**
- `inbound-service` `Config.Validate` outside local requires only `INBOUND_DATABASE_URL` + worker token. Empty/default MinIO keys (`minioadmin`) still start.
- `bootstrap.Run`: if `DatabaseURL == ""`, it logs and `return nil` — process exits 0 without serving. Local Compose always sets the URL; a mis-set env looks like a successful boot.

**Files:**
- Modify: `backend/services/inbound-service/internal/config/config.go`
- Test: `backend/services/inbound-service/internal/config/config_test.go` (create if missing)
- Modify: `backend/services/inbound-service/internal/bootstrap/bootstrap.go`

**Step 1: Test production rejects default S3 keys** (mirror `gateway-service` `ValidateInboundTokens`).

**Step 2: In `Validate`, when `!localEnv`: require non-empty endpoint/bucket and non-default access/secret.**

**Step 3: If `store == nil` after the postgres branch, `return fmt.Errorf("inbound store is required")` instead of `return nil`.** Local without URL is not a supported mode once Compose always supplies it — match other stateful services.

**Step 4:**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./internal/config/ ./internal/bootstrap/ ./... -count=1
```

**Step 5: `node scripts/sync-docs.mjs --write` then fill any purpose stubs in `backend/services/inbound-service/README.md`.** Commit docs with the config change.

---

### Task 8: Unique sender routing (shared address is allowed; shared sender is not)

**Bug:** `00004_allow_shared_receive_address` dropped UNIQUE on `receive_address` (intentional). `sender_email` has no UNIQUE. `GetCompanyByAllowedSender` is `QueryRow` — two companies with the same 1С mailbox silently steal mail.

**Files:**
- Create: `backend/services/inbound-service/internal/migrate/migrations/00006_unique_sender_email.sql`
- Modify: `backend/services/inbound-service/internal/storage/postgres/store.go` only if the upsert error must map to `domain.ErrInvalid`
- Test: service test that a second company cannot take an existing `sender_email`

**SQL:**

```sql
CREATE UNIQUE INDEX IF NOT EXISTS company_inbound_sender_email_lower_uidx
    ON company_inbound (LOWER(sender_email))
    WHERE sender_email <> '';
```

Empty sender is already rejected in `UpdateCompanyInbound`. If a deployed DB already has duplicates, this migration fails loudly — that is the correct ceiling; do not silently `DISTINCT ON` pick a winner. Document the operator step in the inbound README «Эксплуатация»: resolve duplicate `sender_email` before deploy.

**Commands:**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
```

Commit with the migration + mapping + README sentence.

---

### Task 9: Restore ownership docs (no new service)

`inbound-service` and routes `/inbound`, `/brand-rules` already shipped. This task only makes CLAUDE.md / architecture tables tell the truth. Do not add screens.

**Files:**
- Modify: `CLAUDE.md` ownership table + runtime diagram (add inbound-service gRPC `:9101`, own Postgres + bucket)
- Modify: `docs/ARCHITECTURE.md` if the inbound box is incomplete
- Modify: `docs/service-boundaries.md` — already mentions inbound; keep the explicit file-service exception, do not walk it back
- Modify: `backend/services/inbound-service/README.md` — RPC table still says whitelist/`company_admin` for GetCompanyInbound; align to **sender_email** (one mailbox) after Task 3
- Modify: `docs/plans/2026-09-15-inbound-mail-integration-design.md` only a one-line “as-built” note: matching is by `envelope.from` / `sender_email`, not `envelope.to`. Do not rewrite the design as if the old API paths (`/inbound-address`) exist.

**Step:** `node scripts/sync-docs.mjs --write && node scripts/verify-docs.mjs`

Fill purpose stubs; do not hand-edit non-purpose columns inside `<!-- docs-sync:* -->`.

Commit: `docs: record inbound-service in CLAUDE.md owners`

---

### Task 10: Ponytail deletions (over-engineering only)

Do these as one commit after the correctness tasks. Do not extract a shared MinIO package — isolation of inbound S3 was an explicit product decision.

**Delete / shrink:**

1. `backend/services/inbound-service/internal/service/inbound/inbound.go` — field `inboundKey` on `Service` is unused after `New`. Drop the field and the constructor arg; update `bootstrap.go`.
2. `UpdateSettings(..., actorUserID string)` — unused. Drop the parameter through grpc mapper.
3. `frontend/src/ui/admin/InboundScreen.jsx` `CompanyPanel` — `getInboundSettings()` is platform-admin-only and 404s for companies. Remove that `Promise.allSettled` leg.
4. `backend/services/gateway-service/internal/transport/httpapi/brand_rules.go` `listBrandRules` — N+1 `GetBrandPolicy` per brand. **Only if** `ListBrands` already returns policies or a cheap bulk RPC exists. If not, skip (do not add a new RPC in this follow-up).
5. `count32` in inbound.go — `len(filtered)` cannot exceed 20MB-worth of attachments in practice; inline `int32(len(stored))` after a `len > math.MaxInt32` guard or just `int32(len(...))` with a comment. Optional.

**Check:**

```bash
cd backend/services/inbound-service && GOWORK=off go test ./... -count=1
npm run test:component --prefix frontend
```

Commit: `refactor: drop unused inbound constructor args and dead settings fetch`

---

### Task 11: Identity `00001_init.sql` mutation — freeze, do not rewrite again

Already shipped: `00001_init.sql` gained `last_login_at` and `is_primary_admin` **and** additive `00003` / `00004`. Existing databases are saved by `IF NOT EXISTS`. Do not revert `00001`.

**Files:** none unless a comment is useful.

**Step:** Add a 4-line note at the top of `backend/services/identity-service/internal/migrate/migrations/00001_init.sql`:

```sql
-- Applied files are frozen. New columns belong in 0000N_*.sql
-- (00003 last_login, 00004 primary admin) so existing DBs keep migrating.
```

No behavior change. Commit only if the comment is added; otherwise skip.

---

### Task 12: STOP / decision — do not implement in this follow-up

Ask the user before any of these. They are in the inbound design doc but **not** in the running code. Implementing them here would be a new feature, not a review fix.

| Leftover | Where it was promised | Default if unanswered |
| --- | --- | --- |
| `audit-service` events `inbound_webhook_received` / `inbound_rejected` / `inbound_address_updated` | design §architecture | skip (best-effort, not on the ingest hot path) |
| Webhook `rate.Limiter` | design §security | skip; 32 MiB cap + token already exist |
| Auto-create `order_fill` job from an attachment | design «открытые вопросы» | skip (explicit v2) |
| New golang-migrate version table for all services | ponytail/db skill | skip; same re-run-all pattern as identity |

---

### Task 13: Verify the whole gate on the feature branch

After Tasks 1–10:

```bash
cd backend/services/inbound-service && GOWORK=off go test ./...
cd backend/services/gateway-service && GOWORK=off go test ./...
cd backend/services/identity-service && GOWORK=off go test ./...
cd backend/services/brand-service && GOWORK=off go test ./...
cd backend/services/document-service && GOWORK=off go test ./...
npm run test:ui --prefix frontend
make verify
```

`make verify` is the pre-commit gate. Do not `--no-verify`. Merge to `artemch` only when the user asks (`finishing-a-development-branch`, base `artemch`).

---

## Out of scope (do not do)

- Re-implementing inbound, brand rules, North, or Christina PROFF
- Porting browser Excel from `main`
- Sharing inbound MinIO client with `file-service`
- New RPCs / OpenAPI routes / env vars except those required by Tasks 4, 7, 8
- Rewriting `migrate.Up` to a versioned runner

---

## Tests added

Extra coverage for Sept 13–17 behavior (not Task 12):

- identity: Enable of primary admin is `ErrNotFound`; primary can re-enable a secondary (`create_test.go`, `enable_user_test.go`)
- gateway: `company_admin` GET inbound company/messages; `POST /users/{id}/enable` forwards actor+user id and maps identity NotFound
- brand: `UpdateBrandPolicy` persists for `platform_admin` with actor meta
- frontend: `company_admin` InboundScreen sees sender, not platform address panel
