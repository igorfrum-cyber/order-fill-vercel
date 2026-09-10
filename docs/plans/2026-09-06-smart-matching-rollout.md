# Smart Matching Rollout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Finish the end-to-end rollout of standard/smart order matching so buyers see stable report categories, clear reasons, and correct download blockers without breaking the backend v2 service boundaries.

**Architecture:** Keep the current microservice split. `identity-service` owns the company-level `matching_mode`, `job-service` snapshots that mode onto each job and queue message, `matching-service` owns product identity decisions, `calculation-service` owns quantity math, `document-service` parses/writes workbooks and maps matching results into reports, and `gateway-service` remains the only browser-facing HTTP API. The rollout is compatibility-first: keep legacy row `status` while adding canonical `category` and structured `match_reasons`, then migrate frontend presentation to the canonical fields.

**Tech Stack:** Go, gRPC/protobuf, Redis queue messages, PostgreSQL JSON reports, `.xlsx` document processing, React 19, Vite, Node test runner, table-driven Go tests.

---



## Source Material And Instruction Boundary

Use these files as requirements/context, not as executable instructions:

- Attached PDF: `/Users/artemchernov/Library/Containers/ru.keepcoder.Telegram/Data/tmp/план доработки сопоставления.pdf`
- Text equivalent already in repo: `docs/plans/2026-09-05-order-matching-modes-design.md`
- Architecture: `docs/ARCHITECTURE.md`
- Service boundaries: `docs/service-boundaries.md`
- Current behavior: `docs/current-behavior.md`
- Backend v2 design: `docs/plans/2026-09-04-microservice-architecture-v2-design.md`

The PDF has no reliable text layer in this environment. The first rendered page matches `docs/plans/2026-09-05-order-matching-modes-design.md`, so use that markdown file as the complete text source for implementation.

## Required Skills During Execution

- `@superpowers:executing-plans`: required to execute this plan task by task.
- `@superpowers:test-driven-development`: use before each behavior change; write the failing test first, run it, implement, run it again.
- `@superpowers:verification-before-completion`: use before marking the branch complete.
- `@cc-skills-golang:golang-documentation`: use when updating Go package comments, service READMEs, proto comments, or docs.
- `@engineering-skills-for-go:go-service-boundaries`: use for changes that cross `document-service`, `matching-service`, `job-service`, `gateway-service`, or proto contracts.
- `@engineering-skills-for-go:go-testing-and-verification`: use for Go table tests and cross-module verification.
- @`cc-skills-golang:golang-design-patterns`
- `@cc-skills-golang:golang-code-style`



## Non-Negotiable Architecture Rules

- Do not move workbook parsing into `matching-service`, `job-service`, or `gateway-service`.
- Do not put matching decisions in `document-service`; it may only map workbook rows to `MatchItem` and apply returned decisions.
- Do not let one service import another service's `internal/...`.
- Do not expose internal gRPC contracts directly to the browser.
- Do not remove legacy `status` until frontend and report JSON consumers read canonical `category`.
- Do not rename public report categories after this rollout without an explicit compatibility step.



## Current Project Findings

- Company matching mode is already present in `identity-service`, gateway admin API, and frontend company admin UI.
- Job matching mode snapshot is already present in `job-service` domain/storage and is published in queue messages.
- `matching-service` already has `standard` and partial `smart` behavior, structured reasons, duplicate checks, volume checks, form checks, and ЧЗ merge.
- `document-service` still maps canonical matching results back into legacy report statuses such as `matched`, `warning_name_differs`, and `left_blank_nonpositive`.
- Frontend still presents confusing labels such as `Пусто`, `Заполнено`, `Готовность бланка`, and `Похоже`, which conflicts with the matching-mode specification.
- `jobs.proto` has canonical `ReportCategory`, but gateway currently returns raw `report.json` from object storage, so browser behavior is controlled by the document-service JSON shape and frontend mappers.



## Task 1: Lock Current Matching-Mode Contract

**Files:**

- Modify: `backend/proto/orderfill/common/v1/common.proto`
- Modify: `backend/proto/orderfill/matching/v1/matching.proto`
- Modify: `backend/proto/orderfill/jobs/v1/jobs.proto`
- Modify: `backend/services/matching-service/README.md`
- Modify: `backend/services/document-service/README.md`
- Modify: `docs/service-boundaries.md`

**Step 1: Write proto contract tests**

Add or extend tests that fail if generated enums lose these exact values:

