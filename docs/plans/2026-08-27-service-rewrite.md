# Service Rewrite Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Переписать текущий frontend-only Excel-инструмент в сервисную систему с тонким frontend, Go API, Go document worker, очередью задач и подготовленной границей для будущего Python-сервиса.

**Architecture:** Система строится как несколько сервисов: frontend, api-service, document-service, queue, postgres и object storage. На первом этапе источник данных остается Excel, но внутри используется нормализованная доменная модель, чтобы позже подключить 1С и Python без переписывания UI и API.

**Tech Stack:** Go, TypeScript frontend, PostgreSQL, Redis или NATS, S3-compatible storage, Docker, Docker Compose, OpenAPI, structured logging.

---

## Phase 0: Зафиксировать текущее поведение

### Task 1: Собрать behavior map текущего приложения

**Files:**
- Read: `src/app.js`
- Read: `src/workbookProcessor.js`
- Create: `docs/current-behavior.md`

**Step 1: Описать пользовательские сценарии**

Зафиксировать:

- обычное заполнение бланка;
- CHRISTINA HOME/PROFF;
- режим "Север";
- ручные правки;
- CSV-отчет для 1С;
- скачивание файлов.

**Step 2: Описать статусы report rows**

Зафиксировать значения:

- `matched`;
- `matched_by_name`;
- `warning_name_differs`;
- `warning_name_only`;
- `left_blank_nonpositive`;
- `not_in_source`;
- `not_in_blank`;
- `source_duplicate`.

**Step 3: Commit**

```bash
git add docs/current-behavior.md
git commit -m "docs: capture current workbook behavior"
```

### Task 2: Подготовить воспроизводимые фикстуры

**Files:**
- Create: `testdata/README.md`
- Create: `testdata/order-fill/`
- Create: `testdata/north/`
- Modify: `scripts/test-workbook.mjs`

**Step 1: Убрать абсолютные пути**

Текущий `scripts/test-workbook.mjs` использует локальные пути из `/Users/igorfrumes/Downloads`. Нужно перенести тестовые файлы в `testdata` или описать, какие приватные фикстуры нужны и как их положить локально.

**Step 2: Добавить smoke fixtures**

Создать минимальные `.xlsx` фикстуры программно, чтобы тесты не зависели от личных файлов.

**Step 3: Проверить**

```bash
npm run test:workbook
```

Expected: тест проходит на чистой машине после `npm ci`.

**Step 4: Commit**

```bash
git add scripts/test-workbook.mjs testdata
git commit -m "test: make workbook fixtures reproducible"
```

## Phase 1: Выделить доменную модель

### Task 3: Создать domain package в текущем JS-коде

**Files:**
- Create: `src/domain/brandRules.js`
- Create: `src/domain/orderModels.js`
- Create: `src/domain/normalization.js`
- Modify: `src/workbookProcessor.js`

**Step 1: Перенести brand rules**

Вынести `BRAND_RULES`, `brandRule`, `articleNormalizeOptions`, `blankDetectionOptions`.

**Step 2: Перенести normalization**

Вынести:

- `asText`;
- `normalizeHeader`;
- `normalizeArticle`;
- `normalizeName`;
- `parseNumber`;
- `roundHalfUp`.

**Step 3: Проверить**

```bash
npm run test:workbook
npm run build
```

Expected: поведение не изменилось.

**Step 4: Commit**

```bash
git add src/domain src/workbookProcessor.js
git commit -m "refactor: extract workbook domain primitives"
```

### Task 4: Выделить Excel adapter

**Files:**
- Create: `src/excel/xlsxXml.js`
- Modify: `src/workbookProcessor.js`

**Step 1: Перенести низкоуровневую работу с Excel XML**

Вынести:

- `loadXlsx`;
- `saveXlsx`;
- `parseXml`;
- `readSheetCells`;
- `setNumericCell`;
- `setTextCell`;
- `findOrCreateCell`;
- `forceFormulaRecalculation`.

**Step 2: Сохранить публичный API**

