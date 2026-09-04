import test from "node:test";
import assert from "node:assert/strict";

import {
  canProceedPastDuplicates,
  commentGateHint,
  commentGateTitle,
  countByTab,
  displayArticle,
  displayName,
  FILL_COMPOSITION_ORDER,
  fillReadiness,
  matchLayerHint,
  pairedRowCount,
  presentationStatus,
  attentionReason,
  reviewTableHeaders,
  rowMatchesQuery,
  rowMatchesTab,
  visibleFillTabs,
  visibleReportRows,
} from "./rowPresentation.js";

test("presentationStatus maps API row states onto fill-stage tabs", () => {
  assert.equal(presentationStatus({ status: "matched", inserted: 12 }), "filled");
  assert.equal(presentationStatus({ status: "matched_by_name", inserted: 4 }), "filled");
  assert.equal(presentationStatus({ status: "left_blank_nonpositive" }), "empty");
  assert.equal(presentationStatus({ status: "warning_name_differs" }), "check");
  assert.equal(presentationStatus({ status: "warning_name_only" }), "check");
  assert.equal(presentationStatus({ status: "not_in_source" }), "not_in_table");
  assert.equal(presentationStatus({ status: "not_in_blank" }), "not_in_blank");
  assert.equal(presentationStatus({ status: "matched", duplicate: true, inserted: 2 }), "duplicate");
  assert.equal(presentationStatus({ status: "source_duplicate" }), "duplicate");
});

test("countByTab and rowMatchesTab follow presentation status, not raw API status", () => {
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
  assert.equal(counts.filled, 1);
  assert.equal(counts.empty, 1);
  assert.equal(counts.check, 1);
  assert.equal(counts.duplicate, 1);
  assert.equal(counts.not_in_table, 1);
  assert.equal(counts.not_in_blank, 1);
  assert.equal(rows.filter((row) => rowMatchesTab(row, "empty")).length, 1);
});

test("visibleReportRows filters by tab and searches article or name", () => {
  const rows = [
    { status: "left_blank_nonpositive", blankName: "Крем для лица", blankArticle: "AP-100" },
    { status: "left_blank_nonpositive", blankName: "Тоник", blankArticle: "CS-200" },
    { status: "matched", inserted: 3, blankName: "Крем ночной", blankArticle: "AP-300" },
  ];

  assert.equal(visibleReportRows(rows, { tab: "empty", query: "крем" }).length, 1);
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

test("fill composition excludes unmatched rows so the ring can reach 100%", () => {
  assert.deepEqual(FILL_COMPOSITION_ORDER, ["filled", "empty", "check", "duplicate"]);

  const counts = {
    filled: 139,
    empty: 91,
    check: 0,
    duplicate: 2,
    not_in_table: 124,
    not_in_blank: 36,
    all: 392,
  };

  assert.equal(pairedRowCount(counts), 232);
  assert.equal(fillReadiness(counts), 139 / 232);
  assert.equal(pairedRowCount({}), 0);
  assert.equal(fillReadiness({}), 0);
});

test("visibleFillTabs hides empty fill buckets but always keeps matching and all", () => {
  const tabs = visibleFillTabs({
    filled: 139,
    empty: 91,
    check: 0,
    duplicate: 2,
    not_in_table: 124,
    not_in_blank: 0,
    all: 356,
  });

  assert.deepEqual(tabs.map((tab) => tab.key), [
    "empty",
    "duplicate",
    "not_in_table",
    "not_in_blank",
    "filled",
    "all",
  ]);
});

test("reviewTableHeaders use short operational labels", () => {
  const labels = Object.fromEntries(reviewTableHeaders().map((header) => [header.key, header.label]));
  assert.equal(labels.recommended, "Расчёт");
  assert.equal(labels.match, "Похоже");
  assert.equal(labels.inserted, "Вставлено");
});

test("attentionReason explains why a row needs a closer look", () => {
  assert.equal(
    attentionReason({ status: "warning_name_differs" }),
    "Название отличается от таблицы заказа.",
  );
  assert.equal(
    attentionReason({ status: "not_in_source" }),
    "Позиция есть в бланке, но не нашлась в таблице заказа.",
  );
  assert.equal(attentionReason({ status: "matched", inserted: 2 }), "");
});

test("matchLayerHint explains unmatched tabs and stays quiet for fill tabs", () => {
  assert.match(matchLayerHint("not_in_table"), /не нашлись в таблице заказа/i);
  assert.match(matchLayerHint("not_in_blank"), /не нашлись в бланке/i);
  assert.equal(matchLayerHint("empty"), "");
  assert.equal(matchLayerHint("all"), "");
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
