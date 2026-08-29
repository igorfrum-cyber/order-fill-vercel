import test from "node:test";
import assert from "node:assert/strict";

import { duplicateDescription, issueReason, issueReportCsv } from "./issueReport.js";

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
