# QA report

- Date: 2026-09-18
- URL: http://127.0.0.1:3200
- Environment: local Compose (`deploy-frontend-1` :3200, `deploy-gateway-service-1` :8080 `/healthz` → ok). Roles used: owner `art`, purchaser `ivanov`. Earlier guest + `qa_platform` still in this stand’s history. Passwords not recorded.
- Browser: Chromium (Playwright MCP `user-playwright`)
- Areas explored: guest login edges; owner `art` (queue, failed jobs, empty-brand needs_review, company, users, inbound message, jobs/new, north, account); purchaser `ivanov` (upload, real Christina + Angiopharm E2E, brand-mismatch pair, budget, preview, download, jobs filters, north empty + failed merge, account, denied routes, unauth API, purchaser API authz, 390px); wrong file type
- Confirmed issues: 4

## Coverage (what was actually clicked)

### Guest (earlier in this stand)

- `/` login: empty, short password, wrong password, submit disabled until login + password ≥10
- `/c/<slug>` known `art` and unknown `no-such-company`
- `/invite/bogus-token`
- Guest `/overview`, `/jobs`, `/users`, `/account`, unknown path
- Viewport ~390 and ~1280

### `art` — Владелец компании (home `/queue`)

Nav used: Очередь, Люди, Компания, Интеграция 1С, Файлы. Profile shows «Владелец компании». `/jobs/new` reachable by URL even though not in owner nav.

| Screen | Controls exercised |
| --- | --- |
| `/queue` | list of 41 stuck jobs; **Открыть** on two «Ошибка» rows; **Проверить** on empty-brand «На проверке» (`BUtt8022zimg9pk4oy93ZA`, `meW8l4wCs5chDhZfvTf3pA`) |
| `/jobs/CIkXUIz9fZ5ccCoDeIV-VQ` and `/jobs/RBTlschFJDzRFh72DLuQpA` | failed-job fill UI (see BUG-002) |
| `/company` | logo hint, name, login slug, blank-order fields, brand discounts; **Сохранить** disabled while unchanged |
| `/users` | role picker, **Пригласить** disabled without login; listed `qa_owner`, `art`, `qa_admin`, `qa_buyer`, `ivanov`. **Выключить** / **Сброс доступа** not pressed (would disable live accounts) |
| `/inbound` | stats, message table; opened «Тестовое письмо с форматированием»; iframe «Безопасный просмотр» |
| `/jobs`, `/jobs/new`, `/north`, `/account` | opened in the owner session (upload/north/account also fully driven as `ivanov`) |

Not pressed as owner: creating an invite, disabling a user, changing company slug/name, inbound «Настроить» persist, «Выйти со всех устройств».

### `ivanov` — Закупщик (home `/jobs/new`)

Nav used: Работа, Файлы. Profile menu: Мой профиль, Выйти.

| Screen | Controls exercised |
| --- | --- |
| `/jobs/new` | help + 3-step tour (Далее / Назад / Понятно); warehouse checkbox on/off; file pickers; **Обработать** (wrong type + real pairs + Christina×Angiopharm mismatch, double-click); **К выгрузкам** |
| Fill `/jobs/eIJk_RZF9phG4gcNLmydYA` (Christina) | filters, dispute checkboxes «Оставляю как есть», **Заказ до суммы**, **Проверить файлы**, modal **Продолжить проверку** |
| Preview Christina | tabs Бланк / Таблица 1С, **Скачать файлы** |
| Fill `/jobs/Y3pYcV3KfAZjXB819LY7nA` (Angiopharm) | all row filters, search + **Очистить поиск**, qty **Больше**, comment, **Отчёт для 1С**, budget (empty/invalid/discount/floor), **Применить**, 4 dispute checkboxes, preview confirm, **Назад к правкам**, leave-upload **Остаться** |
| Preview Angiopharm | Бланк / Таблица 1С, **Скачать файлы**, refresh on job URL (returned to preview) |
| `/jobs` | brand/status filters (CHRISTINA, Готово, Сбой → «Нет выгрузок с такими фильтрами»), CTA **Соединить северные бланки**; **Открыть** on failed Север (`yp6OAvwDP0f4qi8QvmpijA`); **Проверить** on empty-brand `vh7GMN86A4H6Fn_d88G00A` |
| `/north` | empty **Соединить бланки** → «Добавьте хотя бы один бланк города.» (no API). CHRISTINA HOME+PROFF with sales tables + PROFF blank → failed city-name error on-screen; history **Открыть** then empty form (BUG-004) |
| `/account` | opened 2FA setup (not confirmed), **Сгенерировать** password (not saved), empty **Сохранить** disabled. **Выйти со всех устройств** not pressed |
| Denied URLs | `/company`, `/users`, `/queue`, `/inbound`, `/overview` → silent redirect to `/jobs/new` |
| `/jobs/no-such-id` | `GET` 404 then redirect to `/jobs/new` |
| Viewport | 390px upload (`scrollWidth === 390`, no overflow); restored 1280 |

