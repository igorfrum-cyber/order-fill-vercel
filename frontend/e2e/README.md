# Playwright

- `npm run test:e2e --prefix frontend` — mock-регресс, `*.real.spec.js` не берёт.
- `npm run test:e2e:real --prefix frontend` — живые Excel из `testdata/private`.

Real suite параметризуется matched pairs (`brandPairs.js`): бланк × таблица продаж одного бренда, не декартово каталогов. Нужны `REAL_E2E_LOGIN` / `REAL_E2E_PASSWORD` / `REAL_E2E_OWNER_LOGIN` / `REAL_E2E_OWNER_PASSWORD` и папки `Бланки/` + `таблицы продаж/`. Иначе skip. Коммерческие xlsx в git не класть.
