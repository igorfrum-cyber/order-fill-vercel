# Участие в разработке Order Fill

Этот документ описывает проверяемый рабочий процесс для текущего backend v2 и
React frontend.

## Подготовка окружения

Понадобятся:

- Git;
- Docker с `docker compose`;
- Node.js 24+ и npm;
- Go 1.26.7;
- Buf 1.x;
- Bash и Make.

Полный lint дополнительно использует `golangci-lint` 2.12.2 и `gosec` 2.22.10.
Если утилит нет в `PATH`, `scripts/verify-go.sh` устанавливает закреплённые
версии. Для этого нужен доступ к сети и настроенный Go toolchain.

Buf и сканер уязвимостей можно установить так:

```bash
brew install bufbuild/buf/buf
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
```

Для Linux используйте официальный способ установки Buf, сохранив major version
1; `govulncheck` устанавливается той же Go-командой.

```bash
git clone --recurse-submodules https://github.com/igorfrum-cyber/order-fill-vercel.git
cd order-fill-vercel
npm ci --prefix frontend
cp .env.example .env
make verify
```

## Локальный запуск

```bash
make up
```

Web UI будет доступен на <http://127.0.0.1:3200>, gateway health endpoint — на
<http://127.0.0.1:8080/healthz>. Для просмотра логов используйте `make logs`,
для остановки — `make down`.

## Ветки

Канон разработчика — `artemch`. Задачи делаются в `feat/<slug>` от неё, затем
сливаются в `artemch` по явной просьбе. `dev` — стенд. `main` — песочница
прототипа (другое дерево, Excel в браузере); её не мержат в `artemch`, с неё
снимают поведение и пишут его в текущие сервисы и экраны.

Новый микросервис или новый экран — только после явного согласия: сначала
почему не хватает текущего owner / текущего сценария.

Агент обязан читать skill `.cursor/skills/order-fill-work/SKILL.md`, README
затронутого сервиса и skill документации. Таблицы env/RPC/HTTP в README
регенерируются из кода (`make docs-sync`); хук и pre-commit делают тот же
sync, назначение в таблицах всё равно заполняют руками.

## Где вносить изменения

- Browser UI и API adapters: `frontend/`.
- Публичный HTTP-контракт и его реализация: `backend/services/gateway-service/`.
- Межсервисные protobuf-контракты: `backend/proto/`.
- Бизнес-логика конкретного сервиса: `backend/services/<service>/`.
- Общий Go-код без владения бизнес-данными: `backend/pkg/`.
- Локальный runtime: `backend/deploy/docker-compose.yml`.
- Архитектурные документы и решения: `docs/`.

Сверяйтесь с [`docs/service-boundaries.md`](./docs/service-boundaries.md):
gateway не должен обрабатывать Excel или обращаться к чужому storage напрямую,
а внутренние сервисы не должны становиться browser-facing API.

## Контракты

Канонический публичный OpenAPI расположен в
`backend/services/gateway-service/api/openapi.yaml`. После изменения HTTP API
обновите спецификацию и тесты gateway в том же pull request.

Исходные `.proto` лежат в `backend/proto/orderfill/`. После изменения protobuf:

```bash
make -C backend proto-gen
```

Закоммитьте соответствующие изменения в `backend/proto/gen/go/` и проверьте все
затронутые producer/consumer сервисы.

## Тесты

Быстрые целевые команды:

```bash
npm run test --prefix frontend
npm run test:ui --prefix frontend
npm run test:component --prefix frontend
npm run test:e2e --prefix frontend
(cd backend/services/<service> && GOWORK=off go test ./...)
```

После фичи в UI агент в том же чате запускает `npm run test:ui --prefix frontend`
и смотрит вывод: это unit + Testing Library + Playwright. Локально Playwright
открывает Chromium на экране. Если тесты красные, агент чинит причину и гоняет
suite снова, пока не станет зелёным. Unit-тесты frontend остаются на `node:test`.
Поведение компонентов (`*.ui.test.jsx`) гоняет Vitest + Testing Library, UI в
браузере — Playwright.
Перед первым `test:e2e` один раз установите Chromium:

```bash
npm run test:e2e:install --prefix frontend
```

Включите git hooks один раз в клоне:

```bash
make hooks
```

На любой ветке commit прогоняет `node scripts/sync-docs.mjs --write` и
добавляет обновлённые README сервисов. На ветке `artemch` commit ещё и
запускает `make verify` и отменяется, если gate красный. Перед любым
`git push` тот же полный gate запускается ещё раз. Перед pull request всё
равно запускайте полный gate явно:

```bash
make verify
```

`make verify` запускает детерминированный локальный pre-commit gate. GitHub
Actions использует те же component scripts, а также добавляет race detection,
перемешивание порядка Go-тестов, `govulncheck` и protobuf breaking check.

Перед изменениями зависимостей, аутентификации, авторизации, сетевого слоя или
обработки файлов дополнительно запустите:

```bash
make security
```

Команда требует установленного `govulncheck` и доступа к Go vulnerability DB.

Не запускайте `go mod tidy` сразу из корня: Go workspace находится в `backend/`,
а сервисы являются отдельными модулями. Используйте `make -C backend tidy` или
запустите `go mod tidy` в нужном модуле.

Тестовые workbook-файлы и правила работы с приватными fixtures описаны в
[`testdata/README.md`](./testdata/README.md). Каталог `testdata/private/`
исключён из Git; не добавляйте в репозиторий реальные коммерческие данные.

## Документация

Таблицы env, RPC и список HTTP-маршрутов gateway обновляются из кода:

```bash
make docs-sync
```

`make verify` делает то же самое и падает, если сгенерированные списки не
попали в коммит. Назначение RPC и смысл переменных в таблицах правьте руками —
повторный sync их сохраняет.

При изменении поведения обновите документацию в том же pull request:

- README затронутого микросервиса;
- OpenAPI или protobuf-контракт;
- `.env.example`, если появилась новая пользовательская переменная;
- `docs/ARCHITECTURE.md` и `docs/service-boundaries.md`, если изменились связи
  или владение данными;
- корневой README, если изменились запуск, состав системы или deployment.

Файлы в `docs/plans/` фиксируют дизайн и историю реализации. Они не заменяют
документацию текущего runtime.

## Checklist перед pull request

- Изменение находится в сервисе, который владеет этой ответственностью.
- Работа в `feat/<slug>`, не в `artemch`/`dev`/`main`; слив в `artemch` по явной просьбе.
- Новое поведение покрыто тестами или причина отсутствия теста объяснена.
- `make hooks` включён в клоне; commit на любой ветке синхронизирует README-таблицы, на `artemch` ещё и не обходит `make verify`, а любой push гоняет полный gate.
- `make verify` проходит локально.
- Для security-sensitive изменений проходит `make security`.
- Изменения `go.mod`, `go.sum` и lock-файлов ожидаемы.
- Изменённые API и env vars отражены в документации.
- В diff нет `.env`, секретов и приватных workbook-файлов.
- Миграции добавлены новым файлом; уже применённые миграции не переписаны.

## Коммиты

Делайте небольшие тематические коммиты. Сообщение должно описывать изменение
поведения. Не смешивайте функциональную правку с несвязанным форматированием.
