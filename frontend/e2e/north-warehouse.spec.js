import { expect, test } from "@playwright/test";

test("North uploads office and delivery warehouse as distinct inputs", async ({ page }) => {
  let multipartBody = "";
  let submittedEdits = null;
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1",
        login: "buyer",
        role: "purchaser",
        company_id: "company-1",
        company_name: "Тестовая компания",
        two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [] } });
      return;
    }
    if (path === "/api/v1/jobs/north-merge") {
      multipartBody = (await request.postDataBuffer()).toString("utf8");
      await route.fulfill({ status: 202, json: { id: "job-1" } });
      return;
    }
    if (path === "/api/v1/jobs/job-1") {
      await route.fulfill({ json: { id: "job-1", type: "north_merge", status: submittedEdits ? "completed" : "needs_review", brand: "angiopharm" } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/report") {
      await route.fulfill({ json: {
        has_tyumen_source: true,
        uploaded_cities: ["Сургут"],
        plan_rows: [{
          key: "A1",
          article: "A1",
          name: "Крем",
          cities: [{ key: "surgut", label: "Сургут", quantity: 10 }],
          tyumenStock: 30,
          tyumenInTransit: 0,
          tyumenTarget: 5,
          tyumenFree: 10,
          fromTyumen: 10,
          supplierNeed: 0,
          actualSupplierOrder: 0,
          northNeed: 10,
          hasTyumenSource: true,
          warehouseStock: 10,
          warehouseTransit: 0,
          hasWarehouseStock: true,
        }],
        transfers: [],
        confirmation_groups: [],
        summary: { kind: "angiopharm" },
      } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/edits") {
      submittedEdits = JSON.parse(request.postData() || "{}");
      await route.fulfill({ json: { id: "job-1", status: "finalizing" } });
      return;
    }
    if (path === "/api/v1/jobs/job-1/files") {
      await route.fulfill({ json: { files: [{
        id: "north-output",
        label: "Скачать общий бланк",
        name: "Общий бланк.xlsx",
        content_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        download_path: "/api/v1/jobs/job-1/files/north-output",
      }] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/north");
  await expect(page.getByRole("heading", { name: "Соединить бланки" })).toBeVisible();
  await expect(page.locator('[data-tour="north-warehouse"]')).toContainText("Таблица склада доставки");

  await page.locator('[data-tour="north-cities"] input[type="file"]').setInputFiles({ name: "Сургут.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("city") });
  await page.locator('[data-tour="north-tyumen"] input[type="file"]').setInputFiles({ name: "Тюмень.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("office") });
  await page.locator('[data-tour="north-warehouse"] input[type="file"]').setInputFiles({ name: "Склад доставка.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("warehouse") });

  await expect(page.getByText("Выбрано файлов: 3.")).toBeVisible();
  await page.getByRole("button", { name: "Соединить бланки" }).click();
  await expect.poll(() => multipartBody).toContain('name="warehouse_file"; filename="Склад доставка.xlsx"');
  expect(multipartBody).toContain('name="tyumen_source_file"; filename="Тюмень.xlsx"');
  expect(multipartBody).toContain('name="blank_files"; filename="Сургут.xlsx"');
  await page.getByRole("button", { name: "Соединить", exact: true }).click();
  await expect(page.getByRole("columnheader", { name: "Склад доставки" })).toBeVisible();
  await expect(page.getByText("Доступный остаток ограничен складом доставки.")).toBeVisible();

  await page.getByRole("spinbutton", { name: "Сургут", exact: true }).fill("15");
  await expect(page.getByRole("spinbutton", { name: "Фактический заказ у поставщика для Крем" })).toHaveValue("5");
  await page.getByRole("button", { name: "Скачать файлы" }).click();
  await expect(page.getByRole("link", { name: "Скачать общий бланк" })).toBeVisible();
  expect(submittedEdits).toEqual({ edits: [{ key: "A1", value: "5", comment: "" }] });
});
