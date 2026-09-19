import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test } from "@playwright/test";

import {
  CHRISTINA_PROFF_UI_BLANK,
  CHRISTINA_PROFF_UI_SOURCE,
  christinaProffUiPair,
  isChristinaProffBlank,
  resolvePrivateTestdata,
  scanPrivateBrandPairs,
} from "./brandPairs.js";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const dataRoot = resolvePrivateTestdata(repoRoot);
const login = process.env.REAL_E2E_LOGIN;
const password = process.env.REAL_E2E_PASSWORD;
const ownerLogin = process.env.REAL_E2E_OWNER_LOGIN;
const ownerPassword = process.env.REAL_E2E_OWNER_PASSWORD;
const profileLegalName = "ООО Сквозной E2E";
const allPairs = process.env.REAL_E2E_ALL_PAIRS === "1";
const scanned = scanPrivateBrandPairs(dataRoot);
const christinaUi = christinaProffUiPair(dataRoot);
const pairs = allPairs ? scanned.pairs : christinaUi.pair ? [christinaUi.pair] : [];

function skipReason() {
  if (!scanned.available) return `Нет каталогов Бланки / таблицы продаж в ${dataRoot}`;
  if (allPairs && !scanned.pairs.length) return "В private testdata нет matched pairs по бренду";
  if (!allPairs && !christinaUi.pair) {
    return `Нет пары UI CHRISTINA: ${CHRISTINA_PROFF_UI_BLANK} × ${CHRISTINA_PROFF_UI_SOURCE} в ${dataRoot}`;
  }
  if (!login || !password) return "Задайте REAL_E2E_LOGIN и REAL_E2E_PASSWORD";
  if (!ownerLogin || !ownerPassword) return "Задайте REAL_E2E_OWNER_LOGIN и REAL_E2E_OWNER_PASSWORD";
  return "";
}

const reason = skipReason();

test.describe(allPairs ? "real matched brand pairs" : "real Christina PROFF UI pair", () => {
  test.skip(Boolean(reason), reason);

  if (!pairs.length) {
    test("opt-in real suite", () => {});
    return;
  }

  test.beforeAll(async ({ browser }) => {
    const baseURL = process.env.REAL_E2E_BASE_URL || "http://127.0.0.1:3200";
    const context = await browser.newContext({ baseURL });
    const page = await context.newPage();
    await configureOrderProfile(page, baseURL);
    await context.close();
  });

  for (const pair of pairs) {
    test(`${pair.brand}: ${pair.blank} × ${pair.source}`, async ({ page }, testInfo) => {
      expect(fs.existsSync(pair.sourcePath), `Missing real source workbook: ${pair.sourcePath}`).toBe(true);
      expect(fs.existsSync(pair.blankPath), `Missing real supplier workbook: ${pair.blankPath}`).toBe(true);

      await authenticatePurchaser(page, testInfo);
      await page.goto("/");
      await expect(page.getByRole("heading", { name: "Загрузите файлы" })).toBeVisible();
      await page.getByLabel("Таблица продаж из 1С").setInputFiles(pair.sourcePath);
      await page.getByLabel("Бланк", { exact: true }).setInputFiles(pair.blankPath);
      await page.getByRole("button", { name: "Обработать" }).click();

      await expect(page.getByRole("button", { name: "Проверить файлы" })).toBeVisible({ timeout: 120_000 });
      await expect(page.getByRole("columnheader", { name: "Товар" })).toBeVisible();
      if (pair.brand === "christina") expect(isChristinaProffBlank(pair.blank)).toBe(true);

      const budgetButton = page.getByRole("button", { name: "Заказ до суммы" });
      await expect(budgetButton).toBeVisible();
      const orderTotalText = await page.getByText("Заказ:").textContent();
      await runBudgetPreview(page, moneyFromText(orderTotalText));
    });
  }
});

