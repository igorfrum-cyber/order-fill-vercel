# Order Fill

[![Verify](https://github.com/igorfrum-cyber/order-fill-vercel/actions/workflows/verify.yml/badge.svg)](https://github.com/igorfrum-cyber/order-fill-vercel/actions/workflows/verify.yml)

Order Fill — веб-приложение для заполнения поставщицких Excel-бланков по данным
из 1С, проверки сопоставлений, ручного согласования спорных строк и подготовки
итоговых файлов. Проект также поддерживает режим «Север»: объединение заявок
городов, расчёт перемещений из Тюмени и заказа у поставщика.

Текущая рабочая реализация — frontend на React и backend v2 из независимых
Go-микросервисов. Старое описание браузерной реализации сохранено только как
[историческая справка](./docs/current-behavior.md).

## Как выглядит рабочий поток

```text
Пользователь загружает Excel-файлы
              │
              ▼
React UI ──HTTP──> gateway-service ──gRPC──> file-service / job-service
                                              │              │
                                              │              ▼
                                         PostgreSQL     Redis Stream
                                                             │
                                                             ▼
                                                     document-worker
                                                       │    │    │
                                                       ▼    ▼    ▼
                                                  matching brand calculation
                                                       │
                                                       ▼
                                          отчёт, preview и итоговые .xlsx
```

После запуска доступность публичного API можно проверить так:

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

## Быстрый старт

### Требования

- Docker с поддержкой `docker compose` — для запуска всего приложения;
- Node.js 24+ — для отдельной разработки frontend и нагрузочного runner;
- Go 1.26.7 — для локальной сборки и тестирования backend;
- `golangci-lint` 2.12.2 и `gosec` 2.22.10 — для полного локального lint;
- Buf 1.x — для проверки protobuf-контрактов и генерации gRPC-кода;
- `govulncheck` — для локальной проверки достижимых Go-уязвимостей;
- `make` и Bash — для команд проекта.

Для первого знакомства достаточно Docker: базы данных и внутренние сервисы
поднимаются вместе с приложением.

Для полного локального gate установите Buf и `govulncheck`:

```bash
brew install bufbuild/buf/buf
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
```

### Запуск полного стенда

```bash
cp .env.example .env
docker compose --env-file .env -f deploy/docker-compose.yml up --build
```

То же через Makefile:

```bash
make up
```

После старта:

| Компонент | Адрес | Назначение |
| --- | --- | --- |
| Web UI | <http://127.0.0.1:3200> | Пользовательский интерфейс (Compose публикует только loopback) |
| Gateway liveness | <http://127.0.0.1:8080/healthz> | Проверка процесса gateway |
| Gateway readiness | <http://127.0.0.1:8080/readyz> | Проверка готовности HTTP API |
| MinIO API | <http://127.0.0.1:9000> | Локальное S3-compatible API |
| MinIO Console | <http://127.0.0.1:9001> | Интерфейс локального object storage |
| PostgreSQL | `127.0.0.1:5432` | Локальная БД |
| Redis | `127.0.0.1:6379` | Очередь и временное состояние |

Для первого входа в `local` найдите в логах `identity-service` запись
`bootstrap admin invite` и откройте выданный путь `/invite/...` в Web UI.
Вне local токен в лог не пишется:

```bash
docker compose --env-file .env -f deploy/docker-compose.yml logs identity-service
```

Остановить стенд и контейнеры, сохранив данные в Docker volumes:

```bash
make down
```

## Возможности

### Заполнение заказа

- загрузка исходной выгрузки 1С и одного или нескольких бланков поставщика;
- определение бренда и структуры таблиц;
- сопоставление по артикулам и нормализованным названиям;
- отдельная обработка дубликатов и маркировки «Честный знак»;
- категории отчёта `needs_decision`, `not_in_source`,
  `check_name_or_volume`, `not_in_blank`, `to_order` и `order_not_needed`;
- предпросмотр больших workbook-файлов по окнам строк;
- ручная правка количества с обязательным комментарием при отклонении от
  рассчитанного значения;
- формирование итогового бланка, обновлённой исходной книги и архива файлов.

### Режим «Север»

- приём заполненных бланков городов;
- определение города и типа бланка;
- объединение потребностей по городам;
- расчёт количества из Тюмени, заказа у поставщика и перемещений;
- ручная корректировка фактического заказа;
- выпуск сводных бланков, файлов перемещений и дополнительных таблиц бренда.

### Пользователи и эксплуатация

- компании, пользователи, роли и отдельные страницы входа компаний;
- сессии, смена пароля и отзыв активных сессий;
- TOTP 2FA с backup codes;
- WebAuthn/passkeys: Face ID, Touch ID, Windows Hello и совместимые ключи;
- журнал аудита и операционный экран состояния для `platform_admin`;
- асинхронные jobs с историей, статусом, отчётом и скачиванием результатов.

Подробное текущее поведение обработки файлов описано в
[архитектурной документации](./docs/ARCHITECTURE.md) и документах по отдельным
сценариям в [`docs/`](./docs/README.md).

## Архитектура

Браузер обращается только к `gateway-service`. Frontend-контейнер отдаёт
статические файлы через nginx и проксирует same-origin запросы `/api/` в
gateway. Межсервисные контракты описаны protobuf, а долгие операции не держат
HTTP-запрос открытым: `job-service` публикует сообщение в Redis Stream
`order-fill:jobs`, после чего его обрабатывает `document-worker`.

### Микросервисы

| Сервис | Ответственность | Интерфейсы | Состояние |
| --- | --- | --- | --- |
| [`gateway-service`](./backend/services/gateway-service/README.md) | Публичный HTTP API, сессии на edge, CORS/CSRF, валидация и оркестрация | HTTP `:8080`; gRPC clients | Stateless |
| [`identity-service`](./backend/services/identity-service/README.md) | Пользователи, компании, роли, invites и сессии | gRPC `:9091`; health `:8082` | PostgreSQL |
| [`twofa-service`](./backend/services/twofa-service/README.md) | TOTP, backup codes и защита 2FA-секретов | gRPC `:9092`; health `:8083` | PostgreSQL + Redis |
| [`passkey-service`](./backend/services/passkey-service/README.md) | WebAuthn credentials и ceremonies | gRPC `:9093`; health `:8084` | PostgreSQL + Redis |
| [`job-service`](./backend/services/job-service/README.md) | Jobs, статусы, отчёты, файлы результата и публикация сообщений | gRPC `:9094`; health `:8085` | PostgreSQL + Redis |
| [`file-service`](./backend/services/file-service/README.md) | Метаданные файлов и единственная граница object storage | gRPC `:9095`; health `:8086` | PostgreSQL + S3 |
| [`document-service`](./backend/services/document-service/README.md) | Excel-анализ, preview и асинхронная обработка документов | gRPC API `:9096`; health `:8087`/`:8092` | Redis consumer; файлы через file-service |
| [`matching-service`](./backend/services/matching-service/README.md) | Нормализация и решение о соответствии товарных строк | gRPC `:9097`; health `:8088` | Stateless |
| [`brand-service`](./backend/services/brand-service/README.md) | Каталог брендов, правила и определение бренда | gRPC `:9098`; health `:8089` | Статические правила |
| [`calculation-service`](./backend/services/calculation-service/README.md) | Расчёт заказа, округления, ручные правки и план «Север» | gRPC `:9099`; health `:8090` | Stateless |
| [`audit-service`](./backend/services/audit-service/README.md) | Запись и выборка audit-событий | gRPC `:9100`; health `:8091` | PostgreSQL |

Порты внутренних сервисов опубликованы только внутри Compose-сети. На host по
умолчанию доступны лишь frontend, gateway, PostgreSQL, Redis и MinIO.

### Общие backend-модули

- [`backend/proto`](./backend/proto/) — исходные protobuf-контракты и
  сгенерированные Go clients/servers;
- [`backend/pkg`](./backend/pkg/README.md) — общие пакеты ошибок, логирования,
  health endpoints, метрик, object storage и gRPC transport;
- [`backend/deploy/docker-compose.yml`](./backend/deploy/docker-compose.yml) —
  полный состав локального backend v2;
- [`deploy/docker-compose.yml`](./deploy/docker-compose.yml) — корневая точка
  входа, подключающая backend Compose-файл.

Подробные правила владения кодом и данными: [service boundaries](./docs/service-boundaries.md).

## API и контракты

- Канонический OpenAPI 3.1 публичного HTTP API:
  [`backend/services/gateway-service/api/openapi.yaml`](./backend/services/gateway-service/api/openapi.yaml).
- Стабильный указатель для старых ссылок:
  [`packages/contracts/openapi.yaml`](./packages/contracts/openapi.yaml).
- Исходные внутренние gRPC-контракты:
  [`backend/proto/orderfill`](./backend/proto/orderfill/).
- Конфигурация генерации protobuf: [`backend/proto/buf.gen.yaml`](./backend/proto/buf.gen.yaml).

Публичные группы HTTP-маршрутов: `/api/v1/auth`, `/api/v1/jobs`,
`/api/v1/companies`, `/api/v1/users`, `/api/v1/audit`, `/api/v1/status` и
`/api/v1/public/companies`. Точные методы, тела и ответы следует брать из
канонического OpenAPI-файла.

Перегенерация protobuf-кода:

```bash
make -C backend proto-gen
```

Команда требует установленного Buf и Go-плагинов, перечисленных в
`backend/proto/buf.gen.yaml`.

## Конфигурация

Начальная конфигурация находится в [`.env.example`](./.env.example). Compose
подставляет локальные значения по умолчанию, но явный `.env` делает запуск
воспроизводимым.

### Основные переменные

| Переменная | Локальное значение | Назначение |
| --- | --- | --- |
| `APP_ENV` | `local` | Общий режим окружения; вне local stateful-сервисы включают fail-fast проверки |
| `DATABASE_URL` | PostgreSQL из Compose | DSN общей PostgreSQL-инсталляции |
| `QUEUE_URL` | `redis://redis:6379/0` | Redis для job stream и краткоживущего состояния auth |
| `FILE_S3_ENDPOINT` | `minio:9000` | S3-compatible endpoint для file-service |
| `S3_BUCKET` | `order-fill` | Compose-маппинг на `FILE_S3_BUCKET` |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | `minioadmin` | Compose-маппинг на credentials file-service и MinIO; только для local |
| `FILE_S3_USE_SSL` | `false` | TLS для подключения file-service к object storage |
| `API_ALLOWED_ORIGINS` | localhost/127.0.0.1:3200 | Разрешённые browser origins; список через запятую |
| `BOOTSTRAP_ADMIN_LOGIN` | `admin` | Login bootstrap-администратора |
| `TWOFA_MASTER_KEY` | local development key | Ключ защиты TOTP-секретов; production требует отдельное значение длиной 32+ байта |
| `SESSION_COOKIE_SECURE` | `false` | Secure-флаг session cookie; вне local должен быть `true` |
| `SESSION_COOKIE_DOMAIN` | пусто | Явный cookie domain, если он нужен схеме размещения |
| `WEBAUTHN_RP_ID` | пусто | WebAuthn relying-party ID; в production обязателен |
| `WEBAUTHN_RP_DISPLAY_NAME` | `Order Fill` | Отображаемое имя relying party |
| `GRPC_TLS_MODE` | `insecure` | `insecure`, `tls` или `mtls` для внутреннего gRPC; вне local обязателен `mtls` |
| `GRPC_TLS_CERT_FILE` | пусто | Сертификат клиента/сервера для TLS/mTLS |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ сертификата |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle для проверки peer |
| `GRPC_TLS_SERVER_NAME` | пусто | Необязательное переопределение имени при проверке TLS-сертификата |
| `WORKER_TOKEN` | `local-dev-worker-token` | Общий секрет worker RPC (`job-service`/`file-service`/`document-*`); вне local обязателен не-default ≥16 байт |

Адреса, service-specific env aliases и точные правила валидации перечислены в
README соответствующих микросервисов. Значения секретов нельзя коммитить:
`.env` исключён из Git.

### Минимум для production

В production задайте как минимум:

```dotenv
APP_ENV=production
API_ALLOWED_ORIGINS=https://orderfill.example.com
SESSION_COOKIE_SECURE=true
WEBAUTHN_RP_ID=orderfill.example.com
FILE_S3_ENDPOINT=minio:9000
FILE_S3_USE_SSL=true
S3_ACCESS_KEY=<production-access-key>
S3_SECRET_KEY=<production-secret-key>
TWOFA_MASTER_KEY=<отдельный-секрет-длиной-не-менее-32-байт>
DATABASE_URL=postgres://order_fill:<production-password>@postgres:5432/order_fill?sslmode=require
POSTGRES_PASSWORD=<production-password>
QUEUE_URL=redis://:<production-password>@redis:6379/0
REDIS_PASSWORD=<production-password>
GRPC_TLS_MODE=mtls
WORKER_TOKEN=<production-worker-token>
```

Также должны быть доступны production PostgreSQL, Redis и S3-compatible
storage. Для Compose на одной машине `make https` генерирует внутренний CA и
**отдельный сертификат на каждый сервис** (`scripts/gen-internal-tls.sh`) и
подключает [`deploy/docker-compose.prod.yml`](./deploy/docker-compose.prod.yml):
mTLS на gRPC, SSL у Postgres/MinIO, без публикации 5432/6379/9000 на host.
В контейнер попадает только его ключ; `ca.key` остаётся на хосте. Пути
`GRPC_TLS_*_FILE` overlay задаёт сам. Значения в примере — форма настройки, а
не готовые credentials.

## Разработка

### Только frontend

```bash
npm ci --prefix frontend
npm run dev --prefix frontend
```

Vite слушает `127.0.0.1:3200`. Для работы UI с API gateway должен быть доступен
по адресу, заданному в frontend-конфигурации; production Docker использует
same-origin `/api/` через nginx.

### Backend

Go workspace находится в `backend/go.work` и объединяет `backend/pkg`,
`backend/proto` и все сервисы.

```bash
make -C backend build
make -C backend test
make -C backend check
```

Для запуска отдельного бинарника сначала поднимите или укажите его зависимости,
затем выполните команду из README сервиса. В local-режиме некоторые сервисы
допускают in-memory adapters; это поведение не следует считать production
конфигурацией.

### Полезные команды

| Команда | Что делает |
| --- | --- |
| `make up` | Собирает и запускает весь Compose-стенд |
| `make logs` | Показывает логи всех контейнеров |
| `make down` | Останавливает стенд, не удаляя named volumes |
| `make test` | Запускает frontend, load-runner и Go-тесты |
| `npm run test:ui --prefix frontend` | Unit + component + Playwright; локально Chromium на экране |
| `npm run test:component --prefix frontend` | Vitest + Testing Library: поведение компонентов |
| `npm run test:e2e --prefix frontend` | Playwright: UI и действия пользователя в браузере |
| `make lint` | Проверяет toolchain, frontend lint и Go lint/security/tidy |
| `make docs` | Проверяет обязательные README и локальные Markdown-ссылки |
| `make docs-sync` | Обновляет таблицы env/RPC/HTTP в README сервисов из config.go, proto и router |
| `make contracts` | Сверяет HTTP routes с OpenAPI и запускает Buf lint |
| `make verify` | Выполняет детерминированный локальный pre-commit gate |
| `make hooks` | Включает git hooks: commit на `artemch` и любой push требуют `make verify` |
| `make security` | Запускает `govulncheck` для всех активных Go-модулей |
| `make lan-https` | Поднимает локальный HTTPS для тестирования passkey на телефоне |
| `make https` | Поднимает публичный HTTPS overlay через Caddy с HSTS |
| `make load-order-fill ARGS='…'` | Запускает нагрузочный сценарий через публичный API |

## Тестирование и quality gate

Перед коммитом:

```bash
make hooks
npm ci --prefix frontend
make verify
```

`make hooks` включает репозиторные git hooks. На ветке `artemch` commit
блокируется, пока `make verify` не проходит. Перед **любым** push тот же полный
gate запускается ещё раз. `--no-verify` не используйте: это тот же обход, что и в CI.

`make verify` выполняет:

1. сверку версий Node.js и Go между manifests, Dockerfiles и CI;
2. проверку документации, ссылок, соответствия OpenAPI runtime-маршрутам и
   protobuf lint;
3. frontend lint, unit tests, component tests и production build;
4. для каждого Go-модуля — `gofmt`, `go vet`, `golangci-lint`, `gosec`,
   проверку `go mod tidy`, сборку и тесты;
5. валидацию backend и корневой Docker Compose-конфигурации.

GitHub Actions выполняет те же слои параллельно и дополнительно запускает Go
тесты с `-race -shuffle=on`, `govulncheck` и проверку breaking changes protobuf
относительно базовой ветки pull request.

Первый запуск может скачать Go toolchain, линтеры и зависимости. Тестовые
workbook-файлы и правила работы с приватными данными описаны в
[`testdata/README.md`](./testdata/README.md), нагрузочные сценарии — в
[`docs/load-testing.md`](./docs/load-testing.md).

## Развёртывание

### Локальный HTTPS для passkeys

На macOS команда ниже определяет Wi-Fi IP, создаёт локальный сертификат через
`mkcert`, добавляет Caddy и выводит инструкцию по доверию сертификату на iPhone:

```bash
make lan-https
```

Скрипт может установить `mkcert` через Homebrew, если утилита отсутствует.

### SSH к ноутбуку из интернета

Порт 22 наружу не открывайте. На Mac и на Ubuntu-сервере поставьте
[Tailscale](https://tailscale.com/download):

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
tailscale ip -4
```

Дальше с любого места: `ssh artemch2003@<tailscale-ip>`. Админка роутера для
этого не нужна.

### Автодеплой `dev`

GitHub Actions уже гоняет полный verify на каждый push и pull request. После
успешного **push в `dev`** job `Deploy self-hosted` на ноуте делает
`git pull --ff-only` и `docker compose up -d --build`, не трогая `.env` и
volumes.

Один раз на ноуте:

1. [Settings → Actions → Runners](https://github.com/igorfrum-cyber/order-fill-vercel/settings/actions/runners) → New self-hosted runner, label `order-fill`.
2. `RUNNER_TOKEN=... bash scripts/install-github-runner.sh`
3. `cd ~/actions-runner && sudo ./svc.sh install && sudo ./svc.sh start`

Рабочий поток: коммиты в `artemch` → PR/merge в `dev` → CI зелёный → стек
на ноуте пересобирается сам. На сервер заходить не нужно.

### Публичный HTTPS

Заполните production-блок `.env`, направьте DNS на сервер, откройте порты 80 и
443 и выполните:

```bash
make https
```

Скрипт проверяет `PUBLIC_HOST`, `APP_ENV=production`, HTTPS origin, secure
cookie, `WEBAUTHN_RP_ID`, `WORKER_TOKEN`, `TWOFA_MASTER_KEY`, `GRPC_TLS_MODE=mtls`,
пароли Postgres/Redis и S3, генерирует внутренние сертификаты при отсутствии и
пересобирает **весь** стек с [`deploy/docker-compose.prod.yml`](./deploy/docker-compose.prod.yml)
и Caddy.

### Vercel

[`vercel.json`](./vercel.json) устанавливает и собирает только `frontend/`,
после чего публикует `frontend/dist`. Go-сервисы, PostgreSQL, Redis и object
storage эта конфигурация не разворачивает; для рабочего приложения frontend
нужен отдельно доступный backend и согласованная API-конфигурация.

## Структура репозитория

```text
.
├── frontend/                 React 19, Vite 7, nginx image
├── backend/
│   ├── services/             11 Go-микросервисов
│   ├── proto/                protobuf contracts и generated Go code
│   ├── pkg/                  общие Go-пакеты
│   └── deploy/               полный Docker Compose stack
├── deploy/                   root compose entrypoint и HTTPS overlays
├── packages/contracts/       совместимый указатель на публичный OpenAPI
├── scripts/                  verify, HTTPS, load и test-data utilities
├── docs/                     архитектура, спецификация и design documents
└── testdata/                 публичные fixtures и правила private fixtures
```

## Документация

- [Индекс документации](./docs/README.md)
- [Текущая архитектура](./docs/ARCHITECTURE.md)
- [Границы сервисов](./docs/service-boundaries.md)
- [Техническое задание](./docs/TECHNICAL_SPEC.md)
- [Нагрузочное тестирование](./docs/load-testing.md)
- [OpenAPI](./backend/services/gateway-service/api/openapi.yaml)
- [Карта проекта для AI-инструментов](./llms.txt)

Файлы в `docs/plans/` описывают дизайн и историю реализации. При расхождении с
кодом источниками истины для текущего runtime являются код, Compose,
канонический OpenAPI и README активных сервисов.

## Участие в разработке

Порядок настройки окружения, правила изменения контрактов и checklist перед PR
описаны в [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## Лицензия

В репозитории сейчас нет файла `LICENSE`; условия использования и
распространения явно не заданы. Не считайте отсутствие файла разрешением на
распространение проекта.
