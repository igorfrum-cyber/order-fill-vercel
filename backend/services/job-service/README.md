# job-service

Внутренний gRPC-сервис жизненного цикла задач Order Fill. Он хранит состояние задач, снимок режима сопоставления компании, отчёты и ссылки на файлы, а также публикует команды обработки в Redis Stream. Сам сервис не обрабатывает Excel и не хранит бинарные файлы.

## Ответственность и возможности

- создаёт задачи типов `order_fill` и `north_merge` после проверки входных файлов;
- получает метаданные файлов из `file-service` от имени актора, без worker token, и режим сопоставления компании из `identity-service`;
- фиксирует `matching_mode` в задаче и в сообщении очереди на момент создания;
- публикует стадии `process` и `finalize` в Redis Stream `order-fill:jobs`;
- хранит статусы, progress, ошибки, входные/выходные ссылки и отчёт, а ручные правки публикует worker-у;
- ограничивает чтение задач ролью, компанией и владельцем;
- принимает от `document-worker` progress, результат или ошибку обработки;
- использует PostgreSQL либо непостоянное локальное хранилище;
- предоставляет отдельный HTTP health-интерфейс.

Границы сервиса: бинарные данные принадлежат `file-service`, настройки компаний — `identity-service`, обработка книг и consumer очереди — `document-service`/`document-worker`, публичный HTTP API — `gateway-service`.

## Архитектура и структура

```text
cmd/job/                     точка входа и graceful shutdown
internal/bootstrap/          сборка store, клиентов, publisher и серверов
internal/config/             env-конфигурация и production-валидация
internal/domain/             Job, Report, статусы, роли и authz
internal/service/jobs/       create/list/report/edit/complete use cases
internal/clients/files/      gRPC-клиент метаданных файлов
internal/clients/identity/   gRPC-клиент режима сопоставления компании
internal/queue/              Redis Stream и in-memory publisher
internal/storage/postgres/   jobs/job_reports через pgx
internal/storage/memory/     непостоянный store для разработки
internal/migrate/            встроенные SQL-миграции
internal/transport/grpcapi/  protobuf/gRPC adapter
```

Основной поток:

```text
gateway-service
  -> file-service: метаданные входов
  -> identity-service: matching_mode компании
  -> job-service: запись queued + XADD stage=process
  -> document-worker: обработка и progress
  -> job-service: report/files + needs_review или completed
  -> XADD stage=finalize после ручных правок
```

## API

Контракт: [`jobs.proto`](../../proto/orderfill/jobs/v1/jobs.proto), package `orderfill.jobs.v1`.


<!-- docs-sync:rpc -->
| RPC | Полный gRPC method | Назначение |
| --- | --- | --- |
| `CreateJob` | `/orderfill.jobs.v1.JobService/CreateJob` | Проверяет actor/type/files, создаёт job и публикует `stage=process`. Ошибка `XADD` переводит задачу в `failed`. |
| `GetJob` | `/orderfill.jobs.v1.JobService/GetJob` | Возвращает доступную actor-у задачу; недоступная скрывается как not found. |
| `ListJobs` | `/orderfill.jobs.v1.JobService/ListJobs` | Возвращает только задачи, разрешённые actor-у. PostgreSQL сортирует их по `created_at DESC`; memory store порядок не гарантирует. |
| `GetReport` | `/orderfill.jobs.v1.JobService/GetReport` | Возвращает сохранённые summary и строки отчёта после проверки доступа к задаче. |
| `ListFiles` | `/orderfill.jobs.v1.JobService/ListFiles` | Возвращает входные, затем выходные file references доступной задачи. |
| `SubmitEdits` | `/orderfill.jobs.v1.JobService/SubmitEdits` | Для `needs_review` или `completed` атомарно переводит job в `finalizing` и публикует `stage=finalize`. Конкурентный повтор получает `conflict`. Поле protobuf `Edit.field` сейчас не попадает в queue message. |
| `UpdateProgress` | `/orderfill.jobs.v1.JobService/UpdateProgress` | Обновляет status/message/progress; предназначен для worker-а. Значения status/progress дополнительно не валидируются. |
| `CompleteJob` | `/orderfill.jobs.v1.JobService/CompleteJob` | Сохраняет report и output files. После `finalizing`/`completed` ставит `completed`, иначе `needs_review`. |
| `FailJob` | `/orderfill.jobs.v1.JobService/FailJob` | Переводит задачу в `failed` и записывает сообщение ошибки. |
<!-- /docs-sync:rpc -->

