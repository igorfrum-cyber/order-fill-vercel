import { expect, test } from "@playwright/test";

test("order upload, budget recalculation and both workbook previews stay editable", async ({ page }) => {
  const submissions = [];
  const downloaded = [];
  const previewRows = {
    "source-output": [
      ["Артикул", "Товар", "Категория", "Заказано по факту", "Комментарий"],
      ["A1", "Крем", "B", "", ""],
    ],
    "blank-output": [
      ["Артикул", "Наименование", "Цена", "Количество", "Сумма"],
      ["A1", "Крем", "100", "", ""],
    ],
  };

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1", login: "buyer", role: "purchaser", company_id: "company-1",
        company_name: "Тестовая компания", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [] } });
      return;
    }
    if (path === "/api/v1/jobs/order-fill") {
      const body = (await request.postDataBuffer()).toString("utf8");
      expect(body).toContain('name="source_file"; filename="Продажи.xlsx"');
      expect(body).toContain('name="blank_files"; filename="Бланк.xlsx"');
      await route.fulfill({ status: 202, json: { id: "job-1" } });
      return;
    }
    if (path === "/api/v1/jobs/job-1") {
      await route.fulfill({ json: {
        id: "job-1", type: "order_fill", status: submissions.length ? "completed" : "needs_review",
        brand: "angiopharm", order_month: "2026-09",
      } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/report") {
      await route.fulfill({ json: {
        job_id: "job-1",
        summary: { brand: "ANGIOPHARM", delivery_weeks: 1, to_order: 1 },
        rows: [{
          key: "blank-1:2", status: "matched", category: "to_order",
          match_reasons: { article: "exact", source: "article" },
          blank_id: "blank-1", blank_label: "Бланк.xlsx", blank_row: 2, blank_quantity_col: 4,
          blank_article: "A1", blank_name: "Крем", blank_unit: "50 мл", blank_box_size: "3",
          source_row: 2, source_article: "A1", source_name: "Крем", stock: "0", in_transit: "0",
          recommended: 3, rounded: 3, inserted: 3, editable: true,
          has_budget_data: true, budget_category: "B", budget_demand: 10, budget_price: 100,
        }],
      } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/edits") {
      submissions.push(JSON.parse(request.postData() || "{}"));
      await route.fulfill({ json: { id: "job-1", status: "finalizing" } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/files") {
      await route.fulfill({ json: { files: [
        { id: "source-output", label: "Скачать таблицу заказа", name: "Продажи заполнено.xlsx", download_path: "/api/v1/jobs/job-1/files/source-output" },
        { id: "blank-output", label: "Скачать заполненный бланк", name: "Бланк заполнен.xlsx", download_path: "/api/v1/jobs/job-1/files/blank-output" },
      ] } });
      return;
    }
    const preview = path.match(/^\/api\/v1\/jobs\/job-1\/files\/(source-output|blank-output)\/preview$/);
    if (preview) {
      await route.fulfill({ json: { chunk_rows: 256, sheets: [{
        name: preview[1] === "source-output" ? "Таблица заказа" : "Бланк",
        index: 0, max_row: 2, max_column: 5, header_row: 1,
        quantity_column: 4, comment_column: preview[1] === "source-output" ? 5 : 0,
        columns: [100, 180, 90, 120, 220], row_height: 28,
      }] } });
      return;
    }
    const window = path.match(/^\/api\/v1\/jobs\/job-1\/files\/(source-output|blank-output)\/preview\/window$/);
    if (window) {
      const from = Number(url.searchParams.get("from_row") || 1);
      const to = Number(url.searchParams.get("to_row") || 2);
      await route.fulfill({ json: { from_row: from, to_row: to, rows: previewRows[window[1]].slice(from - 1, to) } });
      return;
    }
    const download = path.match(/^\/api\/v1\/jobs\/job-1\/files\/(source-output|blank-output)$/);
    if (download) {
      downloaded.push(download[1]);
      await route.fulfill({
        body: "xlsx", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        headers: { "Content-Disposition": `attachment; filename="${download[1]}.xlsx"` },
      });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Загрузите файлы" })).toBeVisible();
  await page.locator('[data-tour="source"] input[type="file"]').setInputFiles({ name: "Продажи.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("source") });
  await page.locator('[data-tour="blank"] input[type="file"]').setInputFiles({ name: "Бланк.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("blank") });
  await page.getByRole("button", { name: "Обработать" }).click();

  await expect(page.getByRole("columnheader", { name: "Товар" })).toBeVisible();
  await expect(page.getByText("Заказ:")).toContainText("300,00 ₽");
  await page.getByRole("button", { name: "Заказ до суммы" }).click();
  const dialog = page.getByRole("dialog", { name: "Заказ до суммы" });
  await dialog.getByLabel("Целевая сумма, ₽").fill("600");
  await dialog.getByRole("button", { name: "Рассчитать" }).click();
  await expect(dialog.getByLabel("Предпросмотр бюджета")).toContainText("300,00 ₽ → 600,00 ₽");
  await expect(dialog.getByText("Крем · B")).toBeVisible();
  await dialog.getByRole("button", { name: "Применить" }).click();
  await expect(page.getByRole("textbox", { name: "Количество", exact: true })).toHaveValue("6");
  await expect(page.getByPlaceholder("Почему изменили количество")).toHaveValue(/Добавилось 3 шт/);
  await expect(page.getByRole("button", { name: "Отменить перерасчёт" })).toBeVisible();

  await page.getByRole("button", { name: "Проверить файлы" }).click();
  await page.getByRole("dialog", { name: "Проверьте спорные строки" }).getByRole("button", { name: "Продолжить проверку" }).click();
  await expect(page.getByRole("button", { name: "Таблица 1С" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Бланк" })).toBeVisible();
  await expect(page.getByText("Крем")).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Количество", exact: true })).toHaveValue("6");
  await page.getByRole("button", { name: "Бланк" }).click();
  await expect(page.getByRole("textbox", { name: "Количество", exact: true })).toHaveValue("6");

  await page.getByRole("textbox", { name: "Количество", exact: true }).fill("9");
  await page.getByRole("button", { name: "Скачать файлы" }).click();
  await expect.poll(() => submissions.length).toBe(2);
  expect(submissions[0].edits[0]).toMatchObject({ key: "blank-1:2", value: "6" });
  expect(submissions[1].edits[0]).toMatchObject({ key: "blank-1:2", value: "9" });
  await expect.poll(() => downloaded.sort()).toEqual(["blank-output", "source-output"]);
});
