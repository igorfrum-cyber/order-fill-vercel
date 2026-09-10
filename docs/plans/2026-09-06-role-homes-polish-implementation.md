# Role Homes And Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the signed-in app feel finished and role-correct: URL is the screen, each role has a home and a nav table, order-fill history and review stop feeling raw. Do not change North.

**Architecture:** Keep behavior in presentation helpers (`accessPresentation.js`, a small `appRoutes.js`, existing report helpers). `App.jsx` reads pathname and renders. New product screens later are one row in the nav/route table. Reuse existing APIs (`listJobs`, `listJobFiles`, `resetUser`, `listCompanies`). Only backend change in this wave: `session_expires_at` on `/me`.

**Tech Stack:** React 19, Vite, Tailwind, Node `node:test`, Go net/http gateway. No new npm or Go dependencies. No react-router.

**Spec:** `docs/plans/2026-09-06-role-homes-ops-routing-design.md`

**Do not touch:** `frontend/src/ui/north/**`, `frontend/src/features/jobs/northJobWorkflow.js`, `frontend/src/features/north/**`, north tour copy, north API helpers beyond leaving the existing history button wired.

---

### Task 1: Role homes and nav table

**Files:**

- Modify: `frontend/src/features/auth/accessPresentation.js`
- Modify: `frontend/src/features/auth/accessPresentation.test.js`

**Step 1: Write the failing tests**

Replace `homeScreen` tests and add:

```js
test("homeScreen sends purchasers to order and keepers to their desks", () => {
  assert.equal(homeScreen("purchaser"), "order");
  assert.equal(homeScreen("platform_admin"), "overview");
  assert.equal(homeScreen("company_owner"), "history");
  assert.equal(homeScreen("company_admin"), "history");
});

test("navItemsForRole lists only what the role may open", () => {
  assert.deepEqual(navItemsForRole("purchaser").map((item) => item.id), ["order", "history"]);
  assert.deepEqual(navItemsForRole("company_admin").map((item) => item.id), ["history", "company", "users"]);
  assert.deepEqual(navItemsForRole("company_owner").map((item) => item.id), ["history", "company", "users"]);
  assert.deepEqual(navItemsForRole("platform_admin").map((item) => item.id), ["overview", "history", "companies", "users"]);
});

test("needsSecurityNudge skips purchasers until they finished a job", () => {
  assert.equal(needsSecurityNudge({ role: "purchaser" }), false);
  assert.equal(needsSecurityNudge({ role: "purchaser" }, { completedJob: true }), true);
  assert.equal(needsSecurityNudge({ role: "company_owner" }), true);
  assert.equal(needsSecurityNudge({ role: "purchaser", has_passkey: true }, { completedJob: true }), false);
});
```

Nav labels: purchaser `order` → «Работа», `history` → «Файлы». Company roles: `history` → «Выгрузки». Platform: keep «Обзор», «Выгрузки», «Компании», «Пользователи». Do not add `metrics` or `north` to this table.

**Step 2: Run the test to verify it fails**

```bash
npm run test --prefix frontend -- src/features/auth/accessPresentation.test.js
```

Expected: FAIL (`navItemsForRole` missing, `homeScreen("purchaser")` is `"history"`).

**Step 3: Minimal implementation**

In `accessPresentation.js`:

```js
export function homeScreen(role) {
  if (role === "platform_admin") return "overview";
  if (role === "purchaser") return "order";
  return "history";
}

export function navItemsForRole(role) {
  if (role === "platform_admin") {
    return [
      { id: "overview", path: "/overview", label: "Обзор" },
      { id: "history", path: "/jobs", label: "Выгрузки" },
      { id: "companies", path: "/companies", label: "Компании" },
      { id: "users", path: "/users", label: "Пользователи" },
    ];
  }
  if (role === "purchaser") {
    return [
      { id: "order", path: "/jobs/new", label: "Работа" },
      { id: "history", path: "/jobs", label: "Файлы" },
    ];
  }
  return [
    { id: "history", path: "/jobs", label: "Выгрузки" },
    { id: "company", path: "/company", label: "Компания" },
    { id: "users", path: "/users", label: "Пользователи" },
  ];
}

export function needsSecurityNudge(me, { completedJob = false } = {}) {
  if (!me || me.two_factor_enabled || me.has_passkey) return false;
  if (me.role === "purchaser" && !completedJob) return false;
  return true;
}
```

