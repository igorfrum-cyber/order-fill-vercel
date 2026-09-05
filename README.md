# Заполнение бланка заказа

Текущий локальный рантайм — backend v2:

- `frontend/` ходит в `gateway-service` (`:8080`);
- внутренние сервисы общаются по gRPC;
- Redis — очередь jobs, Excel считает `document-worker`.

Старый `services/api-service` в compose больше не поднимается.

```text
frontend --> gateway-service --> identity / jobs / files / …
                              --> document-worker (Redis)
```

Целевой дизайн:

- `docs/plans/2026-09-04-microservice-architecture-v2-design.md`
- `docs/plans/2026-09-04-backend-v2-microservices-implementation-plan.md`

## Локальный запуск

```bash
cp .env.example .env
docker compose -f deploy/docker-compose.yml up --build
```

- frontend: http://127.0.0.1:3200
- gateway: http://127.0.0.1:8080/healthz
- MinIO console: http://127.0.0.1:9001

Первый вход: смотрите в логах `identity-service` строку `bootstrap admin invite` и примите приглашение на `/invite`.

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

Контракт API: `backend/services/gateway-service/api/openapi.yaml` (указатель: `packages/contracts/openapi.yaml`).
Правила для агентов: `CLAUDE.md`.