Клиентские RPC используют `RequestMeta.actor_user_id`. Роль и company читаются из `identity-service`, metadata `x-actor-role` игнорируется. Разрешения:

- `purchaser` создаёт задачи для непустой компании и видит только собственные задачи этой компании;
- `company_admin` и `company_owner` создают задачи и видят все задачи своей компании;
- `platform_admin` видит все задачи, но `CanCreateJob` сейчас не разрешает этой роли создание;
- при создании задачи `GetObject` идёт от имени актора: чужой `file_id` другой компании `file-service` скрывает как not found;
- `UpdateProgress`, `CompleteJob` и `FailJob` требуют metadata `x-worker-token`, совпадающий с `WORKER_TOKEN`. Вне local токен обязателен и не может быть local default.

Поддерживаются статусы `queued`, `processing`, `needs_review`, `finalizing`, `completed`, `failed`. Для `order_fill` нужен ровно один файл с ролью `source`, не больше одного `warehouse` и от одного до двух `blank`; для `north_merge` нужен минимум один `blank`, а `source` и `warehouse` необязательны. Один и тот же файл (совпадающее имя) дважды не принимается. Все входы должны иметь расширение `.xlsx` или `.xlsm`. Роль берётся из первого сегмента object key, который вернул `file-service`.

Redis message хранится в поле `payload` записи stream `order-fill:jobs` как JSON версии `v1`: `job_id`, `type`, `stage`, `matching_mode`, `inputs`, а также `brand` для process и `edits` для finalize.

HTTP-интерфейс:

| Метод и путь | Ответ | Проверка |
| --- | --- | --- |
| `GET /healthz` | `200 {"status":"ok"}` | Процесс принимает HTTP-запросы. |
| `GET /readyz` | `200 {"status":"ok"}` или `503` | `Ping` PostgreSQL при настроенном DSN; Redis и downstream gRPC не проверяются. |

## Зависимости и взаимодействия

- Go `1.26.7`;
- `google.golang.org/grpc` `v1.83.2`, локальные `backend/proto` и `backend/pkg`;
- `github.com/jackc/pgx/v5` `v5.10.0` для PostgreSQL;
- `github.com/redis/go-redis/v9` `v9.22.0` для публикации в Redis Stream;
- `gateway-service` вызывает клиентские RPC, `document-worker` — trusted state RPC;
- `file-service` предоставляет метаданные входов, `identity-service` — company matching mode, `document-worker` потребляет очередь.

## Конфигурация


<!-- docs-sync:env -->
| Переменная | По умолчанию | Обязательность и назначение |
| --- | --- | --- |
| `JOB_ENV` | значение `APP_ENV`, затем `local` | Окружение сервиса. Пустое значение также считается local. |
| `APP_ENV` | `local` | Общий fallback. Любое значение кроме `local`/пустого включает production-валидацию. |
| `JOB_GRPC_ADDR` | `:9094` | Адрес внутреннего gRPC listener. |
| `JOB_HEALTH_ADDR` | `:8085` | Адрес HTTP listener для health-проверок. |
| `QUEUE_URL` | пусто | Redis URL, например `redis://redis:6379/0`. Вне local обязателен; может содержать пароль и должен считаться secret. Пустое значение включает только внутрипроцессный publisher. |
| `FILE_GRPC_ADDR` | пусто | Адрес `file-service`. Вне local обязателен; фактически нужен для `CreateJob`. |
| `IDENTITY_GRPC_ADDR` | пусто | Адрес `identity-service`. Вне local обязателен. По нему читаются matching mode и роль актора; `x-actor-role` не доверяется. |
| `DATABASE_URL` | пусто | DSN PostgreSQL. Вне local обязателен и должен передаваться как secret; пустое значение включает memory store. |
| `WORKER_TOKEN` | пусто | Секрет worker RPC. Вне local обязателен, не default, ≥16 байт. Compose local: `local-dev-worker-token`. |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`; применяется и к server, и к downstream-клиентам. |
| `GRPC_TLS_CERT_FILE` | пусто | Собственный PEM-сертификат клиента и сервера; обязателен в `tls`/`mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Приватный PEM-ключ; обязателен в `tls`/`mtls`, должен быть secret. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle: обязателен в `mtls`; в `tls` может задавать доверенные root CA для downstream. |
| `GRPC_TLS_SERVER_NAME` | host из target | Явное имя для проверки сертификатов всех downstream gRPC-соединений. |
<!-- /docs-sync:env -->

