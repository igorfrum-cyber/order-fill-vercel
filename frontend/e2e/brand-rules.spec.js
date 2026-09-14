import { expect, test } from "@playwright/test";

test("platform admin opens the read-only brand rules page", async ({ page }) => {
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
    await route.fulfill({ status: 404, json: { message: `Unexpected ${path}` } });
  });

  await page.goto("/brand-rules");
  await expect(page.getByRole("heading", { name: "Правила брендов" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "KLAPP" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Ближайшая кратность" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "× 3" }).first()).toBeVisible();
  await expect(page.getByText("Изменения вносятся вместе с новой версией системы.")).toBeVisible();
});
