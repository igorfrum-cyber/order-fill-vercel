# Order Fill: стартовое задание для агента разработчика

Передай этот файл агенту вместе с доступом к репозиторию. Начни с раздела
«Первое задание», затем используй остальные разделы как рабочую карту.
Все пути ниже считаются от корня репозитория и подходят для нового компьютера.

## Первое задание

Ты помогаешь разработчику настроить и развивать Order Fill. Сначала прочитай
этот файл, `AGENTS.md`, `CLAUDE.md`, `README.md`, `CONTRIBUTING.md` и два
проектных skills из раздела ниже. Проверь текущее состояние файлов, веток и
окружения. Подготовь локальный стенд, запусти проверки и объясни разработчику,
как войти в приложение и где менять нужную ему функциональность.

Доведи настройку до проверяемого результата. Устанавливай недостающие
инструменты обычным способом для обнаруженной ОС, соблюдая разрешения среды.
Если нужны права ОС, доступ к Docker, GitHub или production-серверу, назови
конкретный недостающий доступ. Не запрашивай и не публикуй пароли в чате.
Не объявляй установку завершённой только потому, что сборка или `/healthz`
ответили успешно: проверь контейнеры, вход, интерфейс и тесты отдельно.

Первый запуск означает локальное окружение. Production-развёртывание,
изменение серверных секретов и публикацию выполняй по отдельному поручению
разработчика с указанием целевого сервера и домена.

## Правила и источники истины

Основные инструкции для агента находятся в [AGENTS.md](./AGENTS.md) и
[CLAUDE.md](./CLAUDE.md). Этот файл объясняет начало работы и не заменяет их.
Явное актуальное поручение пользователя имеет приоритет над общими правилами.

Если документация расходится с реализацией, проверяй в таком порядке:

1. Код, `backend/deploy/docker-compose.yml`, канонический OpenAPI и `.proto`.
2. README сервиса, который владеет изменяемым поведением.
3. Корневой README, `docs/ARCHITECTURE.md`, `docs/service-boundaries.md`.
4. `docs/plans/` — история и проектные решения, не доказательство реализации.

Например, версия Go определяется `backend/pkg/go.mod` и `backend/go.work`:
не устанавливай старую версию из отдельного устаревшего абзаца документации.
В системе есть `inbound-service`; его изолированное хранилище входящих писем
описано в текущих `CLAUDE.md`, Compose и README сервиса.

## Что представляет собой эта версия проекта

`igorfrum` синхронизирована с рабочим React + Go приложением из `dev`.
Это полный проект с backend, базами, очередью, object storage и CI.
Старое приложение из корневого `src/` и ветки `main` — исторический прототип.
Не используй его как точку входа для запуска или основу новой реализации.

Текущий поток:

```text
React → gateway-service → gRPC-сервисы
                         file-service → S3/MinIO
                         job-service → Redis Stream → document-worker
                                                      matching / brand / calculation
                         identity / twofa / passkey / audit
                         inbound-service → отдельная БД и bucket вложений
```

Excel разбирается и записывается на backend. Браузер не парсит Excel,
gateway не выполняет бизнес-расчёты. matching, brand и calculation не читают
Excel и не обращаются к object storage. Макросы из `.xlsm` не исполняются.
Обычные файлы идут через file-service; отдельный контур inbound-писем —
через inbound-service. Межсервисные вызовы — protobuf/gRPC либо очередь.

## Получение проекта и skills

Для нового клона:

```bash
git clone --branch igorfrum --recurse-submodules https://github.com/igorfrum-cyber/order-fill-vercel.git
cd order-fill-vercel
git status --short --branch
git submodule status
```

Если проект уже склонирован, сначала проверь незакоммиченные изменения.
Не делай `reset --hard`, не затирай чужую работу. На чистом клоне обнови
нужную ветку через `git fetch origin --prune` и `git pull --ff-only`.
Для загрузки skills выполни:

```bash
git submodule sync --recursive
git submodule update --init --recursive
```

Используй закреплённые коммиты submodules. Не запускай `submodule update
--remote` и не обновляй сами skills в рамках настройки окружения.

