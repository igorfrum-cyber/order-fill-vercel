# document-service

Сервис чтения, анализа и изменения Excel-книг. Репозиторий собирает два процесса из одного Go-модуля: `document-api` предоставляет синхронный gRPC API для анализа загрузок и построения preview, а `document-worker` получает задания из Redis Stream и выполняет полный pipeline заполнения заказа или северного объединения.

`document-service` владеет форматом Excel и представлением отчета, но делегирует хранение файлов в `file-service`, состояние заданий в `job-service`, правила бренда в `brand-service`, расчеты в `calculation-service` и сопоставление товаров в `matching-service`.

## Быстрый старт

Требуется Go `1.26.7`. Процессы запускаются отдельно.

```bash
# gRPC API; для рабочих RPC нужны file-service и brand-service
FILE_GRPC_ADDR=127.0.0.1:9095 \
BRAND_GRPC_ADDR=127.0.0.1:9098 \
go run ./cmd/document-api

# воркер; нужны Redis и все пять gRPC-зависимостей
QUEUE_URL=redis://127.0.0.1:6379/0 \
JOB_GRPC_ADDR=127.0.0.1:9094 \
FILE_GRPC_ADDR=127.0.0.1:9095 \
CALCULATION_GRPC_ADDR=127.0.0.1:9099 \
MATCHING_GRPC_ADDR=127.0.0.1:9097 \
BRAND_GRPC_ADDR=127.0.0.1:9098 \
DOCUMENT_HEALTH_ADDR=:8092 \
go run ./cmd/document-worker
```

Команды предполагают запуск из каталога сервиса. API по умолчанию использует gRPC `:9096` и health `:8087`; воркер gRPC listener не создает и использует только `DOCUMENT_HEALTH_ADDR`.

## Возможности и pipeline

### document-api

- `AnalyzeInputs` загружает объекты через `file-service`, пытается разобрать их как OOXML-книги, находит выгрузку 1С по подписи группы номенклатуры, определяет бренд через `brand-service` и возвращает подпись единственного бланка.
- `BuildPreview` загружает книгу, строит snapshot листов и сохраняет gzip JSON-метаданные/чанки обратно через `file-service`.

### document-worker

- Читает JSON-задания из Redis Stream `order-fill:jobs`, consumer group `document-service`, поле сообщения `payload`.
- Поддерживает типы задания `order_fill` и `north_merge`, стадии `process` и `finalize`.
- Для `order_fill` параллельно загружает исходную таблицу 1С, опциональную таблицу второго склада Тюмени и бланки. Перед расчётом второго склада складываются остатки, товары в пути и продажи одинаковых позиций; несовместимые периоды и дубли отклоняются. Затем сервис определяет месяц и бренд, рассчитывает рекомендации, сопоставляет позиции, заполняет книги, создает отчет и preview. Для строк с ABC-анализом отчет также передает средний спрос и цену из единственной распознанной колонки цены бланка — эти данные используются экраном «Заказ до суммы». Стадия `process` завершает job в состоянии, предназначенном для review; `finalize` применяет ручные правки и завершает результат.
- Для `north_merge` извлекает потребности из городских бланков, опционально читает отдельные остатки офиса Тюмени и склада доставки, строит план через `calculation-service`, сохраняет review-отчет и применяет правки на стадии `finalize`.
- Сохраняет результаты под ключами `jobs/<job_id>/outputs/...`, отчет — `jobs/<job_id>/report.json`, список выходов — `jobs/<job_id>/outputs.json`, идентичность — `jobs/<job_id>/identity.json`, preview — `jobs/<job_id>/preview/<file_id>/...`.
- Сохраняет структуру OOXML, стили, формулы, размеры, объединения ячеек и включает принудительный пересчет формул при открытии результата.

## Границы ответственности

- Входная книга должна открываться как `.xlsx` или `.xlsm`; старый бинарный `.xls` codec не поддерживает, даже если имя проходит через вспомогательную обработку расширения.
- Для обычного `order_fill` требуется одна исходная выгрузка с ролью `source`, от одного до двух бланков с ролью `blank` и допускается одна таблица второго склада Тюмени с ролью `warehouse`. При наличии второго склада обе таблицы должны описывать один период и одинаковое число недель поставки; полная структура продаж и ABC обязательна у основной таблицы, а складская таблица может содержать только идентичность товара, остаток и товар в пути.
- Для `north_merge` требуется хотя бы один городской бланк. Город определяется по имени файла: Сургут, Вартовск, Уренгой или Тюмень. Таблицы офиса Тюмени и склада доставки опциональны.
- Сервис не владеет схемой jobs и object storage: он обращается к ним только через gRPC-контракты владельцев.
- Продуктовая идентичность и математика не должны реализовываться на transport-слое этого сервиса; текущий pipeline вызывает соответствующие сервисы.

## Архитектура

