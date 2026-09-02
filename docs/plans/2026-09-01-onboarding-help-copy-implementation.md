# Onboarding Help Copy Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a first-run quick start, persistent help button, and plain-language UI copy for users who do not know the product yet.

**Architecture:** Keep this frontend-only unless backend first-run persistence is required later. Store dismissed help state in localStorage because it is non-sensitive UI preference data. Centralize Russian UI copy in small data modules so screen components do not keep accumulating text branches.

**Tech Stack:** React 19, Vite, Node test runner, existing CSS/Tailwind utilities.

---

### Task 1: Centralize Human-Facing Role And Help Copy

**Files:**
- Create: `frontend/src/features/help/copy.js`
- Test: `frontend/src/features/help/copy.test.js`
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`

**Step 1: Write the failing test**

Create tests for role labels and quick start sections:

```js
import test from "node:test";
import assert from "node:assert/strict";

import { roleLabel, quickStartForRole } from "./copy.js";

test("roleLabel uses plain Russian labels", () => {
  assert.equal(roleLabel("company_admin"), "Администратор компании");
  assert.equal(roleLabel("platform_admin"), "Администратор сервиса");
  assert.equal(roleLabel("purchaser"), "Закупщик");
});

test("quickStartForRole returns non-technical steps", () => {
  const steps = quickStartForRole("purchaser");
  assert.ok(steps.length >= 3);
  assert.ok(steps.every((step) => !/api|token|cookie|backend|frontend|company_admin/i.test(step)));
});
```

**Step 2: Run test to verify it fails**

Run: `npm run test --prefix frontend -- src/features/help/copy.test.js`

Expected: FAIL because `copy.js` does not exist.

**Step 3: Implement copy module**

Add:

```js
const ROLE_LABELS = {
  purchaser: "Закупщик",
  company_admin: "Администратор компании",
  company_owner: "Владелец компании",
  platform_admin: "Администратор сервиса",
};

export function roleLabel(role) {
  return ROLE_LABELS[role] || "Пользователь";
}

export function quickStartForRole(role) {
  if (role === "platform_admin") {
    return [
      "Создайте компанию и выберите её в списке.",
      "Пригласите владельца или администратора компании.",
      "Проверяйте историю выгрузок и важные действия.",
    ];
  }
  if (role === "company_owner") {
    return [
      "Проверьте название компании и ответственных за доступ.",
      "Пригласите сотрудников, которые будут работать с файлами.",
      "Следите за выгрузками компании и отключайте лишний доступ.",
    ];
  }
  if (role === "company_admin") {
    return [
      "Пригласите сотрудников по одноразовой ссылке.",
      "Проверьте, кому нужен доступ к выгрузкам компании.",
      "Сбрасывайте доступ, если человек потерял пароль.",
    ];
  }
  return [
    "Загрузите таблицу продаж из 1С.",
    "Добавьте бланк поставщика.",
    "Проверьте строки, где сервис просит внимания.",
    "Скачайте готовые файлы.",
  ];
}
```

**Step 4: Replace visible "админ"**

In `frontend/src/ui/admin/AdminScreens.jsx`:
- Replace `админ компании` with `Администратор компании`.
- Replace `Админ компании` with `Администратор компании`.
- Replace `новую выгрузку создаёт закупщик или админ компании` with `новую выгрузку создаёт закупщик или администратор компании`.
- Prefer `roleLabel(user.role)` over local `ROLE_LABELS`.

**Step 5: Run tests**

Run:

```bash
npm run test --prefix frontend
```

Expected: PASS.

**Step 6: Commit**

```bash
git add frontend/src/features/help/copy.js frontend/src/features/help/copy.test.js frontend/src/ui/admin/AdminScreens.jsx
git commit -m "feat: centralize user-facing help copy"
```

---

### Task 2: Add First-Run Quick Start

**Files:**
- Create: `frontend/src/ui/help/QuickStart.jsx`
- Test: `frontend/src/features/help/firstRun.test.js`
- Create: `frontend/src/features/help/firstRun.js`
- Modify: `frontend/src/App.jsx`

**Step 1: Write preference tests**

```js
import test from "node:test";
import assert from "node:assert/strict";

import { quickStartKey, shouldShowQuickStart } from "./firstRun.js";

test("quick start key is scoped by user id", () => {
  assert.equal(quickStartKey({ id: "u1" }), "order-fill:quick-start:u1");
});

test("quick start shows until dismissed", () => {
  const storage = new Map();
  const adapter = { getItem: (key) => storage.get(key) || null };
  assert.equal(shouldShowQuickStart({ id: "u1" }, adapter), true);
});
```

**Step 2: Implement local preference helper**

Expose:
- `quickStartKey(user)`
- `shouldShowQuickStart(user, storage = localStorage)`
- `dismissQuickStart(user, storage = localStorage)`

Never store auth data here.

**Step 3: Create `QuickStart` component**

UI copy:
- Title: `Быстрый старт`
- Subtitle for purchaser: `Четыре шага до готовых файлов.`
- Close button: `Понятно`
- Secondary button: `Показать позже`

Use `quickStartForRole(me.role)` for steps.

**Step 4: Wire into `App.jsx`**

After `me` is loaded and main shell is visible, show modal if `shouldShowQuickStart(me)`.

Do not show on invite screen before password is set.

**Step 5: Verify**

Run:

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 6: Commit**

```bash
git add frontend/src/features/help/firstRun.js frontend/src/features/help/firstRun.test.js frontend/src/ui/help/QuickStart.jsx frontend/src/App.jsx
git commit -m "feat: add first-run quick start"
```

---

### Task 3: Add Persistent Help Button

**Files:**
- Create: `frontend/src/ui/help/HelpDrawer.jsx`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/ui/chrome.jsx`
- Modify: `frontend/src/ui/icons.jsx`

**Step 1: Add help icon**

If no question icon exists, add `IconHelp` to `frontend/src/ui/icons.jsx`.

**Step 2: Add `HelpDrawer`**

Sections:
- `Как сделать выгрузку`
- `Какие файлы нужны`
- `Что значит "Нужно проверить"`
- `Пользователи и доступ`
- `Если не получается войти`

Copy constraints:
- No `API`, `токен`, `cookie`, `backend`, `frontend`, `endpoint`.
- No blamey text.
- Short paragraphs, 1-2 sentences each.

**Step 3: Wire help button**

Add a `?` icon button in the authenticated header and workflow top bars.

**Step 4: Manual visual check**

Run dev server:

```bash
npm run dev --prefix frontend
```

Check desktop and narrow viewport:
- Button is visible.
- Drawer/modal text does not overflow.
- Escape/backdrop closes if modal pattern supports it.

**Step 5: Commit**

```bash
git add frontend/src/ui/help/HelpDrawer.jsx frontend/src/App.jsx frontend/src/ui/chrome.jsx frontend/src/ui/icons.jsx
git commit -m "feat: add in-app help"
```
