import { expect, test } from "@playwright/test";

test("opening a failed job shows the processing error instead of an empty review", async ({ page }) => {
  const reportCalls = [];
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1", login: "art", role: "company_owner", company_id: "company-1",
        company_name: "Тестовая компания", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [{
        id: "failed-job", type: "order_fill", status: "failed", brand: "christina",
        error: { code: "processing_error", message: "Не нашли колонку артикула в бланке." },
      }] } });
      return;
    }
    if (path === "/api/v1/jobs/failed-job") {
      await route.fulfill({ json: {
        id: "failed-job", type: "order_fill", status: "failed", brand: "christina",
        order_month: "2026-08",
        error: { code: "processing_error", message: "Не нашли колонку артикула в бланке." },
      } });
      return;
    }
    if (path === "/api/v1/jobs/failed-job/report") {
      reportCalls.push(path);
      await route.fulfill({ status: 404, json: { code: "not_found", message: "report was not found" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/jobs/failed-job");
  await expect(page.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("Не нашли колонку артикула в бланке.");
  await expect(page.getByText("Критичных проблем нет")).toHaveCount(0);
  expect(reportCalls).toEqual([]);
});

test("opening a needs_review job without a report does not look like an empty review", async ({ page }) => {
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1", login: "art", role: "company_owner", company_id: "company-1",
        company_name: "Тестовая компания", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [{
        id: "review-job", type: "order_fill", status: "needs_review", brand: "",
      }] } });
      return;
    }
    if (path === "/api/v1/jobs/review-job") {
      await route.fulfill({ json: {
        id: "review-job", type: "order_fill", status: "needs_review", brand: "",
      } });
      return;
    }
    if (path === "/api/v1/jobs/review-job/report") {
      await route.fulfill({ status: 404, json: { code: "not_found", message: "report was not found" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/jobs/review-job");
  await expect(page.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("Отчёт по этой выгрузке не сохранился");
  await expect(page.getByText("Критичных проблем нет")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Проверить файлы" })).toHaveCount(0);
});

test("opening a failed north merge from history shows the error instead of an empty north form", async ({ page }) => {
  const northError = "не узнали город по имени файла \"Актуальный_бланк PROFF.xlsx\".";
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1", login: "ivanov", role: "purchaser", company_id: "company-1",
        company_name: "Тестовая компания", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [{
        id: "north-failed", type: "north_merge", status: "failed", brand: "christina",
        error: { code: "processing_error", message: northError },
        created_at: "2026-09-18T10:00:00Z",
      }] } });
      return;
    }
    if (path === "/api/v1/jobs/north-failed") {
      await route.fulfill({ json: {
        id: "north-failed", type: "north_merge", status: "failed", brand: "christina",
        error: { code: "processing_error", message: northError },
      } });
      return;
    }
    if (path === "/api/v1/jobs/north-failed/report") {
      await route.fulfill({ status: 404, json: { code: "not_found", message: "report was not found" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/jobs");
  await page.getByRole("link", { name: "Открыть" }).click();
  await expect(page.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("не узнали город по имени файла");
  await expect(page.getByRole("heading", { name: "Соединить бланки" })).toHaveCount(0);
  await expect(page.getByText("Критичных проблем нет")).toHaveCount(0);
});

test("opening a failed north merge by URL shows the error instead of an empty review", async ({ page }) => {
  const northError = "не узнали город по имени файла \"Актуальный_бланк PROFF.xlsx\".";
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "user-1", login: "ivanov", role: "purchaser", company_id: "company-1",
        company_name: "Тестовая компания", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/jobs" && request.method() === "GET") {
      await route.fulfill({ json: { jobs: [] } });
      return;
    }
    if (path === "/api/v1/jobs/north-failed") {
      await route.fulfill({ json: {
        id: "north-failed", type: "north_merge", status: "failed", brand: "christina",
        error: { code: "processing_error", message: northError },
      } });
      return;
    }
    if (path === "/api/v1/jobs/north-failed/report") {
      await route.fulfill({ status: 404, json: { code: "not_found", message: "report was not found" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/jobs/north-failed");
  await expect(page.getByRole("heading", { name: "Не получилось обработать" })).toBeVisible();
  await expect(page.getByRole("alert")).toContainText("не узнали город по имени файла");
  await expect(page.getByText("Критичных проблем нет")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Проверить файлы" })).toHaveCount(0);
});