`workbookProcessor.js` должен временно реэкспортировать старые функции, чтобы UI не ломался.

**Step 3: Проверить**

```bash
npm run test:workbook
npm run build
```

**Step 4: Commit**

```bash
git add src/excel src/workbookProcessor.js
git commit -m "refactor: extract xlsx xml adapter"
```

## Phase 2: Спроектировать Go-сервисы и Docker baseline

### Task 5: Создать структуру репозитория под сервисы

**Files:**
- Create: `services/api-service/`
- Create: `services/document-service/`
- Create: `frontend/`
- Create: `packages/contracts/`
- Create: `deploy/docker-compose.yml`
- Create: `docs/service-boundaries.md`

**Step 1: Создать каркас**

```text
services/
  api-service/
    cmd/api/
    internal/http/
    internal/jobs/
    internal/storage/
    Dockerfile
  document-service/
    cmd/worker/
    internal/excel/
    internal/domain/
    internal/brands/
    internal/matching/
    internal/orderfill/
    internal/north/
    internal/reports/
    Dockerfile
frontend/
  Dockerfile
packages/
  contracts/
deploy/
```

**Step 2: Описать границы**

В `docs/service-boundaries.md` описать, какие модули не имеют права импортировать друг друга.

**Step 3: Commit**

```bash
git add services packages deploy docs/service-boundaries.md
git commit -m "chore: scaffold service boundaries"
```

### Task 6: Поднять Docker Compose baseline

**Files:**
- Create: `deploy/docker-compose.yml`
- Create: `.env.example`
- Create: `services/api-service/Dockerfile`
- Create: `services/document-service/Dockerfile`
- Create: `frontend/Dockerfile`
- Modify: `README.md`

**Step 1: Описать compose-сервисы**

Compose должен включать:

- `frontend`;
- `api-service`;
- `document-service`;
- `postgres`;
- `redis` или `nats`;
- `minio`.

**Step 2: Добавить healthchecks**

Минимум:

- `api-service`: `GET /healthz`;
- `document-service`: worker readiness endpoint или command healthcheck;
- `postgres`: `pg_isready`;
- `minio`: health endpoint;
- queue: штатная проверка выбранного брокера.

**Step 3: Добавить env contract**

В `.env.example` зафиксировать:

```text
DATABASE_URL=
QUEUE_URL=
S3_ENDPOINT=
S3_BUCKET=
S3_ACCESS_KEY=
S3_SECRET_KEY=
API_PUBLIC_URL=
```

**Step 4: Проверить локальный стенд**

```bash
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml up --build
```

Expected: все сервисы стартуют, healthchecks переходят в healthy или ready.

**Step 5: Commit**

```bash
git add deploy .env.example services frontend README.md
git commit -m "chore: add docker compose baseline"
```

### Task 7: Описать OpenAPI contract

**Files:**
- Create: `packages/contracts/openapi.yaml`

**Step 1: Добавить endpoints**

Описать:

- `POST /api/v1/jobs/order-fill`;
- `POST /api/v1/jobs/north-merge`;
- `GET /api/v1/jobs/{job_id}`;
- `GET /api/v1/jobs/{job_id}/report`;
- `POST /api/v1/jobs/{job_id}/edits`;
- `GET /api/v1/jobs/{job_id}/files`.

**Step 2: Добавить DTO**

Описать:

- `Job`;
- `JobStatus`;
- `OrderFillRequest`;
- `NorthMergeRequest`;
- `ReportRow`;
- `ManualEdit`;
- `OutputFile`;
- `ApiError`.

**Step 3: Проверить**

```bash
npx @redocly/cli lint packages/contracts/openapi.yaml
```

Expected: contract валиден.

**Step 4: Commit**

```bash
git add packages/contracts/openapi.yaml
git commit -m "docs: add service api contract"
```

## Phase 3: Реализовать api-service

### Task 8: Поднять Go API skeleton

**Files:**
- Create: `services/api-service/go.mod`
- Create: `services/api-service/cmd/api/main.go`
- Create: `services/api-service/internal/http/router.go`
- Create: `services/api-service/internal/http/health.go`

