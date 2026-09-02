# Refactor Code Health Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce large-component risk before adding complex account security features.

**Architecture:** Preserve behavior and contracts. Extract pure helpers first, then split UI components along stable seams: app shell, auth/account, admin/users, order review, north merge. Every extraction should be covered by existing tests or small new tests.

**Tech Stack:** React 19, Node test runner, Go tests.

---

### Current Assessment

This is not a rewrite-grade mess. The codebase already has useful separation:
- Go domain/usecase/adapters are split sensibly.
- Frontend business logic under `features/*` has tests.
- API clients are centralized.

The main code-health issue is component mass:
- `frontend/src/ui/order/FillStage.jsx`: 516 lines.
- `frontend/src/ui/north/NorthApp.jsx`: 487 lines.
- `frontend/src/ui/admin/AdminScreens.jsx`: 341 lines.
- `frontend/src/ui/order/OrderFillApp.jsx`: 294 lines.

These files mix state transitions, copy, layout, and row rendering. That is manageable today, but it will make 2FA/passkey/onboarding changes more fragile unless reduced.

---

### Task 1: Extract Role And Access Presentation

**Files:**
- Create: `frontend/src/features/auth/accessPresentation.js`
- Test: `frontend/src/features/auth/accessPresentation.test.js`
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`

**Step 1: Write tests**

Test:
- role label mapping;
- accessible invite roles per actor;
- account summary per role.

**Step 2: Implement pure helper**

Functions:
- `roleLabel(role)`
- `accessSummary(role)`
- `inviteRoleOptions(actorRole)`
- `canInviteRole(actorRole, targetRole)`

**Step 3: Replace duplicated presentation logic**

Remove local role labels from UI components.

**Step 4: Verify**

```bash
npm run test --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/features/auth frontend/src/ui/admin/AdminScreens.jsx frontend/src/ui/auth/AuthScreens.jsx
git commit -m "refactor: centralize access presentation"
```

---

### Task 2: Split Admin Screens

**Files:**
- Create: `frontend/src/ui/admin/JobHistory.jsx`
- Create: `frontend/src/ui/admin/CompaniesScreen.jsx`
- Create: `frontend/src/ui/admin/UsersScreen.jsx`
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`
- Modify: `frontend/src/App.jsx`

**Step 1: Move without behavior changes**

`AdminScreens.jsx` becomes a barrel export:

```js
export { JobHistory } from "./JobHistory.jsx";
export { CompaniesScreen } from "./CompaniesScreen.jsx";
export { UsersScreen } from "./UsersScreen.jsx";
```

**Step 2: Run tests/build**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 3: Commit**

```bash
git add frontend/src/ui/admin frontend/src/App.jsx
git commit -m "refactor: split admin screens"
```

---

### Task 3: Split Auth And Account Screens

**Files:**
- Create: `frontend/src/ui/auth/LoginScreen.jsx`
- Create: `frontend/src/ui/auth/InviteScreen.jsx`
- Create: `frontend/src/ui/auth/AccountScreen.jsx`
- Modify: `frontend/src/ui/auth/AuthScreens.jsx`
- Modify: `frontend/src/App.jsx`

**Step 1: Move components one by one**

Keep `AuthScreens.jsx` as barrel export for compatibility.

**Step 2: Keep shared helpers**

Move only if needed:
- `PasswordHints`
- `AuthCard`

Keep shared UI in `widgets.jsx` only if used outside auth.

**Step 3: Verify**

```bash
npm run build --prefix frontend
```

Expected: PASS.

**Step 4: Commit**

```bash
git add frontend/src/ui/auth frontend/src/App.jsx
git commit -m "refactor: split auth screens"
```

---

### Task 4: Split Order Review Table

**Files:**
- Create: `frontend/src/ui/order/review/ReviewSummary.jsx`
- Create: `frontend/src/ui/order/review/ReviewTabs.jsx`
- Create: `frontend/src/ui/order/review/ReviewTable.jsx`
- Create: `frontend/src/ui/order/review/ReportRow.jsx`
- Modify: `frontend/src/ui/order/FillStage.jsx`

**Step 1: Extract row first**

Move `ReportRow`, `Detail`, `MatchPill`, `rowNeedsComment`.

**Step 2: Extract tabs/filter bar**

Move tab list, query input, and hint banner.

**Step 3: Extract summary header**

Move readiness ring and composition bars.

**Step 4: Verify after each extraction**

```bash
npm run build --prefix frontend
```

Expected: PASS after every commit.

**Step 5: Commit each extraction**

```bash
git add frontend/src/ui/order
git commit -m "refactor: extract review table row"
git commit -m "refactor: extract review filters"
git commit -m "refactor: extract review summary"
```

---

### Task 5: Split North Merge Screen

**Files:**
- Create: `frontend/src/ui/north/NorthUploadPanel.jsx`
- Create: `frontend/src/ui/north/NorthPlanTable.jsx`
- Create: `frontend/src/ui/north/NorthPrompts.jsx`
- Modify: `frontend/src/ui/north/NorthApp.jsx`

**Step 1: Extract dropzones**

Move `MultiDropzone` and `SingleDropzone` first.

**Step 2: Extract plan table**

Move table rendering. Keep calculations in parent until stable.

**Step 3: Extract modal prompts**

Move merge confirmation and short-order warning rendering.

**Step 4: Verify**

```bash
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/ui/north
git commit -m "refactor: split north merge screen"
```

---

### Task 6: Backend Auth Refactor Before 2FA/Passkeys

**Files:**
- Modify: `services/api-service/internal/app/usecase/auth.go`
- Create: `services/api-service/internal/app/usecase/login_flow.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`

**Step 1: Extract login verification**

Move password verification to a helper that returns:
- user;
- generic unauthorized error;
- no session side effects.

This makes TOTP/passkey challenge insertion cleaner.

**Step 2: Extract session issue**

Keep `issueSession` focused and tested.

**Step 3: Verify**

```bash
cd services/api-service && go test ./...
```

Expected: PASS.

**Step 4: Commit**

```bash
git add services/api-service/internal/app/usecase
git commit -m "refactor: prepare auth flow for stronger login methods"
```
