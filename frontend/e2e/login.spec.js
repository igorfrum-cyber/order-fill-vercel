import { expect, test } from "@playwright/test";

test("login form follows password length rules", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Вход" })).toBeVisible();

  const password = page.getByRole("textbox", { name: /Пароль/ });
  await page.getByLabel("Логин").fill("buyer");
  await password.fill("short");
  await expect(page.getByRole("button", { name: "Войти", exact: true })).toBeDisabled();

  await password.fill("long-enough-password");
  await expect(page.getByRole("button", { name: "Войти", exact: true })).toBeEnabled();
});