**Step 1: Написать health endpoint**

Endpoint:

```text
GET /healthz
```

Response:

```json
{"status":"ok"}
```

**Step 2: Добавить тест**

```go
func TestHealthz(t *testing.T) {
  req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
  rec := httptest.NewRecorder()
  router := NewRouter()
  router.ServeHTTP(rec, req)
  require.Equal(t, http.StatusOK, rec.Code)
}
```

**Step 3: Проверить**

```bash
cd services/api-service
go test ./...
```

**Step 4: Commit**

```bash
git add services/api-service
git commit -m "feat: add api service skeleton"
```

### Task 9: Реализовать создание jobs

**Files:**
- Create: `services/api-service/internal/jobs/model.go`
- Create: `services/api-service/internal/jobs/repository.go`
- Create: `services/api-service/internal/storage/object_storage.go`
- Modify: `services/api-service/internal/http/router.go`

**Step 1: Добавить модель job**

Статусы:

```text
queued
processing
needs_review
finalizing
completed
failed
```

**Step 2: Реализовать upload flow**

API должен принять multipart files, сохранить их в object storage и создать запись job.

**Step 3: Опубликовать событие в queue**

Событие:

```json
{
  "job_id": "uuid",
  "type": "order_fill",
  "input_files": []
}
```

**Step 4: Проверить интеграционным тестом**

Использовать fake storage и fake queue.

**Step 5: Commit**

```bash
git add services/api-service
git commit -m "feat: create workbook processing jobs"
```

## Phase 4: Реализовать document-service

### Task 10: Перенести Excel parser/writer в Go

**Files:**
- Create: `services/document-service/internal/excel/workbook.go`
- Create: `services/document-service/internal/excel/xml_reader.go`
- Create: `services/document-service/internal/excel/xml_writer.go`
- Create: `services/document-service/internal/excel/workbook_test.go`

**Step 1: Написать тест чтения workbook**

Тест должен открыть `.xlsx`, прочитать листы, shared strings и значения ячеек.

**Step 2: Реализовать чтение `.xlsx` как zip/XML**

Поведение должно соответствовать текущему `loadXlsx`.

**Step 3: Написать тест записи**

Проверить, что числовая ячейка меняется, workbook сохраняется, `calcChain.xml` удаляется.

**Step 4: Проверить**

```bash
cd services/document-service
go test ./internal/excel/...
```

**Step 5: Commit**

```bash
git add services/document-service/internal/excel
git commit -m "feat: add go xlsx xml adapter"
```

### Task 11: Перенести order-fill логику

**Files:**
- Create: `services/document-service/internal/domain/`
- Create: `services/document-service/internal/brands/`
- Create: `services/document-service/internal/matching/`
- Create: `services/document-service/internal/orderfill/`

**Step 1: Перенести нормализацию**

Покрыть тестами:

- кириллица/латиница в артикулах;
- `ё/е`;
- пробелы и спецсимволы;
- артикулы SOTHYS с дефисом.

**Step 2: Перенести правила брендов**

Покрыть тестами:

- ANGIOPHARM коробки;
- CHRISTINA кратность 3;
- KLAPP ближайшая кратность 3;
- NOVACUTAN минимумы.

**Step 3: Перенести matching**

Покрыть тестами:

- match by article;
- match by name;
- duplicate source rows;
- warning when names differ.

**Step 4: Проверить**

```bash
cd services/document-service
go test ./...
```

**Step 5: Commit**

```bash
git add services/document-service/internal
git commit -m "feat: port order fill domain logic to go"
```

### Task 12: Реализовать worker job execution

**Files:**
- Create: `services/document-service/cmd/worker/main.go`
- Create: `services/document-service/internal/jobs/consumer.go`
- Create: `services/document-service/internal/reports/report.go`

**Step 1: Worker получает job из queue**

Worker должен загрузить input files из object storage.

**Step 2: Worker запускает order-fill или north-merge**