### Real uploads (filenames only, files not committed)

- Christina: `Кристина Тюмень .xlsx` + `Актуальный_бланк PROFF.xlsx` → job `eIJk_RZF9phG4gcNLmydYA`. UI: CHRISTINA, 119 unpaired, 2 duplicates. Order 105 320,00 ₽ → after budget 10% target 110 000: 94 788,00 ₽ → 114 336,00 ₽ (2 lines). Preview ~420 rows. Downloads: `Актуальный_бланк PROFF заполненный.xlsx`, `Кристина Тюмень заполненная таблица.xlsx`. Console clean on this path. `POST /api/v1/jobs/order-fill` 202, `budget-plan` 200, preview windows 200.
- Angiopharm: `Ангио Тюмень .xlsx` + `2026 08 25 Бланк заказа ANGIOPHARM.xlsx` → job `Y3pYcV3KfAZjXB819LY7nA`. UI: ANGIOPHARM, 160 unpaired, 4 name-only matches. Order 4 707 633,00 ₽. Budget target 1 000 000, subtract 30%: stopped at 1-month floor «3 295 343,10 ₽ → 1 396 220,00 ₽», 75 lines. Preview 335 rows. Downloads: filled Angiopharm blank + `Ангио Тюмень заполненная таблица.xlsx`. Also downloaded `отчет для исправления в 1С.csv`. `order-fill` 202, `budget-plan` 200.
- Wrong type: `qa-not-excel.txt` + Angiopharm blank (see BUG-001).
- Mismatch: `Кристина Тюмень .xlsx` + Angiopharm blank → job `qBRmBhHlCG3QZi_QqI-tow`. Brand CHRISTINA, filled 0, 355 «нет в 1С», 144 «нет в бланке», 4 спора. Двойной клик «Обработать» → один `POST` 202.
- North fail: sales as HOME + `Актуальный_бланк PROFF.xlsx` as PROFF → job `yp6OAvwDP0f4qi8QvmpijA` `north_merge` failed (`не узнали город по имени файла`). На `/north` после submit ошибка видна; из списка/URL — нет (BUG-004).
- Not uploaded this round: successful KLAPP / Skin Synergy pairs; no north city-named blanks in `testdata/private/`.

### `qa_platform` (earlier pass, not re-driven this exhaustive round)

overview, jobs view-only, companies, brand-rules, users company picker, inbound, account, role-denied `/north` `/jobs/new` `/queue` `/company`. Buyer/owner logins `qa_buyer` / `qa_owner` were 401 in that pass.

## Confirmed bugs

### BUG-001

Title:
Неверный тип файла: UI принимает `.txt`, API отвечает 500 вместо 400

Severity:
Medium

Area:
Загрузка заказа / gateway + job-service

URL:
http://127.0.0.1:3200/jobs/new

Environment:
purchaser `ivanov`, local :3200

Steps to reproduce:

1. Войти как `ivanov`.
2. На `/jobs/new` выбрать таблицу продаж — файл `.txt` (через file chooser; `accept` браузера можно обойти).
3. Выбрать любой бланк `.xlsx`.
4. Нажать «Обработать».
5. Повторить «Обработать».

Expected:
Клиент не принимает не-Excel, либо сервер отвечает 400 `bad_request`. Текст «Нужен файл Excel: .xlsx или .xlsm.» допустим.

Actual:
Dropzone по `onChange` кладёт `.txt` в состояние (проверка `fileMatchesAccept` есть только на drop). «Обработать» активно. `POST /api/v1/jobs/order-fill` → **500** `{"code":"create_job_failed","message":"invalid job: file \"qa-not-excel.txt\" must be .xlsx or .xlsm"}`. UI всё же показывает дружелюбную фразу. Console: Failed to load resource 500.

Reproducibility:
2/2

Console:
`Failed to load resource: the server responded with a status of 500 (Internal Server Error) @ /api/v1/jobs/order-fill`

Network:
`POST /api/v1/jobs/order-fill` → 500, twice (request indices 14 and 15 in the session)

Screenshot:
`qa/screenshots/51-ivanov-wrong-type-txt.png`, `qa/screenshots/52-ivanov-wrong-type-500.png`

