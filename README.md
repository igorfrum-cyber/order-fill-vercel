# Заполнение бланка заказа

Инструмент переписывается в сервисную схему:

- frontend отвечает за загрузку файлов, просмотр отчета и ручные правки;
- api-service принимает jobs, хранит метаданные и отдает статус;
- document-service владеет Excel-логикой и формированием отчетов;
- Redis используется как очередь, PostgreSQL как хранилище метаданных, MinIO как S3-compatible object storage.

## Что делает

- находит колонки по заголовкам, а не по буквам;
- показывает объем, остаток, товар в пути, рекомендацию и вставленное количество;
- округляет заказ до целой коробки, если добавка не больше 15% или уменьшение не больше 5%;
- автоматически пишет комментарий `до коробки` при таком округлении;
- требует новый комментарий, если менеджер вручную меняет значение в колонке `Вставлено`;
- заполняет в таблице заказа товара колонки `Заказано по факту` и `Комментарий`.

## Локальный запуск

Основной локальный runtime:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Сервисы:

- frontend: http://127.0.0.1:3200
- api-service health: http://127.0.0.1:8080/healthz
- document-service health: http://127.0.0.1:8081/healthz
- MinIO console: http://127.0.0.1:9001

Frontend отдельно, для быстрой верстки:

```bash
npm install
npm run dev
```

Открыть:

```text
http://127.0.0.1:3200
```

## Проверка

```bash
npm run verify
```

## Docker Compose

Скопируйте `.env.example` в `.env` при необходимости локальных переопределений.

```bash
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml up --build
```

Для production-like локального запуска compose уже включает persistent volumes, healthchecks, restart policy и resource limits.

## Контракты

OpenAPI contract лежит в `packages/contracts/openapi.yaml`.
