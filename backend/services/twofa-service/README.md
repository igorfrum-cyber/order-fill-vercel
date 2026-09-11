# twofa-service

`twofa-service` — внутренний gRPC-сервис TOTP-второго фактора. Он создает и проверяет TOTP credentials, шифрует TOTP-секреты перед сохранением, выпускает одноразовые recovery codes и ограничивает число неуспешных проверок.

Сервис не проверяет пароль пользователя и не выпускает cookie-сессии. Password login и сессии принадлежат `identity-service`; публичные setup/enable/disable endpoints предоставляет `gateway-service`.

## Быстрый старт

Требуется Go `1.26.7`. Local-режим без внешней инфраструктуры:

```bash
cd backend/services/twofa-service
go run ./cmd/twofa
```

По умолчанию gRPC слушает `:9092`, health HTTP — `:8083`. Без `DATABASE_URL` credentials хранятся в памяти; без Redis URL rate limit также process-local.

```bash
curl http://127.0.0.1:8083/healthz
curl http://127.0.0.1:8083/readyz
```

Default `TWOFA_MASTER_KEY=local-dev-twofa-master-key` допустим только для local-разработки и не должен использоваться для реальных учетных записей.

## Возможности и границы ответственности

- `Setup`: генерация нового TOTP secret, `otpauth://` URL и PNG QR 200×200; повторный setup запрещен для уже включенного 2FA.
- `Enable`: проверка первого TOTP-кода, включение credential и однократная выдача 8 recovery codes.
- `Verify`: проверка TOTP или recovery code; recovery code хранится как SHA-256 hash и удаляется после успешного использования.
- `Disable`: удаление TOTP credential только после проверки TOTP или recovery code.
- `IsEnabled`: чтение текущего состояния для login flow в `identity-service`.
- AES-256-GCM шифрование TOTP secret перед записью в memory/PostgreSQL store.
- Ограничение: не более 5 неуспешных проверок на пользователя в окне 15 минут.

TOTP использует issuer `Order Fill`, SHA-1, 6 цифр, период 30 секунд и допускает skew ±1 период. Recovery code имеет отображение `XXXX-XXXX`, сравнивается после удаления пробелов/дефиса и приведения к верхнему регистру.

## Архитектура и данные

| Путь | Назначение |
| --- | --- |
| `cmd/twofa` | Точка входа и graceful shutdown по `SIGINT`/`SIGTERM`. |
| `internal/config` | Runtime-конфигурация и production-валидация. |
| `internal/bootstrap` | Вывод AES key, выбор store/limiter, миграции и запуск серверов. |
| `internal/service/twofa` | Setup/enable/disable/is-enabled/verify lifecycle. |
| `internal/totp` | TOTP, QR и recovery codes. |
| `internal/secret` | AES-GCM envelope и SHA-256 token hashing. |
| `internal/ratelimit` | Redis или in-memory счетчик неуспешных проверок. |
| `internal/storage/memory` | Непостоянный credential store. |
| `internal/storage/postgres` | PostgreSQL credential store. |
| `internal/migrate` | Embedded SQL, выполняемый при старте. |
| `internal/transport/grpcapi` | Реализация protobuf RPC. |

Рабочий AES-256 key вычисляется как `SHA-256(TWOFA_MASTER_KEY)`. В таблице `user_totp` хранятся `user_id`, зашифрованный secret, флаг enabled и JSON-массив recovery-code hashes. Compose направляет сервис в общую физическую PostgreSQL БД, хотя таблица принадлежит этому сервису логически.

## Внутренний gRPC API

<!-- docs-sync:rpc -->
| RPC | Полный gRPC method | Назначение |
| --- | --- | --- |
| `Setup` | `/orderfill.twofa.v1.TwoFAService/Setup` | RPC из protobuf-контракта. |
| `Enable` | `/orderfill.twofa.v1.TwoFAService/Enable` | RPC из protobuf-контракта. |
| `Disable` | `/orderfill.twofa.v1.TwoFAService/Disable` | RPC из protobuf-контракта. |
| `IsEnabled` | `/orderfill.twofa.v1.TwoFAService/IsEnabled` | RPC из protobuf-контракта. |
| `Verify` | `/orderfill.twofa.v1.TwoFAService/Verify` | RPC из protobuf-контракта. |
<!-- /docs-sync:rpc -->

Контракт: [`../../proto/orderfill/twofa/v1/twofa.proto`](../../proto/orderfill/twofa/v1/twofa.proto), package `orderfill.twofa.v1`, service `TwoFAService`.

- `Setup(actor_user_id, account_name)` → `secret`, `otpauth_url`, `qr_png`. Если account name пуст, используется actor user ID.
- `Enable(actor_user_id, code)` → raw `recovery_codes`, возвращаемые только этим ответом.
- `Disable(actor_user_id, code)` — удаляет credential после успешного `Verify`. Пустой code отклоняется.
- `IsEnabled(user_id)` → `enabled`.
- `Verify(user_id, code)` → `ok=true`, `used_recovery_code`.

Все RPC доверяют переданному user ID и не выполняют проверку session token. Поэтому listener предназначен только для внутренних клиентов.

