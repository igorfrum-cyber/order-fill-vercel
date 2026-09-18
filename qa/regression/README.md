# Регресс

- **Продукт:** [`frontend/e2e/*.spec.js`](../../frontend/e2e/) — `npm run test:e2e --prefix frontend`. Сюда не копировать.
- **Инфра QA:** [`frontend/e2e/qa-infra.smoke.spec.js`](../../frontend/e2e/qa-infra.smoke.spec.js) — `npm run qa:test --prefix frontend` (конфиг `frontend/playwright.qa.config.js`). Основной suite его игнорирует (`*.smoke.spec.js`).

Артефакты smoke (скрины, HTML-report) пишутся в `qa/screenshots/` и `qa/artifacts/`. Exploratory-сценарии не превращайте в большой заранее написанный E2E.