Keep existing invite/company helpers. `canCreateJobs(role)` = `role !== "platform_admin"` for history CTA (order fill only; leave the existing North button in JobHistory untouched).

**Step 4: Re-run tests**

Same command. Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/features/auth/accessPresentation.js frontend/src/features/auth/accessPresentation.test.js
git commit -m "$(cat <<'EOF'
feat: land each role on its own home screen

Purchasers start at upload; platform admin stays on overview. Nav is one table so later screens do not fork App.jsx.
EOF
)"
```

---

### Task 2: Pathname helpers

**Files:**

- Create: `frontend/src/features/app/routes.js`
- Create: `frontend/src/features/app/routes.test.js`

**Step 1: Failing tests**

```js
test("parseAppPath reads signed-in screens and job ids", () => {
  assert.deepEqual(parseAppPath("/"), { screen: "", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/overview"), { screen: "overview", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs"), { screen: "history", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs/new"), { screen: "order", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/jobs/abc"), { screen: "order", jobId: "abc", unknown: false });
  assert.deepEqual(parseAppPath("/company"), { screen: "company", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/users"), { screen: "users", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/account"), { screen: "account", jobId: "", unknown: false });
  assert.deepEqual(parseAppPath("/companies"), { screen: "companies", jobId: "", unknown: false });
  assert.equal(parseAppPath("/nope").unknown, true);
});

test("parseAppPath ignores invite and company login paths", () => {
  assert.equal(parseAppPath("/invite/tok").screen, "");
  assert.equal(parseAppPath("/c/acme").screen, "");
});

test("pathForScreen writes the URL for a screen", () => {
  assert.equal(pathForScreen("overview"), "/overview");
  assert.equal(pathForScreen("history"), "/jobs");
  assert.equal(pathForScreen("order"), "/jobs/new");
  assert.equal(pathForScreen("order", "abc"), "/jobs/abc");
  assert.equal(pathForScreen("company"), "/company");
});

test("screenAllowed rejects foreign screens", () => {
  assert.equal(screenAllowed("purchaser", "users"), false);
  assert.equal(screenAllowed("purchaser", "order"), true);
  assert.equal(screenAllowed("platform_admin", "order"), false);
  assert.equal(screenAllowed("platform_admin", "overview"), true);
  assert.equal(screenAllowed("company_admin", "company"), true);
});

test("companyQuery reads and writes platform company id", () => {
  assert.equal(companyIdFromSearch("?company=c-1"), "c-1");
  assert.equal(withCompanyQuery("/jobs", "c-1"), "/jobs?company=c-1");
  assert.equal(withCompanyQuery("/jobs", ""), "/jobs");
});
```

Do not add `/jobs/north`. Existing in-memory `north` screen may still render if JobHistory calls `onNew("north")` — keep that callback, do not put it in the URL table yet.

**Step 2:** `npm run test --prefix frontend -- src/features/app/routes.test.js` → FAIL file missing.

**Step 3:** Implement `parseAppPath`, `pathForScreen`, `screenAllowed` (use `navItemsForRole` plus `account` and job resume `order` with id; platform cannot open `order` unless opening an existing job — allow `order` with `jobId` for platform resume from overview, forbid `/jobs/new`).

```js
export function screenAllowed(role, screen, { jobId = "" } = {}) {
  if (screen === "account") return true;
  if (screen === "order" && jobId) return true;
  if (screen === "order" && role === "platform_admin") return false;
  if (screen === "north") return role !== "platform_admin";
  return navItemsForRole(role).some((item) => item.id === screen);
}
```

`unknown` path → caller shows a short 404 then home.

**Step 4:** tests PASS.

**Step 5:** commit `feat: parse app screens from the URL`

---

### Task 3: Wire App to the URL

**Files:**

- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/ui/chrome.jsx` only if nav needs `href`+click

**Step 1:** No new test file if routes are covered. Manual check: login as purchaser lands on `/jobs/new`; refresh stays; `/users` as purchaser redirects home.

**Step 2:** In `App.jsx`:

