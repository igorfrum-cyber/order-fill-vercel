# QA: регресс и AI exploratory

Два разных слоя. Не подменяйте один другим.

| Слой | Назначение | Команда |
| --- | --- | --- |
| Product regression | заранее написанные UI-сценарии | `npm run test:e2e --prefix frontend` |
| QA infra smoke | браузер жив, логин-экран открывается | `npm run qa:test --prefix frontend` |
| AI exploratory | агент сам ходит по UI, ищет баги | Cursor Agent + правило `ai-exploratory-qa` |

Продуктовые спеки остаются в [`frontend/e2e/`](../frontend/e2e/). Здесь — артефакты разведки, карта экранов и инструкция для агента. Инфра-smoke живёт рядом с продуктовыми спеками (`qa-infra.smoke.spec.js`), но основной `test:e2e` его не подхватывает.

## Стенд

```bash
cp .env.example .env   # если ещё нет
make up                # UI http://127.0.0.1:3200  gateway http://127.0.0.1:8080/healthz
```

Только frontend (логин без API, `getMe` падает → форма входа):

```bash
npm run dev --prefix frontend
```

Первый вход на полном стенде: в логах `identity-service` строка `bootstrap admin invite` и путь `/invite/...`. Готового пароля в репозитории нет. Живые Excel-сценарии — `REAL_E2E_*` и `npm run test:e2e:real --prefix frontend`, не этот каталог.

## Команды

Из корня:

```bash
make qa-test          # smoke, Chromium; локально окно на экране
make qa-test-headed   # то же с --headed
make qa-report        # HTML-отчёт последнего qa:test
```

Эквивалент:

```bash
npm run qa:test
npm run qa:test:headed
npm run qa:report
npm run qa:test --prefix frontend
```

`QA_LIVE=1` гоняет smoke против уже поднятого `http://127.0.0.1:3200` (без своего Vite). Другой origin: `QA_BASE_URL=http://127.0.0.1:3200`.

Перед первым прогоном, если Chromium ещё не ставили:

```bash
npm run test:e2e:install --prefix frontend
```

Локально запускайте Playwright вне sandbox (`all`), иначе окно не появится.

## Cursor / MCP

Проектный `.cursor/mcp.json` поднимает `@playwright/mcp` с конфигом `qa/playwright.mcp.json` (скриншоты в `qa/artifacts/mcp`, extra capability `vision`). Console и network в Cursor доступны через native Browser tools (`user-playwright`) без отдельного cap.

После смены `mcp.json` перезапустите MCP в Cursor.

Правило режима: [`.cursor/rules/ai-exploratory-qa.mdc`](../.cursor/rules/ai-exploratory-qa.mdc). Карта экранов: [`exploratory/map.md`](./exploratory/map.md). Шаблон отчёта: [`reports/qa-report.template.md`](./reports/qa-report.template.md). Рабочий отчёт сессии: [`reports/qa-report.md`](./reports/qa-report.md).

Бинарные скрины и HTML Playwright gitignored. Markdown-отчёты — нет.

## Промпт для агента

Скопируйте в чат Cursor Agent при поднятом UI:

```text
Режим AI Exploratory QA по .cursor/rules/ai-exploratory-qa.mdc и qa/README.md.

Стенд: http://127.0.0.1:3200 (gateway http://127.0.0.1:8080/healthz).
Не иди только по frontend/e2e. Сам выбирай пользовательские пути по qa/exploratory/map.md и frontend/src/features/app/routes.js.

Проверь функциональность, края, состояние, UX/UI. После переходов читай console и network, снимай скрины в qa/screenshots/.

Баг только после повторного воспроизведения, console, network, проверки «это не ожидаемо», по возможности взгляда в код. Root cause не выдумывай.

Не меняй продуктовую логику, расчёты, API, БД. Пиши qa/reports/qa-report.md. Подтверждённые баги отдельно от гипотез. Git: не коммить.
```
