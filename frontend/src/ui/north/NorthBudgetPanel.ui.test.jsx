import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, test, vi } from "vitest";

import { NorthBudgetPanel } from "./NorthBudgetPanel.jsx";
import { planOrderBudget } from "../../api/budget.js";

vi.mock("../../api/budget.js", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, planOrderBudget: vi.fn() };
});

beforeEach(() => {
  planOrderBudget.mockReset();
});

test("buyer previews and applies a North budget from the backend", async () => {
  const user = userEvent.setup();
  const onApply = vi.fn();
  const onDiscount = vi.fn();
  planOrderBudget.mockResolvedValue({
    before: 1000, total: 1400, target: 1200, reason: "", complete: true,
    rows: [{ key: "A1", name: "Крем", category: "A", before: 10, quantity: 14, comment: "Добавилось 4 шт. Для закупа до суммы." }],
    lineSteps: [], lineGroups: [],
  });
  const row = {
    key: "A1", name: "Крем", variant: "", hasBudgetData: true, budgetCategory: "A",
    budgetDemand: 100, budgetPrice: 100, actualSupplierOrder: 10, tyumenStock: 0, tyumenInTransit: 0, northNeed: 0,
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

  expect(await screen.findByLabelText("Предпросмотр бюджета")).toHaveTextContent("10 → 14");
  expect(planOrderBudget).toHaveBeenCalledWith(
    expect.objectContaining({ brand: "skin_synergy", target: 1200, discount: 10, delivery_weeks: 1 }),
  );
  expect(planOrderBudget.mock.calls[0][0].rows[0]).toMatchObject({ key: "A1", base_price: 100, quantity: 10 });

  await user.click(screen.getByRole("button", { name: "Применить" }));
  expect(onApply).toHaveBeenCalledWith("A1", 14, "Добавилось 4 шт. Для закупа до суммы.");
  expect(onDiscount).toHaveBeenCalledWith("main", 10);
});

test("North PROFF forwards christina line metadata to the backend", async () => {
  const user = userEvent.setup();
  planOrderBudget.mockResolvedValue({
    before: 5640, total: 5880, target: 6100, reason: "", complete: true,
    rows: [{ key: "proff:MUSE-3", name: "MUSE 3", category: "A+", before: 3, quantity: 6, comment: "Добавилось 3 шт. Для закупа до суммы." }],
    lineSteps: [], lineGroups: [],
  });
  const line = { id: "MUSE", name: "MUSE", article: "3", required: ["0", "1", "2", "3"] };
  const row = {
    key: "proff:MUSE-3", name: "MUSE 3", variant: "proff", hasBudgetData: true,
    budgetCategory: "A+", budgetDemand: 10, budgetPrice: 100, actualSupplierOrder: 3,
    tyumenStock: 0, tyumenInTransit: 0, northNeed: 0, christinaLine: line,
  };
  render(
    <NorthBudgetPanel
      brand="christina"
      plan={{ summary: { deliveryWeeks: 1 } }}
      rows={[row]}
      actualValue={(item) => item.actualSupplierOrder}
      lockedKeys={new Set()}
      discounts={new Map()}
      onApply={() => {}}
      onDiscount={() => {}}
    />,
  );

  await user.click(screen.getByRole("button", { name: /Заказ до суммы/ }));
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));
  await user.type(screen.getByLabelText("Целевая сумма, ₽"), "6100");
  await user.click(screen.getByRole("button", { name: "Рассчитать" }));

  await screen.findByLabelText("Предпросмотр бюджета");
  const sent = planOrderBudget.mock.calls[0][0];
  expect(sent.rows[0]).toMatchObject({ key: "proff:MUSE-3", group: "proff", line });
});

test("North budget rejects malformed values before calling the backend", async () => {
  planOrderBudget.mockResolvedValue({total:100,before:100,complete:true,rows:[]});
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
  expect(await screen.findByRole("alert")).toHaveTextContent("от 0 до 99,99%");
  expect(planOrderBudget.mock.calls.filter(([request])=>!request.quote_only)).toHaveLength(0);
});

