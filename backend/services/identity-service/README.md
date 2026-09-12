# identity-service

`identity-service` — внутренний gRPC-сервис идентификации и авторизации Order Fill. Он владеет компаниями, пользователями, паролями, приглашениями и cookie-сессиями и остается единственным сервисом, который выпускает сессионные токены.

Сервис не хранит TOTP-секреты и passkey credentials: при настроенных адресах он делегирует проверку второго фактора в `twofa-service`, а завершение WebAuthn-входа и проверку наличия ключей — в `passkey-service`.

## Быстрый старт

Требуется Go `1.26.7`. Самодостаточный local-режим с in-memory хранилищем:

```bash
cd backend/services/identity-service
go run ./cmd/identity
```

По умолчанию gRPC слушает `:9091`, health HTTP — `:8082`. При пустом `DATABASE_URL` данные живут только до остановки процесса; интеграции 2FA/passkey при пустых адресах отключены.

```bash
curl http://127.0.0.1:8082/healthz
curl http://127.0.0.1:8082/readyz
```

При первом запуске на пустом хранилище сервис создает пользователя `platform_admin` и пишет одноразовый путь `/invite/<token>` в JSON-лог. Токен действует 72 часа; его нужно считать секретом.

## Возможности и границы ответственности

- Вход по логину и паролю, Argon2id-хеширование паролей и защита unknown-login пути dummy-проверкой.
- Двухшаговый password + TOTP login через одноразовый challenge.
- Завершение passkey-login и выпуск сессии после подтверждения пользователя `passkey-service`.
- Прием одноразовых приглашений, первоначальная установка пароля и сброс доступа.
- Выпуск, проверка, список и отзыв сессий; logout текущей сессии и logout everywhere.
- CRUD-операции текущего объема над компаниями и пользователями: создание, список, профиль, отключение, приглашение и reset access.
- Роли `platform_admin`, `company_owner`, `company_admin`, `purchaser` и ограничения управления своей компанией. Одноранговые роли в компании не управляют друг другом: владелец не отключает другого владельца, администратор — другого администратора. Самого себя отключить и сбросить нельзя.
- Настройка company `matching_mode`: `standard` по умолчанию, `smart` может назначать только `platform_admin`.
- Публичный поиск активной компании по `login_slug`.

Пароль должен иметь длину от 10 до 1024 байт. Сессия действует 8 часов, invite — 72 часа, TOTP login challenge — 5 минут. В БД сохраняется SHA-256 hash сессионного/invite/challenge token, а raw token возвращается только при его выпуске.

## Архитектура и данные

| Путь | Назначение |
| --- | --- |
| `cmd/identity` | Точка входа и graceful shutdown по `SIGINT`/`SIGTERM`. |
| `internal/config` | Переменные окружения и production-валидация. |
| `internal/bootstrap` | Выбор storage, миграции, клиенты 2FA/passkey и запуск серверов. |
| `internal/domain` | Пользователи, компании, роли, slug, сессии и authorization rules. |
| `internal/service/auth` | Login, invite, session, password, TOTP и passkey orchestration. |
| `internal/service/companies` | Компания, публичный slug и matching mode. |
| `internal/service/users` | Приглашение, список, отключение и сброс доступа пользователей. |
| `internal/storage/memory` | Непостоянное process-local хранилище для local/tests. |
| `internal/storage/postgres` | PostgreSQL-реализация store. |
| `internal/migrate` | Встроенная идемпотентная SQL-миграция при старте. |
| `internal/transport/grpcapi` | Реализация protobuf RPC и преобразование доменных ошибок в gRPC status. |

При заданном `DATABASE_URL` сервис проверяет соединение, в одной транзакции выполняет встроенные SQL-файлы и использует PostgreSQL. Он логически владеет таблицами `companies`, `users`, `sessions`, `invite_tokens`, `login_challenges`; compose сейчас направляет несколько сервисов в одну физическую БД.

## Внутренний gRPC API