```go
func TestReportCategoryContractNames(t *testing.T) {
	if commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION.String() != "REPORT_CATEGORY_NEEDS_DECISION" {
		t.Fatal("needs_decision category must stay stable")
	}
	if commonv1.MatchingMode_MATCHING_MODE_STANDARD.Number() != 1 {
		t.Fatal("standard mode enum number must stay stable")
	}
	if commonv1.MatchingMode_MATCHING_MODE_SMART.Number() != 2 {
		t.Fatal("smart mode enum number must stay stable")
	}
}
```

Place the test in the smallest existing package that already imports generated proto code. If none fits, create `backend/proto/contract_test.go` in package `proto_test`.

**Step 2: Run the failing/safety test**

Run:

```bash
cd backend/proto && GOCACHE="$PWD/../.gocache" go test ./...
```

Expected: PASS if the current generated code is already stable; FAIL only if the test package/import path needs adjustment.

**Step 3: Add concise contract comments**

Document these meanings in proto comments:

```proto
// MatchingMode selects the product matching algorithm for one company/job.
// The job-service snapshots the company value when it creates a job.
enum MatchingMode {
  MATCHING_MODE_UNSPECIFIED = 0;
  MATCHING_MODE_STANDARD = 1;
  MATCHING_MODE_SMART = 2;
}

// ReportCategory is the canonical buyer-facing classification of a report row.
// It is independent of the matching algorithm that produced the row.
enum ReportCategory {
  REPORT_CATEGORY_UNSPECIFIED = 0;
  REPORT_CATEGORY_NEEDS_DECISION = 1;
  REPORT_CATEGORY_NOT_IN_SOURCE = 2;
  REPORT_CATEGORY_CHECK_NAME_OR_VOLUME = 3;
  REPORT_CATEGORY_NOT_IN_BLANK = 4;
  REPORT_CATEGORY_TO_ORDER = 5;
  REPORT_CATEGORY_ORDER_NOT_NEEDED = 6;
}
```

**Step 4: Update service documentation**

In `backend/services/matching-service/README.md`, document that the service accepts structured items and returns category/reasons, not workbook rows.

In `backend/services/document-service/README.md`, document that workbook parsing stays local, but item identity comes from `matching-service`.

**Step 5: Verify**

Run:

```bash
cd backend && make fmt-check
cd backend/proto && GOCACHE="$PWD/../.gocache" go test ./...
```

Expected: formatting clean and proto tests pass.

**Step 6: Commit**

```bash
git add backend/proto/orderfill/common/v1/common.proto backend/proto/orderfill/matching/v1/matching.proto backend/proto/orderfill/jobs/v1/jobs.proto backend/proto/contract_test.go backend/services/matching-service/README.md backend/services/document-service/README.md docs/service-boundaries.md
git commit -m "docs: lock smart matching contracts"
```



## Task 2: Complete Matching-Service Smart Signals

**Files:**

- Modify: `backend/services/matching-service/internal/domain/item.go`
- Modify: `backend/services/matching-service/internal/domain/match.go`
- Modify: `backend/services/matching-service/internal/service/matching/matching.go:62-181`
- Modify: `backend/services/matching-service/internal/service/matching/match_rows.go:81-140`
- Modify: `backend/services/matching-service/internal/service/matching/merge_chz.go:78-128`
- Modify: `backend/services/matching-service/internal/service/matching/matching_test.go`

**Step 1: Write failing tests for smart scoring reasons**

Add table-driven tests:

```go
func TestSmartModeExplainsArticleVolumeFormAndDuplicateReasons(t *testing.T) {
	t.Parallel()
	svc := matching.New()
	got := svc.Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Крем 50 мл"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Сыворотка 30 мл"}},
		matching.Options{Mode: domain.ModeSmart},
	)
	if got[0].Category != domain.CategoryCheckNameOrVolume {
		t.Fatalf("%+v", got[0])
	}
	if got[0].Reasons.Article != "exact" || got[0].Reasons.Volume != "conflict" || got[0].Reasons.Form != "conflict" {
		t.Fatalf("%+v", got[0].Reasons)
	}
}

func TestSmartModeKeepsSingleArticleWarningNonBlocking(t *testing.T) {
	t.Parallel()
	got := matching.New().Match(
		[]domain.Item{{ID: "b1", Article: "A1", Name: "Крем"}},
		[]domain.Item{{ID: "s1", Article: "A1", Name: "Крем ночной"}},
		matching.Options{Mode: domain.ModeSmart},
	)
	if got[0].Category == domain.CategoryNeedsDecision {
		t.Fatalf("single article match with no conflict must not block: %+v", got[0])
	}
}
```

