# Passkey Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add passkey login as a passwordless or second-factor option after the account and 2FA foundation is stable.

**Architecture:** Use WebAuthn with server-generated registration/authentication challenges. Store public credentials only; never store private keys. Keep password login as fallback until account recovery, multi-device, and support flows are proven.

**Tech Stack:** Go net/http, PostgreSQL, React 19, browser WebAuthn APIs, likely Go library `github.com/go-webauthn/webauthn`.

---

### Task 1: Add WebAuthn Dependency And Domain Types

**Files:**
- Modify: `services/api-service/go.mod`
- Modify: `services/api-service/go.sum`
- Create: `services/api-service/internal/domain/identity/passkey.go`
- Test: `services/api-service/internal/domain/identity/passkey_test.go`

**Step 1: Choose library**

Use a maintained WebAuthn library:

```bash
cd services/api-service && go get github.com/go-webauthn/webauthn
```

**Step 2: Define passkey credential**

Fields:
- credential id;
- user id;
- public key;
- sign count;
- name;
- created at;
- last used at.

**Step 3: Tests**

Domain tests should cover serialization shape and no raw secrets.

**Step 4: Verify**

```bash
cd services/api-service && go test ./internal/domain/identity
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service
git commit -m "feat: add passkey identity types"
```

---

### Task 2: Persist Passkeys And Challenges

**Files:**
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/migrate_test.go`

**Step 1: Add tables**

```sql
CREATE TABLE IF NOT EXISTS passkey_credentials (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  credential JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS passkey_challenges (
  id TEXT PRIMARY KEY,
  user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
  purpose TEXT NOT NULL,
  challenge JSONB NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL
);
```

**Step 2: Port methods**

Add registration and auth challenge save/get/consume methods.

**Step 3: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add services/api-service
git commit -m "feat: persist passkey credentials"
```

---

### Task 3: Add Passkey Registration API

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `packages/contracts/openapi.yaml`

**Step 1: Endpoints**

Authenticated endpoints:
- `POST /api/v1/auth/passkeys/register/begin`
- `POST /api/v1/auth/passkeys/register/finish`
- `GET /api/v1/auth/passkeys`
- `POST /api/v1/auth/passkeys/{id}/delete`

**Step 2: User-visible naming**

UI will call passkeys:

```text
Вход по Face ID, Touch ID, Windows Hello или ключу безопасности
```

Do not show `WebAuthn`, `credential`, `challenge`.

**Step 3: Tests**

Use library-supported test helpers if available. If browser ceremony is hard to unit test, test handler state transitions and invalid payload handling.

**Step 4: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service packages/contracts/openapi.yaml
git commit -m "feat: add passkey registration api"
```

---

### Task 4: Add Passkey Login API

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `packages/contracts/openapi.yaml`

**Step 1: Endpoints**

Public endpoints:
- `POST /api/v1/auth/passkeys/login/begin`
- `POST /api/v1/auth/passkeys/login/finish`

Finish sets the same `order_fill_session` cookie as password login.

**Step 2: Account discovery**

Start simple:
- user enters login first;
- server begins passkey auth for that user if passkeys exist;
- response is generic when not available.

Avoid username-less passkey login until UX and browser support are tested.

**Step 3: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add services/api-service packages/contracts/openapi.yaml
git commit -m "feat: add passkey login api"
```

---

### Task 5: Add Passkey UI

**Files:**
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Create: `frontend/src/ui/auth/PasskeySettings.jsx`
- Create: `frontend/src/features/auth/passkey.js`
- Test: `frontend/src/features/auth/passkey.test.js`

**Step 1: Browser helpers**

Wrap:
- `navigator.credentials.create`
- `navigator.credentials.get`

Convert base64url fields safely.

**Step 2: Account settings UI**

Buttons:
- `Добавить вход по устройству`
- `Удалить`

Copy:

```text
Можно входить по Face ID, Touch ID, Windows Hello или ключу безопасности. Добавьте хотя бы два устройства, чтобы не потерять доступ.
```

**Step 3: Login UI**

After login is entered:
- show `Войти по устройству` if available.
- keep password path visible.

**Step 4: Verify manually**

Passkeys require real browser testing:
- Chrome/Safari on macOS with Touch ID if available.
- A second device/account recovery path.

Automated tests only cover request conversion and unsupported-browser fallback.

**Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: add passkey login UI"
```
