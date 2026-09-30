import test from "node:test";
import assert from "node:assert/strict";

import { duplicateDescription, isCleanupIssueRow, issueReason, issueReportCsv } from "./issueReport.js";

test("duplicateDescription formats duplicate source rows", () => {
  assert.equal(duplicateDescription([
    { sourceRow: 4, sourceName: "Cream" },
    { sourceRow: 8, sourceName: "Serum" },
  ]), "Строка 4: Cream; Строка 8: Serum");
});

test("issueReason combines status and duplicate explanations", () => {
  const reason = issueReason({
    status: "warning_name_only",
    duplicate: true,
    duplicateCandidates: [{ sourceRow: 4, sourceName: "Cream" }],
  });

  assert.match(reason, /нет артикула/);
  assert.match(reason, /дублирующиеся кандидаты/);
  assert.match(reason, /Строка 4: Cream/);
});

test("issueReportCsv escapes cells and includes manager comment", () => {
  const csv = issueReportCsv([
    {
      status: "not_in_source",
      blankLabel: "HOME",
      blankArticle: "A1",
      blankName: 'Cream "X"',
      blankUnit: "50 ml",
      sourceRow: "",
      sourceArticle: "",
      sourceName: "",
    },
  ], () => ({ comment: "проверить" }));

  assert.match(csv, /"Cream ""X"""/);
  assert.match(csv, /"проверить"/);
});

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

test("cleanup issue detection uses legacy status, canonical category and conflict reasons", () => {
  for (const row of [
    { category: "needs_decision" },
    { category: "not_in_source" },
    { category: "not_in_blank" },
    { matchReasons: { volume: "conflict" } },
    { matchReasons: { form: "conflict" } },
    { matchReasons: { source: "name" } },
    { status: "not_in_source" },
  ]) {
    assert.equal(isCleanupIssueRow(row), true, JSON.stringify(row));
  }
  assert.equal(isCleanupIssueRow({ category: "to_order", status: "matched" }), false);
  assert.equal(isCleanupIssueRow({ matchReasons: { volume: "match", form: "match", source: "article" } }), false);
});

test("issueReason distinguishes Chestny Znak and duplicate-choice cases", () => {
  const ambiguous = issueReason({
    category: "needs_decision",
    matchReasons: { source: "chz", duplicates: "needs_choice", volume: "conflict", form: "conflict" },
  });
  assert.match(ambiguous, /неоднозначный дубль/);
  assert.match(ambiguous, /неоднозначное объединение ЧЗ/);
  assert.match(ambiguous, /конфликт типа/);

  const unmatchedClone = issueReason({
    category: "not_in_blank",
    matchReasons: { source: "chz", duplicates: "chosen_best" },
    duplicate: true,
    duplicateCandidates: [{ sourceRow: 5 }],
  });
  assert.match(unmatchedClone, /ЧЗ без основной строки/);
  assert.match(unmatchedClone, /Есть дублирующиеся кандидаты/);
  assert.match(unmatchedClone, /Дубли в таблице: Строка 5:/);
});

test("issue report emits a header for no rows and defaults absent edit comments", () => {
  const csv = issueReportCsv([{
    status: "source_duplicate",
    duplicate: true,
    duplicateCandidates: [],
  }]);
  assert.equal(csv.split("\n").length, 2);
  assert.match(csv, /В таблице заказа есть несколько строк с одним артикулом/);
  assert.ok(csv.endsWith('""'));
  assert.equal(duplicateDescription([]), "");
});
