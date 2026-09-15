import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

const { updateBrandRule } = vi.hoisted(() => ({ updateBrandRule: vi.fn(async (_brand, rule) => rule) }));

vi.mock("../../api/brands.js", () => ({
  listBrandRules: async () => ({
    rules: [{ brand: "klapp", label: "KLAPP", adjustment: "nearestMultiple", quantity_multiple: 3, min_quantity: 0, blank_layout: "", prefix_aliases: ["MT"], require_unit: null }],
  }),
  updateBrandRule,
}));

import { BrandRulesScreen } from "./BrandRulesScreen.jsx";

test("platform brand rules use the canonical admin table", async () => {
  render(<BrandRulesScreen />);
  expect(await screen.findByText("KLAPP")).toBeInTheDocument();
  expect(screen.getByText("Ближайшая кратность")).toBeInTheDocument();
  expect(screen.getByText("× 3")).toBeInTheDocument();
});

test("platform admin can inspect and save every brand rule field", async () => {
  const user = userEvent.setup();
  render(<BrandRulesScreen />);
  await screen.findByText("KLAPP");
  expect(screen.getByText("Префиксы артикулов")).toBeInTheDocument();
  expect(screen.getByText("Единица измерения")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Редактировать" }));
  await user.clear(screen.getByLabelText("Кратность"));
  await user.type(screen.getByLabelText("Кратность"), "6");
  await user.click(screen.getByRole("button", { name: "Сохранить" }));
  expect(updateBrandRule).toHaveBeenCalledWith("klapp", expect.objectContaining({ quantity_multiple: 6, prefix_aliases: ["MT"] }));
});