Probable root cause:
`ValidateUploads` в job-service возвращает `domain.ErrInvalid`. gRPC `CreateJob` отдаёт эту ошибку без `codes.InvalidArgument`. Gateway `writeGRPCError` для неизвестного кода ставит 500 и `create_job_failed`. Параллельно `SetupUpload.jsx` `onChange` не вызывает `fileMatchesAccept` (только `onDrop`).

Relevant source code:
`backend/services/job-service/internal/domain/job.go` (ValidateUploads); `backend/services/job-service/internal/transport/grpcapi/server.go` CreateJob; `backend/services/gateway-service/internal/transport/httpapi/http.go` writeGRPCError; `frontend/src/ui/order/SetupUpload.jsx` Dropzone onChange vs onDrop

Regression test:
Created

### BUG-002

Title:
Очередь «Ошибка / Откройте, чтобы увидеть причину», а карточка джобы — пустая сверка «Критичных проблем нет»

Severity:
High

Area:
Очередь владельца / resume заказа

URL:
http://127.0.0.1:3200/queue → `/jobs/CIkXUIz9fZ5ccCoDeIV-VQ` и `/jobs/RBTlschFJDzRFh72DLuQpA`

Environment:
owner `art`, local :3200

Steps to reproduce:

1. Войти как `art`, открыть `/queue`.
2. В блоке «Застрявшие выгрузки» нажать «Открыть» у строки со статусом «Ошибка».
3. Повторить на второй строке «Ошибка».

Expected:
По подсказке статуса failed («Не получилось обработать. Откройте строку, чтобы увидеть причину.») на экране видна причина сбоя.

Actual:
`GET /api/v1/jobs/:id` 200, `GET .../report` **404**. Экран заполнения с нулями во всех корзинах, «Нет позиций в этой категории», футер «Критичных проблем нет», кнопка «Проверить файлы» активна. Сообщение об ошибке джобы не показано. Console: report 404.

Reproducibility:
2/2 (два разных failed job id)

Console:
`Failed to load resource: 404 @ /api/v1/jobs/<id>/report`

Network:
job 200; report 404

Screenshot:
`qa/screenshots/62-art-failed-job-empty.png`

Probable root cause:
`loadOrderResume` глотает ошибку отчёта (`getJobReport(jobId).catch(() => null)`), не ветвит `job.status === "failed"` и не прокидывает `job.error`. `FillStage` при пустых строках считает `canProceed` и пишет «Критичных проблем нет».

Relevant source code:
`frontend/src/App.jsx` `loadOrderResume`; `frontend/src/ui/order/FillStage.jsx` footer; `frontend/src/features/report/reportModel.js` `jobStatusHint` / `jobNextAction` for failed

Regression test:
Created

### BUG-003

Title:
«На проверке» без report.json: пустая сверка, «Критичных проблем нет», превью 404, скачивание 500

Severity:
High

Area:
Очередь / resume заказа / файлы объекта

URL:
http://127.0.0.1:3200/queue → `/jobs/BUtt8022zimg9pk4oy93ZA`, `/jobs/meW8l4wCs5chDhZfvTf3pA` (owner `art`); `/jobs/vh7GMN86A4H6Fn_d88G00A` (purchaser `ivanov`)

Environment:
owner `art` and purchaser `ivanov`, local :3200. 9 of 41 `needs_review` jobs (all without brand, created 5–7 Sept) return report 404. Newer branded `needs_review` jobs still have report 200.

Steps to reproduce:

1. Войти как `art`, открыть `/queue`.
2. Нажать «Проверить» у строки «Бланк ·» без бренда (например art · 7 сент., 19:17).
3. Повторить на второй безбрендовой строке (art · 7 сент., 19:14).
4. Нажать «Проверить файлы».
5. Как `ivanov` открыть `/jobs/vh7GMN86A4H6Fn_d88G00A` и снова нажать «Проверить файлы».

Expected:
Джоба `needs_review` без отчёта не выглядит как успешная сверка. Пользователь видит, что данных нет / файлы недоступны. «Проверить файлы» не ведёт на превью с активным «Скачать файлы».

Actual:
`GET /api/v1/jobs/:id` 200, статус `needs_review`, в payload есть input/output имена. `GET .../report` **404**. Экран сверки: все корзины 0, «Нет позиций в этой категории», футер «Критичных проблем нет», «Проверить файлы» активна. Превью: «Не нашли то, что искали.» + «Собираю сетку...» + «Скачать файлы». `GET .../files/output-1` **500** `{"code":"download_failed","message":"not found"}`. Контроль: живой Christina job `eIJk_RZF9phG4gcNLmydYA` скачивается 200, ~46 KB xlsx.

