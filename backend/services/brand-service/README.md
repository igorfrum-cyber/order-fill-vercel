# brand-service

Внутренний stateless gRPC-сервис, который хранит правила обработки поддерживаемых брендов и определяет бренд по группе номенклатуры из выгрузки 1С. Сервис не читает Excel-файлы, не рассчитывает заказ и не хранит данные во внешней БД.

## Быстрый старт

Требуется Go `1.26.7` (версия зафиксирована в `go.mod`). Из каталога сервиса:

```bash
go run ./cmd/brand
```

По умолчанию gRPC слушает `:9098`, а HTTP-проверки состояния — `:8089`.

```bash
curl http://127.0.0.1:8089/healthz
curl http://127.0.0.1:8089/readyz
```

Оба запроса сейчас возвращают `200` и `{"status":"ok"}`.

## Возможности и границы ответственности

- Возвращает правила бренда: тип корректировки количества, кратность, подписи колонок, алиасы префиксов артикула и особенности бланка.
- Перечисляет поддерживаемые ключи брендов.
- Определяет бренд по русскому или латинскому названию группы номенклатуры; для `christina` дополнительно определяет вариант `HOME` или `PROFF` по имени файла.
- Хранит правила статически в коде; миграций, БД и внешнего кеша нет.
- Не проверяет корректность содержимого книги и не выбирает группу номенклатуры из Excel — это делает `document-service`.

Поддерживаются ключи `angiopharm`, `christina`, `klapp`, `levissime`, `novacutan`, `skin_synergy`, `sothys`.

| Бренд | Корректировка | Подтвержденные особенности |
| --- | --- | --- |
| `angiopharm` | `box` | округление по размеру коробки |
| `christina` | `multiple`, кратность 3 | варианты `HOME`/`PROFF` |
| `klapp` | `nearestMultiple`, кратность 3 | ближайшее кратное; колонка количества `order` |
| `levissime` | `box` | алиас префикса `MT`; колонки `order` и `packageQuantity` |
| `novacutan` | `none` | layout `novacutan`, без округления |
| `skin_synergy` | `none` | колонка `exactQuantity`; единица измерения не обязательна |
| `sothys` | `none` | сохраняется дефис артикула; layout `splitVariants` |

## Архитектура

```text
cmd/brand                         точка входа и graceful shutdown
internal/config                   переменные окружения и значения по умолчанию
internal/bootstrap                запуск gRPC и HTTP health-сервера
internal/transport/grpcapi        преобразование protobuf <-> domain
internal/service/brands           определение бренда и выдача правил
internal/storage/static           статический каталог политик
internal/domain                   модель политики бренда
```

Зависимости workspace-модулей `../../pkg` и `../../proto` подключены через `replace` в `go.mod`. Общий `grpcutil` добавляет `x-request-id`, ограничивает сообщение размером 64 MiB и выполняет graceful shutdown при `SIGINT`/`SIGTERM`.

## gRPC API

Сервис реализует `orderfill.brand.v1.BrandService`. Полный контракт находится в [`../../proto/orderfill/brand/v1/brand.proto`](../../proto/orderfill/brand/v1/brand.proto).

| RPC | Полный путь | Назначение |
| --- | --- | --- |
| `GetBrandPolicy` | `/orderfill.brand.v1.BrandService/GetBrandPolicy` | Возвращает `BrandPolicy` по полям `brand` и `variant`. Переданный вариант копируется в ответ. |
| `ListBrands` | `/orderfill.brand.v1.BrandService/ListBrands` | Возвращает упорядоченный список поддерживаемых ключей. |
| `DetectBrand` | `/orderfill.brand.v1.BrandService/DetectBrand` | Определяет ключ по `nomenclature_group`; для Christina анализирует `file_name`. При неизвестном бренде возвращает пустые `brand` и `variant` без gRPC-ошибки. |

REST/HTTP бизнес-API у сервиса нет. HTTP используется только для health endpoints.

Важно: прямой вызов `GetBrandPolicy` с неизвестным ключом сейчас возвращает политику `angiopharm`; он не возвращает `NotFound`. Вызывающая сторона должна передавать ключ из `ListBrands` или результат успешного `DetectBrand`.

## Взаимодействия

`document-api` вызывает `DetectBrand` при предварительном анализе входных файлов. `document-worker` вызывает `DetectBrand` и `GetBrandPolicy`, затем передает полученные правила в обработку книги и в `calculation-service`/`matching-service`. У самого `brand-service` нет исходящих сетевых зависимостей.

## Конфигурация

| Переменная | По умолчанию | Обязательность и смысл |
| --- | --- | --- |
| `BRAND_GRPC_ADDR` | `:9098` | Адрес gRPC listener. |
| `BRAND_HEALTH_ADDR` | `:8089` | Адрес HTTP listener для `/healthz` и `/readyz`. |
| `BRAND_ENV` | `local` | Загружается в конфигурацию, но сейчас не изменяет поведение сервиса. |
| `GRPC_TLS_MODE` | пусто, то есть `insecure` | Общий режим gRPC: `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | PEM-сертификат сервера; вместе с ключом обязателен для `tls` и `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | PEM-ключ сервера; хранить как секрет и не добавлять в репозиторий. |
| `GRPC_TLS_CA_FILE` | пусто | CA bundle; обязателен для `mtls`. |
| `GRPC_TLS_SERVER_NAME` | пусто | Используется только исходящими gRPC-клиентами; у этого сервиса их сейчас нет. |

При неизвестном значении `GRPC_TLS_MODE` или неполном TLS-наборе общий gRPC bootstrap завершает процесс при создании сервера. Health HTTP не защищен gRPC TLS.

## Docker и Compose

`Dockerfile` ожидает build context `backend/`, собирает `./cmd/brand` с `CGO_ENABLED=0`, затем запускает бинарник от пользователя UID `10001` в Alpine 3.22. Например, из корня репозитория:

```bash
docker build -f backend/services/brand-service/Dockerfile -t order-fill-brand backend
docker run --rm -p 9098:9098 -p 8089:8089 order-fill-brand
```

Штатный compose-сервис описан в [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml). Из корня репозитория весь стек запускается командой:

```bash
docker compose -f deploy/docker-compose.yml up --build brand-service
```

Compose публикует порты только внутри своей сети через `expose`; для обращения с хоста используйте явный `ports` override или локальный запуск бинарника.

## Тесты

Из каталога сервиса:

```bash
go test ./...
```

Из каталога `backend/` можно проверить все Go-модули:

```bash
make test
```

Тесты покрывают каталог политик, распознавание русских/латинских названий, варианты Christina, gRPC mapping, defaults конфигурации и health handler.

## Эксплуатационные заметки и ограничения

- `/healthz` и `/readyz` не проверяют внешние зависимости; для stateless-сервиса обе проверки эквивалентны.
- Логи пишутся в stdout в JSON на уровне `info`.
- Сервер завершает работу по `SIGINT`/`SIGTERM`; graceful timeout gRPC/HTTP — 10 секунд.
- Политики изменяются только выпуском новой версии сервиса.
- В проекте нет server reflection и отдельной OpenAPI-спецификации для этого сервиса; источником истины служит protobuf-контракт.