| Когда читать | Skill / путь в репозитории |
| --- | --- |
| Перед изменением проекта | `.cursor/skills/order-fill-work/SKILL.md` |
| Документация, README, env, API, protobuf | `.cursor/skills/order-fill-docs/SKILL.md` |
| Любое изменение Go | `.cursor/skills/use-modern-go/SKILL.md` |
| Выбор профильных Go-навыков | `.cursor/skills/cc-skills-golang/skills/golang-how-to/SKILL.md` |
| Простота и объём решения | `.cursor/skills/ponytail/skills/ponytail/SKILL.md` |

Go-навыки загружай по задаче: testing для тестов, grpc для gRPC, database
для БД, security для безопасности, concurrency/context для конкурентности.
Не загружай весь каталог в контекст агента одновременно.

Если среда агента автоматически не видит `.cursor/skills/`, прочитай нужный
`SKILL.md` явно через файловый инструмент. Наличие расширения или MCP не
обязательно для чтения инструкции. Не требуется переносить глобальные
`.agents/`, `.codex/`, аккаунты, токены и настройки другого разработчика.
Для установки навыков в собственную среду используй её штатный механизм
и исходники из [.gitmodules](./.gitmodules), сохраняя версии проекта.

Перед изменением Go прочитай `use-modern-go` и выполни его wrapper:

```bash
sh .cursor/skills/use-modern-go/scripts/run-tool.sh list --go-version 1.26.9
```

Не обрезай вывод списка рекомендаций. Для конкретного файла предпочитай
`list --file-path <путь-к-файлу.go>`. Версию сверяй с текущим `go.mod`.

Если у агента доступны `systematic-debugging`,
`verification-before-completion`, `finishing-a-development-branch`, используй
их по назначению. Если нет, придерживайся процесса из этого файла: воспроизведи
ошибку, найди причину, внеси минимальную правку, проверь результат. Не объявляй
тесты зелёными без вывода команд и не сливай ветки без поручения пользователя.

## Окружение разработчика

Сначала определи ОС, архитектуру CPU, доступность Docker и занятость портов.
На Windows удобнее использовать WSL2 и работать с клоном внутри Linux-файловой
системы: команды проекта используют Bash, Make и Unix-утилиты.

Проверь:

```bash
git --version
node --version
npm --version
go version
docker version
docker compose version
make --version
buf --version
```

Node нужен версии 24 или новее; Go закреплён проектом, сейчас 1.26.9.
Buf в CI закреплён на 1.72.0. Docker запускает полный стек; Go и Node на
хосте нужны для разработки и локального verify. Установка Docker CLI без
работающего Docker daemon недостаточна.

Для проверки уязвимостей и protobuf-генерации понадобятся инструменты:

```bash
go install github.com/bufbuild/buf/cmd/buf@v1.72.0
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
```

Версии генераторов соответствуют заголовкам текущих файлов в
`backend/proto/gen/go/`; перед установкой сверь эти заголовки и
`backend/proto/buf.gen.yaml`. Сейчас Buf использует local plugins, поэтому
генераторы нужны в PATH. Каталог `$(go env GOPATH)/bin` должен быть в PATH.
`scripts/verify-go.sh` сам устанавливает недостающие закреплённые golangci-lint
и gosec; для первого запуска нужен доступ к сети. GitHub CLI (`gh`) полезен
для PR и CI, но не обязателен для локального запуска.

## Первый локальный запуск

Из корня проекта, сохраняя уже существующий `.env`:

```bash
test -f .env || cp .env.example .env
npm ci --prefix frontend
make hooks
npm run test:e2e:install --prefix frontend
docker compose --env-file .env -f deploy/docker-compose.yml up -d --build
docker compose --env-file .env -f deploy/docker-compose.yml ps
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

`make up` запускает тот же стек с выводом логов в текущем терминале.
UI: `http://127.0.0.1:3200`. В local найди запись `bootstrap admin invite`
в логах identity-service, затем открой её путь `/invite/...` на адресе UI
и заверши регистрацию. Invite — секрет: не включай его в отчёт или коммит.

```bash
docker compose --env-file .env -f deploy/docker-compose.yml logs --tail=100 identity-service
```