Reproducibility:
3/3 (два id у `art`, один у `ivanov`)

Console:
`Failed to load resource: 404 @ /api/v1/jobs/<id>/report`; после «Проверить файлы» 404 `@ .../files/output-2/preview`

Network:
job 200; report 404 `not_found`; files list 200 (ссылки output-1/output-2); download 500 `download_failed` / `not found`; preview 404

Screenshot:
`qa/screenshots/70-empty-needs-review.png`, `qa/screenshots/71-empty-review-preview-404.png`

Probable root cause:
`loadOrderResume` при отсутствии отчёта подставляет пустые `rows` и не отличает «нет report.json» от «пустая сверка». `FillStage` при нуле дублей считает `canProceed` и пишет «Критичных проблем нет». Ссылки на output остаются в job, объекты в file-service уже `not found`; gateway `downloadFile` мапит это в 500.

Relevant source code:
`frontend/src/features/jobs/orderJobWorkflow.js` `loadOrderResume` (`getJobReport(...).catch(() => null)` for non-failed); `frontend/src/ui/order/FillStage.jsx` footer `canProceed`; `backend/services/gateway-service/internal/transport/httpapi/jobs.go` `getReport`, `downloadFile`

Regression test:
Created

### BUG-004

Title:
Сбой северного merge: список обещает причину, «Открыть» даёт пустую форму Севера; прямой URL — пустую сверку бланка

Severity:
High

Area:
Север / история файлов / resume джобы

URL:
http://127.0.0.1:3200/jobs (строка «Север / Ошибка») и http://127.0.0.1:3200/jobs/yp6OAvwDP0f4qi8QvmpijA

Environment:
purchaser `ivanov`, local :3200. Job `yp6OAvwDP0f4qi8QvmpijA`: `type=north_merge`, `status=failed`, `error.message` = «не узнали город по имени файла "Актуальный_бланк PROFF.xlsx"…». На экране `/north` сразу после submit эта фраза была видна.

Steps to reproduce:

1. На `/north` получить failed merge (или взять уже существующий `yp6OAvwDP0f4qi8QvmpijA`).
2. Открыть `/jobs`, у строки «Север / Ошибка» нажать «Открыть» (подсказка статуса: «Не получилось обработать. Откройте строку, чтобы увидеть причину.»). Повторить «Открыть».
3. Отдельно ввести URL `/jobs/yp6OAvwDP0f4qi8QvmpijA`. Повторить переход.

Expected:
Пользователь видит ту же причину, что вернул API / что было на экране Севера после сбоя.

Actual:
- Клик «Открыть» в истории: `onOpen` для `north_merge` делает `go("north")`. URL становится `/north`, файлы не выбраны, бренд сброшен на ANGIOPHARM, статус «Готов к загрузке», текста ошибки нет.
- Прямой URL `/jobs/:id`: `parseAppPath` считает это экраном заказа. Пустая сверка бланка, «Критичных проблем нет», «Проверить файлы», `GET .../report` 404. Тип `north_merge` и `error` с API не показаны.

Reproducibility:
2/2 на клике «Открыть»; 2/2 на прямом URL

Console:
`404 @ /api/v1/jobs/yp6OAvwDP0f4qi8QvmpijA/report` на прямом URL. На клике «Открыть» нового 404 report нет — просто пустой `/north`.

Network:
`GET /api/v1/jobs/yp6OAvwDP0f4qi8QvmpijA` 200 (`type: north_merge`, `status: failed`, error message present); report 404 on deep link

Screenshot:
`qa/screenshots/75-ivanov-north-open-from-history.png`, `qa/screenshots/74-ivanov-north-failed-as-fill.png`

Probable root cause:
`JobHistory` глушит `href` и вызывает `openJob`. `openJob` для `north_merge` только `go("north")` — без job id и без error. `applyLocation` для любого `/jobs/:id` зовёт `loadOrderResume` как order-fill и не смотрит `job.type`.

Relevant source code:
`frontend/src/ui/admin/JobHistory.jsx` onClick `onOpen(job)`; `frontend/src/App.jsx` `openJob` (`job.type === "north_merge"` → `go("north")`) and `applyLocation` → `loadOrderResume`; `frontend/src/features/app/routes.js` `parseAppPath` `/jobs/:id` → screen `order`

Regression test:
Created

## UX observations

