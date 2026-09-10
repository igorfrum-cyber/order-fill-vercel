# gateway-service

`gateway-service` — единственная публичная точка входа в backend Order Fill. Сервис принимает HTTP-запросы браузера, проверяет cookie-сессию через `identity-service` и преобразует запросы в вызовы внутренних gRPC-сервисов.

Сервис не выполняет обработку Excel-файлов, не хранит пользователей, задания и объекты и не выпускает сессии самостоятельно. Владельцами этих данных остаются соответствующие внутренние сервисы.

## Быстрый старт

Требуется Go `1.26.7` (версия зафиксирована в `go.mod`). Для запуска из исходников:

```bash
cd backend/services/gateway-service
go run ./cmd/gateway
```

По умолчанию HTTP-сервер слушает `:8080`, а gRPC-зависимости ищутся на loopback-адресах из таблицы конфигурации ниже. Процесс запустится и без доступных зависимостей, но прикладные запросы к ним завершатся ошибками.

Проверка процесса:

```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

Оба запроса в текущей реализации возвращают `{"status":"ok"}` без проверки зависимостей.

## Возможности и границы ответственности

- HTTP API для входа по паролю, TOTP/recovery code и passkey, управления сессиями и смены пароля.
- Административные операции над компаниями и пользователями с ролевыми ограничениями.
- Прием исходных книг через `multipart/form-data`, создание заданий `order_fill` и `north_merge`, чтение статуса/отчета, отправка ручных правок.
- Скачивание отдельных файлов и ZIP-архива, а также выдача метаданных и окон табличного preview без распаковки всей книги в gateway.
- Публичные метаданные и логотип страницы входа компании по `login_slug`.
- Аудит отдельных административных действий и агрегированный статус инфраструктуры для `platform_admin`.
- CORS, CSRF-проверка POST-запросов, security headers и HTTP-only cookie `order_fill_session`.

Gateway не владеет постоянным хранилищем. `POSTGRES_ADDR` и `REDIS_ADDR` используются только диагностическим `/api/v1/status`.

## Архитектура и структура

| Путь | Назначение |
| --- | --- |
| `cmd/gateway` | Точка входа, обработка `SIGINT`/`SIGTERM`. |
| `internal/config` | Загрузка и проверка runtime-конфигурации. |
| `internal/bootstrap` | Создание gRPC-клиентов и HTTP-сервера. |
| `internal/clients` | Клиенты Identity, TwoFA, Passkey, Job, File и Audit API. |
| `internal/transport/httpapi` | Маршрутизация, auth gate, обработчики, CORS/CSRF и HTTP-представления. |
| `internal/preview` | Чтение gzip-метаданных и чанков preview из object storage через `file-service`. |
| `api/openapi.yaml` | Публичный OpenAPI 3.1 контракт. |

Защищенный запрос проходит так:

1. Gateway читает cookie `order_fill_session`.
2. `IdentityService.ValidateSession` возвращает пользователя и его роль.
3. Обработчик вызывает нужный внутренний gRPC API; для job-запросов роль передается в metadata `x-actor-role`.
4. Ответ gRPC переводится в публичный JSON/файл или в унифицированную HTTP-ошибку.

Исходящие gRPC-вызовы имеют таймаут 60 секунд, если вызывающий код не задал более ранний deadline. HTTP-сервер использует `ReadHeaderTimeout=10s`, `ReadTimeout=60s`, `WriteTimeout=120s`, `IdleTimeout=120s` и лимит заголовков 1 MiB.

## Публичный HTTP API

Полные схемы запросов, ответов и ошибок находятся в [`api/openapi.yaml`](api/openapi.yaml). Реально зарегистрированные маршруты:

### Состояние процесса

- `GET /healthz`
- `GET /readyz`

### Аутентификация и учетная запись

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/login/2fa`
- `POST /api/v1/auth/invite`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/logout-everywhere`
- `GET /api/v1/auth/sessions`
- `POST /api/v1/auth/sessions/{id}/delete`
- `POST /api/v1/auth/password`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/2fa/setup`
- `POST /api/v1/auth/2fa/enable`
- `POST /api/v1/auth/2fa/disable`
- `POST /api/v1/auth/passkeys/register/begin`
- `POST /api/v1/auth/passkeys/register/finish`
- `GET /api/v1/auth/passkeys`
- `POST /api/v1/auth/passkeys/{id}/delete`
- `POST /api/v1/auth/passkeys/login/begin`
- `POST /api/v1/auth/passkeys/login/finish`