- After `me` loads, if pathname is `/` or empty screen, `history.replaceState` to `pathForScreen(homeScreen(me.role))` plus `?company=` for platform.
- `setScreen` also `history.pushState` via `pathForScreen`.
- `popstate` listener reads `parseAppPath`.
- If `screenAllowed` is false or `unknown`, replace with home.
- If `jobId`, keep existing `loadOrderResume`.
- Render `navItemsForRole(me.role)` instead of inline role ifs.
- Platform company: read `companyIdFromSearch`, picker in the header (not per-page). Pass `onCompany` that writes query.
- Leave `screen === "north"` branch and JobHistory `onNew("north")` exactly as today.

Do not import or edit North components except the existing `NorthApp` usage.

**Step 3:** `npm run test --prefix frontend -- src/features/auth/accessPresentation.test.js src/features/app/routes.test.js`

**Step 4:** commit `feat: keep signed-in screens in the URL`

---

### Task 4: History that does not feel unfinished

**Files:**

- Modify: `frontend/src/ui/admin/JobHistory.jsx`
- Modify: `frontend/src/features/report/reportModel.js`
- Modify: `frontend/src/features/report/reportModel.test.js`

**Step 1: Tests for filters**

```js
test("filterJobs keeps brand status and month", () => {
  const jobs = [
    { id: "1", brand: "christina", status: "completed", type: "order_fill", created_at: "2026-09-01T10:00:00Z" },
    { id: "2", brand: "klapp", status: "needs_review", type: "order_fill", created_at: "2026-08-01T10:00:00Z" },
  ];
  assert.equal(filterJobs(jobs, { brand: "christina" }).length, 1);
  assert.equal(filterJobs(jobs, { status: "needs_review" })[0].id, "2");
  assert.equal(filterJobs(jobs, { month: "2026-09" })[0].id, "1");
});
```

**Step 2:** implement `filterJobs` in `reportModel.js` (month = `created_at` `YYYY-MM`).

**Step 3:** `JobHistory`:

- Poll `listJobs` every 8s like Overview (`POLL_MS = 8000`).
- Chips/selects: brand (from `ORDER_BRANDS` plus values present), status (на проверке / готово / сбой / в работе), month.
- Row is `<a href={/jobs/id}>` or button that calls `onOpen`; keyboard reachable. Stop using click-only `<tr>`.
- If `job.status === "completed"`, extra control «Скачать» that `listJobFiles` + existing `downloadJobFile` loop. Do not zip.
- `canCreate`: show «Заполнить бланк закупки». **Do not change** the North button markup/handler.
- Platform: no create buttons (already). Company picker removed here; App header owns it.
- Empty state stays `jobsEmptyState`.
- Loading: a few pulse rows, not a centered «Загрузка…» if the parent still shows that — replace local empty flash.

**Step 4:** tests PASS. `npm run test --prefix frontend -- src/features/report/reportModel.test.js`

**Step 5:** commit `feat: make job history filterable and downloadable`

---

### Task 5: Order-fill review queue and comments

**Files:**

- Modify: `frontend/src/features/report/rowPresentation.js`
- Modify: `frontend/src/features/report/rowPresentation.test.js`
- Modify: `frontend/src/ui/order/FillStage.jsx`
- Modify: `frontend/src/ui/order/review/CommentGate.jsx`
- Modify: `frontend/src/ui/order/SetupUpload.jsx`
- Modify: `frontend/src/features/help/copy.js` only if new strings belong there

**Step 1:**

```js
test("firstReviewTab prefers duplicates then empty then check", () => {
  assert.equal(firstReviewTab({ duplicate: 2, empty: 1, check: 1 }), "duplicate");
  assert.equal(firstReviewTab({ duplicate: 0, empty: 3, check: 1 }), "empty");
  assert.equal(firstReviewTab({ duplicate: 0, empty: 0, check: 2 }), "check");
  assert.equal(firstReviewTab({ duplicate: 0, empty: 0, check: 0, filled: 10 }), "filled");
});

test("reviewQueueLine counts work left", () => {
  assert.match(reviewQueueLine({ duplicate: 2, empty: 1, check: 3 }), /6/);
  assert.equal(reviewQueueLine({ duplicate: 0, empty: 0, check: 0, filled: 4 }), "");
});
```

**Step 2:** implement; `FillStage` initial tab = `firstReviewTab(counts)` (state from counts on first rows, not hardcoded `"empty"`). Show `reviewQueueLine` above tabs.

