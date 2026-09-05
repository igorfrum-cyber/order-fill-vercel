# Документация проекта Order Fill

## Источник истины по архитектуре

Текущие `ARCHITECTURE.md` и `service-boundaries.md` описывают старую схему `api-service` + `document-service`. Их не расширять.

Целевой backend:

- [Backend v2 architecture](./plans/2026-09-04-microservice-architecture-v2-design.md)
- [Backend v2 implementation plan](./plans/2026-09-04-backend-v2-microservices-implementation-plan.md)

Продуктовые решения, которые v2 должен сохранить:

- [Режимы сопоставления](./plans/2026-09-05-order-matching-modes-design.md)
- [Правка количества в превью бланка](./plans/2026-09-05-blank-preview-quantity-edits-design.md)
- [Purchaser upload / preview / downloads](./plans/2026-09-05-purchaser-upload-preview-downloads.md)

## Прочее

- [Техническое задание](./TECHNICAL_SPEC.md)
- [Текущее поведение](./current-behavior.md)

Локальный рантайм: frontend → `gateway-service` → gRPC-сервисы в `backend/`. Excel считает `document-worker`.
