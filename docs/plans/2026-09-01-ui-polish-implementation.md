# UI Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the product feel clearer, calmer, and more deliberate without changing core workbook behavior.

**Architecture:** Keep existing flows and data contracts. Improve information hierarchy, empty states, action copy, navigation, and dense table ergonomics. Do not introduce a marketing landing page; authenticated users should land directly in work.

**Tech Stack:** React 19, existing CSS variables, Tailwind utilities.

---

### Task 1: Improve Global App Shell

**Files:**
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/ui/chrome.jsx`
- Modify: `frontend/src/styles.css`

**Step 1: Header hierarchy**

Show:
- current company;
- role;
- help button;
- profile menu.

User-facing examples:

```text
Компания: Сияние
Владелец компании
```

**Step 2: Replace plain login button**

Use a compact profile control:
- login;
- role label;
- dropdown with `Мой профиль`, `Выйти`.

**Step 3: Add responsive checks**

Narrow viewport should not overflow. Company name truncates.

**Step 4: Verify**

```bash
npm run build --prefix frontend
```

Manual:
- 1440px desktop
- 390px mobile width

**Step 5: Commit**

```bash
git add frontend/src/App.jsx frontend/src/ui/chrome.jsx frontend/src/styles.css
git commit -m "feat: polish app shell"
```

---

### Task 2: Improve History Screen

**Files:**
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`
- Modify: `frontend/src/features/report/reportModel.js`

**Step 1: Better empty state**

Current: `Пока нет выгрузок`

Replace with role-aware text:

```text
Пока нет выгрузок. Начните с бланка закупки или объединения Севера.
```

For platform admin:

```text
Пока нет выгрузок по выбранной компании.
```

**Step 2: Add clearer new-job actions**

Cards:
- `Заполнить бланк закупки`
- `Соединить северные бланки`

Hints:
- `Для заказа поставщику`
- `Для распределения между городами`

**Step 3: Add status helper text**

Add compact status descriptions on hover or via help drawer, not inside every row.

**Step 4: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/ui/admin/AdminScreens.jsx frontend/src/features/report/reportModel.js
git commit -m "feat: clarify export history screen"
```

---

### Task 3: Improve Upload Screens

**Files:**
- Modify: `frontend/src/ui/order/SetupUpload.jsx`
- Modify: `frontend/src/ui/north/NorthApp.jsx`

**Step 1: Make file order obvious**

For order fill:

```text
1. Таблица продаж из 1С
2. Бланк поставщика
```

For North:

```text
1. Бланки городов
2. Таблица Тюмени, если нужно учесть остатки
```

**Step 2: Show selected file count and accepted format**

Plain copy:

```text
Подходят Excel-файлы.
```

**Step 3: Improve error area**

Use inline error block instead of `window.alert` where possible.

**Step 4: Verify**

```bash
npm run build --prefix frontend
```

Manual file selection checks.

**Step 5: Commit**

```bash
git add frontend/src/ui/order/SetupUpload.jsx frontend/src/ui/north/NorthApp.jsx
git commit -m "feat: clarify upload workflows"
```

---

### Task 4: Improve Dense Review Tables

**Files:**
- Modify: `frontend/src/ui/order/FillStage.jsx`
- Modify: `frontend/src/features/report/rowPresentation.js`

**Step 1: Rename ambiguous table labels**

Candidates:
- `Реком.` -> `Расчёт`
- `Совпад.` -> `Похоже`
- `Вставлено` stays if users already know it, otherwise `В бланк`

**Step 2: Add row detail explanation**

Expanded row should explain why row needs attention:

```text
Название отличается от таблицы заказа.
```

or:

```text
Позиция есть в бланке, но не нашлась в таблице заказа.
```

**Step 3: Reduce alert usage**

Replace validation `window.alert` with visible banner above footer.

**Step 4: Verify**

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/ui/order/FillStage.jsx frontend/src/features/report/rowPresentation.js
git commit -m "feat: improve review table clarity"
```

---

### Task 5: Visual Design Pass

**Files:**
- Modify: `frontend/src/styles.css`
- Modify: `frontend/src/ui/widgets.jsx`

**Step 1: Keep palette but reduce purple dominance**

Current palette is serviceable but brand color dominates many controls. Add neutral action variants and reserve purple for primary actions and active navigation.

**Step 2: Standardize radii**

Existing components use `rounded-xl` and `rounded-2xl`. For operational UI, prefer 8-12px and reserve larger radius for auth card/modal only.

**Step 3: Add focus polish**

Ensure buttons and inputs have visible focus rings.

**Step 4: Verify**

Manual:
- keyboard tab through login, header, upload, review table;
- text does not overflow buttons;
- no nested card look.

**Step 5: Commit**

```bash
git add frontend/src/styles.css frontend/src/ui/widgets.jsx
git commit -m "style: tighten operational UI polish"
```
