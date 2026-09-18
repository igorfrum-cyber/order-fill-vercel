# Интеграция входящей почты 1С (inbound-mail)

> As-built: matching is by `envelope.from` / `sender_email`, not `envelope.to`. Public HTTP lives under `/api/v1/inbound/...`, not `/inbound-address`. Gateway records `inbound_webhook_received` / `inbound_rejected` / `inbound_address_updated` via `audit-service` best-effort (ingest stays 2xx/4xx if audit is down) and rate-limits the webhook with `INBOUND_WEBHOOK_RPS` / `INBOUND_WEBHOOK_BURST`. Auto-create `order_fill` job from an attachment is still not implemented: jobs require supplier `blank_files` and a company actor, which the webhook path does not have.

## Зачем это нужно

Автоматизировать приём таблиц продаж из 1С: 1С отправляет письмо на адрес платформы, CloudMailin принимает почту и доставляет её вебхуком, а платформа сохраняет вложения и показывает доставку в едином интерфейсе. Приём писем не зависит от работы основного контура обработки: если бизнес-сервисы лежат, письма все равно принимаются и сохраняются, а CloudMailin ретраит при недоступности нашего эндпоинта.

Границы доступа:

- настройки inbound и состояние доставки — только у `platform_admin`, без содержимого писем;
- адрес приёма для компании и whitelist отправителей 1С — у `company_owner`;
- список писем и статусы своей компании — у `company_owner` и `company_admin` (админ только просмотр);
- содержимое писем и вложения — только владеющая компания. `platform_admin` принципиально не может прочитать чужое письмо.

## Архитектура

Внутренний интегратор `inbound-service` работает как независимый контур за gateway, но для пользователя всё выглядит одной платформой: UI и API единые, session/auth общие.

```text
1С ──SMTP──► CloudMailin (почтовый ящик компании @inbound.<host>)
                 │  webhook POST /api/v1/inbound/webhook
                 ▼
        gateway-service (edge: Bearer INBOUND_WEBHOOK_TOKEN, лимиты, пересылка)
                 │  gRPC (worker-token)
                 ▼
┌─ inbound-service ──────── изолированный контур ─────────────────────────┐
│  • валидация + dedupe по Message-ID                                     │
│  • resolve компания по envelope.to из своей таблицы                      │
│  • вложения ─► свой S3 bucket (order-fill-inbound, отдельные creds)     │
│  • метаданные ─► свой Postgres (inbound-postgres)                       │
│  • статус/ошибка в метаданные                                           │
└─────────────────────────────────────────────────────────────────────────┘
        │ best-effort (не блокирует ответ)
        ▼
  audit-service: inbound_webhook_received / inbound_rejected / inbound_address_updated
```

Webhook возвращает `2xx` только после того, как письмо записано в свой bucket и свою БД. Любая ошибка → не-2xx → CloudMailin ретраит. Повторная доставка с тем же Message-ID → `200` без повторной записи (идемпотентность).

### Изоляция приёмного контура

| Слой | Решение | Причина |
|---|---|---|
| Сервис | Отдельный контейнер `inbound-service` под `backend/services/` | Собственные lifecycle, конфиг, health |
| Postgres | Отдельный инстанс `inbound-postgres` (свои креды, свои таблицы) | Падение/миграция основного Postgres не кладёт приёмку |
| Object storage | Отдельный bucket `order-fill-inbound` + `INBOUND_S3_*` | Основной контур не пересекается; компрометация одного не открывает другое |
| Redis | Не используется в приёмке | Меньше зависимостей в горячем пути |
| Бизнес-сервисы | Нет зависимостей: ни file/job/matching/calculation/identity | Горячий путь только свой bucket + своя БД |

Исключение из правила «file-service — единственная объектная граница»: inbound-контур работает со своим bucket через отдельные credentials. Требует явного пересмотра `docs/service-boundaries.md` и корневого README.

### Долговечность приёма

- CloudMailin хранит письмо и ретраит вебхук с бэкоффом, пока не получит `2xx` — внешний backstop, письмо не теряется даже при полном простое нашего контура.
- Горячий путь не вызывает ни одного бизнес-сервиса; привязка «адрес → компания» читается из собственной таблицы.
- `company_id` в mapping хранится как opaque-строка; существование компании проверяется в identity только в момент настройки адреса (вне горячего пути).

## Модель данных (собственные таблицы inbound-postgres)

`inbound_settings` — одна строка платформенных настроек:

- `enabled` — приём включён/выключен;
- `last_webhook_at`, `webhook_count`, `error_count` — статистика доставки без содержимого.

`company_inbound` — привязка адреса к компании:

- `company_id` (unique);
- `receive_address` (unique) — адрес, на который 1С шлёт письма;
- `allowed_from` [] — whitelist отправителей 1С;
- `enabled`, `created_by`, `updated_at`.

`inbound_message` — метаданные письма:

- `provider_message_id` (unique) — ключ идемпотентности из CloudMailin;
- `envelope_from`, `envelope_to`, `received_at`;
- `company_id` (nullable);
- `status`: `received` → `processed` | `error:unknown_address` | `error:mismatch_from` | `error:no_attachments` | `error:too_large`;
- `error_code`, `attachment_count`, `total_bytes`.