<!-- docs-sync:rpc -->
| RPC | Полный gRPC method | Назначение |
| --- | --- | --- |
| `Login` | `/orderfill.identity.v1.IdentityService/Login` | RPC из protobuf-контракта. |
| `CompleteTwoFactorLogin` | `/orderfill.identity.v1.IdentityService/CompleteTwoFactorLogin` | RPC из protobuf-контракта. |
| `Logout` | `/orderfill.identity.v1.IdentityService/Logout` | RPC из protobuf-контракта. |
| `LogoutEverywhere` | `/orderfill.identity.v1.IdentityService/LogoutEverywhere` | RPC из protobuf-контракта. |
| `ListSessions` | `/orderfill.identity.v1.IdentityService/ListSessions` | RPC из protobuf-контракта. |
| `RevokeSession` | `/orderfill.identity.v1.IdentityService/RevokeSession` | RPC из protobuf-контракта. |
| `ValidateSession` | `/orderfill.identity.v1.IdentityService/ValidateSession` | RPC из protobuf-контракта. |
| `GetMe` | `/orderfill.identity.v1.IdentityService/GetMe` | RPC из protobuf-контракта. |
| `AcceptInvite` | `/orderfill.identity.v1.IdentityService/AcceptInvite` | RPC из protobuf-контракта. |
| `FinishPasskeyLogin` | `/orderfill.identity.v1.IdentityService/FinishPasskeyLogin` | RPC из protobuf-контракта. |
| `ChangePassword` | `/orderfill.identity.v1.IdentityService/ChangePassword` | RPC из protobuf-контракта. |
| `PublicCompany` | `/orderfill.identity.v1.IdentityService/PublicCompany` | RPC из protobuf-контракта. |
| `CreateCompany` | `/orderfill.identity.v1.IdentityService/CreateCompany` | RPC из protobuf-контракта. |
| `ListCompanies` | `/orderfill.identity.v1.IdentityService/ListCompanies` | RPC из protobuf-контракта. |
| `UpdateCompany` | `/orderfill.identity.v1.IdentityService/UpdateCompany` | RPC из protobuf-контракта. |
| `DisableCompany` | `/orderfill.identity.v1.IdentityService/DisableCompany` | RPC из protobuf-контракта. |
| `CreateUser` | `/orderfill.identity.v1.IdentityService/CreateUser` | RPC из protobuf-контракта. |
| `ListUsers` | `/orderfill.identity.v1.IdentityService/ListUsers` | RPC из protobuf-контракта. |
| `DisableUser` | `/orderfill.identity.v1.IdentityService/DisableUser` | RPC из protobuf-контракта. |
| `ResetUserAccess` | `/orderfill.identity.v1.IdentityService/ResetUserAccess` | RPC из protobuf-контракта. |
<!-- /docs-sync:rpc -->

Source of truth — [`../../proto/orderfill/identity/v1/identity.proto`](../../proto/orderfill/identity/v1/identity.proto), package `orderfill.identity.v1`, service `IdentityService`.

### Аутентификация и сессии

- `Login` — проверяет пароль; при включенном 2FA возвращает challenge вместо сессии.
- `CompleteTwoFactorLogin` — проверяет challenge и TOTP/recovery code через `twofa-service`, затем выпускает сессию.
- `Logout`, `LogoutEverywhere`, `ListSessions`, `RevokeSession`.
- `ValidateSession`, `GetMe`.
- `AcceptInvite`, `ChangePassword` — смена пароля удаляет все сессии пользователя.
- `FinishPasskeyLogin` — делегирует WebAuthn assertion и выпускает сессию найденному активному пользователю.

### Компании и пользователи

- `PublicCompany`.
- `CreateCompany`, `ListCompanies`, `UpdateCompany`, `DisableCompany`.
- `CreateUser`, `ListUsers`, `DisableUser`, `ResetUserAccess`.

Сервис принимает actor/user ID в protobuf-запросах и применяет доменные role checks, но на gRPC-слое не проверяет самостоятельную сессию вызывающего. API предназначен только для доверенных внутренних клиентов за сетевой границей; публичным клиентом должен быть gateway.

## Зависимости и взаимодействия

| Зависимость | Когда нужна | Поведение при отсутствии |
| --- | --- | --- |
| PostgreSQL | Постоянное хранение и readiness | В local при пустом URL используется memory store. Вне local обязателен. |
| `twofa-service` | Проверка enabled-state и кода во время login/validation | В local при пустом адресе 2FA-этап не используется. Вне local адрес обязателен. |
| `passkey-service` | Завершение passkey-login и флаг наличия credentials | В local при пустом адресе passkey login недоступен. Вне local адрес обязателен. |

gRPC-клиенты имеют default deadline 60 секунд. Максимальный размер входящего/исходящего сообщения — 64 MiB. Request ID принимается и передается через metadata `x-request-id`.

## Конфигурация