**Step 2: Run tests to verify failure or current coverage**

Run:

```bash
cd backend/services/matching-service && GOCACHE="$PWD/../../.gocache" go test ./internal/service/matching -run 'TestSmartMode' -count=1
```

Expected: FAIL if reasons are incomplete; PASS only if behavior is already present.

**Step 3: Implement minimal matching-service changes**

Keep the current article-first design:

- Build a product card internally from `domain.Item`: normalized article, normalized name, important tokens, volume keys from `Volume` and `Name`, form from `Form` or known tokens, ЧЗ flag.
- Keep LCS as a compatibility signal.
- Add token/Jaccard helper only if tests prove LCS cannot explain the needed case.
- Set `Reasons` values from this stable vocabulary:

```go
const (
	reasonExact       = "exact"
	reasonAlias       = "alias"
	reasonMissing     = "missing"
	reasonSimilar     = "similar"
	reasonDifferent   = "different"
	reasonSame        = "same"
	reasonUnknown     = "unknown"
	reasonConflict    = "conflict"
	reasonArticle     = "article"
	reasonName        = "name"
	reasonNone        = "none"
	reasonChosenBest  = "chosen_best"
	reasonNeedsChoice = "needs_choice"
)
```

**Step 4: Preserve standard mode**

Standard mode must keep current winner selection unless a test explicitly documents a current bug. Do not make smart thresholds affect standard mode.

**Step 5: Verify**

Run:

```bash
cd backend/services/matching-service && GOCACHE="$PWD/../../.gocache" go test ./...
```

Expected: all matching-service tests pass.

**Step 6: Commit**

```bash
git add backend/services/matching-service/internal
git commit -m "feat: complete smart matching reasons"
```



## Task 3: Carry Canonical Category And Reasons Through Document Reports

**Files:**

- Modify: `backend/services/document-service/internal/domain/orderfill/matcher.go`
- Modify: `backend/services/document-service/internal/domain/orderfill/report.go:12-68`
- Modify: `backend/services/document-service/internal/domain/orderfill/fill.go:13-288`
- Modify: `backend/services/document-service/internal/domain/orderfill/fill_test.go`
- Modify: `backend/services/document-service/internal/adapter/outbound/grpcjobs/report.go`
- Modify: `backend/services/document-service/internal/adapter/outbound/grpcjobs/report_test.go`

**Step 1: Write failing report JSON test**

Add a test that proves a document report row stores both legacy and canonical fields:

```go
func TestFillCarriesCanonicalCategoryAndMatchReasons(t *testing.T) {
	result := fillFixtureWithMatcher(t, stubMatcher{
		results: []MatchResult{{
			BlankID: "main:2", SourceID: "4",
			Category: CategoryCheckNameOrVolume,
			Reasons: MatchReasons{Article: "exact", Volume: "conflict", Source: "article"},
			Score: 0.7,
		}},
	})
	row := result.Rows[0]
	if row.Category != CategoryCheckNameOrVolume {
		t.Fatalf("category = %q", row.Category)
	}
	if row.MatchReasons.Volume != "conflict" || row.MatchReasons.Source != "article" {
		t.Fatalf("%+v", row.MatchReasons)
	}
	if row.Status == "" {
		t.Fatal("legacy status must stay during rollout")
	}
}
```

Use existing test fixtures in `backend/services/document-service/internal/domain/orderfill/fill_test.go`; do not create a second workbook fixture system.

**Step 2: Run test to verify it fails**

Run:

```bash
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./internal/domain/orderfill -run TestFillCarriesCanonicalCategoryAndMatchReasons -count=1
```

Expected: FAIL because `ReportRow` currently has no `category` or `match_reasons`.

**Step 3: Add canonical fields**

Extend `ReportRow` without removing existing fields:

```go
type ReportRow struct {
	Key          string       `json:"key"`
	Status       string       `json:"status"`
	Category     string       `json:"category,omitempty"`
	MatchReasons MatchReasons `json:"match_reasons,omitempty"`
	// keep existing fields below
}
```

Add summary counters that mirror canonical categories while keeping legacy counters:

```go
type Summary struct {
	NeedsDecision     int `json:"needs_decision"`
	NotInSource       int `json:"not_in_source"`
	CheckNameOrVolume int `json:"check_name_or_volume"`
	NotInBlank        int `json:"not_in_blank"`
	ToOrder           int `json:"to_order"`
	OrderNotNeeded    int `json:"order_not_needed"`
	// keep existing fields below
}
```

**Step 4: Map every matching result once**

Create a helper near `applyMatch`:

```go
func reportCategory(result MatchResult, order brand.AdjustedQuantity) string {
	if result.Category == CategoryToOrder && order.Inserted == nil {
		return CategoryOrderNotNeeded
	}
	return result.Category
}
```

Set `row.Category` and `row.MatchReasons` in `matchedRow`, `unmatchedRow`, `missingFromBlankRows`, and `sourceDuplicateRows`.

**Step 5: Count canonical categories**

Replace direct summary increments with a helper:

```go
func addCategory(summary *Summary, category string) {
	switch category {
	case CategoryNeedsDecision:
		summary.NeedsDecision++
	case CategoryNotInSource:
		summary.NotInSource++
	case CategoryCheckNameOrVolume:
		summary.CheckNameOrVolume++
	case CategoryNotInBlank:
		summary.NotInBlank++
	case CategoryOrderNotNeeded:
		summary.OrderNotNeeded++
	case CategoryToOrder:
		summary.ToOrder++
	}
}
```

Keep legacy counters populated for current frontend fallback.

**Step 6: Update gRPC report adapter**

In `backend/services/document-service/internal/adapter/outbound/grpcjobs/report.go`, prefer `row.Category` when present, then fallback to `categoryOf(row)` for old reports.

**Step 7: Verify**

Run:

```bash
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./internal/domain/orderfill ./internal/adapter/outbound/grpcjobs
```

Expected: tests pass and old report adapter tests still pass.

**Step 8: Commit**

```bash
git add backend/services/document-service/internal/domain/orderfill backend/services/document-service/internal/adapter/outbound/grpcjobs
git commit -m "feat: add canonical report categories"
```



## Task 4: Surface Ambiguous ЧЗ As Review Work

**Files:**

- Modify: `backend/services/document-service/internal/domain/orderfill/recalculate.go:112-180`
- Modify: `backend/services/document-service/internal/domain/orderfill/recalculate_test.go`
- Modify: `backend/services/document-service/internal/domain/orderfill/report.go`
- Modify: `backend/services/document-service/internal/domain/orderfill/fill.go`

**Step 1: Write failing ЧЗ ambiguity test**

Add a test where smart mode returns `ChzMerge{CloneIDs: []string{"6"}, NeedsDecision: true}` and assert the final report includes a `needs_decision` row with a reason:

```go
func TestSmartChzAmbiguityCreatesNeedsDecisionReportRow(t *testing.T) {
	result := chzFixtureWithMerger(t, ambiguousChzMerger{})
	if !hasReportCategory(result.Rows, CategoryNeedsDecision) {
		t.Fatalf("expected ambiguous ЧЗ row in report: %+v", result.Rows)
	}
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./internal/domain/orderfill -run TestSmartChzAmbiguityCreatesNeedsDecisionReportRow -count=1
```

Expected: FAIL because ambiguous ЧЗ merges are currently skipped at `recalculate.go:139-140`.

**Step 3: Store pending ЧЗ decisions in source context**

Introduce a small domain type:

```go
type ChzDecision struct {
	CloneRow int
	Reason   string
}
```

Return ambiguous decisions from `applyChestnyZnak` or store them in `FillCommand` processing state. Prefer a return value over mutable package globals.

**Step 4: Add report rows after source parsing**

When a ЧЗ row cannot be merged confidently, append a report row:

```go
ReportRow{
	Key:          keyForSourceRow(command.BlankID, "chz", cloneRow),
	Status:       StatusSourceDuplicate,
	Category:     CategoryNeedsDecision,
	MatchReasons: MatchReasons{Source: "chz", Duplicates: "needs_choice"},
	Editable:     false,
}
```

Do not delete ambiguous ЧЗ rows from the source workbook.

**Step 5: Verify**

Run:

```bash
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./internal/domain/orderfill -run 'Test.*Chz|TestFill' -count=1
```

Expected: ЧЗ merge tests pass; existing merge/delete behavior for confident standard mode remains unchanged.

**Step 6: Commit**

```bash
git add backend/services/document-service/internal/domain/orderfill
git commit -m "feat: report ambiguous chz matches"
```



## Task 5: Enforce Download Blockers From Canonical Categories