- Кнопка «Войти» disabled без логина или если пароль короче 10 символов. Сообщение при 401 намеренно одно и то же.
- Passkey на `127.0.0.1` скрыт / «Face ID на этом адресе недоступен»; на `*.localhost` блок быстрого входа есть.
- Неизвестный job id: 404 + тихий редирект на home роли, без тоста.
- Закупщик с чужих маршрутов (`/company`, `/users`, `/queue`, `/inbound`, `/overview`) уходит на `/jobs/new` без пояснения «нет доступа».
- Бюджет Angiopharm честно останавливается у запаса 1 месяц и предлагает пересчитать; чекбокс «Разрешить запас меньше 1 месяца…» сбрасывает форму, повторный «Рассчитать» нужен вручную.
- «Заменить» файл, если chooser закрыть без выбора, очищает слот (onChange → null).
- Help-тур на загрузке: 3 шага, Назад/Далее/Понятно работают.
- Фильтры списка файлов: пустое состояние «Нет выгрузок с такими фильтрами.»
- Сохранить профиль компании disabled, пока нет изменений. Пригласить disabled без логина.
- HTML письма inbound в iframe `sandbox=""`; в console пачка CSP `style-src` на inline стилях srcdoc. Заголовок «Безопасный просмотр» — похоже на намеренную песочницу, не на поломку API (message GET 200, srcdoc не пустой).
- Таблица продаж Christina + бланк Angiopharm: UI пишет «Похоже, Christina», без предупреждения о чужом бланке. Сверка: filled 0, 499 без пары, 4 спора — неверных количеств в заказ не ушло.
- Двойной клик «Обработать» на mismatch: один `POST /api/v1/jobs/order-fill` 202, второй job не создался.
- Без сессии все проверенные `/api/v1/jobs`, `/companies`, `/inbound`, `/brand-rules` → 401.
- Закупщик `ivanov`: чужие джобы той же компании (`qa_buyer` / `art`) → 404; admin GET `/users` `/order-profile` `/inbound` `/companies` `/brand-rules` → 404; POST order-profile → 403. `?company_id=other-company` на list jobs → пустой список, без чужих данных.

## Warnings

- Учётки `qa_buyer` / `qa_owner` в прошлой сессии давали login 401; живые секреты этой сессии — `art` и `ivanov`.
- `GET /api/v1/auth/me` без cookie → 401 в console (ожидаемо).
- Открытие 2FA setup у `ivanov` вызвало `POST /api/v1/auth/2fa/setup` 200; подтверждение кодом **не** делалось. Секрет на экран не копировать в отчёт.
- Исторические джобы «Север / Ошибка» в очереди; живой north merge на файлах с городом в имени не гонялся (нет таких бланков в `testdata/private/`).
- В исходниках уже есть черновик failed-stage для BUG-002 (`OrderFillApp` / `loadOrderResume`); на живом `:3200` failed `north_merge` по URL всё ещё открывается как пустая сверка бланка (BUG-004).

## Potential issues (not confirmed)

- Тихий редирект с несуществующего `/jobs/:id` и с запрещённых ролью URL: нет сообщения. Может быть задумано (`screenAllowed` / `loadOrderResume` catch).
- CSP-ошибки в console при просмотре HTML inbound: шум; содержимое, судя по srcdoc, есть. Не регистрировалось как BUG без доказательства пустого iframe для пользователя.
- На `/inbound` «Адрес для 1С: не задан», при этом приём включён и письма есть — не проверялось, ожидаемо ли это для стенда.
- Mismatch бренда (Christina sales + Angiopharm blank) не блокирует обработку. Сейчас filled=0; не доказано, что при частичном совпадении артикулов уйдут чужие количества.

## Not tested

- Реальные пары KLAPP и Skin Synergy из `testdata/private/` до успешной сверки.
- North merge с файлами, в имени которых есть город (Сургут / Вартовск / Уренгой / Тюмень).
- Второй склад Тюмени на order-fill (чекбокс включали, файл склада не грузили).
- Две вкладки / refresh mid-job на живой обработке (двойной submit на mismatch проверен).
- Подтверждение 2FA, смена пароля, passkey register, «Выйти со всех устройств».
- Приглашение пользователя, выключение сотрудника, смена slug/названия компании, загрузка логотипа.
- Настройки inbound (сохранение email отправителя).
- Повторный полный проход `qa_platform` в этом exhaustive раунде (есть только ранняя разведка).
- Production nginx vs Vite.
- Вторая компания / platform_admin IDOR между тенантами (на стенде живые `art`/`ivanov` в одной компании).

Regression tests: BUG-002 помечен Created другим агентом. BUG-003/004 — Created.