<!-- docs-sync:env -->
| Переменная | Default | Обязательность и назначение |
| --- | --- | --- |
| `IDENTITY_ENV` | значение `APP_ENV`, затем `local` | Среда сервиса; имеет приоритет над `APP_ENV`. |
| `APP_ENV` | `local` | Общая среда. Пустая строка и `local` считаются local-режимом. |
| `IDENTITY_GRPC_ADDR` | `:9091` | gRPC listener. |
| `IDENTITY_HEALTH_ADDR` | `:8082` | Отдельный HTTP listener liveness/readiness. |
| `BOOTSTRAP_ADMIN_LOGIN` | `admin` | Логин первоначального `platform_admin`, создаваемого только если пользователей нет. |
| `TWOFA_GRPC_ADDR` | пусто | Адрес `twofa-service`; обязателен вне local. |
| `PASSKEY_GRPC_ADDR` | пусто | Адрес `passkey-service`; обязателен вне local. |
| `DATABASE_URL` | пусто | PostgreSQL DSN. Обязателен вне local; обычно содержит пароль и должен храниться как secret. |
| `GRPC_TLS_MODE` | `insecure` | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CERT_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_KEY_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CA_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_SERVER_NAME` | пусто | Необязательное имя для проверки TLS-сертификата исходящих gRPC-клиентов. |
<!-- /docs-sync:env -->

Общая конфигурация gRPC server и исходящих клиентов:

| Переменная | Default | Назначение |
| --- | --- | --- |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | PEM certificate сервера; вместе с key обязателен для `tls` и `mtls`. В `mtls` тот же certificate используется клиентом сервиса. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ; обязателен для `tls`/`mtls`, хранить как secret. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle; обязателен в `mtls`, необязателен для исходящих `tls` соединений. |
| `GRPC_TLS_SERVER_NAME` | hostname из target | Явное имя для проверки сертификатов `twofa-service` и `passkey-service`. |

Вне local процесс не стартует без `DATABASE_URL`, `TWOFA_GRPC_ADDR` и `PASSKEY_GRPC_ADDR`. `DATABASE_URL` не может использовать пароль `order_fill` и должен задавать `sslmode=require` (или `verify-ca`/`verify-full`). `GRPC_TLS_MODE` должен быть `tls` или `mtls`.

## Запуск с PostgreSQL

Пример из директории сервиса при уже запущенном PostgreSQL:

```bash
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
TWOFA_GRPC_ADDR=127.0.0.1:9092 \
PASSKEY_GRPC_ADDR=127.0.0.1:9093 \
go run ./cmd/identity
```

Для isolated local-debug можно не задавать все три переменные и использовать memory store, но состояние и bootstrap-admin будут созданы заново после перезапуска.

## Docker и Compose

Dockerfile требует build context `backend/`, собирает статический бинарник и запускает его в Alpine 3.22 от UID `10001`:

```bash
docker build -f backend/services/identity-service/Dockerfile -t order-fill-identity backend
```

В [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml) сервис доступен только внутри compose-сети на `9091` и `8082`, зависит от healthy PostgreSQL, TwoFA и Passkey и проходит healthcheck через `/healthz`.

Из корня репозитория:

```bash
docker compose -f deploy/docker-compose.yml up --build identity-service
```

Compose поднимет зависимости. Полный стек запускается `make up`; переменные для локального окружения приведены в [`../../../.env.example`](../../../.env.example).

## Тесты

```bash
cd backend/services/identity-service
go test ./...
```

Тесты покрывают конфигурацию, password/token utilities, доменные правила, auth/session/invite flows, компании, storage и миграции. Команда `make test` из корня проверяет весь репозиторий.

## Эксплуатационные заметки и ограничения

- `GET /healthz` всегда отвечает `200`, если HTTP listener работает. `GET /readyz` вызывает `pool.Ping` только при PostgreSQL store; в memory-режиме всегда отвечает `200`.
- Readiness не проверяет `twofa-service` и `passkey-service`.
- При каждом старте с PostgreSQL все embedded SQL migration-файлы выполняются заново в одной транзакции; отдельной таблицы версий миграций нет, поэтому SQL обязан оставаться идемпотентным.
- In-memory store process-local: не подходит для нескольких реплик и теряет компании, пользователей, приглашения и сессии при рестарте.
- Bootstrap invite печатается в лог в открытом виде. Ограничьте доступ к логам и завершите активацию до истечения 72 часов.
- Смена/потеря PostgreSQL удаляет доступ к сессиям; raw session token в БД не хранится.
- gRPC по умолчанию незашифрован и не содержит interceptor аутентификации вызывающего. В production нужны TLS/mTLS и сетевые политики.
- В репозитории нет файла `LICENSE`; условия распространения сервиса не зафиксированы.