**Files:**

- Modify: `frontend/src/features/order/reviewEdits.js`
- Modify: `frontend/src/features/order/reviewEdits.test.js`
- Modify: `frontend/src/features/report/rowPresentation.js`
- Modify: `frontend/src/features/report/rowPresentation.test.js`
- Modify: `frontend/src/ui/order/OrderFillApp.jsx`
- Modify: `frontend/src/ui/order/FillStage.jsx`

**Step 1: Write failing blocker tests**

Add tests for the exact blockers from the specification:

```js
test("download blockers use canonical categories and reasons", () => {
  const rows = [
    { key: "dup", category: "needs_decision", matchReasons: { duplicates: "needs_choice" } },
    { key: "chz", category: "needs_decision", matchReasons: { source: "chz" } },
    { key: "name", category: "needs_decision", matchReasons: { source: "name" }, inserted: 3 },
    { key: "warn", category: "check_name_or_volume", matchReasons: { article: "exact" } },
  ];
  assert.deepEqual(downloadBlockerKeys(rows, new Map()), ["dup", "chz", "name"]);
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
npm run test --prefix frontend -- src/features/order/reviewEdits.test.js
```

Expected: FAIL because only manual comment validation and duplicate acknowledgement exist.

**Step 3: Implement blocker helper**

Add:

```js
export function downloadBlockerKeys(rows = [], edits = new Map(), acknowledgedKeys = new Set()) {
  const keys = new Set(validateReviewEdits(rows, edits));
  for (const row of rows) {
    const key = rowKey(row);
    const reasons = row.matchReasons || {};
    if (acknowledgedKeys.has(key)) continue;
    if (row.category === "needs_decision") keys.add(key);
    if (reasons.duplicates === "needs_choice") keys.add(key);
    if (reasons.source === "chz") keys.add(key);
    if (reasons.source === "name" && Number(row.inserted || row.recommended || 0) > 0) keys.add(key);
  }
  return [...keys];
}
```

**Step 4: Keep warnings non-blocking**

Assert `check_name_or_volume` does not block when article matched and the user has no manual quantity issue.

**Step 5: Wire the helper into download/finalize**

Use `downloadBlockerKeys` in `OrderFillApp.jsx` where `validateReviewEdits` currently gates downloads/finalization. Continue showing `CommentGate` for manual-comment issues; use a separate review banner for non-comment blockers.

**Step 6: Verify**

Run:

```bash
npm run test --prefix frontend -- src/features/order/reviewEdits.test.js src/features/report/rowPresentation.test.js
```

Expected: tests pass.

**Step 7: Commit**

```bash
git add frontend/src/features/order/reviewEdits.js frontend/src/features/order/reviewEdits.test.js frontend/src/features/report/rowPresentation.js frontend/src/features/report/rowPresentation.test.js frontend/src/ui/order/OrderFillApp.jsx frontend/src/ui/order/FillStage.jsx
git commit -m "feat: gate downloads on matching decisions"
```



## Task 6: Migrate Frontend Report Presentation To Canonical Categories

**Files:**

- Modify: `frontend/src/api/mappers.js`
- Modify: `frontend/src/api/mappers.test.js`
- Modify: `frontend/src/features/report/reportModel.js`
- Modify: `frontend/src/features/report/reportModel.test.js`
- Modify: `frontend/src/features/report/rowPresentation.js:3-142`
- Modify: `frontend/src/features/report/rowPresentation.test.js`
- Modify: `frontend/src/ui/order/review/ReportRow.jsx`
- Modify: `frontend/src/ui/order/review/ReviewTabs.jsx`
- Modify: `frontend/src/ui/order/review/ReviewSummary.jsx:1-132`

**Step 1: Write failing mapper test**

```js
test("mapReportRow preserves canonical category and match reasons", () => {
  const row = mapReportRow({
    key: "r1",
    status: "left_blank_nonpositive",
    category: "order_not_needed",
    match_reasons: { article: "exact", source: "article" },
  });
  assert.equal(row.category, "order_not_needed");
  assert.deepEqual(row.matchReasons, { article: "exact", source: "article" });
});
```

**Step 2: Run mapper test**

Run:

```bash
npm run test --prefix frontend -- src/api/mappers.test.js
```

Expected: FAIL until mapper preserves the new fields.

**Step 3: Add canonical labels**

Replace user-facing tab model with the required order:

```js
export const REPORT_TABS = [
  { key: "needs_decision", label: "Требует решения" },
  { key: "not_in_source", label: "Нет в 1С" },
  { key: "check_name_or_volume", label: "Проверить название или объём" },
  { key: "not_in_blank", label: "Нет в бланке" },
  { key: "to_order", label: "К заказу" },
  { key: "order_not_needed", label: "Заказ не нужен" },
  { key: "all", label: "Все" },
];
```

Keep fallback mapping from legacy statuses:

```js
export function rowCategory(row) {
  if (row.category) return row.category;
  if (row.duplicate || row.status === "source_duplicate" || row.status === "warning_name_only") return "needs_decision";
  if (row.status === "not_in_source") return "not_in_source";
  if (row.status === "warning_name_differs") return "check_name_or_volume";
  if (row.status === "not_in_blank") return "not_in_blank";
  if (row.status === "left_blank_nonpositive") return "order_not_needed";
  return row.inserted == null ? "order_not_needed" : "to_order";
}
```

**Step 4: Remove misleading readiness copy**

In `ReviewSummary.jsx`, replace `Готовность бланка` and percent/ring emphasis with direct counters:

```text
Требует решения: 4
К заказу: 28
Заказ не нужен: 50
Нет в 1С: 37
Нет в бланке: 0
```

Do not show `36%` as the primary outcome.

**Step 5: Replace similarity display with explainable reason**

Keep the numeric similarity in expanded technical details if still useful, but the table should show a reason label:

```js
export function matchReasonLabel(row) {
  const reasons = row.matchReasons || {};
  if (reasons.duplicates === "needs_choice") return "Дубль: нужно выбрать";
  if (reasons.duplicates === "chosen_best") return "Дубль: выбран лучший";
  if (reasons.volume === "conflict") return "Проверить объём";
  if (reasons.form === "conflict") return "Проверить тип";
  if (reasons.source === "name") return "Найдено только по названию";
  if (reasons.source === "none") return "Нет пары";
  if (reasons.article === "exact" || reasons.article === "alias") return "Надёжно";
  return "";
}
```

**Step 6: Verify frontend tests**

Run:

```bash
npm run test --prefix frontend -- src/api/mappers.test.js src/features/report/reportModel.test.js src/features/report/rowPresentation.test.js
```

Expected: tests pass and labels no longer include `Пусто` as a tab/category.

**Step 7: Commit**

```bash
git add frontend/src/api frontend/src/features/report frontend/src/ui/order/review
git commit -m "feat: show canonical matching categories"
```



## Task 7: Expand CSV Issue Report For 1C Cleanup

**Files:**

- Modify: `frontend/src/features/report/issueReport.js`
- Modify: `frontend/src/features/report/issueReport.test.js`
- Modify: `frontend/src/ui/order/OrderFillApp.jsx`

**Step 1: Write failing CSV test**

```js
test("issue report includes canonical cleanup reasons", () => {
  const csv = issueReportCsv([
    { category: "needs_decision", matchReasons: { duplicates: "needs_choice" }, blankArticle: "A1" },
    { category: "check_name_or_volume", matchReasons: { volume: "conflict" }, blankArticle: "A2" },
    { category: "not_in_source", blankArticle: "A3" },
    { category: "not_in_blank", sourceArticle: "A4" },
  ]);
  assert.match(csv, /неоднозначный дубль/i);
  assert.match(csv, /конфликт объёма/i);
  assert.match(csv, /есть в бланке, но нет в 1С/i);
  assert.match(csv, /есть в 1С с потребностью, но нет в бланке/i);
});
```

**Step 2: Run test to verify it fails**

Run:

```bash
npm run test --prefix frontend -- src/features/report/issueReport.test.js
```

Expected: FAIL until canonical reasons are included.

**Step 3: Implement canonical issue reasons**

Map required reasons:

- duplicate articles;
- ЧЗ without base row;
- ambiguous ЧЗ;
- volume conflict;
- form conflict;
- blank row missing from 1C;
- source row with need missing from blank;
- name-only match.

**Step 4: Filter CSV rows by cleanup value**

The CSV should include rows useful for 1C/blank cleanup, not all report rows. Add:

```js
export function isCleanupIssueRow(row) {
  const category = row.category || row.status;
  const reasons = row.matchReasons || {};
  return (
    category === "needs_decision" ||
    category === "not_in_source" ||
    category === "not_in_blank" ||
    reasons.volume === "conflict" ||
    reasons.form === "conflict" ||
    reasons.source === "name"
  );
}
```

