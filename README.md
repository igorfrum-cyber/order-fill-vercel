# Заполнение бланка заказа

Сервисная схема:

- `frontend/` — загрузка файлов, отчёт и ручные правки;
- `services/api-service/` — jobs, метаданные, статус;
- `services/document-service/` — Excel-логика и отчёты;
- Redis — очередь, PostgreSQL — метаданные, MinIO — файлы.

Браузер Excel не разбирает.

## Локальный запуск

```bash
cp .env.example .env
docker compose -f deploy/docker-compose.yml up --build
```

- frontend: http://127.0.0.1:3200
- api-service: http://127.0.0.1:8080/healthz
- document-service: http://127.0.0.1:8081/healthz
- MinIO console: http://127.0.0.1:9001

Только UI:

```bash
npm ci --prefix frontend
npm run dev --prefix frontend
```

## Проверка перед коммитом

Тот же набор шагов, что в GitHub Actions:

```bash
make verify
```

`make lint` — только версии, ESLint, gofmt, vet, golangci-lint, gosec и `go mod tidy`.

Контракт API: `packages/contracts/openapi.yaml`.
Правила для агентов: `CLAUDE.md`.
