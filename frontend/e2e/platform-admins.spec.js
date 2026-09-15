import { expect, test } from "@playwright/test";

test("primary platform admin invites a protected secondary administrator", async ({ page }) => {
  const admins = [
    { id: "root", login: "root", role: "platform_admin", is_primary_admin: true, activated: true },
  ];
  let invitedLogin = "";

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "root", login: "root", role: "platform_admin", is_primary_admin: true, two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/companies") {
      await route.fulfill({ json: { companies: [] } });
      return;
    }
    if (path === "/api/v1/platform-admins" && request.method() === "GET") {
      await route.fulfill({ json: { users: admins } });
      return;
    }
    if (path === "/api/v1/platform-admins" && request.method() === "POST") {
      invitedLogin = request.postDataJSON().login;
      admins.push({ id: "ops", login: invitedLogin, role: "platform_admin", is_primary_admin: false, activated: false });
      await route.fulfill({ status: 201, json: { user: admins[1], invite_url: "/invite/admin-token" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/users");
  await expect(page.getByRole("heading", { name: "Администраторы сервиса" })).toBeVisible();
  const primaryCard = page.getByRole("listitem").filter({ hasText: "root" });
  await expect(primaryCard).toContainText("Главный администратор");
  await expect(primaryCard.getByRole("button", { name: "Выключить" })).toHaveCount(0);

  await page.getByRole("textbox", { name: "Логин администратора" }).fill("ops-admin");
  await page.getByRole("button", { name: "Пригласить администратора" }).click();

  await expect(page.getByLabel("Ссылка-приглашение")).toHaveValue(/\/invite\/admin-token$/);
  await expect(page.getByRole("listitem").filter({ hasText: "ops-admin" })).toContainText("Ждёт активации");
  expect(invitedLogin).toBe("ops-admin");
});
