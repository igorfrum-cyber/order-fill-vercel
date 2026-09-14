import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("../../api/brands.js", () => ({
  listBrandRules: async () => ({
    rules: [{ brand: "klapp", label: "KLAPP", adjustment: "nearestMultiple", quantity_multiple: 3, blank_layout: "" }],
  }),
}));

import { BrandRulesScreen } from "./BrandRulesScreen.jsx";

test("platform brand rules use the canonical admin table", async () => {
  render(<BrandRulesScreen />);
  expect(await screen.findByText("KLAPP")).toBeInTheDocument();
  expect(screen.getByText("Ближайшая кратность")).toBeInTheDocument();
  expect(screen.getByText("× 3")).toBeInTheDocument();
});
