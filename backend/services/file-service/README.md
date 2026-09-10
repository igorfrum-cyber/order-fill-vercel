# file-service

Внутренний gRPC-сервис бинарных объектов Order Fill. Он владеет содержимым файлов и их метаданными, предоставляет загрузку/чтение и сборку ZIP-архивов, но не разбирает Excel и не определяет бизнес-смысл документов.

## Ответственность и возможности

- сохраняет объект по переданному ключу либо генерирует ключ `objects/<id>/<safe-name>`;
- возвращает объект по `id` или ключу вместе со всем бинарным содержимым;
- поддерживает двухшаговую загрузку `CreateUpload` → `FinalizeUpload`; повторная финализация возвращает уже созданный объект;
- собирает несколько существующих объектов в ZIP и сохраняет архив как новый объект;
- нормализует имя до basename, удаляя компоненты Unix- и Windows-пути;
- хранит blob-данные в MinIO/S3-compatible storage и метаданные в PostgreSQL либо использует память процесса в local-режиме;
- создаёт bucket и применяет SQL-миграцию метаданных при старте;
- предоставляет liveness/readiness по отдельному HTTP listener.

Сервис не предоставляет публичный HTTP upload/download API. `gateway-service` загружает пользовательские файлы через gRPC, а `document-service` читает исходные и сохраняет сформированные артефакты.

## Архитектура и структура

```text
cmd/file/                         точка входа, сигналы SIGINT/SIGTERM
internal/bootstrap/               выбор хранилищ, миграции, серверы
internal/config/                  env-конфигурация и production-валидация
internal/domain/                  Object, Upload и безопасные имена
internal/service/files/           put/get/finalize/archive use cases
internal/storage/objectstore/     MinIO adapter и in-memory blob store
internal/storage/postgres/        метаданные objects/uploads через pgx
internal/storage/memory/          непостоянные метаданные
internal/migrate/                 встроенная SQL-миграция
internal/transport/grpcapi/       protobuf/gRPC adapter
```

Blob store и metadata store выбираются независимо. `FILE_S3_ENDPOINT` включает MinIO, `DATABASE_URL` включает PostgreSQL. Поэтому технически возможен смешанный режим; для устойчивой работы оба внешних хранилища должны быть настроены вместе.

## API

Контракт: [`files.proto`](../../proto/orderfill/files/v1/files.proto), package `orderfill.files.v1`.

| RPC | Полный gRPC method | Назначение |
| --- | --- | --- |
| `PutObject` | `/orderfill.files.v1.FileService/PutObject` | Сохраняет `body`, имя и content type. Пустой content type становится `application/octet-stream`. |
| `GetObject` | `/orderfill.files.v1.FileService/GetObject` | Ищет сначала по непустому `id`, иначе по `key`, и возвращает метаданные и всё тело. |
| `CreateUpload` | `/orderfill.files.v1.FileService/CreateUpload` | Создаёт запись загрузки и возвращает `upload_id`; бинарные данные ещё не принимаются. |
| `FinalizeUpload` | `/orderfill.files.v1.FileService/FinalizeUpload` | Сохраняет переданное тело для `upload_id`; повторный вызов возвращает метаданные первого объекта. |
| `CreateArchive` | `/orderfill.files.v1.FileService/CreateArchive` | Читает объекты по ID (при отсутствии — пробует значение как key), формирует ZIP в памяти и сохраняет его. |

`RequestMeta` присутствует в protobuf-запросах, но текущие обработчики его не используют. Внутри сервиса нет проверки пользователя или компании.

HTTP-интерфейс:

| Метод и путь | Ответ | Проверка |
| --- | --- | --- |
| `GET /healthz` | `200 {"status":"ok"}` | Процесс принимает HTTP-запросы. |
| `GET /readyz` | `200 {"status":"ok"}` или `503` | `Ping` PostgreSQL, если он настроен; MinIO/S3 здесь не проверяется. |

Максимальный размер входящего и исходящего gRPC-сообщения — 64 MiB. Поскольку тело передаётся одним полем `bytes`, полезный размер файла должен оставаться ниже этого предела с учётом protobuf overhead.

## Зависимости и взаимодействия

- Go `1.26.7`;
- `google.golang.org/grpc` `v1.83.2` и локальный `backend/proto` для контракта;
- локальный `backend/pkg` для gRPC transport, health и JSON-логов;
- `github.com/minio/minio-go/v7` `v7.3.0` для S3-compatible object storage;
- `github.com/jackc/pgx/v5` `v5.10.0` для метаданных в PostgreSQL;
- `gateway-service`, `job-service` и `document-service` — подтверждённые внутренние клиенты. Сам file-service не вызывает другие gRPC-сервисы.

## Конфигурация

