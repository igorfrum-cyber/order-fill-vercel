---
name: order-fill-docs
description: >-
  Writes and updates Order Fill documentation after code changes. Use when
  writing or reviewing documentation, README, CONTRIBUTING, CLAUDE.md, llms.txt,
  docs/, service READMEs, OpenAPI, protobuf, godoc, env vars, gateway routes,
  architecture, or when the user mentions документация, README, OpenAPI, proto,
  конфигурация, or описание API. Regenerates env/RPC/HTTP tables from code and
  keeps human purpose text. Supersedes samber/cc-skills-golang@golang-documentation
  for this repository's READMEs; still apply that skill for Go doc comments.
---

# Order Fill documentation

This company skill supersedes `samber/cc-skills-golang@golang-documentation` for
Order Fill README/docs. Still read that skill for godoc comments and Example tests.

## Sources of truth

Runtime code, Compose, canonical OpenAPI, and protobuf win over prose. Service
READMEs describe that service only. Do not invent ownership across boundaries.

## Generated lists

Env tables, RPC tables, and gateway HTTP routes live between `<!-- docs-sync:* -->`
markers. After changing `config.go`, `.proto`, or `router.go`:

```bash
node scripts/sync-docs.mjs --write
```

Keep the purpose column; the sync preserves it. Do not hand-edit other columns
inside the markers. New keys/RPCs get a stub purpose — fill it in the same change.

## Required in the same change as behavior

- affected `backend/services/<service>/README.md` (sections Конфигурация, Тесты, Эксплуатация)
- OpenAPI and gateway tests if the public HTTP contract changed
- `.env.example` and Compose if an operator env var changed
- `docs/ARCHITECTURE.md` / `docs/service-boundaries.md` if ownership or flow changed
- root README if launch, topology, or deploy changed

`make verify` runs the sync and fails if generated lists were not committed.

## Prose

Code shows what happens. Docs say why, when to use it, and the ceiling. No
marketing, no restating the signature, no undocumented future work. Russian in
user-facing README/docs unless the file is already English (`CLAUDE.md`, `llms.txt`).
