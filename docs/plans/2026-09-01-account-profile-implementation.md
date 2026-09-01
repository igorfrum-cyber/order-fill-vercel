# Account Profile Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the minimal password page with a useful personal account screen that explains access, security, and available actions.

**Architecture:** Keep the first iteration mostly frontend-only using current `/api/v1/auth/me` data. Add backend session-management endpoints only for "sign out everywhere" and active session display. Do not expose raw session tokens or technical details in UI.

**Tech Stack:** React 19, Go net/http, PostgreSQL via pgx, existing auth usecase.

---

### Task 1: Rename Screen From "Пароль" To "Мой Профиль"

**Files:**
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Modify: `frontend/src/App.jsx`
- Test: `frontend/src/features/help/copy.test.js`

**Step 1: Add account copy helpers**

Extend `frontend/src/features/help/copy.js`:

```js
export function accessSummaryForRole(role) {
  if (role === "platform_admin") return "Вы можете создавать компании, помогать с доступом и смотреть историю по всем компаниям.";
  if (role === "company_owner") return "Вы управляете доступом сотрудников и видите выгрузки своей компании.";
  if (role === "company_admin") return "Вы приглашаете сотрудников, сбрасываете доступ и видите выгрузки своей компании.";
  return "Вы создаёте выгрузки, проверяете строки и скачиваете готовые файлы.";
}
```

**Step 2: Update screen content**

In `AccountScreen`:
- Header: `Мой профиль`
- Fields:
  - `Логин`
  - `Компания`
  - `Доступ`
- Security block title: `Безопасность`
- Password form title: `Сменить пароль`

Recommended UI text:

```text
Ваш пароль знаете только вы. Если доступ нужен другому человеку, создайте отдельного пользователя.
```

**Step 3: Keep password behavior unchanged**

The existing `changePassword` call stays as-is.

**Step 4: Run frontend tests**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/ui/auth/AuthScreens.jsx frontend/src/App.jsx frontend/src/features/help/copy.js frontend/src/features/help/copy.test.js
git commit -m "feat: expand account profile screen"
```

---

### Task 2: Add "Sign Out Everywhere"

**Files:**
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`

**Step 1: Write backend usecase test**

Add test:

```go
func TestLogoutEverywhereInvalidatesAllSessions(t *testing.T) {
    auth, store := newTestAuthStore(t)
    user := seedPurchaser(t, store, "buyer", "correct-horse")
    first, _ := auth.Login(context.Background(), user.Login, "correct-horse")
    second, _ := auth.Login(context.Background(), user.Login, "correct-horse")

    if err := auth.LogoutEverywhere(context.Background(), user); err != nil {
        t.Fatal(err)
    }

    if _, err := auth.SessionUser(context.Background(), identity.HashSecret(first.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
        t.Fatalf("first session still valid: %v", err)
    }
    if _, err := auth.SessionUser(context.Background(), identity.HashSecret(second.RawToken)); !errors.Is(err, identity.ErrUnauthorized) {
        t.Fatalf("second session still valid: %v", err)
    }
}
```

**Step 2: Implement usecase**

Add to authenticator interface:

```go
LogoutEverywhere(ctx context.Context, actor identity.User) error
```

Implementation:

```go
func (a *Auth) LogoutEverywhere(ctx context.Context, actor identity.User) error {
    return a.store.DeleteSessionsForUser(ctx, actor.ID)
}
```

**Step 3: Add route**

Route: `POST /api/v1/auth/logout-everywhere`

UI text:

```text
Выйти со всех устройств
```

Confirmation:

```text
Вы выйдете здесь и на других устройствах. Войти снова можно будет по логину и паролю.
```

**Step 4: Verify**

```bash
cd services/api-service && go test ./...
npm run test --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api-service frontend/src
git commit -m "feat: allow users to sign out everywhere"
```

---

### Task 3: Optional Active Sessions List

**Files:**
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`

**Step 1: Add session metadata**

Add columns later if wanted:
- `sessions.created_at`
- `sessions.last_seen_at`
- `sessions.user_agent`

Do not store IP in first iteration unless there is a real admin/support need.

**Step 2: Show plain text**

UI:

```text
Активные входы
Текущий браузер
Другие устройства: 2
```

Avoid exposing raw browser fingerprints or technical user-agent strings.

**Step 3: Verify**

Run full API and frontend test suites.
