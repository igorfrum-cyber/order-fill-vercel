---
name: order-fill-work
description: >-
  Use when implementing, fixing, porting from main, adding a feature, screen, or
  microservice, or starting any Order Fill code change. Also when the user
  mentions ветка, artemch, main, микросервис, экран, or слив.
---

# Order Fill work

Runtime code, the service README, proto, OpenAPI, and CLAUDE.md owners win.
Friend's `main` is product intent, never a merge source and never a UI kit.

## Start

1. Read this skill, CLAUDE.md owners, and the affected service README.
2. Load the skills for the files you will touch (table below).
3. If HEAD is `artemch`, `dev`, `main`, or `igorfrum`: create `feat/<slug>` from `artemch` before edits. Use `.worktrees/<slug>` when the workspace already has unrelated dirty files. Do not commit to those four branches. Merge into `artemch` only when the user asks.

## Place the change

Put behavior in the existing owner. A new `backend/services/<name>` or a new routed screen: **STOP**. Say why current owners/screens fail, the proposed name, and data/RPCs or route+role. Wait for explicit «да». Then implement.

UI: `frontend/src/ui/` with `widgets.jsx`, `chrome.jsx`, CSS variables, plain Russian. Do not copy `src/app.js` or `workbookProcessor.js`.

## Docs (same change)

Load `.cursor/skills/order-fill-docs/SKILL.md`. After env/RPC/HTTP source changes, tables regenerate via hook/`node scripts/sync-docs.mjs --write`. Fill purpose stubs; write why in the human README sections.

## Skills to load

| Touch | Load |
| --- | --- |
| README, proto, OpenAPI, `config.go`, env | `order-fill-docs` |
| Go | `use-modern-go` and `golang-how-to` (then its table) |
| frontend UI | existing screens; then `npm run test:ui --prefix frontend` |
| unexpected failure | systematic-debugging |

Ponytail is already always on. Do not invent a new visual language.

## Finish

Do not merge yourself. When the user asks to land, use finishing-a-development-branch with base `artemch`, not `main`.