Проверь вход и навигацию в браузере. Для проверки обработки Excel используй
доступные разрешённые fixtures; отсутствие приватных файлов не считай
успешным тестом реальной обработки. Каталог `testdata/private/` не приходит
из Git и заполняется владельцем данных отдельно.

Для быстрой разработки UI при работающем backend можно остановить только
Compose frontend и запустить Vite, освободив порт 3200:

```bash
docker compose --env-file .env -f deploy/docker-compose.yml stop frontend
npm run dev --prefix frontend
```

Локальный Vite перенаправляет `/api` на `127.0.0.1:8080`.
`make down` останавливает стенд с сохранением volumes. Не применяй `down -v`
или `docker system prune --volumes` для исправления обычных ошибок запуска.

## Карта кода

| Задача | Владелец / точка входа |
| --- | --- |
| UI, маршруты, состояние, API-клиент | `frontend/src/`, `frontend/src/ui/`, `frontend/src/api/` |
| Публичный HTTP и session edge | `backend/services/gateway-service/` |
| Канонический OpenAPI | `backend/services/gateway-service/api/openapi.yaml` |
| Пользователи, компании, приглашения | `backend/services/identity-service/` |
| TOTP / WebAuthn | `backend/services/twofa-service/`, `backend/services/passkey-service/` |
| Задачи и очередь | `backend/services/job-service/` |
| Файлы и object storage | `backend/services/file-service/` |
| Excel, preview, document-worker | `backend/services/document-service/` |
| Сопоставление / бренды / расчёты | `backend/services/matching-service/`, `brand-service/`, `calculation-service/` |
| Входящие письма 1С | `backend/services/inbound-service/` |
| Audit | `backend/services/audit-service/` |
| Межсервисные контракты | `backend/proto/orderfill/` |
| Общие Go-пакеты | `backend/pkg/` |
| Запуск и production overlays | `backend/deploy/`, `deploy/`, `scripts/` |
| CI и агрегат Required checks | `.github/workflows/verify.yml` |

Перед правкой прочитай README владельца, его config, реализацию и ближайшие
тесты. Для поиска сначала используй `rg` / `rg --files`. Не импортируй
чужой `internal/` и не создавай второй источник бизнес-правил в gateway/UI.
Новый микросервис или отдельный routed screen предварительно согласуй.

## Ветки и ежедневная работа

`artemch` — общая база разработки. `dev` — публикация на существующий сервер.
`igorfrum` содержит переданную другу версию. `main` остаётся старым прототипом.
Название ветки `igorfrum` не разрешает прямые коммиты в неё.

Перед новой задачей на чистой рабочей папке:

```bash
git fetch origin --prune
git switch -c feat/<краткое-имя-задачи> origin/artemch
```

Если есть чужие изменения, сохрани их и используй отдельный worktree.
Вноси изменения в feature-ветке. Сначала найди owner и сформулируй ожидаемое
поведение, затем сделай правку и проверь её. Не смешивай задачу с массовым
рефакторингом, обновлением всех зависимостей или сменой дизайна.

Коммит — только после `make verify`. Push запускает его ещё раз через hook.
Для PR явно указывай целевую ветку: default branch репозитория — `main`,
поэтому не полагайся на значение по умолчанию в интерфейсе GitHub или `gh`.
Обычный PR направляй в `artemch`; синхронизацию с `igorfrum` и выпуск в `dev`
делай по отдельному поручению. Не выполняй force push и не обходи hooks.

Правила защиты веток проверяй в GitHub: локальный hook не заменяет серверную
защиту. На момент передачи её включение требовало доступа владельца; этот
файл не является подтверждением, что защита уже включена.

## Проверки перед передачей изменения

```bash
make verify
```

Команда проверяет toolchain, документацию, OpenAPI, Buf, frontend lint/tests/
component/build, Go lint/SAST/tidy/build/tests и Compose. После изменения UI:

```bash
npm run test:ui --prefix frontend
```

Это unit + component + Playwright. Локально Chromium открывается на экране;
агенту нужен доступ среды к запуску браузера. Моки E2E не подтверждают работу
реального backend. Для live Excel-сценариев смотри `frontend/e2e/README.md`
и `testdata/README.md`; учётные данные и fixtures передаются отдельно.

При изменении зависимостей, auth, сети или обработки файлов дополнительно:

