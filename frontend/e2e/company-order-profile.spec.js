import { expect, test } from "@playwright/test";

test("company owner validates and saves requisites used in supplier blanks", async ({ page }) => {
  let savedProfile;
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: {
        id: "owner-1", login: "owner", role: "company_owner", company_id: "company-1",
        company_name: "Тестовая компания", login_slug: "test-company", two_factor_enabled: true,
      } });
      return;
    }
    if (path === "/api/v1/companies/company-1/order-profile" && request.method() === "GET") {
      await route.fulfill({ json: {
        legal_name: "ООО Тест",
        consignee: "Склад Тест",
        address: "Тюмень, ул. Тестовая, 1",
        contact_name: "Иван Иванов",
        contact_phone: "+7 900 000-00-00",
        carrier: "Деловые линии",
        delivery_payer: "Получатель",
        delivery_destination: "До терминала",
        brand_terms: [{ brand: "angiopharm", discount_set: true, discount_basis_points: 3500 }],
      } });
      return;
    }
    if (path === "/api/v1/companies/company-1/order-profile" && request.method() === "POST") {
      savedProfile = request.postDataJSON();
      await route.fulfill({ json: savedProfile });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unexpected ${request.method()} ${path}` } });
  });

  await page.goto("/company");
  await expect(page.getByRole("heading", { name: "Данные для бланков заказа" })).toBeVisible();
  await expect(page.getByLabel("Юридическое лицо")).toHaveValue("ООО Тест");
  await expect(page.getByLabel("Скидка ANGIOPHARM, %")).toHaveValue("35");

  await page.getByLabel("Телефон получателя").fill("ошибка");
  await page.getByRole("button", { name: "Сохранить данные для бланков" }).click();
  await expect(page.getByRole("alert")).toContainText("Проверьте телефон");
  expect(savedProfile).toBeUndefined();

  await page.getByLabel("Телефон получателя").fill("+7 912 345-67-89");
  await page.getByLabel("Скидка ANGIOPHARM, %").fill("30,25");
  const responsePromise = page.waitForResponse((response) =>
    new URL(response.url()).pathname === "/api/v1/companies/company-1/order-profile" && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Сохранить данные для бланков" }).click();
  await responsePromise;
  await expect(page.getByText("Реквизиты сохранены и будут применены к новым заказам.")).toBeVisible();
  expect(savedProfile).toMatchObject({
    legal_name: "ООО Тест",
    contact_phone: "+7 912 345-67-89",
    brand_terms: expect.arrayContaining([
      expect.objectContaining({ brand: "angiopharm", discount_set: true, discount_basis_points: 3025 }),
    ]),
  });
});
