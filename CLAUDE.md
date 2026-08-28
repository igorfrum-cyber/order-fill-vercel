# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Overview

Frontend-only Vercel tool for filling supplier order workbooks from Excel exports.

- `src/app.js` owns the browser UI, file upload flow, reports, manual edits, and downloads.
- `src/workbookProcessor.js` owns Excel parsing, column detection, item matching, brand rules, order calculation, and workbook output.
- `scripts/test-workbook.mjs` is the regression suite for workbook processing behavior.
- `docs/` describes the target service architecture and migration plan; it is not the current runtime.

The app processes files locally in the browser. Do not add a backend dependency to the current app unless the task explicitly moves into the service rewrite.

## Common Commands

Install dependencies first on a fresh checkout:

```bash
npm ci
```

Run locally:

```bash
npm run dev
```

Build and test:

```bash
npm run test:workbook
npm run build
npm run verify
```

`npm run verify` is the local gate for this repository.

## Pre-Commit Gate

### Mandatory before every commit

Run the full local gate from the repository root and commit only if it passes:

```bash
npm run verify
```

This currently expands to:

```bash
npm run test:workbook
npm run build
```

Run both even for a small change. Workbook logic and UI build failures often surface outside the file being edited.

### When dependencies changed

If `package.json` or `package-lock.json` changed, run a clean install before the gate:

```bash
npm ci
npm run verify
```

Commit `package-lock.json` together with `package.json` when dependency versions change.

## Working Rules

- Keep workbook calculations deterministic and explainable unless the user explicitly asks for an ML experiment.
- Preserve browser-only processing for the current Vercel app.
- Do not execute macros from uploaded `.xlsm` files.
- Keep generated output out of git: `dist/`, `test-output/`, `.vercel/`, and `node_modules/` are ignored.
- Match existing brand-specific behavior before generalizing shared logic.
- Add or update workbook regression coverage when changing matching, rounding, period validation, or order quantity calculations.
