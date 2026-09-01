# Auth, tenancy, job history Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Use @superpowers:test-driven-development — no production code before a failing test.

**Goal:** Закрыть публичный API сессиями, изолировать джобы по компаниям и ролям, дать ленту выгрузок с повторной правкой отчёта; криптография, CSRF, rate limit и 404-изоляция входят в тот же релиз.

**Architecture:** Пользователи, сессии, invite и аудит живут в `api-service` (домен `identity` + `authz`, Postgres, HTTP middleware). `document-service` не меняет Excel-правила. Фронт ходит на тот же origin через Vite/nginx proxy, cookie `HttpOnly`, без токена в Web Storage. Спека: `docs/plans/2026-09-01-auth-tenancy-design.md`.

**Tech Stack:** Go 1.25, `net/http`, `golang.org/x/crypto/argon2`, `crypto/rand`, `crypto/subtle`, pgx, Vite/React, OpenAPI в `packages/contracts/openapi.yaml`.

---

## Норматив безопасности (не пропускать)

- Пароль: argon2id, соль из `crypto/rand`. Сессия и invite: 32 байта `crypto/rand`, в БД только SHA-256.
- Логин: одинаковый 401; dummy-hash если пользователя нет. Не логировать пароль/cookie/сырой токен.
- Cookie `order_fill_session`: HttpOnly, SameSite=Lax, Path=/; Secure только при `SESSION_COOKIE_SECURE=true`.
- POST: `X-Requested-With: fetch` + Origin из allowlist. Чужая джоба → 404.
- `/healthz` открыт. `/metrics` — `platform_admin`.
- SQL с `$1`. `VITE_*` без секретов.

Перед каждым коммитом из корня: `make verify` (как в CLAUDE.md), если менялся затронутый модуль — хотя бы `gofmt` + тесты пакета. Не `--no-verify`.

---

### Task 1: Домен identity — роли, пароль, токены

**Files:**
- Create: `services/api-service/internal/domain/identity/identity.go`
- Create: `services/api-service/internal/domain/identity/password.go`
- Create: `services/api-service/internal/domain/identity/token.go`
- Test: `services/api-service/internal/domain/identity/password_test.go`
- Test: `services/api-service/internal/domain/identity/token_test.go`

**Step 1: Write the failing test**

```go
func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.fatal(err)
	}
	if err := VerifyPassword(hash, "correct-horse"); err != nil {
		t.Fatalf("expected match: %v", err)
	}
	if err := VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestRejectShortPassword(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomTokenNotStoredInPlain(t *testing.T) {
	raw, err := NewSecret()
	if err != nil {
		t.fatal(err)
	}
	if len(raw) < 32 {
		t.Fatalf("too short: %d", len(raw))
	}
	sum := HashSecret(raw)
	if sum == raw {
		t.Fatal("must not store raw token")
	}
	if !SecretEqual(sum, HashSecret(raw)) {
		t.Fatal("expected equal hashes")
	}
}
```

Исправить `t.fatal` → `t.Fatal` в реальном файле.

**Step 2: Run test to verify it fails**

```bash
cd services/api-service && go test ./internal/domain/identity/ -count=1
```

Expected: FAIL, пакет не существует.

**Step 3: Write minimal implementation**

- `Role`: `platform_admin`, `company_admin`, `purchaser`.
- `User`: ID, CompanyID `*string`, Login, PasswordHash, Role, DisabledAt `*time.Time`.
- `Company`: ID, Name, DisabledAt.
- argon2id: time=1, memory=64MiB, threads=4, key=32, salt=16; хранить PHC-строку `$argon2id$v=19$m=65536,t=1,p=4$...`.
- `NewSecret()`: 32 байта, encode base64.RawURLEncoding.
- `HashSecret`: sha256 hex.
- `SecretEqual`: `subtle.ConstantTimeCompare`.
- Dummy hash: `MustDummyPasswordHash()` один раз в `sync.Once` для login.

**Step 4: Run tests**

```bash
cd services/api-service && go test ./internal/domain/identity/ -count=1
```

Expected: PASS. `gofmt -w` на новых файлах.

**Step 5: Commit**

```bash
git add services/api-service/internal/domain/identity
git commit -m "$(cat <<'EOF'
feat: add identity password and secret hashing

EOF
)"
```

---

### Task 2: Авторизация джобы — 404, не 403