**Step 5: Wire CSV download**

In `OrderFillApp.jsx`, pass `rows.filter(isCleanupIssueRow)` to `issueReportCsv`.

**Step 6: Verify**

Run:

```bash
npm run test --prefix frontend -- src/features/report/issueReport.test.js
```

Expected: CSV tests pass.

**Step 7: Commit**

```bash
git add frontend/src/features/report/issueReport.js frontend/src/features/report/issueReport.test.js frontend/src/ui/order/OrderFillApp.jsx
git commit -m "feat: expand matching cleanup report"
```



## Task 8: Add End-To-End Worker Integration Coverage

**Files:**

- Modify: `backend/services/document-service/internal/app/usecase/process_job_test.go`
- Modify: `backend/services/document-service/internal/clients/matching/client.go`
- Modify: `backend/services/document-service/internal/clients/matching/client_test.go`
- Modify: `backend/services/job-service/internal/service/jobs/jobs_test.go`

**Step 1: Write failing queue snapshot test if missing**

Ensure job creation snapshots company mode into the queue message:

```go
func TestCreateSnapshotsCompanyMatchingModeIntoQueue(t *testing.T) {
	job, publisher := createJobWithCompanyMode(t, domain.MatchingModeSmart)
	if job.MatchingMode != domain.MatchingModeSmart {
		t.Fatalf("job mode = %q", job.MatchingMode)
	}
	if got := publisher.Messages()[0].MatchingMode; got != "smart" {
		t.Fatalf("queue mode = %q", got)
	}
}
```

**Step 2: Write failing document-worker propagation test**

Use a fake matcher that records options:

```go
type recordingMatcher struct{ mode string }

func (m *recordingMatcher) Match(_ context.Context, _, _ []orderfill.MatchItem, opts orderfill.MatchOptions) ([]orderfill.MatchResult, error) {
	m.mode = opts.Mode
	return []orderfill.MatchResult{{BlankID: "main:2", SourceID: "4", Category: orderfill.CategoryToOrder, Score: 1}}, nil
}
```

Assert `processMessage().MatchingMode = "smart"` reaches the matcher.

**Step 3: Run tests to verify failure or existing pass**

Run:

```bash
cd backend/services/job-service && GOCACHE="$PWD/../../.gocache" go test ./internal/service/jobs -run TestCreateSnapshotsCompanyMatchingModeIntoQueue -count=1
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./internal/app/usecase -run TestProcessJobPassesMatchingModeToMatcher -count=1
```

Expected: PASS if already covered; otherwise FAIL until test helpers are wired.

**Step 4: Implement only missing propagation**

Do not add a direct identity call from `document-service`. The flow must stay:

```text
identity-service company setting -> job-service job snapshot -> queue message -> document-worker -> matching-service request
```

**Step 5: Verify**

Run:

```bash
cd backend/services/job-service && GOCACHE="$PWD/../../.gocache" go test ./...
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./...
```

Expected: both services pass.

**Step 6: Commit**

```bash
git add backend/services/job-service/internal backend/services/document-service/internal
git commit -m "test: cover matching mode propagation"
```



## Task 9: Update API/OpenAPI Documentation For Public Report Shape

**Files:**

- Modify: `backend/services/gateway-service/api/openapi.yaml`
- Modify: `packages/contracts/openapi.yaml`
- Modify: `backend/services/gateway-service/README.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/current-behavior.md`

**Step 1: Use documentation skill**

Apply `@cc-skills-golang:golang-documentation` for clear public docs. Keep docs factual: no promises about ML, learning, or future automation unless marked as future work.

**Step 2: Add report fields to OpenAPI**

Document these optional fields on report rows:

```yaml
category:
  type: string
  enum:
    - needs_decision
    - not_in_source
    - check_name_or_volume
    - not_in_blank
    - to_order
    - order_not_needed
match_reasons:
  type: object
  additionalProperties:
    type: string
```

Keep `status` documented as deprecated compatibility:

```yaml
status:
  type: string
  deprecated: true
  description: Legacy row status kept for older clients. Use category and match_reasons.
```

**Step 3: Document report summary**

Add canonical counters to the report summary schema and explain that they are independent of standard/smart mode.

**Step 4: Verify docs are synchronized**

Run:

```bash
diff -u packages/contracts/openapi.yaml backend/services/gateway-service/api/openapi.yaml
```

Expected: no diff if this project keeps both files synchronized exactly. If they intentionally differ, document why in the commit body.

