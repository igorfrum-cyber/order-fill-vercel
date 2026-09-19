# Playwright

- `npm run test:e2e --prefix frontend` — mock-регресс, `*.real.spec.js` не берёт.
- `npm run test:e2e:real --prefix frontend` — живые Excel из `testdata/private`.

По умолчанию real suite гоняет одну UI-пару CHRISTINA PROFF:

- бланк `Актуальный_бланк PROFF (1).xlsx`
- таблица продаж `Кристина Тюмень .xlsx`

Полный matched-pairs матрикс (все бренды) — `REAL_E2E_ALL_PAIRS=1`.

Нужны `REAL_E2E_LOGIN` / `REAL_E2E_PASSWORD` / `REAL_E2E_OWNER_LOGIN` /
`REAL_E2E_OWNER_PASSWORD` и папки `Бланки/` + `таблицы продаж/`. Иначе skip.
В git worktree private ищется и в checkout выше `.worktrees/`. Коммерческие
xlsx в git не класть. Другой каталог — `ORDER_FILL_PRIVATE_TESTDATA`.
