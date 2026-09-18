# inbound-service

Внутренний gRPC-сервис, который принимает входящую почту 1С, доставленную CloudMailin на шлюз webhook-ом, и сохраняет её в изолированный контур хранения: отдельный инстанс PostgreSQL и отдельный S3-бакет `order-fill-inbound`. Сервис не читает Excel-файлы, не запускает обработку заказов и не общается с бизнес-сервисами в hot path приёма.

## Быстрый старт

Требуется Go `1.26.7` (версия зафиксирована в `go.mod`). Из каталога сервиса:

```bash
go run ./cmd/inbound
```

По умолчанию gRPC слушает `:9101`, а HTTP-проверки состояния — `:8093`.

```bash
curl http://127.0.0.1:8093/healthz
curl http://127.0.0.1:8093/readyz
```

`/readyz` дополнительно проверяет PostgreSQL (когда задан `INBOUND_DATABASE_URL`) и доступность S3-бакета.

## Возможности и границы ответственности

- Принимает один webhook-пейлоад CloudMailin, сохраняет его транзакционно и только после этого отвечает успехом.
- Дубликаты отсекаются по `provider_message_id` — повторный webhook возвращает успех без повторной записи.
- Вложения принимаются только как inline base64 `content`; URL не открываются (анти-SSRF). Одно вложение ограничено 20 МБ в раскодированном виде, всё тело запроса — 32 МБ.
- Текстовая и HTML-части письма сохраняются в собственной БД и доступны владельцу/админу компании через просмотр письма. HTML открывается в изолированном iframe без разрешения скриптов.
- При неизвестном адресе, отправителе вне whitelist, отсутствии вложений или слишком большом вложении сохраняются только минимальные метаданные со статусом `error:*` — содержимое не хранится, ответ 200 (CloudMailin перестаёт ретраить).
- Отвечает за настройки приёма (platform-wide), адрес и `sender_email` компании (один ящик 1С), ленту сообщений и скачивание вложений для владельца компании.
- Не авторизует пользователей напрямую — роли приходят от gateway в `RequestMeta`, вложения защищены учётными данными собственного S3-инстанса.
- Не создаёт заказы и jobs — письма остаются в контуре до явного импорта. Автосоздание `order_fill` из вложения не делается: заданию нужны бланки поставщика и актор компании, которых нет на webhook.

## Архитектура

```text
cmd/inbound                        точка входа и graceful shutdown
internal/config                    переменные окружения и значения по умолчанию
internal/bootstrap                 запуск gRPC, HTTP health-сервера, миграций и S3-бакета
internal/transport/grpcapi         преобразование protobuf <-> domain и проверка ролей
internal/service/inbound           логика приёма webhook и management-операции
internal/storage/postgres          store: настройки, компании, сообщения, вложения
internal/storage/objectstore       S3/MinIO boundary для содержимого вложений
internal/migrate                   встроенная миграция схемы
internal/domain                    модель и ошибки контура
```

Зависимости workspace-модулей `../../pkg` и `../../proto` подключены через `replace` в `go.mod`. Общий `grpcutil` добавляет `x-request-id`, ограничивает сообщение размером 64 MiB и выполняет graceful shutdown при `SIGINT`/`SIGTERM`.

## gRPC API

Сервис реализует `orderfill.inbound.v1.InboundService`. Полный контракт находится в [`../../proto/orderfill/inbound/v1/inbound.proto`](../../proto/orderfill/inbound/v1/inbound.proto).


<!-- docs-sync:rpc -->
| RPC | Полный путь | Назначение |
| --- | --- | --- |
| `IngestWebhook` | `/orderfill.inbound.v1.InboundService/IngestWebhook` | Принимает raw-пейлоад CloudMailin от gateway. Требует worker-токен. |
| `GetSettings` | `/orderfill.inbound.v1.InboundService/GetSettings` | Возвращает глобальные настройки приёма. Требует роль `platform_admin`. |
| `UpdateSettings` | `/orderfill.inbound.v1.InboundService/UpdateSettings` | Включает/останавливает приём. Требует роль `platform_admin`. |
| `GetCompanyInbound` | `/orderfill.inbound.v1.InboundService/GetCompanyInbound` | Возвращает адрес приёма и `sender_email` компании. Требует роль `company_owner` или `company_admin` этой компании. |
| `UpdateCompanyInbound` | `/orderfill.inbound.v1.InboundService/UpdateCompanyInbound` | Сохраняет адрес приёма, `sender_email` и включение. Требует роль `company_owner`. |
| `ListMessages` | `/orderfill.inbound.v1.InboundService/ListMessages` | Лента писем компании с метаданными вложений. Требует роль владельца/админа компании. |
| `GetMessage` | `/orderfill.inbound.v1.InboundService/GetMessage` | Просмотр тела одного письма в plain text/HTML. Требует роль владельца/админа компании. |
| `GetMessageFile` | `/orderfill.inbound.v1.InboundService/GetMessageFile` | Скачивание одного сохранённого вложения. Требует роль владельца/админа компании. |
| `ListDeliveries` | `/orderfill.inbound.v1.InboundService/ListDeliveries` | Статус-лента поступлений без содержимого и вложений. Требует роль `platform_admin`. |
<!-- /docs-sync:rpc -->

REST/HTTP бизнес-API у сервиса нет. HTTP используется только для health endpoints. Публичный webhook-контур принимает gateway (`POST /api/v1/inbound/webhook`), он же является единственным вызывающим этот gRPC-сервис.

## Взаимодействия

