# Company Login Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Give each company a personalized login URL and entry screen without exposing a public company directory.

**Architecture:** Add a stable, unique `login_slug` to companies and resolve it through a public read-only endpoint. The login screen may display company name and optional short help text, but authentication remains the same global login/password flow. Do not reveal users, roles, history, or disabled companies.

**Tech Stack:** Go net/http, PostgreSQL, React 19, Vite.

---

### Task 1: Add Company Login Slug

**Files:**
- Modify: `services/api-service/internal/domain/identity/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/migrate_test.go`
- Modify: `services/api-service/internal/app/usecase/admin.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`

**Step 1: Write migration test**

Add expected migration strings:

```go
"login_slug",
"UNIQUE",
```

**Step 2: Extend domain**

Add:

```go
LoginSlug string
```

to `identity.Company`.

**Step 3: Generate slug on company creation**

For first version:
- Lowercase company name.
- Transliterate simple Cyrillic or fallback to company id suffix.
- Keep only `a-z`, `0-9`, `-`.
- Ensure uniqueness in database.

YAGNI rule: support manual slug editing later, not now.

**Step 4: Database migration**

Add column:

```sql
ALTER TABLE companies ADD COLUMN IF NOT EXISTS login_slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS companies_login_slug_uidx ON companies(login_slug) WHERE login_slug IS NOT NULL;
```

Backfill existing companies using a deterministic fallback like `company-<short-id>`.

**Step 5: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api-service
git commit -m "feat: add company login slugs"
```

---

### Task 2: Add Public Company Lookup Endpoint

**Files:**
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/admin.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth_test.go`
- Modify: `packages/contracts/openapi.yaml`

**Step 1: Add endpoint test**

Expected behavior:
- `GET /api/v1/public/companies/acme/login` returns `{ "name": "Acme", "login_slug": "acme" }`.
- Disabled company returns 404.
- Unknown slug returns 404.

**Step 2: Add port method**

```go
GetCompanyByLoginSlug(ctx context.Context, slug string) (identity.Company, error)
```

**Step 3: Add route before auth gate requirement**

Allow unauthenticated only for:

```text
/api/v1/public/companies/{slug}/login
```

**Step 4: Keep response minimal**

Never return:
- company id;
- user list;
- whether a login exists;
- any operational metadata.

**Step 5: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api-service packages/contracts/openapi.yaml
git commit -m "feat: expose company login screen metadata"
```

---

### Task 3: Add Personalized Login Route

**Files:**
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Test: `frontend/src/api/auth.test.js`

**Step 1: Add API client function**

```js
export function getCompanyLogin(slug) {
  return apiClient.request(`/api/v1/public/companies/${encodeURIComponent(slug)}/login`);
}
```

**Step 2: Parse route**

Support:

```text
/c/:slug
```

Do not replace `/invite/:token`.

**Step 3: Update login screen copy**

Default:

```text
Вход
```

Company route:

```text
Вход для сотрудников «{company.name}»
Работайте только с файлами своей компании.
```

Help text:

```text
Нет доступа? Попросите владельца или администратора компании прислать приглашение.
```

**Step 4: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src packages/contracts/openapi.yaml
git commit -m "feat: add personalized company login screen"
```