Результат:

- JSON report;
- draft output files;
- job status update.

**Step 3: Worker пишет ошибки**

Ошибки должны иметь код, message и details.

**Step 4: Проверить**

```bash
cd services/document-service
go test ./...
```

**Step 5: Commit**

```bash
git add services/document-service
git commit -m "feat: process document jobs in worker"
```

## Phase 5: Обновить frontend

### Task 13: Перевести frontend на API

**Files:**
- Modify: `src/app.js`
- Create: `src/api/client.js`
- Create: `src/api/jobs.js`

**Step 1: Заменить локальную обработку на API calls**

Frontend должен:

- отправить файлы;
- получить `job_id`;
- poll статус;
- показать отчет;
- отправить edits;
- скачать output files.

**Step 2: Убрать прямые импорты workbookProcessor**

`src/app.js` не должен импортировать Excel processor.

**Step 3: Проверить**

```bash
npm run build
```

**Step 4: Commit**

```bash
git add src
git commit -m "refactor: move frontend to job api"
```

## Phase 6: Production packaging

### Task 14: Уточнить production packaging

**Files:**
- Modify: `deploy/docker-compose.yml`
- Modify: `README.md`

**Step 1: Проверить compose как основной dev runtime**

Compose уже должен запускать весь локальный стенд. На этом этапе нужно добавить production-like настройки: volumes, restart policy, resource limits и отдельные profiles для dev/prod.

**Step 2: Добавить команды запуска**

```bash
docker compose -f deploy/docker-compose.yml up --build
```

**Step 3: Проверить полный smoke flow**

Загрузить минимальные тестовые Excel-файлы через UI и получить результат.

**Step 4: Commit**

```bash
git add deploy .env.example README.md
git commit -m "chore: harden docker service stack"
```

### Task 15: Добавить наблюдаемость

**Files:**
- Modify: `services/api-service/`
- Modify: `services/document-service/`

**Step 1: Structured logs**

Каждый лог должен включать:

- `service`;
- `job_id`;
- `event`;
- `duration_ms`;
- `error_code`.

**Step 2: Metrics**

Добавить базовые метрики:

- jobs created;
- jobs completed;
- jobs failed;
- processing duration;
- file size.

**Step 3: Проверить**

```bash
go test ./...
```

**Step 4: Commit**

```bash
git add services
git commit -m "feat: add service logs and metrics"
```

## Phase 7: Удалить старое ядро из frontend

### Task 16: Удалить browser workbook processor

**Files:**
- Remove: `src/workbookProcessor.js`
- Remove or rewrite: `scripts/test-workbook.mjs`
- Modify: `package.json`

**Step 1: Убедиться, что frontend не импортирует processor**

```bash
rg "workbookProcessor|loadXlsx|fillWorkbook|saveXlsx" src
```

Expected: нет импортов из frontend.

**Step 2: Перенести оставшиеся тесты в Go**

Все бизнес-тесты должны жить в `services/document-service`.

**Step 3: Проверить**

```bash
npm run build
cd services/api-service && go test ./...
cd ../document-service && go test ./...
```

**Step 4: Commit**

```bash
git add src scripts package.json services
git commit -m "refactor: remove browser workbook engine"
```

## Completion Criteria

Проект считается переписанным, когда:

- frontend не обрабатывает Excel локально;
- все Excel-операции выполняются в `document-service`;
- API работает через jobs;
- файлы хранятся в object storage;
- статусы и отчеты хранятся в PostgreSQL;
- обработка запускается через queue;
- есть docker-compose для локального стенда;
- каждый собственный сервис имеет Dockerfile;
- локальная разработка запускается через Docker Compose;
- есть тесты на правила брендов, matching, Excel read/write и full job flow;
- текущие пользовательские сценарии не потеряны.

## Follow-up после MVP

- добавить пользователей и организации;
- добавить роли и права;
- добавить интеграцию с 1С;
- добавить Python calculation-service;
- добавить справочники брендов через UI;
- добавить billing/licensing для коммерческого продукта.