CloudMailin присылает письмо на webhook gateway через заголовок `Authorization`.
В настройках CloudMailin укажите либо точное значение `INBOUND_WEBHOOK_TOKEN`,
либо добавьте к нему префикс схемы авторизации и пробел; gateway принимает оба варианта и сравнивает
секрет constant-time. Для HTTPS webhook используйте URL
`/api/v1/inbound/webhook`. Gateway передаёт raw-пейлоад сюда через
`IngestWebhook` (worker-токен). Сервис в одной транзакции записывает сообщение и
метаданные вложений в PostgreSQL, а сами тела вложений кладёт в S3-бакет
`order-fill-inbound` с отдельными `INBOUND_S3_*` учётными данными MinIO.
После успешного ingest gateway пишет в `audit-service` событие
`inbound_webhook_received` или `inbound_rejected` best-effort: падение аудита
не меняет HTTP-ответ CloudMailin. Управление настройками и чтение писем идёт
через те же gRPC-вызовы из gateway по запросу UI. Смена адреса компании даёт
`inbound_address_updated`.

## Конфигурация


<!-- docs-sync:env -->
| Переменная | По умолчанию | Обязательность и смысл |
| --- | --- | --- |
| `INBOUND_GRPC_ADDR` | `:9101` | Адрес gRPC listener. |
| `INBOUND_HEALTH_ADDR` | `:8093` | Адрес HTTP listener для `/healthz` и `/readyz`. |
| `INBOUND_ENV` | `APP_ENV`, затем `local` | Вне local `Validate` требует `INBOUND_DATABASE_URL`, S3 endpoint/bucket и не-дефолтные ключи. |
| `APP_ENV` | `local` | Общий fallback окружения; пустое значение и `local` включают local-режим. |
| `INBOUND_DATABASE_URL` | пусто | PostgreSQL контура приёма; вне local обязателен. |
| `WORKER_TOKEN` | `local-dev-worker-token` | Общий секрет для worker-вызовов от gateway; вне local обязателен, ≥16 байт. |
| `INBOUND_S3_ENDPOINT` | `minio:9000` | Endpoint S3/MinIO для вложений; вне local обязателен. |
| `INBOUND_S3_ACCESS_KEY` | `minioadmin` | Access key вложений; `minioadmin` вне local запрещён. |
| `INBOUND_S3_SECRET_KEY` | `minioadmin` | Secret key вложений; `minioadmin` вне local запрещён. Не коммитить. |
| `INBOUND_S3_BUCKET` | `order-fill-inbound` | Имя бакета вложений; вне local обязателен. |
| `INBOUND_S3_USE_SSL` | пусто | `true` включает TLS для S3-клиента. |
| `GRPC_TLS_MODE` | `insecure` | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CERT_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_KEY_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_CA_FILE` | пусто | Читается из окружения; назначение см. config.go. |
| `GRPC_TLS_SERVER_NAME` | пусто | Необязательное имя для проверки TLS-сертификата исходящих gRPC-клиентов. |
<!-- /docs-sync:env -->

Приложение генерирует общий `grpcutil` также из `GRPC_TLS_*` переменных (как во всех сервисах workspace); локальный режим работает через `GRPC_TLS_MODE=insecure`.

## Docker и Compose

`Dockerfile` ожидает build context `backend/`, собирает `./cmd/inbound` с `CGO_ENABLED=0`, затем запускает бинарник от пользователя UID `10001` в Alpine 3.22. Например, из корня репозитория:

```bash
docker build -f backend/services/inbound-service/Dockerfile -t order-fill-inbound backend
docker run --rm -p 9101:9101 -p 8093:8093 order-fill-inbound
```

Штатный compose включает сервис `inbound-service` и выделенный инстанс PostgreSQL `inbound-postgres` (host port `5433:5432`). Из корня репозитория весь стек запускается командой:

```bash
make up
```

Compose публикует порты только внутри своей сети через `expose`; для обращения с хоста используйте явный `ports` override или локальный запуск бинарника.

## Тесты

Из каталога сервиса:

```bash
GOWORK=off go test ./...
```

Из каталога `backend/` можно проверить все Go-модули:

```bash
make test
```

Тесты используют fake-store и fake-object-store и покрывают приём webhook: неизвестный адрес, дедупликацию, остановленный приём, отправителя вне whitelist, отсутствие вложений, успешную запись с объектом в S3, некорректный base64, превышение лимита вложения, нечувствительный к регистру whitelist и fallback content-type.

## Эксплуатационные заметки и ограничения

- Перед выкладкой с миграцией `00006_unique_sender_email` разрешите дубликаты `sender_email` вручную: индекс уникален по `LOWER(sender_email)` и упадёт, если две компании уже делят один ящик 1С.
- Пустой `INBOUND_DATABASE_URL` теперь завершает процесс с ошибкой (`inbound store is required`), а не с кодом 0.
- Логи пишутся в stdout в JSON на уровне `info`.
- Сервер завершает работу по `SIGINT`/`SIGTERM`; graceful timeout gRPC/HTTP — 10 секунд.
- `ingest` идемпотентен по `provider_message_id`; ошибки webhook пишутся как сообщения со статусом `error:*` без содержимого.
- Лимит частоты webhook живёт на gateway (`INBOUND_WEBHOOK_RPS` / `INBOUND_WEBHOOK_BURST`); inbound-service его не дублирует.
- В проекте нет server reflection и отдельной OpenAPI-спецификации для этого сервиса; источником истины служит protobuf-контракт, публичная спецификация webhook живёт в gateway `api/openapi.yaml`.
