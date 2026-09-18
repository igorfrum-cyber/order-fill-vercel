import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";

import { OrderFillApp } from "./OrderFillApp.jsx";

test("failed resume shows the job error instead of an empty review", () => {
  render(
    <OrderFillApp
      companyId="company-1"
      resumeJob={{
        jobId: "job-failed",
        brand: "christina",
        month: "2026-08",
        status: "failed",
        error: "Не нашли колонку артикула в бланке.",
        rows: [],
        results: [],
        edits: new Map(),
        outputFiles: [],
        finalized: false,
      }}
      onHome={() => {}}
    />,
  );

  expect(screen.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  expect(screen.getByRole("alert")).toHaveTextContent("Не нашли колонку артикула в бланке.");
  expect(screen.queryByText("Критичных проблем нет")).toBeNull();
  expect(screen.queryByRole("button", { name: "Проверить файлы" })).toBeNull();
});

test("needs_review resume without a report shows the broken job instead of an empty review", () => {
  render(
    <OrderFillApp
      companyId="company-1"
      resumeJob={{
        jobId: "job-review",
        brand: "",
        month: "",
        status: "needs_review",
        error: "Отчёт по этой выгрузке не сохранился. Сверку открыть нельзя.",
        rows: [],
        results: [],
        edits: new Map(),
        outputFiles: [],
        finalized: false,
      }}
      onHome={() => {}}
    />,
  );

  expect(screen.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  expect(screen.getByRole("alert")).toHaveTextContent("Отчёт по этой выгрузке не сохранился");
  expect(screen.queryByText("Критичных проблем нет")).toBeNull();
  expect(screen.queryByRole("button", { name: "Проверить файлы" })).toBeNull();
});