```bash
make security
npm audit --prefix frontend
```

Go-тест отдельного сервиса запускается из его модуля с `GOWORK=off`.
После изменения `.proto` выполни `make -C backend proto-gen`, обнови обе
стороны RPC и закоммить исходники вместе с сгенерированным Go-кодом.
Generated-файлы вручную не исправляй. При изменении HTTP API обновляй
OpenAPI, gateway-тесты и клиент вместе. Миграции добавляй новыми файлами.
Таблицы env/RPC/HTTP обновляй через `make docs-sync`; проверь diff.

CI добавляет Go race/shuffle и проверку уязвимостей. `Required checks`
агрегирует обязательные CI-этапы, но не включает Playwright и `npm audit`.
Не называй зелёный CI доказательством прохождения этих двух проверок.

## Развёртывание и проверка выпуска

Push в `igorfrum` запускает CI, но **не публикует** её на существующий сервер.
Автодеплой в текущем workflow выполняется только после push в `dev` и
успешных обязательных проверок. Не меняй deploy branch или self-hosted runner
для настройки личного компьютера друга.

Если поручен выпуск на существующий стенд: проверь diff и тесты, выполни
согласованные слияния, push, дождись CI для конкретного SHA и отдельно
`Deploy self-hosted`. Успех подтверждай по логу `deploy ok` и состоянию
контейнеров. Если доступен адрес приложения, проверь UI и API после выпуска.

Для отдельного production-сервера сначала согласуй сервер, домен, владельца
данных и способ доступа. Далее следуй разделу развёртывания в README и
`scripts/prod-env.sh`: отдельные секреты, HTTPS origins, secure cookies,
`WEBAUTHN_RP_ID`, `APP_ENV=production`, `GRPC_TLS_MODE=mtls`, TLS overlays,
PostgreSQL/Redis/S3 и сохранение volumes. `make https` подготавливает TLS и
поднимает production-стек; это действие выполняется на целевом сервере.
Установка GitHub runner описана в README и `scripts/install-github-runner.sh`.
Для отдельного сервера не меняй маршрутизацию существующего runner вслепую.
Vercel-конфигурация разворачивает только frontend и сама backend не запускает.

`.env`, TLS private keys, пароли, токены, invite-ссылки и реальные workbook
не должны попадать в Git, этот файл или диагностический отчёт. Локальные
значения `.env.example` не подходят для production. Не копируй `.env`
другого разработчика или действующего сервера в новый стенд.

## Если что-то не работает

Сначала воспроизведи сбой и зафиксируй команду, exit code и короткий фрагмент
ошибки без секретов. Проверь, где проблема: окружение, сеть, конфигурация,
контракт или приложение. Меняй одну причину за раз.

- Docker недоступен: проверь daemon и права, не удаляй volumes.
- `npm ci` / Go download не проходят: проверь DNS/TLS/прокси и доступ к
  registry, затем повтори загрузку. Не отключай TLS-проверку сертификатов.
- Docker build падает на `mirror.gcr.io`: проверь сетевой доступ; единичный
  таймаут допускает повтор этапа. Повторяющийся сбой требует диагностики.
- E2E зависает на `page.goto` с ожиданием `load`: проверь сетевые запросы,
  включая внешние Google Fonts. Не увеличивай timeout и не удаляй assertions
  без подтверждения причины. Не запускай параллельно два набора с одним
  каталогом test-results.
- Порт 3200 занят: выясни, работает ли Compose frontend или другой Vite.
- `/readyz` зелёный: всё равно проверь downstream и пользовательский сценарий.
- Нужны права GitHub Admin: укажи это владельцу; Write достаточно для кода,
  но не для настройки защиты веток.

## Что должно быть в отчёте агента

Укажи ОС и версии инструментов, ветку/SHA, что установлено, какие контейнеры
работают, как разработчику открыть UI и войти. Отдельно перечисли команды
проверок с результатами, обнаруженные ограничения и следующие действия.
Различай «пройдено», «упало» и «не запускалось». Для изменений перечисли
затронутые файлы и статус commit/PR/push/deploy. Не заменяй проверку обещанием
«должно работать» и не выдумывай выполненные действия на чужом компьютере.
