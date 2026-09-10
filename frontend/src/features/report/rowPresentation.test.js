import test from "node:test";
import assert from "node:assert/strict";

import {
  canProceedPastDuplicates,
  commentGateHint,
  commentGateTitle,
  matchingDecisionBanner,
  countByTab,
  displayArticle,
  displayName,
  matchLayerHint,
  matchReasonLabel,
  presentationStatus,
  attentionReason,
  reviewTableHeaders,
  rowCategory,
  rowMatchesQuery,
  rowMatchesTab,
  visibleFillTabs,
  visibleReportRows,
  firstReviewTab,
  reviewQueueLine,
  REPORT_TABS,
} from "./rowPresentation.js";

test("presentationStatus maps API row states onto canonical categories", () => {
  assert.equal(presentationStatus({ status: "matched", inserted: 12 }), "to_order");
  assert.equal(presentationStatus({ status: "matched_by_name", inserted: 4 }), "to_order");
  assert.equal(presentationStatus({ status: "left_blank_nonpositive" }), "order_not_needed");
  assert.equal(presentationStatus({ status: "warning_name_differs" }), "check_name_or_volume");
  assert.equal(presentationStatus({ status: "warning_name_only" }), "needs_decision");
  assert.equal(presentationStatus({ status: "not_in_source" }), "not_in_source");
  assert.equal(presentationStatus({ status: "not_in_blank" }), "not_in_blank");
  assert.equal(presentationStatus({ status: "matched", duplicate: true, inserted: 2 }), "needs_decision");
  assert.equal(presentationStatus({ status: "source_duplicate" }), "needs_decision");
  assert.equal(presentationStatus({ category: "order_not_needed", status: "left_blank_nonpositive" }), "order_not_needed");
});

test("countByTab and rowMatchesTab follow canonical categories", () => {
  const rows = [
    { status: "matched", inserted: 12, blankName: "Крем", blankArticle: "A1" },
    { status: "left_blank_nonpositive", blankName: "Тоник", blankArticle: "A2" },
    { status: "warning_name_only", blankName: "Сыворотка", blankArticle: "A3" },
    { status: "not_in_source", blankName: "Маска", blankArticle: "A4" },
    { status: "not_in_blank", blankName: "Флюид", blankArticle: "B1" },
    { status: "matched", duplicate: true, inserted: 6, blankName: "Пилинг", blankArticle: "A5" },
  ];

  const counts = countByTab(rows);
  assert.equal(counts.all, 6);
  assert.equal(counts.to_order, 1);
  assert.equal(counts.order_not_needed, 1);
  assert.equal(counts.needs_decision, 2);
  assert.equal(counts.not_in_source, 1);
  assert.equal(counts.not_in_blank, 1);
  assert.equal(rows.filter((row) => rowMatchesTab(row, "order_not_needed")).length, 1);
});

test("visibleReportRows filters by tab and searches article or name", () => {
  const rows = [
    { status: "left_blank_nonpositive", blankName: "Крем для лица", blankArticle: "AP-100" },
    { status: "left_blank_nonpositive", blankName: "Тоник", blankArticle: "CS-200" },
    { status: "matched", inserted: 3, blankName: "Крем ночной", blankArticle: "AP-300" },
  ];

  assert.equal(visibleReportRows(rows, { tab: "order_not_needed", query: "крем" }).length, 1);
  assert.equal(visibleReportRows(rows, { tab: "all", query: "AP-" }).length, 2);
  assert.equal(rowMatchesQuery(rows[1], "тоник"), true);
});

test("displayArticle and displayName fall back to source identity when the row is missing from the blank", () => {
  const row = {
    status: "not_in_blank",
    blankArticle: "",
    blankName: "",
    sourceArticle: "A400",
    sourceName: "Сыворотка",
    recommended: 8,
  };

  assert.equal(displayArticle(row), "A400");
  assert.equal(displayName(row), "Сыворотка");
});

test("REPORT_TABS follow the required buyer-facing order and do not include Пусто", () => {
  assert.deepEqual(REPORT_TABS.map((tab) => tab.key), [
    "needs_decision",
    "not_in_source",
    "check_name_or_volume",
    "not_in_blank",
    "to_order",
    "order_not_needed",
    "all",
  ]);
  assert.equal(REPORT_TABS.some((tab) => tab.label === "Пусто"), false);
});