Тело письма (`plain` и `html`) хранится в inbound PostgreSQL для просмотра владельцем/админом компании. HTML показывается только в sandbox iframe без выполнения скриптов; вложения по-прежнему хранятся в bucket через inbound-контур.

## Разграничение доступа

| Действие | platform_admin | company_owner | company_admin | purchaser |
|---|---|---|---|---|
| Глобальные настройки + состояние доставки | ✅ (без содержимого) | ❌ | ❌ | ❌ |
| Адрес приёма компании + whitelist | ❌ | ✅ настройка | ✅ просмотр | ❌ |
| Список писем + статусы своей компании | ❌ | ✅ | ✅ read-only | ❌ |
| Тело/вложения письма | ❌ всегда | ✅ своя | ✅ своя read-only | ❌ |

`platform_admin` видит только статус-ленту доставки и агрегированную статистику (счётчики, timestamps, размеры) без `subject`, отправителя-адреса вне статистики и вложений.

## API (gateway, OpenAPI)

- `POST /api/v1/inbound/webhook` — публичный, только Bearer `INBOUND_WEBHOOK_TOKEN`, без сессии.
- `GET /api/v1/inbound/settings` — `platform_admin`.
- `GET/POST /api/v1/inbound/deliveries` — `platform_admin` (статус-лента без содержимого).
- `PUT /api/v1/companies/{company_id}/inbound-address` — `company_owner`.
- `GET /api/v1/companies/{company_id}/inbound-address` — `owner`/`admin`.
- `GET /api/v1/companies/{company_id}/inbound/messages` — `owner`/`admin` (у admin без mutating полей).
- `GET /api/v1/companies/{company_id}/inbound/messages/{message_id}/files/{file_id}` — только владеющая компания.

## Security (проверено по golang-security)

- Аутентификация внешнего источника: `INBOUND_WEBHOOK_TOKEN` из env, в проде не-default и ≥16 байт; сравнение через `crypto/subtle.ConstantTimeCompare`.
- Лимиты: body ≤32 МБ (паттерн из scratch), лимит числа вложений и суммарного объёма, rate-limit на webhook (`golang.org/x/time/rate`) — защита от DoS.
- SSRF: вложения принимаются только inline `content` (base64); `attachment.url` не дёргается — reject (как в scratch `main.go`).
- Timing/XSS: метаданные в UI через штатный React-эскейпинг; никакого ручного HTML.
- Транспорт: prod — HTTPS снаружи, gRPC `mtls` внутри (`make https` генерирует cert на сервис), S3 inbound по SSL.
- Логи: содержимое писем и вложений нигде не логируется.
- `platform_admin` не имеет маршрутов и credentials на bucket писем: защита двойная (gateway ACL + отсутствие доступа к бакету).

## Frontend

Один раздел «Интеграция 1С» в `frontend/src/ui/` с ролевыми экранами:

- `platform_admin`: настройки приёма (вкл/выкл), статус-лента доставки, агрегированная статистика.
- `company_owner`: адрес приёма + whitelist + список писем своей компании со статусами и скачиванием вложений.
- `company_admin`: список писем read-only (статусы видит, менять/скачивать может по решению владельца — в v1 только просмотр статусов, скачивание без правки).

Обновление статусов — лёгкий polling `GET .../inbound/messages` (интервал ~5–10 с), тот же паттерн, что `frontend/src/api/jobs.js` (pollJob). SSE/WebSocket не вводятся.

## Этапы работ

1. Дизайн-документ (этот файл).
2. `backend/proto/orderfill/inbound/v1/inbound.proto` + `make -C backend proto-gen` + Buf lint.
3. Скаффолд `inbound-service` по шаблону `brand-service`/`job-service` (config/domain/service/storage/transport/migrate), миграции — новым файлом, свой `inbound-postgres` в compose, свои `INBOUND_S3_*`.
4. Контейнер, `backend/go.work`, `backend/deploy/docker-compose.yml`, health.
5. Gateway: webhook-хендлер, маршруты, `api/openapi.yaml`, тесты.
6. Audit-события через `audit-service` (best-effort, не блокирует webhook).
7. Frontend-раздел + `npm run test:ui --prefix frontend` в той же сессии.
8. README inbound-service + root README, `.env.example`, `docs/service-boundaries.md`, `docs/ARCHITECTURE.md`; `make docs-sync`; `make verify`; `make security`.
9. Настройка CloudMailin: почтовый ящик `@inbound.<host>`, webhook → `/api/v1/inbound/webhook` с Bearer-токеном, формат JSON с inline-вложениями; документация шага.

## Открытые вопросы

- Авто-создание `order_fill` job из принятой таблицы продаж — отдельная итерация; в v1 письмо сохраняется и доступно компании как вложение.
- Судьба писем на неизвестный адрес: содержимое не сохраняется, платформа видит только факт доставки на конкретный адрес и время (для диагностики).