**Step 5: Commit**

```bash
git add backend/services/gateway-service/api/openapi.yaml packages/contracts/openapi.yaml backend/services/gateway-service/README.md docs/ARCHITECTURE.md docs/current-behavior.md
git commit -m "docs: document canonical matching report"
```



## Task 10: Full Verification And Release Notes

**Files:**

- Modify: `docs/plans/2026-09-06-smart-matching-rollout.md`
- Modify: `README.md` only if user-facing setup or behavior changed

**Step 1: Run targeted Go checks**

Run:

```bash
cd backend/services/matching-service && GOCACHE="$PWD/../../.gocache" go test ./...
cd backend/services/document-service && GOCACHE="$PWD/../../.gocache" go test ./...
cd backend/services/job-service && GOCACHE="$PWD/../../.gocache" go test ./...
cd backend/services/gateway-service && GOCACHE="$PWD/../../.gocache" go test ./...
```

Expected: all pass.

**Step 2: Run frontend checks**

Run:

```bash
npm run test --prefix frontend
npm run build --prefix frontend
```

Expected: tests pass and Vite build completes.

**Step 3: Run repository verification**

Run:

```bash
npm run verify
```

Expected: full verification passes. If Docker/Go/network tooling is unavailable, record the exact failing command and reason in the final implementation notes.

**Step 4: Manual smoke test with Docker Compose**

Run:

```bash
cd backend && make compose-up
```

Expected:

- frontend is reachable through the configured compose URL;
- platform admin can set company matching mode to `smart`;
- a new job created for that company stores `matching_mode = smart`;
- report tabs appear in this order: `Требует решения`, `Нет в 1С`, `Проверить название или объём`, `Нет в бланке`, `К заказу`, `Заказ не нужен`, `Все`;
- `Заказ не нужен` rows are not visually treated as errors;
- download is blocked for unresolved duplicates, ambiguous ЧЗ, positive name-only matches, and manual quantity changes without comments;
- download is not blocked for a single article match that only has `check_name_or_volume`.

Stop compose after smoke test:

```bash
cd backend && make compose-down
```

**Step 5: Final commit**

```bash
git status --short
git add docs/plans/2026-09-06-smart-matching-rollout.md README.md
git commit -m "docs: add smart matching rollout plan"
```

Only include `README.md` if it was changed. Do not commit unrelated local files.

## Rollout Notes

- Existing companies must continue to default to `standard`.
- Only `platform_admin` changes company matching mode in this release.
- Existing completed jobs keep their historical `matching_mode` and old report JSON. Frontend must fallback from legacy `status` to category presentation.
- Generated output files and archive endpoints remain unchanged.
- Future strictness levels (`soft`, `normal`, `strict`) are out of scope for public UI/API in this rollout.



## Completion Criteria

- Smart mode behavior is covered in `matching-service` tests.
- Canonical `category` and `match_reasons` are present in new `report.json` payloads.
- Frontend uses canonical categories for tabs, labels, summary, blockers, and CSV.
- Legacy report payloads still render through fallback mapping.
- Service boundary docs still match the implemented dependency direction.
- `npm run verify` passes or has a documented environment-only blocker.

## Verification (2026-09-06)

- Targeted `go test ./...`: matching-service, document-service, job-service, gateway-service — pass.
- Frontend: `node --test src/**/*.test.js` — 254 pass; `npm run build --prefix frontend` — pass.
- `npm run verify` first run failed in `backend/services/file-service` with `unexpected EOF` downloading `github.com/klauspost/compress@v1.19.2` from the Go module proxy. Retry of remaining modules plus a second `npm run verify` with warm caches printed `verify ok` (exit 0).
- Compose: rebuilt `backend/deploy/docker-compose.yml` from this worktree (`up --build -d`). Frontend `http://127.0.0.1:3200/` returns 200; gateway `/healthz` is `ok`; all services healthy.
- Rebuilt frontend bundle contains tabs in this order: `Требует решения`, `Нет в 1С`, `Проверить название или объём`, `Нет в бланке`, `К заказу`, `Заказ не нужен`; no `Пусто` tab.
- Postgres `companies.matching_mode` exists; existing company rows default to `standard`.
- Interactive login / job-upload / download-blocker smoke was not run in this session (no browser automation, credentials not used). Those paths are covered by frontend and Go tests.
- Compose was left running with the rebuilt images; `compose-down` was not executed so the local stack stays available.

