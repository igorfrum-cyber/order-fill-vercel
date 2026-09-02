# Login Security Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close practical auth and HTTP hardening gaps before adding heavier login features.

**Architecture:** Keep current cookie-session architecture. Add bounded request parsing, stricter production config checks, stronger browser headers, broader audit events, and safer UI messages without changing the user-facing login flow.

**Tech Stack:** Go net/http, PostgreSQL, React 19, nginx config.

---

### Task 1: Bound JSON Body Reads Everywhere

**Files:**
- Modify: `services/api-service/internal/adapter/inbound/httpapi/handlers.go`
- Test: `services/api-service/internal/adapter/inbound/httpapi/auth_test.go`

**Step 1: Write failing test**

Add test for oversized edits payload:

```go
func TestSubmitEditsRejectsOversizedBody(t *testing.T) {
    body := strings.NewReader(strings.Repeat("x", authJSONLimit+1))
    request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-1/edits", body)
    request.Header.Set("X-Requested-With", "fetch")
    request.Header.Set("Origin", "http://127.0.0.1:3200")
    // Build router with fake auth and finder if needed.
    // Expected: 400, not unbounded memory read.
}
```

**Step 2: Add shared JSON decode helper usage**

Replace direct:

```go
json.NewDecoder(r.Body).Decode(&payload)
```

in `submitEdits` with existing `decodeJSON(w, r, &payload)`.

**Step 3: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add services/api-service/internal/adapter/inbound/httpapi
git commit -m "fix: bound edits request bodies"
```

---

### Task 2: Add Production-Safe Config Validation

**Files:**
- Modify: `services/api-service/internal/platform/config/config.go`
- Modify: `services/api-service/internal/platform/config/config_test.go`
- Modify: `deploy/docker-compose.yml`

**Step 1: Add config tests**

Cases:
- local mode allows `API_ALLOWED_ORIGINS=*`.
- production mode rejects `API_ALLOWED_ORIGINS=*`.
- production mode requires `SESSION_COOKIE_SECURE=true`.

**Step 2: Add env**

Add:

```go
Environment string
```

Read `APP_ENV`, default `local`.

**Step 3: Validate**

If `APP_ENV=production`:
- `API_ALLOWED_ORIGINS` must not be `*`.
- `SESSION_COOKIE_SECURE` must be true.

**Step 4: Update compose**

Keep local compose as:

```yaml
APP_ENV: ${APP_ENV:-local}
```

**Step 5: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api-service/internal/platform/config deploy/docker-compose.yml
git commit -m "fix: require safe auth config in production"
```

---

### Task 3: Strengthen Browser Headers

**Files:**
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth_test.go`
- Modify: `frontend/nginx.conf`

**Step 1: Add test for headers**

Assert responses include:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy` present
- `Content-Security-Policy` present

**Step 2: Add conservative CSP**

For frontend nginx:

```nginx
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
```

For API JSON responses, set at least:

```go
header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
```

If CSP breaks Vite dev, scope CSP to nginx production first and keep API headers minimal.

**Step 3: Verify**

```bash
npm run build --prefix frontend
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add frontend/nginx.conf services/api-service/internal/adapter/inbound/httpapi
git commit -m "fix: strengthen browser security headers"
```

---

### Task 4: Expand Audit Events

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/app/usecase/admin.go`
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/admin.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`

**Step 1: Define audit actions**

Use constants:
- `login_success`
- `logout`
- `password_changed`
- `invite_created`
- `access_reset`
- `user_disabled`
- `company_disabled`
- existing `job_view`, `file_download`, `archive_download`

**Step 2: Record without secrets**

Never log:
- password;
- cookie;
- raw invite link;
- uploaded file content.

**Step 3: Add tests**

Assert important actions append audit rows.

**Step 4: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service
git commit -m "feat: audit account and access changes"
```

---

### Task 5: Improve Login UX Copy

**Files:**
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Modify: `frontend/src/features/help/copy.js`

**Step 1: Replace error text**

Use:

```text
Не получилось войти. Проверьте логин и пароль или запросите новую ссылку.
```

Do not distinguish missing user, disabled user, wrong password, or disabled company.

**Step 2: Add access recovery hint**

```text
Нет доступа? Попросите владельца или администратора компании прислать приглашение.
```

**Step 3: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 4: Commit**

```bash
git add frontend/src
git commit -m "feat: improve login access copy"
```
