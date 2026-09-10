# passkey-service

`passkey-service` — внутренний gRPC-сервис WebAuthn/passkey. Он проводит регистрацию и аутентификацию credentials, хранит публичные credential-данные и счетчики подписей и держит короткоживущие одноразовые WebAuthn ceremony sessions.

Приватные ключи остаются в аутентификаторе пользователя и сервисом не принимаются. Выпуск cookie-сессии и проверка активности пользователя находятся в `identity-service`; passkey-service после успешного assertion возвращает только `user_id`.

## Быстрый старт

Требуется Go `1.26.7`. Для локального запуска без внешней инфраструктуры:

```bash
cd backend/services/passkey-service
go run ./cmd/passkey
```

По умолчанию gRPC слушает `:9093`, health HTTP — `:8084`. При пустых `DATABASE_URL` и Redis URL credentials и ceremony sessions хранятся в памяти процесса.

```bash
curl http://127.0.0.1:8084/healthz
curl http://127.0.0.1:8084/readyz
```

Для выполнения реальной WebAuthn ceremony нужен клиент, формирующий браузерный credential JSON и передающий точный `Origin`; публичный HTTP adapter находится в `gateway-service`.

## Возможности и границы ответственности

- Начало и завершение регистрации passkey.
- Начало и завершение обычного или discoverable login.
- Список безопасного публичного представления credentials и удаление credential только владельцем по `user_id`.
- Хранение credential ID, public key, signature counter, transports, AAGUID, library JSON, времени создания и последнего использования.
- Однократное потребление WebAuthn challenge с TTL 2 минуты.
- Проверка origin и вычисление/валидация WebAuthn Relying Party ID.
- Обновление signature counter и `last_used_at` после успешного входа.
- Отказ от сохранения credential JSON, если в нем обнаружены поля, похожие на private key или secret.

Сервис не выпускает сессии, не хранит логины/пароли и не проверяет права вызывающего по сессии. Авторизацию actor ID обеспечивает доверенный внутренний клиент — gateway/identity — и сетевой периметр.

## Архитектура и данные

| Путь | Назначение |
| --- | --- |
| `cmd/passkey` | Точка входа и graceful shutdown по `SIGINT`/`SIGTERM`. |
| `internal/config` | Runtime-конфигурация и production-валидация. |
| `internal/bootstrap` | Выбор store, миграции, Redis ceremony store и запуск серверов. |
| `internal/service/passkey` | Оркестрация begin/finish, TTL и credential lifecycle. |
| `internal/webauthn` | Интеграция `go-webauthn`, origin/RP ID policy и преобразование credentials. |
| `internal/ceremony` | Redis или in-memory одноразовые challenge/session records. |
| `internal/storage/memory` | Непостоянный credential store для local/tests. |
| `internal/storage/postgres` | PostgreSQL credential store. |
| `internal/migrate` | Embedded SQL, выполняемый при старте. |
| `internal/transport/grpcapi` | Реализация protobuf RPC. |

При заданном `DATABASE_URL` сервис проверяет БД, выполняет embedded SQL и использует таблицу `passkey_credentials`. В ней нет приватных ключей: сохраняются публичная credential-информация и serialized state библиотеки. Redis хранит JSON ceremony session под ключом `passkey:ceremony:<challenge_id>` с expiry.

## Внутренний gRPC API

Контракт: [`../../proto/orderfill/passkey/v1/passkey.proto`](../../proto/orderfill/passkey/v1/passkey.proto), package `orderfill.passkey.v1`, service `PasskeyService`.

- `BeginRegistration` — принимает `actor_user_id` и `origin`, возвращает `challenge_id` и WebAuthn `options_json`.
- `FinishRegistration` — однократно потребляет registration challenge, проверяет origin/response и сохраняет credential.
- `ListCredentials` — возвращает только ID, отображаемое имя, `created_at` и `last_used_at`.
- `DeleteCredential` — удаляет credential при совпадении `actor_user_id` и владельца.
- `BeginLogin` — возвращает assertion options и challenge.
- `FinishLogin` — проверяет assertion, обновляет credential и возвращает подтвержденный `user_id`.

Текущая bootstrap-сборка создает сервис без user directory. Поэтому переданный в `BeginLogin.login` логин сейчас не используется для поиска пользователя: всегда запускается discoverable login, а credential/user определяется при `FinishLogin`.

Максимальный размер gRPC-сообщения — 64 MiB. Request ID поддерживается через metadata `x-request-id`.

## WebAuthn origin и RP ID

- `origin` обязан быть валидным `http://` или `https://` URL с host.
- HTTP разрешен только для loopback (`localhost`, `*.localhost`, `127.0.0.1`, `::1`); для остальных хостов нужен HTTPS.
- Не-loopback IP запрещен как WebAuthn host: для LAN/production нужен домен.
- При пустом `WEBAUTHN_RP_ID` RP ID равен hostname origin.
- При заданном RP ID origin должен совпасть с ним или быть его непосредственным/вложенным поддоменом по suffix-проверке.
- Для public-suffix-подобных local значений `localhost` и `local` на поддомене используется полный origin host, а не общий RP ID.

Точный `Origin` должен одинаково передаваться на begin и finish. Смена RP ID способна сделать уже зарегистрированные credentials непригодными для входа.

## Зависимости

