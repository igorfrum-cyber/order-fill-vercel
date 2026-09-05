# Blank Preview Quantity Edits Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make matched blank quantity cells editable in preview so the same edit updates 1C «Заказано по факту» and live blank sums.

**Architecture:** Keep one `edits` Map. Give blank quantity overlays a `key` so ExcelGrid renders an input. Extend the existing formula evaluator just enough for Angiopharm `IFERROR` / `IF(x=0,"",…)` and cached prices. No backend or API change.

**Tech Stack:** React 19, Vite, Node `node:test`.

---

### Task 1: Editable blank quantity overlays

**Files:**
- Modify: `frontend/src/features/preview/previewEdits.js`
- Test: `frontend/src/features/preview/previewEdits.test.js`

**Step 1: Write the failing test**

Change `without making the blank editable` to assert the blank overlay is a quantity overlay with the same key as the 1C fact cell. Add a comment on the blank overlay from the edit.

**Step 2: Run to verify it fails**

```bash
npm run test --prefix frontend -- src/features/preview/previewEdits.test.js
```

Expected: FAIL because blank overlay has no `key`.

**Step 3: Minimal implementation**

In `blankOverlays`, set `key` and `comment` on the quantity overlay, same shape as the 1C fact cell without a comment column.

**Step 4: Run tests**

Same command. Expected: PASS.

---

### Task 2: Angiopharm-capable formula recompute

**Files:**
- Modify: `frontend/src/features/preview/formulas.js`
- Test: `frontend/src/features/preview/formulas.test.js`

**Step 1: Write failing tests**

- `IFERROR(D2*E2,"")` updates after qty overlay; `D2` is an unevaluable formula with cached price.
- `IFERROR(IF(E2*H2=0,"",E2*H2),"")` follows qty.
- Cached price must not block a computable chain like `H2=I2*(1-J2)`, `J2=H2*E2`.

**Step 2: Run to verify fail**

```bash
npm run test --prefix frontend -- src/features/preview/formulas.test.js
```

**Step 3: Minimal implementation**

Tokenizer: `=` and `""` (quoted text → 0). Parser: comparison `=`, `IFERROR`, `IF`. After the pending loop stalls, retry once using cached `formula_values` for still-uncomputed formula cells.

**Step 4: Run tests.** Expected: PASS.

---

### Task 3: Copy

**Files:**
- Modify: `frontend/src/ui/order/PreviewStage.jsx`
- Modify: `frontend/src/features/help/copy.js`
- Test: `frontend/src/features/help/copy.test.js` if it asserts the old sentence.

Hint: quantity in the blank and «Заказано по факту» are the same number; sums recalculate.
