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

## Production knobs

Для production задайте как минимум:

- `APP_ENV=production`
- `API_ALLOWED_ORIGINS=https://<public-host>`
- `SESSION_COOKIE_SECURE=true`
- `WEBAUTHN_RP_ID=<public-host>`
- `FILE_S3_ENDPOINT`, `FILE_S3_USE_SSL=true`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`
- `TWOFA_MASTER_KEY` с непустым production secret длиной от 32 байт
- `GRPC_TLS_MODE=mtls` плюс `GRPC_TLS_CERT_FILE`, `GRPC_TLS_KEY_FILE`, `GRPC_TLS_CA_FILE`

Без этих значений stateful сервисы в production падают на старте вместо in-memory fallback.

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