| Зависимость | Использование | Local fallback |
| --- | --- | --- |
| PostgreSQL | Постоянные credentials, signature counters, timestamps | In-memory map при пустом `DATABASE_URL`. |
| Redis | Ceremony session/challenge с атомарным `GETDEL` | Process-local map при пустых `QUEUE_URL`/`REDIS_URL`. |
| `go-webauthn/webauthn` | Создание options и проверка attestation/assertion | Нет. |
| `gateway-service` | Публичные HTTP endpoints регистрации и login begin | Не является runtime-зависимостью самого процесса. |
| `identity-service` | Вызывает `FinishLogin` и выпускает сессию | Не является исходящим клиентом passkey-service. |

## Конфигурация

| Переменная | Default | Обязательность и назначение |
| --- | --- | --- |
| `PASSKEY_ENV` | значение `APP_ENV`, затем `local` | Среда сервиса; имеет приоритет над `APP_ENV`. |
| `APP_ENV` | `local` | Общая среда. Пустая строка и `local` включают local-режим. |
| `PASSKEY_GRPC_ADDR` | `:9093` | gRPC listener. |
| `PASSKEY_HEALTH_ADDR` | `:8084` | HTTP listener liveness/readiness. |
| `DATABASE_URL` | пусто | PostgreSQL DSN; обязателен вне local. Может содержать пароль — хранить как secret. |
| `QUEUE_URL` | значение `REDIS_URL`, затем пусто | Redis URL для ceremonies; имеет приоритет над `REDIS_URL`, обязателен вне local совместно с fallback-правилом. Может содержать пароль. |
| `REDIS_URL` | пусто | Fallback Redis URL, если `QUEUE_URL` не задан. |
| `WEBAUTHN_RP_ID` | пусто | Явный RP ID; обязателен вне local. Не включает scheme или port. |
| `WEBAUTHN_RP_DISPLAY_NAME` | `Order Fill` | Имя relying party в WebAuthn UI. |

Конфигурация входящего gRPC TLS:

| Переменная | Default | Назначение |
| --- | --- | --- |
| `GRPC_TLS_MODE` | `insecure` | `insecure`/`disabled`/`off`, `tls` или `mtls`. |
| `GRPC_TLS_CERT_FILE` | пусто | PEM certificate сервера; вместе с key обязателен в `tls` и `mtls`. |
| `GRPC_TLS_KEY_FILE` | пусто | Закрытый ключ сервера; обязателен в `tls`/`mtls`, хранить как secret. |
| `GRPC_TLS_CA_FILE` | пусто | CA для проверки клиентских сертификатов; обязателен в `mtls`. |

Вне local сервис не стартует без `DATABASE_URL`, Redis URL и `WEBAUTHN_RP_ID`.

## Запуск с PostgreSQL и Redis

```bash
cd backend/services/passkey-service
DATABASE_URL='postgres://order_fill:order_fill@127.0.0.1:5432/order_fill?sslmode=disable' \
QUEUE_URL='redis://127.0.0.1:6379/0' \
WEBAUTHN_RP_ID=localhost \
go run ./cmd/passkey
```

Для company-поддоменов localhost лучше оставить `WEBAUTHN_RP_ID` пустым, чтобы RP ID вычислялся как `<slug>.localhost`; это соответствует подсказке в корневом `.env.example`.

## Docker и Compose

Dockerfile требует контекст `backend/`, собирает статический бинарник и запускает Alpine 3.22 от UID `10001`:

```bash
docker build -f backend/services/passkey-service/Dockerfile -t order-fill-passkey backend
```

В [`../../deploy/docker-compose.yml`](../../deploy/docker-compose.yml) сервис доступен только внутри compose-сети на `9093`/`8084`, зависит от healthy PostgreSQL и Redis и проверяется через `/healthz`.

```bash
docker compose -f deploy/docker-compose.yml up --build passkey-service
```

Команда выполняется из корня репозитория; compose поднимает зависимости. Полный стек: `make up`. Пример переменных: [`../../../.env.example`](../../../.env.example).

## Тесты

```bash
cd backend/services/passkey-service
go test ./...
```

Тесты покрывают конфигурацию, credential safety, Redis/in-memory ceremonies, WebAuthn RP ID policy, service flows, PostgreSQL migration и store. `make test` из корня запускает все тесты репозитория.

## Эксплуатационные заметки и ограничения

- `/healthz` всегда отвечает `200`. `/readyz` проверяет только PostgreSQL, если он включен; Redis в readiness не входит.
- Ошибка Redis при сохранении ceremony игнорируется, поэтому begin RPC может вернуть challenge, который затем невозможно завершить. Мониторинг Redis должен быть отдельным.
- In-memory credentials и ceremonies теряются при рестарте и не разделяются между репликами. Без Redis несколько реплик не смогут гарантировать, что finish попадет к процессу, создавшему challenge.
- Ceremony challenge одноразовый и живет 2 минуты. Даже неуспешная finish-попытка потребляет его.
- RPC доверяют переданным `actor_user_id`/`user_id`; отдельного auth interceptor нет. Не публикуйте gRPC listener наружу.
- Смена `WEBAUTHN_RP_ID` или доменной схемы требует учета уже зарегистрированных credentials.
- При `GRPC_TLS_MODE=insecure` трафик не шифруется; production deployment должен использовать TLS/mTLS и сетевые политики.
- В репозитории нет файла `LICENSE`; условия распространения сервиса не зафиксированы.