test("visibleFillTabs keep the canonical order", () => {
  assert.deepEqual(visibleFillTabs().map((tab) => tab.key), REPORT_TABS.map((tab) => tab.key));
});

test("reviewTableHeaders use a reason column instead of similarity", () => {
  const labels = Object.fromEntries(reviewTableHeaders().map((header) => [header.key, header.label]));
  assert.equal(labels.recommended, "Расчёт");
  assert.equal(labels.match, "Причина");
  assert.equal(labels.inserted, "Вставлено");
});

test("matchReasonLabel prefers an explainable reason over a percent", () => {
  assert.equal(matchReasonLabel({ matchReasons: { duplicates: "needs_choice" } }), "Дубль: нужно выбрать");
  assert.equal(matchReasonLabel({ matchReasons: { volume: "conflict" } }), "Проверить объём");
  assert.equal(matchReasonLabel({ matchReasons: { article: "exact" } }), "Надёжно");
});

test("attentionReason explains why a row needs a closer look", () => {
  assert.equal(
    attentionReason({ status: "warning_name_differs" }),
    "Название отличается от таблицы заказа.",
  );
  assert.equal(
    attentionReason({ status: "not_in_source" }),
    "Позиция есть в бланке, но не нашлась в 1С.",
  );
  assert.equal(attentionReason({ status: "matched", inserted: 2 }), "");
});

test("matchLayerHint explains unmatched tabs and stays quiet for fill tabs", () => {
  assert.match(matchLayerHint("not_in_source"), /не нашлись в 1С/i);
  assert.match(matchLayerHint("not_in_blank"), /не нашлись в бланке/i);
  assert.equal(matchLayerHint("order_not_needed"), "");
  assert.equal(matchLayerHint("all"), "");
});

test("matchingDecisionBanner is a review banner, not a comment gate", () => {
  assert.match(matchingDecisionBanner, /требуют решения/i);
});

test("comment gate copy tells the reviewer why they cannot open files yet", () => {
  assert.equal(commentGateTitle, "Сначала напишите, почему изменили количество");
  assert.match(commentGateHint, /не пускаем в файлы/i);
});

test("canProceedPastDuplicates requires every duplicate key to be acknowledged", () => {
  assert.equal(canProceedPastDuplicates({ duplicateKeys: [], acknowledgedKeys: new Set() }), true);
  assert.equal(canProceedPastDuplicates({ duplicateKeys: ["a", "b"], acknowledgedKeys: new Set() }), false);
  assert.equal(canProceedPastDuplicates({ duplicateKeys: ["a", "b"], acknowledgedKeys: new Set(["a"]) }), false);
  assert.equal(canProceedPastDuplicates({ duplicateKeys: ["a", "b"], acknowledgedKeys: new Set(["a", "b"]) }), true);
});

test("displayArticle and displayName keep blank identity when both sides exist", () => {
  const row = {
    status: "matched",
    blankArticle: "AP-100",
    blankName: "Крем",
    sourceArticle: "AP-100",
    sourceName: "Крем для лица",
  };

  assert.equal(displayArticle(row), "AP-100");
  assert.equal(displayName(row), "Крем");
});

test("firstReviewTab prefers unresolved decisions", () => {
  assert.equal(firstReviewTab({ needs_decision: 2, order_not_needed: 1, check_name_or_volume: 1 }), "needs_decision");
  assert.equal(firstReviewTab({ needs_decision: 0, not_in_source: 3, check_name_or_volume: 1 }), "not_in_source");
  assert.equal(firstReviewTab({ needs_decision: 0, not_in_source: 0, check_name_or_volume: 2 }), "check_name_or_volume");
  assert.equal(firstReviewTab({ needs_decision: 0, to_order: 10 }), "to_order");
});

test("reviewQueueLine counts work left", () => {
  assert.match(reviewQueueLine({ needs_decision: 6 }), /6/);
  assert.equal(reviewQueueLine({ needs_decision: 0, to_order: 4 }), "");
});

test("rowCategory prefers canonical category over legacy status", () => {
  assert.equal(rowCategory({ category: "order_not_needed", status: "left_blank_nonpositive" }), "order_not_needed");
});
