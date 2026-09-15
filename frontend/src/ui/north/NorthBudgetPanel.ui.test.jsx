import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

import { NorthBudgetPanel } from "./NorthBudgetPanel.jsx";

test("buyer previews and applies a North budget with discount", async () => {
  const user = userEvent.setup();
  const onApply = vi.fn();
  const onDiscount = vi.fn();
  const row = {
    key: "A1",
    name: "Крем",
    variant: "",
    hasBudgetData: true,
    budgetCategory: "A",
    budgetDemand: 100,
    budgetPrice: 100,
    actualSupplierOrder: 10,
    tyumenStock: 0,
    tyumenInTransit: 0,
    northNeed: 0,
  };
  render(
    <NorthBudgetPanel
      brand="skin_synergy"
      plan={{ summary: { deliveryWeeks: 1 } }}
      rows={[row]}
      actualValue={(item) => item.actualSupplierOrder}
      lockedKeys={new Set()}
      discounts={new Map()}
      onApply={onApply}
      onDiscount={onDiscount}
    />,
  );

  await user.click(screen.getByRole("button", { name: "Заказ до суммы" }));
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));
  await user.type(screen.getByLabelText("Целевая сумма, ₽"), "1200");
  await user.clear(screen.getByLabelText("Скидка, %"));
  await user.type(screen.getByLabelText("Скидка, %"), "10");
  await user.click(screen.getByRole("button", { name: "Рассчитать" }));

  expect(screen.getByLabelText("Предпросмотр бюджета")).toHaveTextContent("10 → 14");
  await user.click(screen.getByRole("button", { name: "Применить" }));
  expect(onApply).toHaveBeenCalledWith("A1", 14, "Добавилось 4 шт. Для закупа до суммы.");
  expect(onDiscount).toHaveBeenCalledWith("main", 10);
});

test("North budget rejects malformed values", async () => {
  const user = userEvent.setup();
  render(
    <NorthBudgetPanel
      brand="skin_synergy"
      plan={{ summary: { deliveryWeeks: 1 } }}
      rows={[{ key: "A1", name: "Крем", hasBudgetData: true, budgetCategory: "A", budgetDemand: 10, budgetPrice: 100 }]}
      actualValue={() => 1}
      lockedKeys={new Set()}
      discounts={new Map()}
      onApply={() => {}}
      onDiscount={() => {}}
    />,
  );
  await user.click(screen.getByRole("button", { name: "Заказ до суммы" }));
  await user.clear(screen.getByLabelText("Скидка, %"));
  await user.type(screen.getByLabelText("Скидка, %"), "100");
  await user.click(screen.getByRole("button", { name: "Рассчитать" }));
  expect(screen.getByRole("alert")).toHaveTextContent("от 0 до 99,99%");
});
