import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test } from "vitest";

import { FillStage } from "./FillStage.jsx";

function NameOnlyHarness() {
  const rows = [
    {
      key: "a",
      status: "warning_name_only",
      category: "needs_decision",
      matchReasons: { source: "name" },
      blankId: "main",
      blankRow: 4,
      blankArticle: "WH02",
      blankName: "Детокс сыворотка",
      editable: true,
      inserted: 0,
    },
  ];
  const [acks, setAcks] = useState(() => new Set());
  return (
    <FillStage
      brand="angiopharm"
      rows={rows}
      edits={new Map([["a", { value: "0", comment: "" }]])}
      onEdit={() => {}}
      invalidKeys={new Set()}
      summary={{ brand: "ANGIOPHARM", orderMonthLabel: "сентябрь 2026" }}
      status=""
      busy={false}
      banner=""
      onDownloadFiles={() => {}}
      onIssueReport={() => {}}
      acknowledgedDuplicates={acks}
      onAcknowledgedDuplicates={(next) => setAcks(typeof next === "function" ? next(acks) : next)}
    />
  );
}

function StockOptionHarness() {
  const [include, setInclude] = useState(false);
  return (
    <FillStage
      brand="angiopharm"
      rows={[]}
      edits={new Map()}
      onEdit={() => {}}
      invalidKeys={new Set()}
      summary={{}}
      status=""
      busy={false}
      banner=""
      onDownloadFiles={() => {}}
      onIssueReport={() => {}}
      acknowledgedDuplicates={new Set()}
      onAcknowledgedDuplicates={() => {}}
      includeBlankStocks={include}
      onIncludeBlankStocksChange={setInclude}
    />
  );
}

test("name-only rows can be acknowledged so files can be checked", async () => {
  const user = userEvent.setup();
  render(<NameOnlyHarness />);

  expect(screen.getByRole("button", { name: "Проверить файлы" })).toBeDisabled();
  await user.click(screen.getByRole("checkbox", { name: /оставляю/i }));
  expect(screen.getByRole("button", { name: "Проверить файлы" })).toBeEnabled();
});

test("blank stock export is an explicit opt-in", async () => {
  const user = userEvent.setup();
  render(<StockOptionHarness />);

  const checkbox = screen.getByRole("checkbox", { name: "Добавить остатки в бланк" });
  expect(checkbox).not.toBeChecked();
  await user.click(checkbox);
  expect(checkbox).toBeChecked();
});
