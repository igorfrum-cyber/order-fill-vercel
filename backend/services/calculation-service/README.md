# calculation-service

Внутренний stateless gRPC-сервис расчетов заказа. Он выполняет ABC-классификацию, рассчитывает рекомендуемое и скорректированное количество, строит северный план распределения и проверяет ручные правки. Сервис принимает уже структурированные строки и не читает Excel, не определяет бренд и не хранит результаты.

## Быстрый старт

Требуется Go `1.26.7`. Из каталога сервиса:

```bash
go run ./cmd/calculation
```

По умолчанию gRPC слушает `:9099`, HTTP-проверки состояния — `:8090`.

```bash
curl http://127.0.0.1:8090/healthz
curl http://127.0.0.1:8090/readyz
```

## Возможности и правила расчета

- `CalculateOrderRecommendations` ранжирует строки по выручке и назначает ABC-категории по накопленной доле: `A+` до 50%, `A` до 80%, `B` до 95%, далее `C`.
- Рекомендуемое количество учитывает продажи по месяцам, остаток, товар в пути, категорийный коэффициент и срок поставки. Если `delivery_weeks <= 0`, используется 4 недели; внутри расчета значение ограничивается снизу одной неделей.
- Новинки с 1–3 последовательными ненулевыми последними месяцами получают суффикс `/New` и отдельный расчет целевого остатка.
- При `city_rule = "urengoy"` используется специальный расчет по максимальным месячным продажам.
- `CalculateAdjustedQuantity` округляет исходное число математически (`floor(x+0.5)`) и применяет политику `none`, `box`, `multiple`, `nearestMultiple` или `minimum`. Явные поля политики имеют приоритет; если `adjustment` пуст, действует встроенное сопоставление по ключу бренда.
- `CalculateNorthPlan` распределяет доступный остаток Тюмени в порядке Вартовск → Уренгой → Сургут и рассчитывает заказ поставщику. Для `klapp` применяется ближайшее кратное 3, для `novacutan` — минимум 100 с округлением к десяткам.
- `ValidateManualEdits` требует непустой комментарий для ненулевого количества и, если передан список строк, блокирует неизвестные `row_id`.

## Архитектура

```text
cmd/calculation                    точка входа и обработка сигналов
internal/config                    адреса listener и окружение
internal/bootstrap                 запуск gRPC и HTTP health-сервера
internal/transport/grpcapi         protobuf/domain mapping
internal/service/calculation       рекомендации, округление, north plan, правки
internal/domain                    структуры строк заказа и северного плана
```

Сервис не имеет БД, кеша и исходящих сетевых вызовов. Workspace-модули `../../pkg` и `../../proto` подключены через `replace`. Общий gRPC-слой переносит `x-request-id`, ограничивает сообщения размером 64 MiB и выполняет graceful shutdown.

## gRPC API

Сервис реализует `orderfill.calculation.v1.CalculationService`. Поля запросов и ответов описаны в [`../../proto/orderfill/calculation/v1/calculation.proto`](../../proto/orderfill/calculation/v1/calculation.proto).


<!-- docs-sync:rpc -->
| RPC | Полный путь | Назначение |
| --- | --- | --- |
| `CalculateOrderRecommendations` | `/orderfill.calculation.v1.CalculationService/CalculateOrderRecommendations` | Рассчитывает ABC-метрики, целевой запас и `recommended_qty`; сохраняет порядок входных строк. |
| `CalculateAdjustedQuantity` | `/orderfill.calculation.v1.CalculationService/CalculateAdjustedQuantity` | Возвращает округленное число, вставляемое количество, автокомментарий и признак корректировки по коробке/кратности. |
| `CalculateNorthPlan` | `/orderfill.calculation.v1.CalculationService/CalculateNorthPlan` | Объединяет потребности городов по артикулам с остатками Тюмени и строит план перемещения/заказа. |
| `RecalculateNorthRow` | `/orderfill.calculation.v1.CalculationService/RecalculateNorthRow` | Пересчитывает одну строку, трактуя `edited_qty` как потребность Сургута. |
| `ValidateManualEdits` | `/orderfill.calculation.v1.CalculationService/ValidateManualEdits` | Возвращает `ok` и список блокирующих `row_id`. |
<!-- /docs-sync:rpc -->

REST/HTTP бизнес-API отсутствует. `RequestMeta`, где он есть в protobuf, сейчас не участвует в расчетах.

## Взаимодействия

Основной клиент — `document-worker`: он передает извлеченные из книг строки, правила бренда, потребности северных городов и получает рассчитанные значения для Excel и отчета. Правила бренда сервис напрямую не запрашивает; их получает вызывающая сторона от `brand-service` и передает в `CalculateAdjustedQuantity`.

## Конфигурация


<!-- docs-sync:env -->
| Переменная | По умолчанию | Обязательность и смысл |
| --- | --- | --- |
| `CALCULATION_GRPC_ADDR` | `:9099` | Адрес gRPC listener. |
| `CALCULATION_HEALTH_ADDR` | `:8090` | Адрес HTTP listener для `/healthz` и `/readyz`. |
| `CALCULATION_ENV` | `APP_ENV`, затем `local` | Вне local `Validate` требует `GRPC_TLS_MODE=mtls`. |
| `APP_ENV` | `local` | Общий fallback окружения; пустое значение и `local` включают local-режим. |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | Сертификат сервера; обязателен вместе с ключом для `tls` и `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ; хранить как секрет вне репозитория. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle; обязателен для `mtls`. |
| `GRPC_TLS_SERVER_NAME` | пусто | Настройка исходящих клиентов; сейчас сервис сам gRPC-вызовы не выполняет. |
<!-- /docs-sync:env -->

Health HTTP остается отдельным незашифрованным listener независимо от gRPC TLS.

## Docker и Compose

`Dockerfile` собирает `./cmd/calculation` из build context `backend/` с `CGO_ENABLED=0`. Финальный Alpine 3.22 запускает бинарник от UID `10001`.

```bash
# из корня репозитория
docker build -f backend/services/calculation-service/Dockerfile -t order-fill-calculation backend
docker run --rm -p 9099:9099 -p 8090:8090 order-fill-calculation
```

Сервис включен в [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml):

```bash
docker compose -f deploy/docker-compose.yml up --build calculation-service
```

Compose использует внутренние адреса `calculation-service:9099` и healthcheck `http://localhost:8090/healthz`; порты не публикуются на хост.

## Тесты

```bash
# из каталога сервиса
go test ./...

# все Go-модули, из backend/
make test
```

Тесты покрывают рекомендации, брендовые коэффициенты, округление, north plan, ручные правки, gRPC mapping, defaults конфигурации и health handler.

## Эксплуатационные заметки и ограничения

- `/healthz` и `/readyz` всегда возвращают готовность и не выполняют дополнительные проверки.
- Все расчеты выполняются в памяти в рамках unary RPC; долговременного состояния и фоновых задач нет.
- При одинаковых `id` строк ABC-метрики в map перезаписываются, поэтому вызывающая сторона должна передавать уникальные непустые идентификаторы.
- Порядок строк north plan формируется обходом map и не гарантирован; потребитель не должен полагаться на него.
- Значения `NaN`/`Inf` и отрицательные бизнес-поля отдельно на границе gRPC не валидируются; вызывающая сторона должна передавать конечные допустимые числа.
- Логи — JSON в stdout; завершение по `SIGINT`/`SIGTERM`, graceful timeout 10 секунд.
- Server reflection и отдельной OpenAPI-спецификации нет; protobuf — источник истины.