Вне local сервис требует PostgreSQL, Redis, file-service, identity-service и `WORKER_TOKEN`. Production-валидация требует `GRPC_TLS_MODE=mtls` и запрещает пароль Postgres `order_fill`, `sslmode=disable` и Redis URL без пароля.

## Локальный запуск

Требуется Go `1.26.7`. Для реально работающего создания задач также нужны запущенные `file-service`, `identity-service` и Redis; PostgreSQL нужен для сохранения состояния между рестартами.

Пример запуска с инфраструктурой на localhost:

```bash
cd backend/services/job-service
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
QUEUE_URL='redis://127.0.0.1:6379/0' \
FILE_GRPC_ADDR='127.0.0.1:9095' \
IDENTITY_GRPC_ADDR='127.0.0.1:9091' \
  go run ./cmd/job
```

Health-проверки:

```bash
curl -fsS http://127.0.0.1:8085/healthz
curl -fsS http://127.0.0.1:8085/readyz
```

Запуск без env возможен только как ограниченный memory-mode: процесс и health endpoints стартуют, но сообщения не попадут в Redis, данные исчезнут при рестарте, а `CreateJob` не имеет настроенного клиента файлов.

## Docker и Compose

`Dockerfile` требует build context `backend/`, собирает `./cmd/job` с `CGO_ENABLED=0` и запускает бинарник под UID 10001:

```bash
docker build -f backend/services/job-service/Dockerfile backend
```

Основной compose-файл — [`deploy/docker-compose.yml`](../../../deploy/docker-compose.yml). В [`backend/deploy/docker-compose.yml`](../../deploy/docker-compose.yml) job-service зависит от здоровых PostgreSQL, Redis, file-service и identity-service. gRPC/health порты доступны в compose-сети через `expose`, но не опубликованы на host.

```bash
docker compose -f deploy/docker-compose.yml up --build job-service
```

Для полного асинхронного цикла нужен также `document-worker`; запуск только job-service и его declared dependencies позволяет публиковать сообщения, но не гарантирует их обработку.

## Тесты и проверки

```bash
cd backend/services/job-service
go test ./...
go vet ./...
go build ./cmd/job
```

Из `backend/` доступен `make check` для всех Go-модулей. Unit-тесты покрывают upload constraints, authz, matching-mode snapshot, queue message, complete/edits, конфигурацию, SQL migration contents и часть PostgreSQL error mapping. Интеграционных тестов с реальными PostgreSQL, Redis и gRPC-сервисами нет.

## Эксплуатационные заметки и ограничения

- `/readyz` проверяет только PostgreSQL. Недоступность Redis, file-service или identity-service обнаружится при соответствующем RPC, но не healthcheck-ом.
- Создание job и публикация в Redis не объединены транзакцией. Ошибка `XADD` помечает уже сохранённую задачу как `failed`; автоматического outbox/retry в сервисе нет.
- `SubmitEdits` переводит статус через compare-and-swap. Ошибка публикации откатывает `finalizing` обратно; гонка двух вызовов даёт один `finalize` в очереди.
- Redis publisher не закрывает клиент явно; delivery acknowledgement находится на стороне consumer-а. Job-service только выполняет `XADD`.
- `UpdateProgress` / `CompleteJob` / `FailJob` требуют `x-worker-token`. Статус/progress дополнительно не валидируются.
- Пользовательские RPC (`Create`/`Get`/`List`/`SubmitEdits`/`GetReport`/`ListFiles`) берут роль и company_id из `identity-service` по `actor_user_id`. Metadata `x-actor-role` игнорируется.
- Memory store и memory publisher предназначены для тестов/локальной разработки: данные и очередь теряются при завершении процесса, worker не может прочитать in-memory сообщения другого процесса.
- Local-валидация разрешает пустой `FILE_GRPC_ADDR`, но `CreateJob` без файлового клиента разыменует отсутствующую зависимость и может аварийно завершить процесс. Для вызова этого RPC адрес обязателен и в local.
- Миграции применяются целиком при каждом старте без таблицы версий. PostgreSQL хранит file references и report как JSONB.
- gRPC-клиенты имеют общий default timeout 60 секунд, если caller не установил deadline. Graceful stop сервера ограничен 10 секундами.

Общая архитектура: [`docs/ARCHITECTURE.md`](../../../docs/ARCHITECTURE.md); границы сервисов: [`docs/service-boundaries.md`](../../../docs/service-boundaries.md).
