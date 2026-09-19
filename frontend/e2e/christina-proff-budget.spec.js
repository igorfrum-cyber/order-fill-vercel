import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test } from "@playwright/test";

import {
  CHRISTINA_PROFF_UI_BLANK,
  CHRISTINA_PROFF_UI_SOURCE,
  christinaProffUiPair,
  resolvePrivateTestdata,
} from "./brandPairs.js";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const { pair } = christinaProffUiPair(resolvePrivateTestdata(repoRoot));
const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";

test("Christina PROFF UI uploads real Tyumen + PROFF (1) and shows сверка timings", async ({ page }) => {
  test.skip(!pair, `Нужны файлы ${CHRISTINA_PROFF_UI_BLANK} и ${CHRISTINA_PROFF_UI_SOURCE} в testdata/private`);

  expect(fs.existsSync(pair.sourcePath)).toBe(true);
  expect(fs.existsSync(pair.blankPath)).toBe(true);

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const routePath = url.pathname;
    if (routePath === "/api/v1/auth/me") {
      await route.fulfill({
        json: {
          id: "user-1",
          login: "buyer",
          role: "purchaser",
          company_id: "company-1",
          company_name: "Тестовая компания",
          two_factor_enabled: true,
        },
      });
      return;
    }
    if (routePath === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [] } });
      return;
    }
    if (routePath === "/api/v1/jobs/order-fill") {
      const body = (await request.postDataBuffer()).toString("utf8");
      expect(body).toContain(CHRISTINA_PROFF_UI_SOURCE.normalize("NFC"));
      expect(body).toContain(CHRISTINA_PROFF_UI_BLANK.normalize("NFC"));
      await route.fulfill({ status: 202, json: { id: "job-christina" } });
      return;
    }
    if (routePath === "/api/v1/jobs/job-christina") {
      await route.fulfill({
        json: {
          id: "job-christina",
          type: "order_fill",
          status: "needs_review",
          brand: "christina",
          order_month: "2026-09",
        },
      });
      return;
    }
    if (routePath === "/api/v1/jobs/job-christina/report") {
      await route.fulfill({
        json: {
          job_id: "job-christina",
          summary: { brand: "CHRISTINA", delivery_weeks: 1, to_order: 1 },
          rows: [
            {
              key: "proff:3",
              status: "matched",
              category: "to_order",
              match_reasons: { article: "exact", source: "article" },
              blank_id: "proff",
              blank_label: pair.blank,
              blank_row: 3,
              blank_quantity_col: 4,
              blank_article: "3",
              blank_name: "MUSE 3",
              blank_unit: "шт",
              blank_box_size: "3",
              source_row: 2,
              source_article: "3",
              source_name: "MUSE 3",
              stock: "0",
              in_transit: "0",
              recommended: 3,
              rounded: 3,
              inserted: 3,
              editable: true,
              has_budget_data: true,
              budget_category: "A+",
              budget_demand: 10,
              budget_price: 100,
              christina_line: { id: "MUSE", name: "MUSE", article: "3", required: ["0", "1", "2", "3"] },
            },
          ],
        },
      });
      return;
    }
    if (routePath === "/api/v1/order/budget-plan") {
      await route.fulfill({
        json: {
          before: 5640,
          total: 5880,
          target: 6100,
          reason: "",
          complete: true,
          christina_proff_mode: "compare",
          rows: [
            {
              key: "proff:3",
              name: "MUSE 3",
              category: "A+",
              before: 3,
              quantity: 6,
              comment: "Добавилось 3 шт. Для закупа до суммы. Дополнение комплектов MUSE.",
            },
          ],
          line_steps: [{ id: "MUSE", name: "MUSE", sets: 6, added: 300, saved: 60 }],
          line_groups: [{ id: "MUSE", name: "MUSE", valid: true, sets: 6, saving: 120, net: 5880 }],
          fast_total: 5880,
          fast_complete: true,
          fast_rows: [
            {
              key: "proff:3",
              name: "MUSE 3",
              category: "A+",
              before: 3,
              quantity: 6,
              comment: "Добавилось 3 шт. Для закупа до суммы. Дополнение комплектов MUSE.",
            },
          ],
          fast_line_steps: [{ id: "MUSE", name: "MUSE", sets: 6, added: 300, saved: 60 }],
          compare: { match: true, standard_ms: 12, fast_ms: 3, mismatches: [] },
        },
      });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${routePath}` } });
  });

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Загрузите файлы" })).toBeVisible();
  await page.getByLabel("Таблица продаж из 1С").setInputFiles({
    name: CHRISTINA_PROFF_UI_SOURCE.normalize("NFC"),
    mimeType: xlsxType,
    buffer: fs.readFileSync(pair.sourcePath),
  });
  await page.getByLabel("Бланк", { exact: true }).setInputFiles({
    name: CHRISTINA_PROFF_UI_BLANK.normalize("NFC"),
    mimeType: xlsxType,
    buffer: fs.readFileSync(pair.blankPath),
  });
  await page.getByRole("button", { name: "Обработать" }).click();

  await expect(page.getByRole("columnheader", { name: "Товар" })).toBeVisible();
  await page.getByRole("button", { name: "Заказ до суммы" }).click();
  const dialog = page.getByRole("dialog", { name: "Заказ до суммы" });
  await dialog.getByLabel("Целевая сумма, ₽").fill("6100");
  await dialog.getByRole("button", { name: "Рассчитать" }).click();

  const preview = dialog.getByLabel("Предпросмотр бюджета");
  await expect(preview).toContainText(/5[\s\u00a0]?640,00\s*₽\s*→\s*5[\s\u00a0]?880,00\s*₽/);
  await expect(preview).toContainText("Сверка CHRISTINA: совпало. Текущий 12 мс, быстрый 3 мс.");
  await expect(dialog.getByRole("link", { name: "Текущий JSON" })).toBeVisible();
  await expect(dialog.getByRole("link", { name: "Быстрый JSON" })).toBeVisible();
  await expect(dialog.getByRole("link", { name: "Расхождения JSON" })).toBeVisible();
});