```text
cmd/document-api                   синхронный gRPC-процесс
cmd/document-worker                Redis consumer и фоновой pipeline
internal/bootstrap                 сборка зависимостей и lifecycle
internal/transport/grpcapi         DocumentService RPC
internal/adapter/inbound/queue     Redis Streams consumer
internal/adapter/outbound/xlsx     OOXML codec без внешнего Excel-процесса
internal/adapter/outbound/grpcjobs адаптеры JobService/FileService
internal/app/usecase               process/finalize и прогресс задания
internal/app/port                  внутренние порты и JSON queue contract
internal/clients                   gRPC-клиенты brand/calculation/matching/files/jobs
internal/domain/orderfill          обнаружение колонок, заполнение, отчет, правки
internal/domain/north              извлечение и review северного плана
internal/domain/preview            snapshot и gzip-чанки preview
internal/domain/spreadsheet        абстракция книги/листа/стиля
```

Для параллельной загрузки, сохранения и preview используется `errgroup` с ограничением concurrency на отдельных стадиях. Внешние unary gRPC-вызовы без собственного deadline получают timeout 60 секунд; максимальный размер gRPC-сообщения — 64 MiB.

## gRPC API

`document-api` реализует `orderfill.documents.v1.DocumentService`. Контракт: [`../../proto/orderfill/documents/v1/documents.proto`](../../proto/orderfill/documents/v1/documents.proto).


<!-- docs-sync:rpc -->
| RPC | Полный путь | Назначение и результат |
| --- | --- | --- |
| `AnalyzeInputs` | `/orderfill.documents.v1.DocumentService/AnalyzeInputs` | Принимает `job_id` и `input_file_ids`; возвращает `brand` и `blank_label`. Ошибка `InvalidArgument` означает отсутствие распознанной исходной книги, неизвестный бренд или неверное число бланков. |
| `BuildPreview` | `/orderfill.documents.v1.DocumentService/BuildPreview` | Принимает `job_id` и `file_id`, сохраняет preview-объекты и возвращает `snapshot_id`, равный `file_id`. Ошибка разбора книги возвращается как `InvalidArgument`. |
<!-- /docs-sync:rpc -->

Если API запущен без `FILE_GRPC_ADDR`, он поднимает gRPC-сервер, но оба RPC отвечают `Unavailable: document api is not configured`. При заданном `FILE_GRPC_ADDR` обязательно также задать `BRAND_GRPC_ADDR`: bootstrap иначе завершается ошибкой. Публичного REST API у процесса нет.

### Контракт очереди

JSON из поля `payload` соответствует `internal/app/port.JobMessage`:

```json
{
  "job_id": "job-123",
  "type": "order_fill",
  "stage": "process",
  "brand": "",
  "order_month": "",
  "matching_mode": "smart",
  "company_id": "co-1",
  "inputs": [
    {"role": "source", "name": "sales.xlsx", "storage_key": "uploads/source"},
    {"role": "blank", "name": "blank.xlsx", "storage_key": "uploads/blank"}
  ],
  "edits": []
}
```

Сообщение публикует `job-service`; отправлять его вручную обычно не требуется.

## Зависимости и взаимодействия

| Зависимость | Используется процессом | Для чего |
| --- | --- | --- |
| Redis (`QUEUE_URL`) | worker | Stream `order-fill:jobs`, group `document-service`. |
| `job-service` | worker | Прогресс, ошибки, завершение задания и список выходных файлов. |
| `file-service` | API и worker | Входные, выходные и служебные объекты. |
| `brand-service` | API и worker | Определение бренда и правила обработки. |
| `calculation-service` | worker | Рекомендации, корректировка количества и north plan. |
| `matching-service` | worker | Сопоставление строк и объединение ЧЗ. |

Исходящих HTTP-запросов нет. Все межсервисные бизнес-вызовы выполняются по gRPC.

Подтвержденные исходящие RPC: `FileService.GetObject`/`PutObject`, `JobService.UpdateProgress`/`CompleteJob`/`FailJob`, `BrandService.DetectBrand`/`GetBrandPolicy`, `CalculationService.CalculateOrderRecommendations`/`CalculateAdjustedQuantity`/`CalculateNorthPlan`, `MatchingService.MatchRows`/`MergeChestnyZnak`. Их protobuf-контракты находятся в `../../proto/orderfill/*/v1`.

## Конфигурация

Пустая переменная трактуется как отсутствие значения и заменяется указанным default.


