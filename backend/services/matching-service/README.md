# matching-service

Внутренний stateless gRPC-сервис идентификации товарных строк. Он сопоставляет уже разобранные позиции источника и бланка по артикулу, названию, объему и форме, объясняет решение структурированными причинами и формирует каноническую категорию отчета. Excel и объектное хранилище остаются зоной ответственности `document-service`.

## Быстрый старт

Требуется Go `1.26.7`. Из каталога сервиса:

```bash
go run ./cmd/matching
```

По умолчанию gRPC слушает `:9097`, HTTP-проверки состояния — `:8088`.

```bash
curl http://127.0.0.1:8088/healthz
curl http://127.0.0.1:8088/readyz
```

## Возможности

- Нормализует артикулы: заменяет визуально совпадающие кириллические символы латиницей, переводит в верхний регистр, удаляет прочие символы и опционально сохраняет дефис.
- Нормализует названия: приводит регистр и пробелы, удаляет пунктуацию и служебные слова `АН`/`ANGIOPHARM`.
- Ищет точное или alias-соответствие артикула, включая числовой артикул с настраиваемыми префиксами.
- Выбирает лучший дубликат по похожести названия (LCS), повышая оценку при совпадении объема и снижая при конфликте.
- Фиксирует конфликты объема и формы товара и возвращает причины `article`, `name`, `volume`, `form`, `duplicates`, `source`.
- Поддерживает режимы `STANDARD` и `SMART`. Smart-режим требует ручного решения при неоднозначных близких дубликатах.
- Объединяет строки «Честный знак» с базовой строкой того же артикула; в smart-режиме отмечает неоднозначное слияние.

Возвращаемые категории определены общим protobuf-контрактом: `NEEDS_DECISION`, `NOT_IN_SOURCE`, `CHECK_NAME_OR_VOLUME`, `NOT_IN_BLANK`, `TO_ORDER`, `ORDER_NOT_NEEDED`. Текущая реализация `MatchRows` непосредственно формирует `NEEDS_DECISION`, `NOT_IN_SOURCE`, `CHECK_NAME_OR_VOLUME` и `TO_ORDER`; остальные категории доступны общему downstream-контракту.

## Архитектура

```text
cmd/matching                     точка входа и обработка сигналов
internal/config                  listener-адреса и окружение
internal/bootstrap               запуск gRPC и health HTTP
internal/transport/grpcapi       protobuf/domain mapping
internal/normalize               нормализация артикула, заголовка и имени
internal/service/matching        индекс, scoring, дубликаты и ЧЗ merge
internal/domain                  Item, Result, Reasons, Category и Mode
```

Состояние создается на один вызов и не сохраняется. Внешних сетевых зависимостей нет. Общий модуль `../../pkg` обеспечивает request ID, лимит gRPC-сообщения 64 MiB и graceful shutdown; сообщения берутся из `../../proto`.

## gRPC API

Сервис реализует `orderfill.matching.v1.MatchingService`; полный контракт — [`../../proto/orderfill/matching/v1/matching.proto`](../../proto/orderfill/matching/v1/matching.proto), категории и режимы — [`../../proto/orderfill/common/v1/common.proto`](../../proto/orderfill/common/v1/common.proto).

| RPC | Полный путь | Назначение |
| --- | --- | --- |
| `MatchRows` | `/orderfill.matching.v1.MatchingService/MatchRows` | Сопоставляет `blank_items` с `source_items`; принимает режим, `prefix_aliases` и `preserve_hyphen`, возвращает результат, score, причины и отсортированные candidate IDs. |
| `MergeChestnyZnak` | `/orderfill.matching.v1.MatchingService/MergeChestnyZnak` | Возвращает целевую строку, список ЧЗ-клонов и `needs_decision`. |
| `NormalizeArticle` | `/orderfill.matching.v1.MatchingService/NormalizeArticle` | Возвращает нормализованный артикул с опциональным сохранением дефиса. |
| `NormalizeName` | `/orderfill.matching.v1.MatchingService/NormalizeName` | Возвращает нормализованное название. |

Неуказанный режим трактуется как `STANDARD`. REST/HTTP бизнес-API нет; HTTP используется только для health endpoints. Поле `RequestMeta` сейчас не влияет на matching.

## Взаимодействия

`document-worker` разбирает книги в структурированные `Item`, получает правила `prefix_aliases` и `preserve_hyphen` у `brand-service`, вызывает `MatchRows`/`MergeChestnyZnak` и переносит категории и причины обратно в отчет и Excel. Сам `matching-service` не вызывает другие сервисы.

## Конфигурация

| Переменная | По умолчанию | Обязательность и смысл |
| --- | --- | --- |
| `MATCHING_GRPC_ADDR` | `:9097` | Адрес gRPC listener. |
| `MATCHING_HEALTH_ADDR` | `:8088` | Адрес HTTP listener для `/healthz` и `/readyz`. |
| `MATCHING_ENV` | `local` | Загружается, но сейчас не меняет поведение. |
| `GRPC_TLS_MODE` | пусто, то есть `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | Сертификат сервера; с ключом обязателен для `tls`/`mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ; хранить как секрет вне репозитория. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle; обязателен для `mtls`. |
| `GRPC_TLS_SERVER_NAME` | пусто | Настройка исходящих клиентов; у сервиса их сейчас нет. |

## Docker и Compose

`Dockerfile` ожидает build context `backend/`, собирает `./cmd/matching` с `CGO_ENABLED=0` и запускает бинарник от UID `10001` в Alpine 3.22.

```bash
# из корня репозитория
docker build -f backend/services/matching-service/Dockerfile -t order-fill-matching backend
docker run --rm -p 9097:9097 -p 8088:8088 order-fill-matching
```

Compose-конфигурация: [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml).

```bash
docker compose -f deploy/docker-compose.yml up --build matching-service
```

В compose сервис доступен другим контейнерам как `matching-service:9097`; порты на хост не публикуются.

## Тесты

```bash
# из каталога сервиса
go test ./...

# все Go-модули, из backend/
make test
```

Тесты покрывают нормализацию, точные/alias/name matches, конфликты объема и формы, выбор дубликатов, smart-режим, объединение ЧЗ, gRPC mapping, defaults конфигурации и health handler.

## Эксплуатационные заметки и ограничения

- `/healthz` и `/readyz` всегда возвращают `200`; зависимостей для проверки сейчас нет.
- Name fallback рассматривает только source-строки без артикула. Если у source есть другой артикул, одного похожего названия недостаточно.
- Для fallback минимальный score равен `0.72`; близкие кандидаты с разницей менее `0.08` отклоняются. Конфликтная строка по найденному артикулу помечается при score ниже `0.32`; smart-дубликаты требуют score не ниже `0.85` и отрыв не менее `0.10`.
- Слияние ЧЗ требует похожесть имени не ниже `0.9` и отсутствие конфликта объема/формы; в smart-режиме используются дополнительные пороги неоднозначности.
- Списки `prefix_aliases` должны быть переданы в уже ожидаемом регистре: отдельной нормализации префикса сервис не выполняет.
- Дубликаты без ID дедуплицируются по паре `article:name`; стабильные уникальные ID улучшают объяснимость результата.
- Логи — JSON в stdout; graceful shutdown по `SIGINT`/`SIGTERM` занимает до 10 секунд.
- Server reflection и OpenAPI отсутствуют; protobuf — источник истины.
