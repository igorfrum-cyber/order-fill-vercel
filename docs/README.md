# Документация проекта Order Fill

## Источник истины по архитектуре

Текущий backend v2:

- [Архитектура](./ARCHITECTURE.md)
- [Service boundaries](./service-boundaries.md)

Дизайн и план миграции:

- [Backend v2 architecture](./plans/2026-09-04-microservice-architecture-v2-design.md)
- [Backend v2 implementation plan](./plans/2026-09-04-backend-v2-microservices-implementation-plan.md)

Продуктовые решения, которые v2 должен сохранить:

- [Режимы сопоставления](./plans/2026-09-05-order-matching-modes-design.md)
- [Правка количества в превью бланка](./plans/2026-09-05-blank-preview-quantity-edits-design.md)
- [Purchaser upload / preview / downloads](./plans/2026-09-05-purchaser-upload-preview-downloads.md)

## Прочее

- [Техническое задание](./TECHNICAL_SPEC.md)
- [Историческое поведение frontend до backend v2](./current-behavior.md)

Локальный рантайм: frontend → `gateway-service` → gRPC-сервисы в `backend/`. Excel считает `document-worker`.
