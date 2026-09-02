# 2FA TOTP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add app-based two-factor authentication for higher-risk accounts.

**Architecture:** Implement TOTP as a second step after password verification. Store only encrypted or protected TOTP secrets and hashed recovery codes. Enforce 2FA for owners/admins after rollout, but first ship opt-in setup in "Мой профиль".

**Tech Stack:** Go net/http, PostgreSQL, React 19, Web Crypto where useful, likely Go TOTP library such as `github.com/pquerna/otp/totp`.

---

### Task 1: Add 2FA Domain Model

**Files:**
- Modify: `services/api-service/go.mod`
- Modify: `services/api-service/internal/domain/identity/identity.go`
- Create: `services/api-service/internal/domain/identity/totp.go`
- Test: `services/api-service/internal/domain/identity/totp_test.go`

**Step 1: Choose library**

Use a proven TOTP library. Do not hand-roll OTP generation/verification.

Candidate:

```bash
cd services/api-service && go get github.com/pquerna/otp
```

**Step 2: Add domain type**

Fields needed later:
- user id;
- secret;
- enabled time;
- recovery code hashes.

**Step 3: Add tests**

Test:
- valid TOTP code verifies;
- wrong code fails;
- recovery code is accepted once;
- recovery code hash does not store raw code.

**Step 4: Verify**

```bash
cd services/api-service && go test ./internal/domain/identity
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service/go.mod services/api-service/go.sum services/api-service/internal/domain/identity
git commit -m "feat: add totp identity primitives"
```

---

### Task 2: Persist 2FA Settings

**Files:**
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/migrate_test.go`

**Step 1: Migration**

Add table:

```sql
CREATE TABLE IF NOT EXISTS user_totp (
  user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  secret TEXT NOT NULL,
  enabled_at TIMESTAMPTZ,
  recovery_code_hashes JSONB NOT NULL DEFAULT '[]'
)
```

**Step 2: Port methods**

Add:
- `SaveTOTPSetup`
- `GetTOTP`
- `EnableTOTP`
- `DisableTOTP`
- `ReplaceRecoveryCodes`

**Step 3: Tests**

Repository migration and DTO tests should confirm raw recovery codes never appear in stored fields.

**Step 4: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service
git commit -m "feat: persist two-factor settings"
```

---

### Task 3: Add Login Challenge Flow

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `packages/contracts/openapi.yaml`

**Step 1: Desired API behavior**

Password-only account:
- `POST /auth/login` returns current user and sets session.

2FA-enabled account:
- `POST /auth/login` returns `{ "two_factor_required": true, "challenge_id": "..." }`.
- No full session cookie yet.
- `POST /auth/login/2fa` with challenge + code completes login and sets session.

**Step 2: Store challenge**

Add short-lived login challenge storage:
- table `login_challenges`;
- hash challenge secret;
- user id;
- expires in 5 minutes;
- consumed once.

**Step 3: Tests**

Cases:
- 2FA user does not get session after password step.
- valid code completes login.
- challenge is single-use.
- expired challenge fails.
- wrong code fails with same generic message.

**Step 4: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service packages/contracts/openapi.yaml
git commit -m "feat: require two-factor challenge on login"
```

---

### Task 4: Add 2FA Setup UI

**Files:**
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Create: `frontend/src/ui/auth/TwoFactorSetup.jsx`
- Create: `frontend/src/ui/auth/TwoFactorLogin.jsx`

**Step 1: Account setup flow**

UI steps:
- Button: `Включить защиту кодом`
- Show QR code or manual setup key.
- Input: `Код из приложения`
- Show recovery codes once.

User-facing copy:

```text
Код нужен при входе с новым паролем. Сохраните запасные коды: каждый работает один раз.
```

**Step 2: Login flow**

After password step:
- title `Подтвердите вход`
- field `Код из приложения`
- link/button `Использовать запасной код`

**Step 3: Tests**

Add API tests for request shapes. Component tests are optional unless test setup already supports rendering.

**Step 4: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: add two-factor login UI"
```

---

### Task 5: Enforce 2FA For Sensitive Roles

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/app/usecase/admin.go`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Modify: `frontend/src/features/help/copy.js`

**Step 1: Add policy**

First rollout:
- owners/admins see a banner after login if 2FA is not enabled.
- second rollout can block sensitive actions until 2FA is enabled.

Do not block all users on day one.

**Step 2: UI copy**

```text
Для управления доступом включите вход с кодом. Это защищает сотрудников и файлы компании.
```

**Step 3: Verify**

Run full verify:

```bash
npm run verify
```

Expected: PASS.

**Step 4: Commit**

```bash
git add services/api-service frontend/src
git commit -m "feat: require stronger protection for access managers"
```
