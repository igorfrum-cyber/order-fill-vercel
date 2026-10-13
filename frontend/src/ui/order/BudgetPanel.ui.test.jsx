import { useState } from "react";
import { render, screen, waitFor } from "@testing-library/react";
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

test("HOME and PROFF use independent pricing and apply separate budgets atomically", async () => {
 const user=userEvent.setup();const onEdit=vi.fn();const onPricingChange=vi.fn();
 const rows=[{key:"h",blankLabel:"HOME",blankId:"blank-1",editable:true,inserted:2,budgetPrice:100,budgetPrices:[{column:3,label:"Цена HOME",price:100}],budgetCategory:"A",budgetDemand:10},{...proffRow,key:"p",blankLabel:"PROFF",blankId:"blank-2",budgetPrices:[{column:5,label:"Цена PROFF",price:200}],budgetPrice:200}];
 let blocked=true;
 planOrderBudget.mockImplementation(async (payload)=>{
  if(payload.quote_only)return {total:850,before:850,complete:true,rows:payload.rows.map((row)=>({...row,before:row.quantity,price:100,discount:row.discount_override}))};
  const row=payload.rows[0];return {total:payload.target,before:100,complete:row.group!=="proff"||!blocked,reason:row.group==="proff"&&blocked?"overSix":"",rows:[{key:row.key,before:row.quantity,quantity:6,price:row.group==="home"?90:160,discount:row.discount_override}],lineGroups:[],lineSteps:[]};
 });
 render(<BudgetPanel brand="christina" rows={rows} edits={new Map([["h",{value:2}],["p",{value:3}]])} onEdit={onEdit} onPricingChange={onPricingChange}/>);
 expect(await screen.findByText("850,00 ₽")).toBeVisible();
 await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
 await user.click(screen.getByLabelText("Отдельные суммы HOME и PROFF"));
 await user.click(screen.getByLabelText("Вычесть скидку HOME"));await user.clear(screen.getByLabelText("Скидка HOME, %"));await user.type(screen.getByLabelText("Скидка HOME, %"),"10");
 await user.click(screen.getByLabelText("Вычесть скидку PROFF"));await user.clear(screen.getByLabelText("Скидка PROFF, %"));await user.type(screen.getByLabelText("Скидка PROFF, %"),"20");
 await user.type(screen.getByLabelText("Целевая сумма HOME, ₽"),"600");await user.type(screen.getByLabelText("Целевая сумма PROFF, ₽"),"900");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 expect(screen.getByRole("button",{name:"Применить"})).toBeDisabled();expect(onEdit).not.toHaveBeenCalled();
 blocked=false;
 await user.click(screen.getByLabelText("Разрешить запас больше 6 месяцев и рассчитать снова"));
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 await user.click(screen.getByRole("button",{name:"Применить"}));
 expect(onEdit).toHaveBeenCalledTimes(2);
 expect(onPricingChange).toHaveBeenCalledWith([{row_key:"h",price_column:3,base_price:100,price:90,discount:10},{row_key:"p",price_column:5,base_price:200,price:160,discount:20}]);
});

test("combined HOME PROFF target passes different discounts in one server request",async()=>{
 const user=userEvent.setup();planOrderBudget.mockImplementation(async(payload)=>({before:123,total:123,complete:true,rows:payload.rows.map(row=>({key:row.key,before:row.quantity,quantity:row.quantity,price:50,discount:row.discount_override})),lineSteps:[],lineGroups:[]}));
 const rows=[{key:"h",blankLabel:"HOME",editable:true,budgetPrice:100},{key:"p",blankLabel:"PROFF",editable:true,budgetPrice:200}];
 render(<BudgetPanel brand="christina" rows={rows} edits={new Map()} onEdit={()=>{}}/>);
 await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
 expect(screen.getByLabelText("Отдельные суммы HOME и PROFF")).not.toBeChecked();
 await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"300");
 await user.click(screen.getByLabelText("Вычесть скидку HOME"));await user.clear(screen.getByLabelText("Скидка HOME, %"));await user.type(screen.getByLabelText("Скидка HOME, %"),"10");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));
 const plan=planOrderBudget.mock.calls.map(([request])=>request).find(request=>!request.quote_only);
 expect(plan.target).toBe(300);expect(plan.rows).toHaveLength(2);expect(plan.rows[0].discount_override).toBe(10);expect(plan.rows[1].discount_override).toBe(0);
});