| Переменная | По умолчанию | Обязательность и назначение |
| --- | --- | --- |
| `FILE_GRPC_ADDR` | `:9095` | Адрес внутреннего gRPC listener. |
| `FILE_HEALTH_ADDR` | `:8086` | Адрес HTTP listener для health-проверок. |
| `FILE_ENV` | значение `APP_ENV`, затем `local` | Окружение сервиса. Пустое значение также считается local. |
| `APP_ENV` | `local` | Общий fallback. Любое значение кроме `local`/пустого включает строгую проверку внешних хранилищ. |
| `DATABASE_URL` | пусто | DSN PostgreSQL для метаданных. Вне local обязателен и должен передаваться как secret. |
| `FILE_S3_ENDPOINT` | пусто | Endpoint MinIO/S3. Вне local обязателен. При схеме `http://`/`https://` схема определяет transport; без схемы используется `FILE_S3_USE_SSL`. |
| `FILE_S3_ACCESS_KEY` | `minioadmin` | Access key object storage. Вне local непустой и не может быть `minioadmin`; secret. |
| `FILE_S3_SECRET_KEY` | `minioadmin` | Secret key object storage. Вне local непустой и не может быть `minioadmin`; secret. |
| `FILE_S3_BUCKET` | `order-fill` | Bucket. При настройке endpoint пустой bucket приводит к ошибке старта. |
| `FILE_S3_USE_SSL` | `false` в local, `true` вне local | Только строка `true` включает TLS; вне local значение `false` запрещено. |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | PEM-сертификат сервера; обязателен для `tls`/`mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Приватный PEM-ключ; обязателен для `tls`/`mtls`, должен быть secret. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle; обязателен для проверки клиентов в `mtls`. |

Вне local сервис отказывается запускаться без PostgreSQL, S3 endpoint, значения `FILE_S3_USE_SSL=true` и нестандартных credentials. При этом endpoint с явной схемой `http://` переопределяет флаг уже после валидации, поэтому оператор должен отдельно исключить такой endpoint. Валидация также не проверяет TLS-параметры внутри `DATABASE_URL`.

## Локальный запуск

Требуется Go `1.26.7`. Самый простой непостоянный режим не требует PostgreSQL или MinIO:

```bash
cd backend/services/file-service
go run ./cmd/file
```

Проверка health:

```bash
curl -fsS http://127.0.0.1:8086/healthz
curl -fsS http://127.0.0.1:8086/readyz
```

Для устойчивого локального хранения запустите PostgreSQL и MinIO, затем из каталога сервиса:

```bash
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
FILE_S3_ENDPOINT='127.0.0.1:9000' \
FILE_S3_ACCESS_KEY='minioadmin' \
FILE_S3_SECRET_KEY='minioadmin' \
FILE_S3_BUCKET='order-fill' \
FILE_S3_USE_SSL='false' \
  go run ./cmd/file
```

При старте сервис проверяет PostgreSQL, применяет миграцию, проверяет bucket и создаёт его при отсутствии.

## Docker и Compose

`Dockerfile` собирается только с context `backend/`, потому что копирует общие модули `pkg` и `proto`. Runtime — Alpine 3.22 с `wget`, процесс работает под UID 10001:

```bash
docker build -f backend/services/file-service/Dockerfile backend
```

Основной compose-файл: [`deploy/docker-compose.yml`](../../../deploy/docker-compose.yml). В [`backend/deploy/docker-compose.yml`](../../deploy/docker-compose.yml) сервис зависит от PostgreSQL и MinIO; gRPC/health порты доступны только внутри compose-сети. Обратите внимание на отображение имён: compose читает `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, но передаёт процессу `FILE_S3_ACCESS_KEY`, `FILE_S3_SECRET_KEY`, `FILE_S3_BUCKET`.

```bash
docker compose -f deploy/docker-compose.yml up --build file-service
```

MinIO API публикуется на `127.0.0.1:9000`, консоль — на `127.0.0.1:9001`; сам file-service на host не публикуется.

## Тесты и проверки

```bash
cd backend/services/file-service
go test ./...
go vet ./...
go build ./cmd/file
```

Из `backend/` доступен `make check` для всех Go-модулей. Unit-тесты покрывают нормализацию имён, put/get, отсутствующий объект, ZIP, идемпотентную финализацию, ошибки метаданных, endpoint parsing и конфигурацию. Интеграционных тестов с реальными PostgreSQL/MinIO нет.

## Эксплуатационные заметки и ограничения

- При отсутствии `FILE_S3_ENDPOINT` blob-данные хранятся в памяти; при отсутствии `DATABASE_URL` там же хранятся метаданные. Перезапуск теряет соответствующую часть состояния.
- Не настраивайте только одно постоянное хранилище: PostgreSQL без устойчивых blob-данных оставляет метаданные на потерянные объекты, а MinIO без PostgreSQL теряет индекс ID/key после рестарта.
- `/readyz` проверяет только PostgreSQL и может отвечать `200`, когда MinIO недоступен.
- Production-проверка `FILE_S3_USE_SSL=true` сама по себе не гарантирует TLS: явная схема `http://` в `FILE_S3_ENDPOINT` заставляет MinIO-клиент использовать незашифрованное соединение.
- `PutObject`, `FinalizeUpload` и `GetObject` передают файл целиком в памяти. `CreateArchive` также читает все входы и строит весь ZIP в RAM; streaming и multipart upload не реализованы.
- При ошибке сохранения метаданных после успешной записи blob возможен объект без metadata. Между blob store и PostgreSQL нет общей транзакции.
- Повторный `PutObject` с тем же key перезаписывает blob и upsert-ит metadata с новым ID; старый ID перестаёт находиться в PostgreSQL.
- `FinalizeUpload` идемпотентен после сохранения `object_id`, но конкурентные первые вызовы не сериализованы на уровне use case.
- API не выполняет tenant-проверку и не имеет delete/list операций. Доступ должен быть ограничен доверенной внутренней сетью и, в production, mTLS.
- Миграция запускается при каждом старте без отдельного журнала версий.

Общая архитектура: [`docs/ARCHITECTURE.md`](../../../docs/ARCHITECTURE.md); границы ответственности: [`docs/service-boundaries.md`](../../../docs/service-boundaries.md).