<!-- docs-sync:env -->
| Переменная | По умолчанию | API | Worker | Смысл |
| --- | --- | --- | --- | --- |
| `DOCUMENT_ENV` | `APP_ENV`, затем `local` | validation | validation | Окружение сервиса; имеет приоритет над `APP_ENV`. |
| `APP_ENV` | `local` | fallback | fallback | Общее окружение, если `DOCUMENT_ENV` пуст. |
| `DOCUMENT_GRPC_ADDR` | `:9096` | listener | не используется | Адрес gRPC API. |
| `DOCUMENT_HEALTH_ADDR` | `:8087` | listener | listener | HTTP `/healthz` и `/readyz`; для одновременного локального запуска процессов задайте разные порты. |
| `QUEUE_URL` | пусто | не используется | практически обязательно | Redis URL, например `redis://redis:6379/0`; может содержать пароль, поэтому считается секретом. |
| `JOB_GRPC_ADDR` | пусто | не используется | обязательно для работы | Адрес `job-service`. |
| `FILE_GRPC_ADDR` | пусто | обязательно для рабочих RPC | обязательно | Адрес `file-service`. |
| `CALCULATION_GRPC_ADDR` | пусто | не используется | обязательно | Адрес `calculation-service`. |
| `MATCHING_GRPC_ADDR` | пусто | не используется | обязательно | Адрес `matching-service`. |
| `BRAND_GRPC_ADDR` | пусто | обязателен вместе с File | обязательно | Адрес `brand-service`. |
| `WORKER_TOKEN` | пусто | исходящие file RPC | исходящие job/file RPC | Секрет `x-worker-token`. Вне local обязателен, не default, ≥16 байт. |
| `GRPC_TLS_MODE` | `insecure` | да | да | Общий режим клиента/сервера: `insecure`/`disabled`/`off`, `tls`, `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | при TLS | при mTLS-клиенте | PEM certificate/key pair; обязателен серверу в `tls`/`mtls`, клиенту в `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | при TLS | при mTLS-клиенте | Закрытый PEM-ключ; хранить как секрет вне репозитория. |
| `GRPC_TLS_CA_FILE` | пусто | опционально в TLS, обязательно в mTLS | опционально в TLS, обязательно в mTLS | Доверенный CA bundle. |
| `GRPC_TLS_SERVER_NAME` | target host | для клиентов | для клиентов | Явное TLS server name для всех исходящих gRPC-соединений процесса. |
<!-- /docs-sync:env -->

В окружении, отличном от `local` (без учета регистра), `ValidateAPI` требует `FILE_GRPC_ADDR`, `BRAND_GRPC_ADDR`, `WORKER_TOKEN` и `GRPC_TLS_MODE=mtls`. `ValidateWorker` дополнительно требует все внешние адреса и Redis URL с паролем.

## Docker и Compose

Один `Dockerfile` собирает выбранный бинарник через build argument `BIN` (`document-api` по умолчанию). Build context должен быть `backend/`.

```bash
# из корня репозитория
docker build -f backend/services/document-service/Dockerfile \
  --build-arg BIN=document-api -t order-fill-document-api backend

docker build -f backend/services/document-service/Dockerfile \
  --build-arg BIN=document-worker -t order-fill-document-worker backend
```

Финальный Alpine 3.22 запускает бинарник от UID `10001`. Штатные сервисы `document-api` и `document-worker` определены в [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml):

```bash
# из корня репозитория; поднимутся и объявленные зависимости
docker compose -f deploy/docker-compose.yml up --build document-api document-worker
```

В compose API использует health-порт `8087`, worker — `8092`; бизнес-порты доступны только внутри compose network. `QUEUE_URL` по умолчанию compose — `redis://redis:6379/0`.

## Тесты

```bash
# из каталога сервиса
go test ./...

# все Go-модули, из backend/
make test
```

Пакет содержит unit- и integration-style тесты без внешнего Excel: queue consumer, use cases, gRPC API, клиенты, распознавание колонок/периодов/брендов, заполнение и финальные правки, north report, preview, OOXML round-trip, формулы и стили. Тяжелый large-file тест пропускается, если не задан `ORDERFILL_BENCH`.

## Эксплуатация и текущие ограничения

- `/healthz` и `/readyz` возвращают `200` без проверки Redis или gRPC-зависимостей. У worker health server запускается до входа в consumer loop; положительный ответ не гарантирует доступность очереди.
- Consumer обрабатывает по одному сообщению, блокирует чтение максимум на 3 секунды и пытается забрать pending-сообщения, простаивающие 30 минут. Имя consumer строится из hostname и PID.
- Некорректный payload подтверждается (`XACK`) после логирования. Ошибка handler оставляет сообщение в PEL для повторной доставки через `XAutoClaim`; автоматического ack после handler error нет. Use case по возможности записывает failure в `job-service`.
- OOXML-парсер отклоняет части с более чем 250000 открывающих XML-тегов и номера строк выше Excel-лимита 1048576.
- API пропускает файлы, которые не удалось разобрать, во время `AnalyzeInputs`; если среди оставшихся не найдена выгрузка 1С, возвращается `InvalidArgument`.
- Preview хранится как `application/gzip`: `meta.json.gz` и чанки по 256 строк. Запись preview выполняется параллельно и может создать часть объектов до возврата ошибки.
- Сохранение нескольких output-файлов также не является транзакционным; при частичном сетевом сбое возможны уже записанные объекты до отметки job как failed.
- Значение `order_month` из сообщения не используется на стадии первичной обработки `order_fill`: месяц выводится из исходной книги. Поле применяется как часть контракта сообщения, но не как источник расчета в текущем use case.
- Логи пишутся в stdout в JSON. Процессы реагируют на `SIGINT`/`SIGTERM`; gRPC/HTTP graceful shutdown ограничен 10 секундами. Worker завершает health server после остановки consumer.
- Метрики предусмотрены внутренним портом, но production bootstrap передает `nil`; отдельного `/metrics` endpoint сейчас нет.
- Server reflection и отдельной OpenAPI-спецификации нет; protobuf и `JobMessage` в коде являются источниками истины.