test("North selected retail column and discount apply even when quantity is unchanged", async () => {
 const user=userEvent.setup();const onPriceColumn=vi.fn();const onDiscount=vi.fn();
 planOrderBudget.mockResolvedValue({before:180,total:180,target:180,complete:true,reason:"",rows:[{key:"A1",name:"Cream",before:2,quantity:2}],lineSteps:[],lineGroups:[]});
 render(<NorthBudgetPanel brand="skin_synergy" plan={{summary:{deliveryWeeks:1}}} rows={[{key:"A1",name:"Cream",hasBudgetData:true,budgetCategory:"A",budgetDemand:10,budgetPrice:70,budgetPrices:[{column:3,label:"Цена розничная",price:100},{column:4,label:"Цена со скидкой",price:70}],actualSupplierOrder:2}]} actualValue={(row)=>row.actualSupplierOrder} lockedKeys={new Set()} discounts={new Map()} onApply={()=>{}} onDiscount={onDiscount} onPriceColumn={onPriceColumn}/ >);
 await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
 await user.selectOptions(screen.getByLabelText("Колонка цены"),"3");
 await user.clear(screen.getByLabelText("Скидка, %"));await user.type(screen.getByLabelText("Скидка, %"),"10");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 expect(planOrderBudget.mock.calls.map(([request])=>request).find(request=>!request.quote_only).rows[0].base_price).toBe(100);
 await user.click(screen.getByRole("button",{name:"Применить"}));
 expect(onPriceColumn).toHaveBeenCalledWith("main",3);expect(onDiscount).toHaveBeenCalledWith("main",10);
});

test("North current PROFF total uses an unchanged backend quote including set savings",async()=>{
 planOrderBudget.mockImplementation(async(request)=>({before:5640,total:5640,complete:true,rows:request.rows.map(row=>({...row,before:row.quantity}))}));
 const rows=[18,18,18,3].map((quantity,index)=>({key:`p${index}`,name:`MUSE ${index}`,variant:"proff",hasBudgetData:true,budgetPrice:100,budgetDemand:10,budgetCategory:"A",actualSupplierOrder:quantity,christinaLine:{id:"MUSE",article:String(index),required:["0","1","2","3"]}}));
 render(<NorthBudgetPanel brand="christina" plan={{summary:{deliveryWeeks:1}}} rows={rows} actualValue={row=>row.actualSupplierOrder} lockedKeys={new Set()} discounts={new Map()} onApply={()=>{}} onDiscount={()=>{}}/>);
 expect(await screen.findByText("5 640,00 ₽")).toBeVisible();
 expect(planOrderBudget.mock.calls[0][0].quote_only).toBe(true);
 expect(planOrderBudget.mock.calls[0][0].rows.map(row=>row.quantity)).toEqual([18,18,18,3]);
});

for(const [reason,flag,label] of [["overSix","allow_over_six","Разрешить запас больше 6 месяцев и рассчитать снова"],["belowOne","allow_below_one","Разрешить запас меньше 1 месяца и рассчитать снова"]]){
 test(`North ${reason} permission is explicit and revoked by target changes`,async()=>{
  const user=userEvent.setup();const requests=[];
  planOrderBudget.mockImplementation(async(request)=>{
   if(!request.quote_only)requests.push(request);
   return {before:6000,total:request.target || 6000,complete:request.quote_only || Boolean(request[flag]),reason:request.quote_only || request[flag] ? "" : reason,rows:[{key:"a",before:60,quantity:60}],lineSteps:[],lineGroups:[]};
  });
  render(<NorthBudgetPanel brand="skin_synergy" plan={{summary:{deliveryWeeks:1}}} rows={[{key:"a",name:"Cream",hasBudgetData:true,budgetPrice:100,budgetDemand:10,budgetCategory:"A",actualSupplierOrder:60}]} actualValue={row=>row.actualSupplierOrder} lockedKeys={new Set()} discounts={new Map()} onApply={()=>{}} onDiscount={()=>{}}/>);
  await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"6200");
  await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
  expect(screen.getByRole("button",{name:"Применить"})).toBeDisabled();
  await user.click(screen.getByLabelText(label));await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
  expect(requests.at(-1)[flag]).toBe(true);
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"6300");
  await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
  expect(requests.at(-1)[flag]).toBe(false);
 });
}
