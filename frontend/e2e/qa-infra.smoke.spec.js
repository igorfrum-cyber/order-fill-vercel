import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { expect, test } from "@playwright/test";

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../qa");
const screenshotDir = path.join(qaRoot, "screenshots");

test.describe("QA infra smoke", () => {
  test("Chromium launches", async ({ page }) => {
    await page.setContent("<main><h1>Order Fill QA infra</h1></main>");
    await expect(page.getByRole("heading", { name: "Order Fill QA infra" })).toBeVisible();
  });

  test("login screen renders", async ({ page }) => {
    fs.mkdirSync(screenshotDir, { recursive: true });
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
    await expect(page.getByLabel("Логин")).toBeVisible();
    await expect(page.getByRole("textbox", { name: /Пароль/ })).toBeVisible();
    await page.screenshot({
      path: path.join(screenshotDir, "qa-infra-login.png"),
      fullPage: true,
    });
  });
});