**Files:**
- Create: `services/api-service/internal/domain/authz/job.go`
- Test: `services/api-service/internal/domain/authz/job_test.go`
- Modify: `services/api-service/internal/domain/job/job.go` — поля `CompanyID string`, `CreatedBy string`

**Step 1: Failing tests**

Таблица:

| actor | job | allowed |
|---|---|---|
| purchaser A | created_by A, same company | yes |
| purchaser A | created_by B, same company | no |
| company_admin | any job same company | yes |
| company_admin | other company | no |
| platform_admin | any | yes |
| any | job.CompanyID empty (legacy) | only platform_admin |

`CanAccessJob(actor identity.User, j job.Job) bool`

**Step 2–4:** реализовать только булев предикат. Маппинг в HTTP 404 — в Task 6.

**Step 5: Commit** `feat: authorize job access by company and role`

---

### Task 3: Postgres — users, companies, sessions, invites, job columns

**Files:**
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go` — `Migrate` statements
- Create: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Test: `services/api-service/internal/adapter/outbound/postgres/identity_test.go` — если нет testcontainers, тестировать SQL-сборку через репозиторий с интеграцией **только если** в модуле уже есть postgres-тесты. Сейчас их нет (только dto). Тогда: узкий unit на scan/DTO + migrate statements как строки, проверяемые тестом `TestMigrateStatementsIncludeUsers`.

Не поднимать Docker в unit-тесте, если репозиторий этого ещё не делает.

Добавить в `Migrate` (плейсхолдеры в runtime queries, не в DDL):

```sql
CREATE TABLE IF NOT EXISTS companies (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  disabled_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  company_id TEXT REFERENCES companies(id),
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  disabled_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS invite_tokens (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY,
  at TIMESTAMPTZ NOT NULL,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  company_id TEXT,
  job_id TEXT
);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS company_id TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS created_by TEXT;
```

Порт `port.IdentityStore` в `services/api-service/internal/app/port/identity.go`: CreateCompany, CreateUser, GetUserByLogin, GetUserByID, SetPasswordHash, DisableUser, CreateSession, GetSessionUser(hash), DeleteSessionsForUser, CreateInvite, ConsumeInvite (atomic: delete returning user_id), InsertAudit.

**Commit:** `feat: persist users, sessions, and job ownership columns`

---

### Task 4: Use cases — login, invite, logout, bootstrap

**Files:**
- Create: `services/api-service/internal/app/usecase/auth.go`
- Test: `services/api-service/internal/app/usecase/auth_test.go` (in-memory fake store, как `create_job_test.go`)

Поведение:

- `Login(login, password)`: грузим user; если нет — `VerifyPassword(dummy, password)` и `ErrUnauthorized`; если disabled или company disabled — тот же `ErrUnauthorized`; успех — новая сессия 7 дней, сырой токен наружу один раз.
- `AcceptInvite(raw, password)`: hash raw, consume invite, HashPassword, SetPasswordHash, DeleteSessionsForUser, новая сессия. Повтор того же токена — `ErrUnauthorized`.
- `Logout(hash)`: удалить эту сессию (или все — проще все для user).
- `Bootstrap(ctx, login)`: если users пусто — создать platform_admin без пароля + invite 72h, вернуть raw для лога. Если users не пусто — no-op.

Тесты: wrong password; unknown login; disabled; invite once; expired invite (clock injected).

**Commit:** `feat: add login and invite use cases`

---

### Task 5: HTTP — cookie, CSRF, auth middleware

**Files:**
- Create: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Create: `services/api-service/internal/adapter/inbound/httpapi/csrf.go`
- Test: `services/api-service/internal/adapter/inbound/httpapi/auth_test.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/cors.go` — credentials: если cookie auth, `Allow-Credentials: true` и **запретить** `*` (если `*`, не ставить credentials).
- Modify: `services/api-service/cmd/api/main.go` — прокинуть auth deps
- Modify: `services/api-service/internal/platform/config/config.go` — `SESSION_COOKIE_SECURE`, `BOOTSTRAP_ADMIN_LOGIN`

Публичные без сессии: `GET /healthz`, `POST /api/v1/auth/login`, `POST /api/v1/auth/invite`.

`POST /api/v1/auth/login` JSON `{"login","password"}`, MaxBytesReader 8KiB. Set-Cookie. Не возвращать токен в JSON.

CSRF на POST кроме login/invite: `X-Requested-With` == `fetch`; если `Origin` не пустой — должен быть в allowlist.

Контекст: `identity.User` через `context.WithValue`.

Тесты httptest: login sets HttpOnly cookie; job POST without cookie 401; POST without custom header 403; healthz 200.

CORS тесты: wildcard больше не с credentials. Обновить `TestRouterAllowsBrowserOriginByDefault` — allowlist из конфига, не `*`+cookie.

**Commit:** `feat: authenticate API with session cookies`

---

### Task 6: Закрыть существующие job routes

**Files:**
- Modify: `services/api-service/internal/adapter/inbound/httpapi/handlers.go`
- Modify: `services/api-service/internal/app/usecase/create_job.go` — `CreatedBy`, `CompanyID` в `CreateJobCommand` и `job.New`
- Modify: `services/api-service/internal/adapter/outbound/postgres/repository.go` — INSERT/SELECT новых колонок
- Test: `services/api-service/internal/adapter/inbound/httpapi/authz_test.go` — два fake user, одна джоба; purchaser B получает 404 на get/report/edits/files/preview/download

Helper `loadAccessibleJob(r) (job.Job, bool)` — not found и !CanAccessJob одинаково 404.

Create: purchaser/company_admin берут `company_id` из сессии. Platform_admin — поле формы/JSON `company_id`, иначе 400.

**Commit:** `fix: deny cross-tenant job access with 404`

---

### Task 7: Лента джоб `GET /api/v1/jobs`

**Files:**
- Modify: port + postgres `ListJobs(ctx, filter)` filter: companyID, createdBy (optional), limit
- Create usecase `ListJobs`
- Router GET
- Presenter: id, type, status, brand, order_month, created_at, created_by_login, company_id, progress
- Test: purchaser видит только свои; admin фирмы — все в фирме

**Commit:** `feat: list jobs for export history`

---

### Task 8: Компании, пользователи, invite reset

**Files:** HTTP + usecase + tests

- Platform: `POST /api/v1/companies` `{name}` → company + invite для первого admin (login в теле) **или** два шага: создать компанию, затем `POST /api/v1/companies/{id}/users` `{login, role}`. Предпочтительно два шага. Ответ создания пользователя: `{user, invite_url}` один раз (сырой токен только здесь).
- `POST /api/v1/users/{id}/reset` — новая invite, DeleteSessionsForUser, password_hash сбросить в ``.
- `POST /api/v1/users/{id}/disable`, `POST /api/v1/companies/{id}/disable`
- company_admin может создавать только `purchaser` и `company_admin` в **своей** company_id. Не может создавать platform_admin.
- GET lists.

Тесты: purchaser 404 на create company; company_admin не создаёт юзера другой фирмы.

**Commit:** `feat: manage companies and invite users`

---

### Task 9: Аудит

**Files:** usecase hook + postgres

Писать `audit_events` когда:

- platform_admin открывает джобу/компанию не «свою» (у него нет company);
- скачивание file или archive.

Не писать тело файла. GET `/api/v1/audit` только platform_admin (простая лента, limit 100) — YAGNI UI можно отложить, API пусть будет.

Тест: purchaser download не пишет `platform_view`; platform getJob пишет `job_view`.

**Commit:** `feat: audit platform access and downloads`

---

### Task 10: Rate limit, timeouts, headers, metrics

**Files:**
- Create: `services/api-service/internal/adapter/inbound/httpapi/ratelimit.go`
- Test: `.../ratelimit_test.go` — 6 логинов подряд с одного ключа → 429
- Modify: `cmd/api/main.go` Server:

```go
ReadHeaderTimeout: 10 * time.Second,
ReadTimeout:       60 * time.Second,
WriteTimeout:      120 * time.Second,
IdleTimeout:       90 * time.Second,
MaxHeaderBytes:    1 << 20,
```

WriteTimeout 120s из‑за скачивания zip; при необходимости позже вынести.

- Headers middleware: nosniff, DENY frame, Referrer-Policy no-referrer.
- `/metrics` требует platform_admin.
- Login limiter: ключ `ip + "\x00" + login`, 10 / 15 min.
- Create job: 30 / hour per user id.

**Commit:** `fix: rate-limit auth and harden HTTP server`

---

### Task 11: OpenAPI

**Files:** `packages/contracts/openapi.yaml`

- `securitySchemes.sessionCookie` cookie `order_fill_session`
- global `security: [{sessionCookie: []}]`
- `/healthz`, login, invite: `security: []`
- 401 responses
- новые paths: auth, jobs list, companies, users
- Job schema: `company_id`, `created_by`

**Commit:** `docs: describe session auth in OpenAPI`

---

### Task 12: Same-origin proxy

**Files:**
- Modify: `frontend/nginx.conf` — `location /api/` proxy_pass `http://api-service:8080`; пробросить Cookie, Host; не буферизовать большие download без нужды (`proxy_buffering off` на archive если надо)
- Modify: `frontend/vite.config.js` — `server.proxy['/api']` и healthz на `http://127.0.0.1:8080`
- Modify: `frontend/src/api/client.js` — `credentials: 'include'`; на POST JSON/FormData заголовок `X-Requested-With: fetch`; `apiBaseUrl()` по умолчанию `''` (same origin). Убрать дефолт `http://127.0.0.1:8080` чтобы cookie не уезжал на другой origin.
- Modify: `frontend/src/api/client.test.js`
- Modify: `frontend/Dockerfile` — `ARG VITE_API_BASE_URL=` пусто
- Modify: `deploy/docker-compose.yml` — не передавать публичный API URL в Vite; frontend зависит от api-service по сети compose (`proxy_pass http://api-service:8080`)
- CORS allowlist: `http://127.0.0.1:3200` для случая без proxy; compose `API_ALLOWED_ORIGINS=http://127.0.0.1:3200`

**Commit:** `fix: serve API on the frontend origin`

---

### Task 13: Frontend — сессия, логин, invite

**Files:**
- Create: `frontend/src/api/auth.js` + `auth.test.js`
- Create: `frontend/src/ui/auth/LoginScreen.jsx`, `InviteScreen.jsx`
- Modify: `frontend/src/App.jsx` — `/invite/:token` по `window.location.pathname` (роутера нет: простой parse). Без `me` — LoginScreen. 401 на apiClient — сброс me и логин.
- Никакого `localStorage.setItem('token')`.

Тесты: client шлёт credentials и CSRF header (mock fetch).

**Commit:** `feat: add login and invite screens`

---

### Task 14: Frontend — лента, админ компаний/юзеров

**Files:**
- `frontend/src/ui/jobs/JobHistory.jsx` — список, клик открывает существующий OrderFill/North по type+id (переиспользовать workflow, не дублировать Excel-логику)
- `frontend/src/ui/admin/CompaniesScreen.jsx`, `UsersScreen.jsx` — по роли из `me`
- chrome: выход logout POST

Закупщик не видит экраны админки (только UX). API всё равно 404.

**Commit:** `feat: show export history and admin consoles`

---

### Task 15: Compose, bootstrap, порты инфры

**Files:**
- Modify: `deploy/docker-compose.yml` — postgres/redis/minio `ports: ["127.0.0.1:5432:5432"]` и аналогично 6379, 9000, 9001. API 8080 можно оставить на 127.0.0.1. Frontend 3200.
- `.env.example`: `BOOTSTRAP_ADMIN_LOGIN=admin`, `SESSION_COOKIE_SECURE=false`, `API_ALLOWED_ORIGINS=http://127.0.0.1:3200`
- `cmd/api/main.go`: после migrate вызвать Bootstrap, залогировать invite **без** повторного логирования на каждый рестарт (только если создали).

**Commit:** `fix: bind datastore ports to localhost and bootstrap admin`

---

### Task 16: Сквозная проверка

**Step 1:** `make verify` из корня.

Expected: toolchain, frontend verify, оба Go модуля зелёные.

**Step 2:** Ручной чеклист (когда stack поднят):

- Без cookie `POST /api/v1/jobs/order-fill` → 401
- `/healthz` → 200
- Invite bootstrap из лога → пароль → лента
- Второй закупщик не открывает `job_id` первого (curl cookie)
- Повторная правка completed джобы и скачивание

**Step 3:** Если verify падает — чинить, новый коммит, не `--no-verify`.

---

## Порядок и зависимости

1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10. 11 можно параллельно с 8. 12 до 13 (иначе cookie не прилипнет). 14 после 13. 15 в любой момент после 4. 16 в конце.

`document-service` не трогать, кроме случая если INSERT jobs сломает worker SELECT — тогда добавить чтение новых колонок как игнор.

## Вне скоупа (не делать в этих задачах)

SSO, JWT, 2FA, SMTP, версии xlsx, биллинг, мульти-компанейский пользователь, кастомные роли.
