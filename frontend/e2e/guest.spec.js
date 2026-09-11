import { expect, test } from "@playwright/test";

test("signed-out visitor sees login on app routes", async ({ page }) => {
  await page.goto("/overview");
  await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
  await page.goto("/jobs");
  await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
  await page.goto("/users");
  await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
});

test("unknown company login path still shows the login form", async ({ page }) => {
  await page.goto("/c/no-such-company");
  await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();
});
