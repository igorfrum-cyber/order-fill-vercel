# Ops Overview Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Give platform admins a home dashboard of dependency health, live jobs, and access audit; show company user hierarchy with last login.

**Architecture:** Keep workbook flows unchanged. Add a platform-admin status use case that probes existing adapters. Enrich audit and user list from data already in Postgres. Frontend presents copy, tiles, and a three-band org layout.

**Tech Stack:** Go api-service, React 19, existing CSS variables.

---

### Task 1: Status probes and API

**Files:**
- Modify: `services/api-service/internal/platform/config/config.go`
- Modify: `services/api-service/internal/app/port/identity.go`
- Create: `services/api-service/internal/app/usecase/status.go`
- Create: `services/api-service/internal/app/usecase/status_test.go`
- Modify: `services/api-service/internal/adapter/outbound/queue/queue.go`
- Modify: `services/api-service/internal/adapter/outbound/objectstore/objectstore.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/router.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/admin.go`
- Modify: `services/api-service/cmd/api/main.go`
- Modify: `packages/contracts/openapi.yaml`
- Modify: `deploy/docker-compose.yml`
- Modify: `.env.example`

**Steps:** TDD `Status.Snapshot` hides from non-admins and reports probe results. Add `Ping` on queue and object store. `GET /api/v1/status` returns `{ components: [{ id, ok }] }`. Default `DOCUMENT_HEALTH_URL=http://document-service:8081/healthz` in compose.

**Verify:** `bash scripts/verify-go.sh services/api-service`

**Commit:** `feat: expose platform dependency status`

---

### Task 2: Audit feed and last login

**Files:**
- Modify: `services/api-service/internal/app/port/identity.go`
- Modify: `services/api-service/internal/domain/identity/identity.go`
- Modify: `services/api-service/internal/adapter/outbound/postgres/identity.go`
- Modify: `services/api-service/internal/app/usecase/admin.go`
- Modify: `services/api-service/internal/app/usecase/auth_test.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/admin.go`
- Modify: `services/api-service/internal/adapter/inbound/httpapi/auth.go`
- Modify: `packages/contracts/openapi.yaml`

**Steps:** Filter list audit to access actions; join actor login and company name. `LastLogins` from `login_success`. Attach `LastSeenAt` on listed users.

**Verify:** `bash scripts/verify-go.sh services/api-service`

**Commit:** `feat: show access audit and last login`

---

### Task 3: Frontend presentation helpers

**Files:**
- Create: `frontend/src/features/ops/statusPresentation.js`
- Create: `frontend/src/features/ops/statusPresentation.test.js`
- Create: `frontend/src/features/ops/auditPresentation.js`
- Create: `frontend/src/features/ops/auditPresentation.test.js`
- Create: `frontend/src/features/auth/userHierarchy.js`
- Create: `frontend/src/features/auth/userHierarchy.test.js`

**Steps:** TDD Russian tile copy, audit sentences, last-seen labels, three role bands.

**Verify:** `npm test --prefix frontend`

**Commit:** `feat: add ops and hierarchy copy`

---

### Task 4: Overview screen and hierarchy UI

**Files:**
- Modify: `frontend/src/api/auth.js`
- Modify: `frontend/src/ui/icons.jsx`
- Create: `frontend/src/ui/admin/OverviewScreen.jsx`
- Modify: `frontend/src/ui/admin/UsersScreen.jsx`
- Modify: `frontend/src/ui/admin/AdminScreens.jsx`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/features/help/copy.js`
- Modify: `frontend/src/features/help/copy.test.js`

**Steps:** Platform admin lands on Обзор. Users screen renders three bands of cards with last login. Tour target `overview`.

**Verify:** `npm run verify --prefix frontend`

**Commit:** `feat: add platform overview and company hierarchy`
