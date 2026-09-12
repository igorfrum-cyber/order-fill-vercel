# Архитектура

## Текущее состояние

Проект работает как backend v2: браузер общается только с публичным
`gateway-service`, а бизнес-операции разнесены по внутренним Go-сервисам.
Старые root-level Go-модули удалены из рабочего пути.

```text
browser
  |
  v
frontend nginx
  |
  v
gateway-service  --gRPC--> identity-service
      |          --gRPC--> job-service
      |          --gRPC--> file-service
      |          --gRPC--> twofa-service
      |          --gRPC--> passkey-service
      |
      +-- Redis stream --> document-worker

internal services --> PostgreSQL
file-service      --> S3-compatible object storage
document-worker   --gRPC--> file / brand / matching / calculation / jobs
```

Frontend в production отдает статические файлы через nginx и проксирует
`/api/` в `gateway-service:8080`. Прямых browser-запросов к внутренним
сервисам нет.

## Принципы

- Frontend не содержит бизнес-логики обработки документов.
- Единственная внешняя HTTP-граница backend: `gateway-service`.
- Межсервисные вызовы внутри backend идут по protobuf/gRPC.
- Долгие Excel-операции выполняются в `document-worker`, а не в HTTP request
  lifecycle.
- Jobs, входные файлы, результаты и audit-события хранятся в durable
  зависимостях.
- Production-конфигурация fail-fast: без PostgreSQL, Redis, object storage,
  корректного CORS, secure cookies, `WORKER_TOKEN` и `GRPC_TLS_MODE=mtls` сервис
  не стартует. `make https` / production overlay выпускает отдельный сертификат
  на каждый сервис (`scripts/gen-internal-tls.sh`) и монтирует в контейнер
  только его ключ.
- Внутренний insecure gRPC допустим только для локальной разработки. Для
  production обязателен `GRPC_TLS_MODE=mtls`: клиент проверяет DNS-имя пира,
  сервер требует клиентский сертификат того же CA. Allowlist RPC по имени
  клиента (кто именно может вызвать Complete/GetMe) — следующий потолок.

## Сервисы

### frontend

Пользовательский интерфейс: загрузка файлов, выбор компании и режима,
отображение статуса job, просмотр отчета, ручные правки и скачивание результата.
Excel не парсит.

### gateway-service

Публичный HTTP API продукта.

Отвечает за:

- CORS, secure headers, cookies и session middleware;
- проверку авторизации на публичных маршрутах;
- прием пользовательских запросов и файлов;
- оркестрацию gRPC-вызовов во внутренние сервисы;
- streaming/download endpoints;
- ограничение размера upload и preview windows.

### identity-service

Владение пользователями, компаниями, членством, сессиями и bootstrap invite.
PostgreSQL является обязательным в production.

### twofa-service

TOTP-секреты, backup codes и шифрование 2FA данных. В production требует
durable PostgreSQL/Redis и непустой 32+ byte master key.

### passkey-service

WebAuthn/passkey ceremonies и credentials. В production запрещает HTTP origins
и требует корректный RP ID.

### job-service

Владение job metadata, статусами, report rows, output metadata и публикацией
work messages в Redis stream.

### file-service

Единственный владелец object storage операций для входных и выходных файлов.
Объекты несут `company_id`; Get/Put сверяют его с identity-актором или `WORKER_TOKEN`.
Локально может работать с MinIO, в production должен использовать
S3-compatible storage с непустыми credentials и TLS endpoint.

### document-service

Один Go-модуль с двумя процессами:

- `document-api`: внутренний gRPC API `AnalyzeInputs` / `BuildPreview`; текущий
  gateway напрямую к нему не подключён;
- `document-worker`: Redis consumer, который выполняет тяжелую Excel-обработку.

Отвечает за чтение `.xlsx`, нормализацию, правила брендов, режим
"Север", генерацию отчетов, preview sidecar objects и итоговых workbook files.
Сопоставление товаров не считает сам: передаёт структурированные строки в
`matching-service` и записывает возвращённые `category` / `match_reasons` в
`report.json`. Режим сопоставления берёт из snapshot `matching_mode` в Redis
message, а не из identity-service.

### brand-service, matching-service, calculation-service

Внутренние вычислительные сервисы под правила брендов, сопоставление и расчеты.
Они не доступны из браузера напрямую. `matching-service` возвращает канонические
категории отчёта; Excel не читает.

### audit-service

Запись audit-событий пользовательских и системных действий в PostgreSQL.

## Хранилища

### PostgreSQL

Хранит пользователей, компании, сессии, passkeys, 2FA, jobs, статусы, report
rows, output metadata и audit log.

### Redis

Используется для очереди `order-fill:jobs`, rate/session-related volatile
state и межпроцессной координации там, где нужна короткоживущая mutable state.

### Object Storage

S3-compatible хранилище для входных Excel-файлов, preview chunks, draft/final
outputs и архивов.

## Поток "Заполнение бланка"

```text
1. Пользователь загружает source workbook и blank workbook во frontend.
2. frontend отправляет multipart request в gateway-service.
3. gateway-service проверяет session и вызывает file-service для сохранения inputs.
4. gateway-service вызывает job-service для создания job.
5. job-service сохраняет metadata в PostgreSQL и публикует Redis message.
6. document-worker читает сообщение из consumer group.
7. document-worker получает input files через file-service/object storage.
8. document-worker читает Excel, вызывает matching-service с matching_mode из сообщения очереди и сохраняет report.json плюс output artifacts.
9. document-worker обновляет job status/report/output metadata через job-service.
10. frontend читает status/report/preview через gateway-service.
11. Пользователь отправляет ручные правки.
12. job-service публикует finalize message.
13. document-worker применяет правки и сохраняет финальные файлы.
14. frontend скачивает результат через gateway-service.
```

## Поток "Север"

```text
1. Пользователь загружает заполненные бланки городов.
2. gateway-service создает north-merge job через job-service.
3. document-worker определяет города и типы бланков.
4. document-worker собирает потребности по городам.
5. document-worker учитывает остаток, товар в пути и целевой запас Тюмени.
6. document-worker строит план: из Тюмени / у поставщика.
7. frontend показывает расчет для проверки.
8. Пользователь правит фактический заказ у поставщика.
9. document-worker формирует общий бланк, перемещения и таблицу заказа.
```

## Production Минимум

Минимальный production-стенд:

```text
2 frontend replicas behind HTTPS ingress
2 gateway-service replicas
2 identity-service replicas
2 job-service replicas
2 file-service replicas
2 document-api replicas
2-5 document-worker replicas
1+ PostgreSQL primary/managed cluster
1+ Redis/managed queue
1 S3-compatible object storage
```

`document-worker` масштабируется отдельно от HTTP API, потому что именно он
выполняет CPU/RAM тяжелую работу. `gateway-service` должен оставаться тонким
edge/orchestration слоем.