async function runBudgetPreview(page, currentTotal) {
  await page.getByRole("button", { name: "Заказ до суммы" }).click();
  const dialog = page.getByRole("dialog", { name: "Заказ до суммы" });
  await dialog.getByRole("radio", { name: "Вычесть скидку" }).check();
  await dialog.getByLabel("Скидка, %").fill("10");
  await dialog.getByLabel("Целевая сумма, ₽").fill(String(Math.ceil(currentTotal + 1_000)));

  let plan;
  const capture = async (response) => {
    if (!response.url().includes("/api/v1/order/budget-plan") || !response.ok()) return;
    plan = await response.json();
  };
  page.on("response", capture);
  try {
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
    await expect.poll(() => plan?.complete === true).toBe(true);
    await expect(dialog.getByRole("button", { name: "Применить" })).toBeEnabled();

    const kits = dialog.getByText("PROFF: комплекты и дополнительная скидка");
    const sawKits = await kits.isVisible();
    if (sawKits) {
      await kits.click();
      await expect(dialog.getByRole("columnheader", { name: "Линия" }).first()).toBeVisible();
    }

    await dialog.getByRole("button", { name: "Применить" }).click();
    await expect(page.getByText(/скидка 10%/)).toBeVisible();
    if (sawKits) {
      const comments = page.getByPlaceholder("Почему изменили количество");
      await expect.poll(async () => {
        const values = [];
        for (const input of await comments.all()) values.push(await input.inputValue());
        return values.some((value) => /комплект/i.test(value));
      }).toBeTruthy();
    }
  } finally {
    page.off("response", capture);
  }
}

async function authenticatePurchaser(page, testInfo) {
  const origin = new URL(testInfo.project.use.baseURL).origin;
  const auth = await page.request.post("/api/v1/auth/login", {
    data: { login, password },
    headers: { Origin: origin, "X-Requested-With": "fetch" },
  });
  const body = await auth.text();
  expect(auth.ok(), `Login failed: ${auth.status()} ${body}`).toBe(true);
  expect(JSON.parse(body).role).toBe("purchaser");
}

async function configureOrderProfile(page, baseURL) {
  const origin = new URL(baseURL).origin;
  const headers = { Origin: origin, "X-Requested-With": "fetch" };
  const auth = await page.request.post("/api/v1/auth/login", {
    data: { login: ownerLogin, password: ownerPassword },
    headers,
  });
  const body = await auth.text();
  expect(auth.ok(), `Owner login failed: ${auth.status()} ${body}`).toBe(true);
  const owner = JSON.parse(body);
  expect(owner.role).toBe("company_owner");

  const saved = await page.request.post(`/api/v1/companies/${owner.company_id}/order-profile`, {
    data: {
      legal_name: profileLegalName,
      consignee: "ООО Сквозной E2E",
      address: "Тюмень, ул. Тестовая, 1",
      contact_name: "Иван Иванов",
      contact_phone: "+7 912 345-67-89",
      carrier: "Деловые линии",
      delivery_payer: "Получатель",
      delivery_destination: "До терминала",
      brand_terms: [
        { brand: "angiopharm", discount_set: true, discount_basis_points: 2750 },
        { brand: "skin_synergy", discount_set: true, discount_basis_points: 3000 },
        { brand: "klapp", discount_set: true, discount_basis_points: 2500 },
        { brand: "christina", discount_set: true, discount_basis_points: 3000 },
      ],
    },
    headers,
  });
  expect(saved.ok(), `Saving company order profile failed: ${saved.status()} ${await saved.text()}`).toBe(true);
  await page.request.post("/api/v1/auth/logout", { headers });
}

function moneyFromText(value) {
  const normalized = String(value || "")
    .replace(/[^\d,.-]/g, "")
    .replace(",", ".");
  const amount = Number(normalized);
  expect(Number.isFinite(amount), `Cannot parse order total from: ${value}`).toBe(true);
  return amount;
}
