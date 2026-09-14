import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test } from "@playwright/test";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const dataRoot = process.env.ORDER_FILL_PRIVATE_TESTDATA || path.join(repoRoot, "testdata/private");
const sourceFile = path.join(dataRoot, "таблицы продаж/Ангио Тюмень .xlsx");
const blankFile = path.join(dataRoot, "Бланки/2026 08 25 Бланк заказа ANGIOPHARM.xlsx");
const login = process.env.REAL_E2E_LOGIN;
const password = process.env.REAL_E2E_PASSWORD;
const realPairs = [
  {
    name: "Skin Synergy",
    source: "таблицы продаж/Скин Синерджи Тюмень .xlsx",
    blank: "Бланки/_Бланк заказа Skin Synergy от 26.08.2026.xlsx",
  },
  {
    name: "KLAPP",
    source: "таблицы продаж/Клапп Тюмень .xlsx",
    blank: "Бланки/Бланк Заказа KLAPP август 2026 (1).xlsx",
  },
  {
    name: "Christina",
    source: "таблицы продаж/Кристина Тюмень .xlsx",
    blank: "Бланки/Актуальный_бланк PROFF.xlsx",
  },
];

test("real ANGIOPHARM files pass budget calculation, preview and download", async ({ page }, testInfo) => {
  expect(login, "Set REAL_E2E_LOGIN for a purchaser account").toBeTruthy();
  expect(password, "Set REAL_E2E_PASSWORD for that account").toBeTruthy();
  expect(fs.existsSync(sourceFile), `Missing real source workbook: ${sourceFile}`).toBe(true);
  expect(fs.existsSync(blankFile), `Missing real supplier workbook: ${blankFile}`).toBe(true);

  await authenticatePurchaser(page, testInfo);

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Загрузите файлы" })).toBeVisible();
  await page.getByLabel("Таблица продаж из 1С").setInputFiles(sourceFile);
  await page.getByLabel("Бланк", { exact: true }).setInputFiles(blankFile);
  await page.getByRole("button", { name: "Обработать" }).click();

  const budgetButton = page.getByRole("button", { name: "Заказ до суммы" });
  await expect(budgetButton).toBeVisible({ timeout: 120_000 });
  const orderTotalText = await page.getByText("Заказ:").textContent();
  const currentTotal = moneyFromText(orderTotalText);

  await budgetButton.click();
  const dialog = page.getByRole("dialog", { name: "Заказ до суммы" });
  await dialog.getByRole("radio", { name: "Вычесть скидку" }).check();
  await dialog.getByLabel("Скидка, %").fill("10");
  await dialog.getByLabel("Целевая сумма, ₽").fill(String(Math.ceil(currentTotal + 1_000)));
  await dialog.getByRole("button", { name: "Рассчитать" }).click();

  const overSix = dialog.getByRole("checkbox", { name: /Разрешить запас больше 6 месяцев/ });
  if (await overSix.isVisible()) {
    await overSix.check();
    await dialog.getByRole("button", { name: "Рассчитать" }).click();
  }
  const belowOne = dialog.getByRole("checkbox", { name: /Разрешить запас меньше 1 месяца/ });
  if (await belowOne.isVisible()) {
    await belowOne.check();
    await dialog.getByRole("button", { name: "Рассчитать" }).click();
  }

  await expect(dialog.getByLabel("Предпросмотр бюджета")).toBeVisible();
  await expect(dialog.getByLabel("Предпросмотр бюджета")).toContainText(/Изменено позиций: [1-9]/);
  await dialog.getByRole("button", { name: "Применить" }).click();
  await expect(page.getByText(/скидка 10%/)).toBeVisible();

  for (const checkbox of await page.getByRole("checkbox", { name: /Оставляю как есть/ }).all()) {
    await checkbox.check();
  }
  await page.getByRole("button", { name: "Проверить файлы" }).click();

  const commentGate = page.getByRole("dialog", { name: "Проверьте комментарии" });
  if (await commentGate.isVisible()) {
    await commentGate.getByRole("button", { name: "Поставщик" }).click();
    await commentGate.getByRole("button", { name: /Продолжить/ }).click();
  }
  const warning = page.getByRole("dialog", { name: "Проверьте спорные строки" });
  if (await warning.isVisible()) {
    await warning.getByRole("button", { name: "Продолжить проверку" }).click();
  }

  await expect(page.getByRole("button", { name: "Таблица 1С" })).toBeVisible({ timeout: 120_000 });
  await expect(page.getByRole("button", { name: "Бланк" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Найти артикул" })).toBeVisible();
  await expect(page.getByText(/строк · до/)).toBeVisible();

  const downloads = [];
  page.on("download", (download) => downloads.push(download));
  await page.getByRole("button", { name: "Скачать файлы" }).click();
  await expect.poll(() => downloads.length, { timeout: 120_000 }).toBeGreaterThanOrEqual(2);

  for (const [index, download] of downloads.entries()) {
    const output = testInfo.outputPath(`${index}-${download.suggestedFilename()}`);
    await download.saveAs(output);
    const content = fs.readFileSync(output);
    expect(content.length).toBeGreaterThan(1_000);
    expect(content.subarray(0, 2).toString()).toBe("PK");
  }
});

for (const pair of realPairs) {
  test(`real ${pair.name} workbooks reach the editing screen`, async ({ page }, testInfo) => {
    const actualSource = path.join(dataRoot, pair.source);
    const actualBlank = path.join(dataRoot, pair.blank);
    expect(fs.existsSync(actualSource), `Missing real source workbook: ${actualSource}`).toBe(true);
    expect(fs.existsSync(actualBlank), `Missing real supplier workbook: ${actualBlank}`).toBe(true);
    await authenticatePurchaser(page, testInfo);

    await page.goto("/jobs/new");
    await page.getByLabel("Таблица продаж из 1С").setInputFiles(actualSource);
    await page.getByLabel("Бланк", { exact: true }).setInputFiles(actualBlank);
    await page.getByRole("button", { name: "Обработать" }).click();

    await expect(page.getByRole("button", { name: "Проверить файлы" })).toBeVisible({ timeout: 120_000 });
    await expect(page.getByRole("columnheader", { name: "Товар" })).toBeVisible();
  });
}

async function authenticatePurchaser(page, testInfo) {
  expect(login, "Set REAL_E2E_LOGIN for a purchaser account").toBeTruthy();
  expect(password, "Set REAL_E2E_PASSWORD for that account").toBeTruthy();
  const origin = new URL(testInfo.project.use.baseURL).origin;
  const auth = await page.request.post("/api/v1/auth/login", {
    data: { login, password },
    headers: { Origin: origin, "X-Requested-With": "fetch" },
  });
  const body = await auth.text();
  expect(auth.ok(), `Login failed: ${auth.status()} ${body}`).toBe(true);
  expect(JSON.parse(body).role).toBe("purchaser");
}

function moneyFromText(value) {
  const normalized = String(value || "")
    .replace(/[^\d,.-]/g, "")
    .replace(",", ".");
  const amount = Number(normalized);
  expect(Number.isFinite(amount), `Cannot parse order total from: ${value}`).toBe(true);
  return amount;
}
