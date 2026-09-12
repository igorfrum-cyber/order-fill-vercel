# audit-service

Внутренний gRPC-сервис хранения аудиторских событий Order Fill. Сервис записывает события и возвращает их в хронологическом порядке; он не предоставляет публичный HTTP API и не определяет, какие действия должны считаться аудитируемыми.

## Ответственность и возможности

- принимает тип события, автора, компанию, задачу и произвольную JSON-строку с деталями;
- создаёт для события случайный URL-safe идентификатор и UTC timestamp;
- фильтрует список событий по компании; пустой фильтр для `platform_admin` возвращает события всех компаний, для остальных — только свою компанию из identity;
- хранит данные в PostgreSQL либо, только для локальной разработки, в памяти процесса;
- запускает встроенные SQL-миграции при старте с PostgreSQL;
- предоставляет отдельные liveness- и readiness-проверки по HTTP.

За формирование и отправку событий отвечает вызывающая сторона. В текущей системе `gateway-service` вызывает `Record`, в частности для административных действий. `ListEvents` проверяет роль через `identity-service`. Сервис не проверяет формат `payload_json` и не изменяет данные других сервисов.

## Архитектура и структура

```text
cmd/audit/                  точка входа, сигналы SIGINT/SIGTERM
internal/bootstrap/         сборка зависимостей, gRPC и health HTTP
internal/config/            переменные окружения и production-валидация
internal/domain/            модель аудиторского события
internal/service/audit/     запись и выборка событий
internal/storage/memory/    непостоянное локальное хранилище
internal/storage/postgres/  pgx pool и запросы к audit_events
internal/migrate/           встроенная SQL-миграция
internal/transport/grpcapi/ адаптер protobuf/gRPC
```

Путь выполнения: gRPC-обработчик → прикладной сервис → `Store`. При непустом `DATABASE_URL` bootstrap подключает PostgreSQL, выполняет миграции и использует таблицу `audit_events`; иначе выбирается память процесса.

## API

Контракт: [`audit.proto`](../../proto/orderfill/audit/v1/audit.proto), package `orderfill.audit.v1`.


<!-- docs-sync:rpc -->
| RPC | Полный gRPC method | Назначение |
| --- | --- | --- |
| `Record` | `/orderfill.audit.v1.AuditService/Record` | Записывает событие и возвращает его `id`. `actor_user_id` и `company_id` читаются из `RequestMeta`. |
| `ListEvents` | `/orderfill.audit.v1.AuditService/ListEvents` | Возвращает события в порядке `created_at` по возрастанию. `platform_admin` может запросить все компании пустым фильтром; остальные всегда ограничены своей компанией из identity. Без identity пустой фильтр отклоняется. |
<!-- /docs-sync:rpc -->

HTTP-интерфейс ограничен служебными endpoint-ами:

| Метод и путь | Ответ | Проверка |
| --- | --- | --- |
| `GET /healthz` | `200 {"status":"ok"}` | Процесс принимает HTTP-запросы. |
| `GET /readyz` | `200 {"status":"ok"}` или `503` | `Ping` PostgreSQL, если он настроен; в memory-режиме всегда `200`. |

gRPC-сервер принимает сообщения размером до 64 MiB и переносит `x-request-id` из metadata в контекст запроса. TLS настраивается общими переменными из `backend/pkg/grpcutil`.

## Зависимости и взаимодействия

- Go `1.26.7`;
- `google.golang.org/grpc` `v1.83.2` и локальный модуль `backend/proto` для контракта;
- локальный модуль `backend/pkg` для gRPC transport, health и JSON-логов;
- `github.com/jackc/pgx/v5` `v5.10.0` для PostgreSQL;
- `gateway-service` — подтверждённый producer аудиторских событий; прямых downstream-вызовов у audit-service нет.

## Конфигурация