**Step 3:** `CommentGate` — three buttons that set comment: `Округление`, `Нет на складе`, `Поставщик`. Click fills the focused/current row comment if empty or always sets that row.

**Step 4:** `SetupUpload` — if `looksLikeChristinaSource(sourceFile?.name)` show «Похоже, Christina: достаточно одного бланка.» No North copy.

**Step 5:** tests + commit `feat: lead order review from leftover rows`

---

### Task 6: Help, invite copy-again, loading chrome

**Files:**

- Modify: `frontend/src/features/help/copy.js`
- Modify: `frontend/src/features/help/copy.test.js`
- Modify: `frontend/src/ui/help/HelpDrawer.jsx`
- Modify: `frontend/src/ui/admin/UsersScreen.jsx`
- Modify: `frontend/src/App.jsx` loading placeholder
- Modify: `frontend/src/ui/admin/UsersScreen.jsx` invite `<input>` → labeled field (`Field` from widgets)

**Step 1:** `helpSectionsForRole(role)` returns subset: purchaser without «Обзор сервиса»; platform keeps it; company roles keep users/company, drop overview.

**Step 2:** `UserCard` if `canManage && !user.last_seen_at && !user.disabled_at`: button «Скопировать ссылку снова» calling the same `onReset` path (existing reset API). Do not invent a new endpoint.

**Step 3:** App boot: skeleton block, not faint «Загрузка…». Invite login field uses `Field label="Логин"`.

**Step 4:** tests in `copy.test.js`. Commit `feat: trim help and recover lost invite links`

---

### Task 7: Session expiry on /me and banner

**Files:**

- Modify: `backend/services/gateway-service/internal/transport/httpapi/auth.go` (`presentUser` / `me`)
- Modify: gateway OpenAPI `Me` schema
- Test: existing gateway http tests if any; otherwise `frontend/src/features/auth/session.js` + test for `sessionExpiringSoon(expiresAt, now)`
- Modify: `frontend/src/App.jsx` banner

**Step 1:** frontend helper tests:

```js
test("sessionExpiringSoon is true inside ten minutes", () => {
  const now = new Date("2026-09-06T12:00:00Z");
  assert.equal(sessionExpiringSoon("2026-09-06T12:05:00Z", now), true);
  assert.equal(sessionExpiringSoon("2026-09-06T14:00:00Z", now), false);
  assert.equal(sessionExpiringSoon("", now), false);
});
```

**Step 2:** Gateway `presentUser` adds `session_expires_at` RFC3339 from the current session. Trace `userFrom` — if session expiry is not on `User`, load it from identity session list current cookie or pass expiry through request context. Smallest path: if `User` has no expiry, add it where the session is validated.

If expiry is not in the request user struct, grep `userFrom` and identity `ValidateSession` response; add one field, do not redesign sessions.

**Step 3:** Banner: «Сессия скоро закончится. Сохраните правки.» with link to account. Not for the passkey nudge.

**Step 4:** `gofmt`, `go test` in gateway-service for the touched package. Frontend session tests PASS.

**Step 5:** commit `feat: warn before the session cookie dies`

---

### Task 8: Narrow preview honesty

**Files:**

- Modify: `frontend/src/ui/order/PreviewStage.jsx`

**Step 1:** If `window.matchMedia("(max-width: 768px)").matches`, show «Превью таблицы удобнее на компьютере. Файлы можно скачать.» plus download buttons. Do not load the grid. Desktop unchanged.

**Step 2:** no North files.

**Step 3:** commit `fix: skip cramped excel preview on phones`

---

### Task 9: Verify

```bash
npm run test --prefix frontend -- src/features/auth/accessPresentation.test.js src/features/app/routes.test.js src/features/report/reportModel.test.js src/features/report/rowPresentation.test.js src/features/help/copy.test.js src/features/auth/session.test.js
```

From `backend/services/gateway-service` if Task 7 touched Go:

```bash
gofmt -w .
go test ./internal/transport/httpapi/...
```

Do not run North-focused tests as a reason to change North.

---

## Later (not this plan)

- Prometheus scrape, `GET /api/v1/metrics/query`, `/metrics` page, status groups for v2 services.
- Retry failed jobs, auto-swap uploaded files.
- URL for North when that flow is finished.
