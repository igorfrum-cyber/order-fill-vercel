import { expect, test } from "@playwright/test";

test("platform admin configures inbound receive address", async ({ page }) => {
  let savedAddress = "";

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: { id: "admin", login: "admin", role: "platform_admin", two_factor_enabled: true } });
      return;
    }
    if (path === "/api/v1/companies") {
      await route.fulfill({ json: { companies: [] } });
      return;
    }
    if (path === "/api/v1/inbound/settings" && request.method() === "GET") {
      await route.fulfill({ json: { enabled: true, receive_address: savedAddress || "", webhook_count: 12, error_count: 1, last_webhook_at: "2026-09-15T10:00:00Z" } });
      return;
    }
    if (path === "/api/v1/inbound/settings" && request.method() === "POST") {
      const body = request.postDataJSON();
      savedAddress = body.receive_address;
      await route.fulfill({ json: { enabled: body.enabled ?? true, receive_address: savedAddress, webhook_count: 12, error_count: 1 } });
      return;
    }
    if (path === "/api/v1/inbound/deliveries") {
      await route.fulfill({ json: { deliveries: [] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/inbound");
  await expect(page.getByRole("heading", { name: "Адрес приёма" })).toBeVisible();

  const addressInput = page.locator("input[type='email']");
  await addressInput.fill("newaddr@cloudmailin.net");
  await page.getByRole("button", { name: "Сохранить" }).click();

  await expect(addressInput).toHaveValue("newaddr@cloudmailin.net");
  expect(savedAddress).toBe("newaddr@cloudmailin.net");
});

test("company owner configures sender email and sees platform address", async ({ page }) => {
  let savedSender = "";

  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: { id: "u1", login: "owner", role: "company_owner", company_id: "c1", two_factor_enabled: true } });
      return;
    }
    if (path === "/api/v1/inbound/companies/c1" && request.method() === "GET") {
      await route.fulfill({ json: { company_id: "c1", receive_address: "7e1432246b724f3bcd6c@cloudmailin.net", sender_email: savedSender, enabled: true } });
      return;
    }
    if (path === "/api/v1/inbound/companies/c1" && request.method() === "POST") {
      const body = request.postDataJSON();
      savedSender = body.sender_email;
      await route.fulfill({ json: { company_id: "c1", receive_address: "7e1432246b724f3bcd6c@cloudmailin.net", sender_email: savedSender, enabled: body.enabled } });
      return;
    }
    if (path === "/api/v1/inbound/companies/c1/messages") {
      await route.fulfill({ json: { messages: [] } });
      return;
    }
    if (path === "/api/v1/inbound/settings") {
      await route.fulfill({ json: { enabled: true, receive_address: "7e1432246b724f3bcd6c@cloudmailin.net" } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/inbound?company=c1");
  await expect(page.getByText("7e1432246b724f3bcd6c@cloudmailin.net")).toBeVisible();

  await page.getByRole("button", { name: "Настроить" }).click();
  const senderInput = page.locator("input[type='email'][placeholder='1c@company.ru']");
  await senderInput.fill("1c@testcompany.ru");
  await page.getByRole("button", { name: "Сохранить" }).click();

  await expect(page.getByText("1c@testcompany.ru")).toBeVisible();
  expect(savedSender).toBe("1c@testcompany.ru");
});
