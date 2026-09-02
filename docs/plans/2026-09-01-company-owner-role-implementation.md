# Company Owner Role Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Introduce a clear "Владелец компании" role for the main client-side account owner while keeping administrator workflows simple.

**Architecture:** Add a new role value `company_owner` with stronger permissions than `company_admin` inside its own company. Keep `platform_admin` as internal service admin. Existing company admins can continue to work; migration can either leave them as admins or promote the first company admin per company after product confirmation.

**Tech Stack:** Go domain/authz, PostgreSQL text roles, React role labels and user management UI.

---

### Task 1: Add Role To Domain And Authorization

**Files:**
- Modify: `services/api-service/internal/domain/identity/identity.go`
- Modify: `services/api-service/internal/domain/authz/job.go`
- Modify: `services/api-service/internal/domain/authz/job_test.go`
- Modify: `services/api-service/internal/app/usecase/admin.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`

**Step 1: Write authorization tests**

Cases:
- owner can create jobs for own company.
- owner can read all jobs in own company.
- owner cannot read another company.
- owner can manage company users.
- company admin cannot disable owner.
- platform admin can manage owner.

**Step 2: Add role**

```go
RoleCompanyOwner Role = "company_owner"
```

**Step 3: Update authz**

Treat owner like company admin for job access:

```go
case identity.RoleCompanyOwner, identity.RoleCompanyAdmin:
    return entity.CompanyID != "" && entity.CompanyID == actor.CompanyID
```

Allow job creation:

```go
case identity.RolePurchaser, identity.RoleCompanyAdmin, identity.RoleCompanyOwner:
```

**Step 4: Update user management**

Rules:
- `platform_admin` can create owner/admin/purchaser.
- `company_owner` can create company admin and purchaser in own company.
- `company_admin` can create purchaser only, unless product decides otherwise.
- `company_admin` cannot disable or reset `company_owner`.

**Step 5: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api-service
git commit -m "feat: add company owner role"
```

---

### Task 2: Update API Contract And Presenters

**Files:**
- Modify: `packages/contracts/openapi.yaml`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/presenter.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/admin.go`

**Step 1: Update role enums**

Add `company_owner` wherever role enum appears.

**Step 2: Keep payload compatibility**

Existing `company_admin` users must still be valid.

**Step 3: Verify**

```bash
npm run test
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add packages/contracts/openapi.yaml services/api-service
git commit -m "chore: document company owner role"
```

---

### Task 3: Update Users UI

**Files:**
- Modify: `frontend/src/features/help/copy.js`
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`

**Step 1: Add role labels**

User-facing labels:
- `company_owner`: `Владелец компании`
- `company_admin`: `Администратор компании`
- `platform_admin`: `Администратор сервиса`

**Step 2: Update invite role dropdown**

Show options by current actor:
- platform admin: owner, administrator, purchaser;
- owner: administrator, purchaser;
- administrator: purchaser.

This likely requires passing `me` into `UsersScreen`.

**Step 3: Add explanatory copy**

Use this text near dropdown:

```text
Выберите, что человек сможет делать в компании. Доступ можно отключить позже.
```

**Step 4: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: show company owner role in users UI"
```