### Компании, пользователи и аудит

- `GET|POST /api/v1/companies`
- `POST /api/v1/companies/{company_id}/login-slug`
- `POST /api/v1/companies/{company_id}/profile`
- `POST /api/v1/companies/{company_id}/logo`
- `POST /api/v1/companies/{company_id}/logo/clear`
- `POST /api/v1/companies/{company_id}/disable`
- `GET|POST /api/v1/companies/{company_id}/users`
- `POST /api/v1/users/{user_id}/disable`
- `POST /api/v1/users/{user_id}/reset`
- `GET /api/v1/audit`
- `GET /api/v1/status`
- `GET /api/v1/public/companies/{slug}/login`
- `GET /api/v1/public/companies/{slug}/logo`

### Задания и файлы

- `GET /api/v1/jobs`
- `POST /api/v1/jobs/order-fill`
- `POST /api/v1/jobs/north-merge`
- `GET /api/v1/jobs/{job_id}`
- `GET /api/v1/jobs/{job_id}/report`
- `POST /api/v1/jobs/{job_id}/edits`
- `GET /api/v1/jobs/{job_id}/files`
- `GET /api/v1/jobs/{job_id}/archive`
- `GET /api/v1/jobs/{job_id}/files/{file_id}`
- `GET /api/v1/jobs/{job_id}/files/{file_id}/preview`
- `GET /api/v1/jobs/{job_id}/files/{file_id}/preview/window`
- `GET /api/v1/jobs/{job_id}/files/{file_id}/preview/find`

Без cookie доступны health/readiness, login, завершение 2FA-login, прием invite, начало/завершение passkey-login и публичные маршруты компании. Остальные маршруты проходят через `ValidateSession`. POST-запросы дополнительно требуют `X-Requested-With: fetch`; если указан `Origin`, он должен входить в разрешенный список.

Сессионная cookie имеет `HttpOnly`, `SameSite=Lax`, TTL 8 часов и получает `Secure`/`Domain` из конфигурации. JSON для auth/admin ограничен 8 KiB, JSON задания — 1 MiB, все multipart-тело задания — 64 MiB. Логотип ограничен 512 KiB и форматами PNG, JPEG или WebP.

## Внутренние зависимости

| Зависимость | Использование |
| --- | --- |
| `identity-service` | Вход, invite, сессии, пользователи, компании, проверка сессии и выпуск сессии после passkey-login. |
| `twofa-service` | Настройка, включение и отключение TOTP. |
| `passkey-service` | WebAuthn registration/login ceremony и список/удаление credentials. |
| `job-service` | Создание и чтение заданий, отчетов, файлов и ручных правок. |
| `file-service` | Загрузка и скачивание объектов, архивы, логотипы и preview-чанки. |
| `audit-service` | Запись и чтение событий аудита. Запись выполняется best-effort с таймаутом 750 ms. |
| document worker, PostgreSQL, Redis | Только проверки для `/api/v1/status`; gateway не обращается к их данным напрямую. |

gRPC-контракты находятся в [`../../proto/orderfill`](../../proto/orderfill). Максимальный размер gRPC-сообщения — 64 MiB.

## Конфигурация

Значения читаются только из переменных окружения. Пустая строка считается отсутствующим значением и заменяется default.