<!-- docs-sync:env -->
| Переменная | По умолчанию | Обязательность и назначение |
| --- | --- | --- |
| `AUDIT_ENV` | значение `APP_ENV`, затем `local` | Окружение именно этого сервиса. Пустое значение также считается локальным. |
| `APP_ENV` | `local` | Общий fallback для `AUDIT_ENV`. Любое значение, кроме регистронезависимого `local` и пустой строки, включает production-проверку конфигурации. |
| `AUDIT_GRPC_ADDR` | `:9100` | Адрес внутреннего gRPC listener. |
| `AUDIT_HEALTH_ADDR` | `:8091` | Адрес HTTP listener для health-проверок. |
| `DATABASE_URL` | пусто | DSN PostgreSQL. Вне local обязателен; содержит учётные данные и должен передаваться как secret. При пустом значении события хранятся только в памяти. |
| `IDENTITY_GRPC_ADDR` | пусто | Адрес `identity-service` для ACL `ListEvents`. Вне local обязателен. |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. Неизвестное значение останавливает запуск. |
| `GRPC_TLS_CERT_FILE` | пусто | Путь к PEM-сертификату сервера; обязателен для `tls` и `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Путь к приватному PEM-ключу; обязателен для `tls` и `mtls`, должен монтироваться как secret. |
| `GRPC_TLS_CA_FILE` | пусто | Путь к CA bundle; обязателен для проверки клиентских сертификатов в режиме `mtls`. |
| `GRPC_TLS_SERVER_NAME` | пусто | Необязательное имя для проверки TLS-сертификата исходящих gRPC-клиентов. |
<!-- /docs-sync:env -->

Production-валидация не допускает запуск без `DATABASE_URL` и `IDENTITY_GRPC_ADDR` и требует `GRPC_TLS_MODE=mtls`. Она не проверяет, что DSN использует TLS: это обязанность deployment-конфигурации.

## Локальный запуск

Требуется Go `1.26.7`, указанный в `go.mod`, и доступные TCP-порты. Из каталога сервиса:

```bash
cd backend/services/audit-service
go run ./cmd/audit
```

Без переменных окружения сервис работает с памятью процесса. Проверка:

```bash
curl -fsS http://127.0.0.1:8091/healthz
curl -fsS http://127.0.0.1:8091/readyz
```

Для постоянного локального хранения сначала запустите PostgreSQL и передайте доступный с хоста DSN, например:

```bash
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
  go run ./cmd/audit
```

При подключении сервис создаёт/обновляет схему в одной транзакции до открытия listener-ов.

## Docker и Compose

`Dockerfile` ожидает build context `backend/`, собирает статический бинарник `./cmd/audit` в образе Go 1.26 Alpine и запускает его непривилегированным пользователем UID 10001:

```bash
docker build -f backend/services/audit-service/Dockerfile backend
```

Основной compose-файл репозитория — [`deploy/docker-compose.yml`](../../../deploy/docker-compose.yml); он подключает [`backend/deploy/docker-compose.yml`](../../deploy/docker-compose.yml). Compose передаёт PostgreSQL, адреса `:9100`/`:8091`, общую gRPC TLS-конфигурацию и ждёт healthcheck `GET /healthz`. Порты сервиса объявлены через `expose`, но не публикуются на host.

```bash
docker compose -f deploy/docker-compose.yml up --build audit-service
```

## Тесты и проверки

Из каталога сервиса:

```bash
go test ./...
go vet ./...
go build ./cmd/audit
```

Из `backend/` можно проверить все Go-модули командой `make check`. Тесты сервиса покрывают конфигурацию, memory store/service и совместимость начальной SQL-миграции; интеграционного теста с реальным PostgreSQL в этом модуле нет.

## Эксплуатационные заметки и ограничения

- Memory-режим теряет все события при перезапуске и подходит только для локальной разработки.
- Readiness проверяет только PostgreSQL. Он не подтверждает семантическую корректность данных или доступность вызывающих сервисов.
- Миграции выполняются при каждом старте и не ведут отдельную таблицу версий; SQL должен оставаться повторно применимым.
- `payload_json` хранится как `TEXT`: валидность JSON, схема и маскирование чувствительных данных не проверяются. Не передавайте пароли, токены и другие секреты.
- `ListEvents` без `company_id` для не-admin отклоняется или сужается до компании актора. Пагинации нет.
- Остановка по `SIGINT`/`SIGTERM` сначала пытается завершить gRPC-запросы корректно; предельное время graceful stop и HTTP shutdown — 10 секунд.

Общее устройство системы описано в [`docs/ARCHITECTURE.md`](../../../docs/ARCHITECTURE.md), границы сервисов — в [`docs/service-boundaries.md`](../../../docs/service-boundaries.md).