test("known northern source disables ordinary budget except ANGIO",async()=>{
 planOrderBudget.mockResolvedValue({total:100,rows:[],complete:true});
 const props={rows:[{key:"a",editable:true,budgetPrice:100}],edits:new Map(),onEdit:()=>{},sourceCity:"Сургут"};
 const {rerender}=render(<BudgetPanel {...props} brand="skin_synergy"/>);
 expect(screen.getByRole("button",{name:"Заказ до суммы"})).toBeDisabled();
 rerender(<BudgetPanel {...props} brand="angiopharm"/>);expect(screen.getByRole("button",{name:"Заказ до суммы"})).toBeEnabled();
});

function RepeatBudgetHarness() {
 const row={key:"a",blankName:"Cream",editable:true,inserted:10,budgetPrice:100,budgetCategory:"A",budgetDemand:100};
 const [edits,setEdits]=useState(new Map([["a",{value:10,comment:""}]]));
 return <BudgetPanel brand="angiopharm" rows={[row]} edits={edits} onEdit={(key,patch)=>setEdits(current=>new Map(current).set(key,{...current.get(key),...patch}))}/>;
}

test("second automatic budget remains adjustable and explicit pin prevents adjustment",async()=>{
 const user=userEvent.setup();const requests=[];
 planOrderBudget.mockImplementation(async(request)=>{
  const row=request.rows[0];
  if(!request.quote_only)requests.push(request);
  const quantity=request.quote_only || row.locked ? row.quantity : request.target/100;
  return {before:row.quantity*100,total:quantity*100,target:request.target,complete:true,rows:[{key:row.key,name:"Cream",before:row.quantity,quantity,price:100,discount:0}],lineSteps:[],lineGroups:[]};
 });
 render(<RepeatBudgetHarness/>);
 for(const target of ["1200","1400"]){
  await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
  await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),target);
  await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
  await user.click(screen.getByRole("button",{name:"Применить"}));
 }
 expect(requests[0].rows[0]).toMatchObject({quantity:10,locked:false});
 expect(requests[1].rows[0]).toMatchObject({quantity:12,locked:false});
 await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
 await user.click(screen.getByText("Закрепить позиции"));await user.click(screen.getByLabelText("Закрепить Cream"));
 await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"1500");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 expect(requests[2].rows[0]).toMatchObject({quantity:14,locked:true,excluded:true});
});

test("changing ordinary target revokes a previous coverage permission",async()=>{
 const user=userEvent.setup();const requests=[];
 planOrderBudget.mockImplementation(async(request)=>{
  if(!request.quote_only)requests.push(request);
  return {before:1000,total:request.target || 1000,complete:request.quote_only || request.allow_over_six,reason:request.quote_only || request.allow_over_six ? "" : "overSix",rows:[{key:"a",before:10,quantity:10}],lineSteps:[],lineGroups:[]};
 });
 render(<RepeatBudgetHarness/>);
 await user.click(screen.getByRole("button",{name:"Заказ до суммы"}));
 await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"6200");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 await user.click(screen.getByLabelText("Разрешить запас больше 6 месяцев и рассчитать снова"));
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 expect(requests.at(-1).allow_over_six).toBe(true);
 await user.clear(screen.getByLabelText("Целевая сумма, ₽"));await user.type(screen.getByLabelText("Целевая сумма, ₽"),"6300");
 await user.click(screen.getByRole("button",{name:"Рассчитать"}));await screen.findByLabelText("Предпросмотр бюджета");
 expect(requests.at(-1).allow_over_six).toBe(false);
});

test("resuming SOTHYS preserves each block price column",async()=>{planOrderBudget.mockResolvedValue({total:420,rows:[],complete:true});const rows=[{key:"a",editable:true,blankId:"blank-1",inserted:2,budgetPrice:100,budgetPrices:[{column:8,price:100,label:"Цена (по блоку объема)"}]},{key:"b",editable:true,blankId:"blank-1",inserted:2,budgetPrice:200,budgetPrices:[{column:13,price:200,label:"Цена (по блоку объема)"}]}];render(<BudgetPanel brand="sothys" rows={rows} edits={new Map(rows.map(r=>[r.key,{value:2}]))} onEdit={()=>{}} pricing={[{row_key:"a",price_column:8,discount:30},{row_key:"b",price_column:13,discount:30}]}/>);await waitFor(()=>expect(planOrderBudget).toHaveBeenCalled());expect(planOrderBudget.mock.calls[0][0].rows.map(r=>r.base_price)).toEqual([100,200]);});