| Переменная | Default | Обязательность и назначение |
| --- | --- | --- |
| `GATEWAY_ENV` | значение `APP_ENV`, затем `local` | Среда именно gateway; имеет приоритет над `APP_ENV`. |
| `APP_ENV` | `local` | Общая среда, если `GATEWAY_ENV` не задан. Пустая строка и `local` включают local-режим. |
| `GATEWAY_ADDR` | `:8080` | Адрес публичного HTTP listener. |
| `IDENTITY_GRPC_ADDR` | `127.0.0.1:9091` | Адрес `identity-service`. |
| `TWOFA_GRPC_ADDR` | `127.0.0.1:9092` | Адрес `twofa-service`. |
| `PASSKEY_GRPC_ADDR` | `127.0.0.1:9093` | Адрес `passkey-service`. |
| `JOB_GRPC_ADDR` | `127.0.0.1:9094` | Адрес `job-service`. |
| `FILE_GRPC_ADDR` | `127.0.0.1:9095` | Адрес `file-service`. |
| `AUDIT_GRPC_ADDR` | `127.0.0.1:9100` | Адрес `audit-service`. |
| `WORKER_HEALTH_URL` | `http://127.0.0.1:8092/healthz` | HTTP URL document worker для `/api/v1/status`. |
| `FILE_HEALTH_URL` | `http://127.0.0.1:8086/healthz` | HTTP URL `file-service` для `/api/v1/status`. |
| `POSTGRES_ADDR` | `127.0.0.1:5432` | TCP-адрес PostgreSQL для `/api/v1/status`. |
| `REDIS_ADDR` | `127.0.0.1:6379` | TCP-адрес Redis для `/api/v1/status`. |
| `API_ALLOWED_ORIGINS` | `http://127.0.0.1:3200,http://localhost:3200` в local; пусто вне local | Разделенный запятыми allowlist CORS/CSRF. Вне local обязателен, `*` запрещен, все origins должны быть валидными HTTPS URL. |
| `SESSION_COOKIE_SECURE` | `false` в local, `true` вне local | Только строка `true` включает Secure. Вне local значение обязано быть истинным. |
| `SESSION_COOKIE_DOMAIN` | пусто | Необязательный атрибут Domain cookie. |

Все исходящие gRPC-клиенты используют общую TLS-конфигурацию:

| Переменная | Default | Назначение |
| --- | --- | --- |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. Не влияет на публичный HTTP listener. |
| `GRPC_TLS_CA_FILE` | пусто | PEM CA для проверки серверов; обязателен в `mtls`, необязателен в `tls` (тогда используются системные CA). |
| `GRPC_TLS_CERT_FILE` | пусто | Клиентский сертификат; обязателен в `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ клиентского сертификата; обязателен в `mtls`, хранить как secret. |
| `GRPC_TLS_SERVER_NAME` | hostname из target | Явное имя сервера для TLS-проверки исходящих соединений. |

Пример production-минимума:

```bash
APP_ENV=production \
API_ALLOWED_ORIGINS=https://orderfill.example.com \
SESSION_COOKIE_SECURE=true \
GATEWAY_ADDR=:8080 \
go run ./cmd/gateway
```

Адреса внутренних сервисов и gRPC TLS material при этом также должны соответствовать окружению.

## Docker и Compose

Dockerfile — multi-stage сборка статического бинарника. Контекст сборки обязан быть `backend/`; runtime-образ Alpine 3.22 содержит `wget` для healthcheck и запускает процесс от UID `10001`:

```bash
docker build -f backend/services/gateway-service/Dockerfile -t order-fill-gateway backend
```

Основной compose-файл репозитория [`../../../deploy/docker-compose.yml`](../../../deploy/docker-compose.yml) подключает backend-compose и публикует gateway только на `127.0.0.1:8080`. Из корня репозитория:

```bash
docker compose -f deploy/docker-compose.yml up --build gateway-service
```

Зависимости из `depends_on` будут подняты автоматически. Полный локальный стек запускается командой `make up`; значения для него описаны в [`../../../.env.example`](../../../.env.example).

## Тесты и проверки

Из директории сервиса:

```bash
go test ./...
```

Из корня репозитория `make test` запускает тесты всех Go-модулей и frontend. Тесты gateway покрывают конфигурационную валидацию, CORS/CSRF и auth gate, ограничения ролей/логотипов, маршруты и preview.

После изменения публичного API нужно синхронизировать OpenAPI-контракт и проверить его генерацию командами из [`../../Makefile`](../../Makefile).

## Эксплуатационные заметки и ограничения

- `/healthz` — liveness, `/readyz` сейчас всегда отвечает `200` и не отражает доступность gRPC-зависимостей.
- `/api/v1/status` доступен только `platform_admin` и проверяет worker, PostgreSQL, Redis и file-service с общим deadline 2 секунды. Identity, TwoFA, Passkey, Job и Audit в эту диагностику не входят.
- Вне local сервис отказывается запускаться с insecure cookie, пустым CORS allowlist, wildcard origin или origin без HTTPS.
- `GRPC_TLS_MODE=insecure` — default для разработки. В production внутреннюю сеть нужно защищать TLS/mTLS и сетевыми политиками.
- В репозитории нет файла `LICENSE`; условия распространения сервиса в README не зафиксированы.
