import { expect, test } from "@playwright/test";

test("platform admin inspects and updates a complete brand rule", async ({ page }) => {
	let savedRule;
  await page.route("**/api/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/v1/auth/me") {
      await route.fulfill({ json: { id: "admin-1", login: "platform", role: "platform_admin", two_factor_enabled: true } });
      return;
    }
    if (path === "/api/v1/companies") {
      await route.fulfill({ json: { companies: [] } });
      return;
    }
    if (path === "/api/v1/brand-rules") {
      await route.fulfill({ json: { rules: [
        { brand: "klapp", label: "KLAPP", adjustment: "nearestMultiple", quantity_multiple: 3 },
        { brand: "christina", label: "CHRISTINA", adjustment: "multiple", quantity_multiple: 3 },
      ] } });
      return;
    }
		if (path === "/api/v1/brand-rules/klapp" && route.request().method() === "POST") {
			savedRule = route.request().postDataJSON();
			await route.fulfill({ json: savedRule });
			return;
		}
    await route.fulfill({ status: 404, json: { message: `Unexpected ${path}` } });
  });

  await page.goto("/brand-rules");
  await expect(page.getByRole("heading", { name: "Правила брендов" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "KLAPP" })).toBeVisible();
  await expect(page.getByText("Ближайшая кратность").first()).toBeVisible();
  await expect(page.getByText("× 3").first()).toBeVisible();
  await expect(page.getByText("Префиксы артикулов").first()).toBeVisible();
  await page.getByRole("button", { name: "Редактировать" }).first().click();
  await page.getByRole("spinbutton", { name: "Кратность", exact: true }).fill("6");
  await page.getByRole("button", { name: "Сохранить" }).click();
  await expect(page.getByText("× 6").first()).toBeVisible();
  expect(savedRule).toMatchObject({ brand: "klapp", adjustment: "nearestMultiple", quantity_multiple: 6 });
});