Максимальный размер gRPC-сообщения — 64 MiB. Request ID поддерживается через metadata `x-request-id`.

## Зависимости

| Зависимость | Использование | Local fallback |
| --- | --- | --- |
| PostgreSQL | Постоянное хранение encrypted TOTP credential и recovery hashes | In-memory map при пустом `DATABASE_URL`. |
| Redis | Счетчик ошибок `twofa:fail:<user_id>` с TTL 15 минут | Process-local limiter при пустых Redis URL. |
| `identity-service` | Вызывает `IsEnabled`/`Verify` в login flow | Не является исходящей зависимостью twofa-service. |
| `gateway-service` | Вызывает setup/enable/disable от имени аутентифицированного пользователя | Не является исходящей зависимостью twofa-service. |

## Конфигурация


<!-- docs-sync:env -->
| Переменная | Default | Обязательность и назначение |
| --- | --- | --- |
| `TWOFA_ENV` | значение `APP_ENV`, затем `local` | Среда сервиса; имеет приоритет над `APP_ENV`. |
| `APP_ENV` | `local` | Общая среда. Пустая строка и `local` включают local-режим. |
| `TWOFA_GRPC_ADDR` | `:9092` | gRPC listener. |
| `TWOFA_HEALTH_ADDR` | `:8083` | HTTP listener liveness/readiness. |
| `TWOFA_MASTER_KEY` | `local-dev-twofa-master-key` | Корневой секрет шифрования. Вне local обязателен, не может равняться default и должен иметь не менее 32 байт. Хранить как secret. |
| `DATABASE_URL` | пусто | PostgreSQL DSN; обязателен вне local. Обычно содержит пароль — хранить как secret. |
| `QUEUE_URL` | значение `REDIS_URL`, затем пусто | Redis URL для rate limit; имеет приоритет над `REDIS_URL`. Вне local один из URL обязателен. Может содержать пароль. |
| `REDIS_URL` | пусто | Fallback Redis URL, если `QUEUE_URL` не задан. |
| `GRPC_TLS_MODE` | `insecure` | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CERT_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_KEY_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CA_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_SERVER_NAME` | пусто | Необязательное имя для проверки TLS-сертификата исходящих gRPC-клиентов. |
<!-- /docs-sync:env -->

Конфигурация входящего gRPC TLS:

| Переменная | Default | Назначение |
| --- | --- | --- |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | PEM certificate сервера; вместе с key обязателен в `tls` и `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ сервера; обязателен в `tls`/`mtls`, хранить как secret. |
| `GRPC_TLS_CA_FILE` | пусто | CA для проверки клиентских сертификатов; обязателен в `mtls`. |

Вне local процесс не стартует без PostgreSQL, Redis и безопасного master key.

## Запуск с PostgreSQL и Redis

```bash
cd backend/services/twofa-service
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
QUEUE_URL='redis://127.0.0.1:6379/0' \
TWOFA_MASTER_KEY='replace-with-a-stable-secret-of-at-least-32-bytes' \
go run ./cmd/twofa
```

Master key должен быть стабильным между рестартами и репликами. Смена значения без миграции/перешифрования делает ранее сохраненные TOTP secrets нерасшифровываемыми.

## Docker и Compose

Dockerfile требует build context `backend/`, собирает статический бинарник и запускает Alpine 3.22 от UID `10001`:

```bash
docker build -f backend/services/twofa-service/Dockerfile -t order-fill-twofa backend
```

В [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml) сервис доступен только внутри compose-сети на `9092`/`8083`, зависит от healthy PostgreSQL и Redis и проверяется через `/healthz`.

```bash
docker compose -f deploy/docker-compose.yml up --build twofa-service
```

Команда выполняется из корня репозитория; compose поднимает зависимости. Полный стек: `make up`. Пример переменных: [`../../../.env.example`](../../../.env.example).

## Тесты

```bash
cd backend/services/twofa-service
go test ./...
```

Тесты покрывают конфигурацию, AES-GCM, token/recovery-code handling, TOTP, rate limit, service flows, storage и миграции. `make test` из корня запускает весь набор тестов.

## Эксплуатационные заметки и ограничения

- `/healthz` всегда отвечает `200`. `/readyz` проверяет только PostgreSQL при включенном persistent store; Redis не проверяется.
- При недоступном Redis limiter работает fail-closed: ошибка Redis блокирует попытку Verify.
- In-memory credential store и limiter теряют состояние при рестарте и не разделяются между репликами. Это сбрасывает и настройки 2FA, и защиту от перебора.
- Recovery codes показываются в raw-виде только при `Enable`; после этого восстановить их из хешей нельзя.
- `Disable` требует валидный TOTP или recovery code.
- При каждом старте PostgreSQL migration-файлы повторно выполняются в одной транзакции, без version table; SQL должен оставаться идемпотентным.
- Изменение или потеря `TWOFA_MASTER_KEY` блокирует чтение существующих TOTP credentials. Не логируйте и не коммитьте production key.
- gRPC default — незашифрованный `insecure`; в production используйте TLS/mTLS и сетевые политики.
- В репозитории нет файла `LICENSE`; условия распространения сервиса не зафиксированы.
