import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, test, vi } from "vitest";

import { BudgetPanel } from "./BudgetPanel.jsx";
import { planOrderBudget } from "../../api/budget.js";

vi.mock("../../api/budget.js", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, planOrderBudget: vi.fn() };
});

beforeEach(() => {
  planOrderBudget.mockReset();
});

const proffRow = {
  key: "MUSE:3", editable: true, hasBudgetData: true, inserted: 3,
  blankName: "MUSE 3", blankId: "proff", budgetCategory: "A+", budgetDemand: 10,
  budgetPrice: 100, stock: 0, inTransit: 0, blankBoxSize: 3,
  christinaLine: { id: "MUSE", name: "MUSE", article: "3", required: ["0", "1", "2", "3"] },
};

test("buyer previews a PROFF budget and applies it from the backend response", async () => {
  const user = userEvent.setup();
  const onEdit = vi.fn();
  planOrderBudget.mockResolvedValue({
    before: 5640, total: 5880, target: 6100, reason: "", complete: true,
    rows: [{ key: "MUSE:3", name: "MUSE 3", category: "A+", before: 3, quantity: 6, comment: "Добавилось 3 шт. Для закупа до суммы. Дополнение комплектов MUSE." }],
    lineSteps: [{ id: "MUSE", name: "MUSE", sets: 6, added: 300, saved: 60 }],
    lineGroups: [{ id: "MUSE", name: "MUSE", valid: true, sets: 6, saving: 120, net: 5880 }],
  });

  render(
    <BudgetPanel
      brand="christina"
      deliveryWeeks={1}
      rows={[proffRow]}
      edits={new Map([["MUSE:3", { value: 3, comment: "" }]])}
      onEdit={onEdit}
    />,
  );

  await user.click(screen.getByRole("button", { name: "Заказ до суммы" }));
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));
  await user.type(screen.getByLabelText("Целевая сумма, ₽"), "6100");
  await user.click(screen.getByRole("button", { name: "Рассчитать" }));

  const preview = await screen.findByLabelText("Предпросмотр бюджета");
  expect(preview).toHaveTextContent("3 → 6");
  expect(preview).toHaveTextContent("60,00"); // set discount from line_steps
  expect(preview).toHaveTextContent("MUSE");
  expect(planOrderBudget).toHaveBeenCalledWith(
    expect.objectContaining({ brand: "christina", target: 6100, delivery_weeks: 1 }),
  );
  expect(planOrderBudget.mock.calls[0][0].rows[0]).toMatchObject({
    key: "MUSE:3", base_price: 100, group: "proff", line: expect.objectContaining({ id: "MUSE" }),
  });

  await user.click(screen.getByRole("button", { name: "Применить" }));
  expect(onEdit).toHaveBeenCalledTimes(1);
  const [key, patch] = onEdit.mock.calls[0];
  expect(key).toBe("MUSE:3");
  expect(patch.value).toBe(6);
  expect(patch.comment).toContain("Дополнение комплектов MUSE.");
});
