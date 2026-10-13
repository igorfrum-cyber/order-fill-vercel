import test from "node:test";
import assert from "node:assert/strict";

import { appendBudgetComment, budgetPatches, budgetRequestRows, budgetPricingEntries } from "./budgetWorkflow.js";

test("budgetRequestRows sends raw report numbers and locks a manual deviation", () => {
  const rows = [{
    key: "row-1", editable: true, hasBudgetData: true, inserted: 3,
    blankName: "Крем", blankBoxSize: "3", blankId: "proff", budgetCategory: "B", budgetDemand: 10,
    budgetPrice: 100, stock: "2", inTransit: "1", christinaLine: { id: "MUSE", name: "MUSE", article: "0", required: ["0"] },
  }];
  const input = budgetRequestRows(rows, new Map([["row-1", { value: 6, comment: "решение" }]]));
  assert.deepEqual(input[0], {
    key: "row-1", name: "Крем", category: "B", quantity: 6, base_price: 100,
    demand: 10, stock: 2, transit: 1, outbound: 0, box_size: 3,
    group: "proff", line: { id: "MUSE", name: "MUSE", article: "0", required: ["0"] },
    locked: true, excluded: false, unsafe: false,
  });
});

test("budgetRequestRows carries the base price without discounting it", () => {
  const rows = [{
    key: "row-1", editable: true, hasBudgetData: true, inserted: 2,
    blankName: "Крем", budgetCategory: "B", budgetDemand: 10, budgetPrice: 199.99,
  }];
  const [row] = budgetRequestRows(rows, new Map());
  assert.equal(row.base_price, 199.99);
  assert.equal(row.group, "main");
  assert.equal(row.line, null);
});

test("budgetPatches keeps the previous comment and takes the note from the plan row", () => {
  const edits = new Map([["row-1", { value: 3, comment: "проверено" }]]);
  const plan = { rows: [{ key: "row-1", name: "Крем", before: 3, quantity: 6, comment: "Добавилось 3 шт. Для закупа до суммы." }] };
  const patches = budgetPatches(plan, edits);
  assert.equal(patches[0].next.value, 6);
  assert.match(patches[0].next.comment, /^проверено; Добавилось 3 шт\./);
  assert.deepEqual(patches[0].previous, { value: 3, comment: "проверено" });
});

test("appendBudgetComment joins non-empty parts", () => {
  assert.equal(appendBudgetComment("ручная правка", "Уменьшено"), "ручная правка; Уменьшено");
  assert.equal(appendBudgetComment("", "Уменьшено"), "Уменьшено");
  assert.equal(appendBudgetComment("ручная правка", ""), "ручная правка");
});

test("budget pricing uses the selected original base and covers unchanged quantities", () => {
 const rows = [{key: "r1", editable: true, hasBudgetData: true, budgetPrice: 70,
  budgetPrices: [{column: 2, label: "Цена", price: 100}, {column: 3, label: "Цена со скидкой", price: 70}]}];
 const plan = {rows: [{key: "r1", before: 2, quantity: 2, price:70, discount:30}]};
 assert.deepEqual(budgetPricingEntries(rows, plan, 30, "2"), [{row_key: "r1", price_column: 2, base_price: 100, discount: 30, price: 70}]);
 assert.equal(budgetRequestRows(rows, new Map(), "2")[0].base_price, 100);
});

test("Christina PROFF uses the pricing group even when the job blank id is positional",()=>{const rows=[{key:"a",editable:true,hasBudgetData:true,blankId:"blank-1",blankLabel:"PROFF",budgetDemand:10,budgetCategory:"A",budgetPrice:100}];const [row]=budgetRequestRows(rows,new Map());assert.equal(row.group,"proff");});

test("budget keeps manual-only rows and marks reused source identity unsafe", () => {
 const rows=[{key:"home:1",editable:true,blankLabel:"HOME",sourceRow:4,inserted:2,budgetPrice:100},{key:"proff:1",editable:true,blankLabel:"PROFF",sourceRow:4,inserted:3,budgetPrice:200}];
 const result=budgetRequestRows(rows,new Map([["home:1",{value:2}],["proff:1",{value:3}]]),{home:{priceColumn:"",pricingMode:"gross",discount:"10"},proff:{priceColumn:"",pricingMode:"gross",discount:"20"}});
 assert.equal(result.length,2);assert.equal(result[0].unsafe,true);assert.equal(result[1].unsafe,true);
 assert.equal(result[0].discount_override,10);assert.equal(result[1].discount_override,20);
});

test("export pricing takes authoritative backend prices for independent groups",()=>{
 const rows=[{key:"h",blankLabel:"HOME",budgetPrices:[{column:2,price:100}],budgetPrice:100},{key:"p",blankLabel:"PROFF",budgetPrices:[{column:4,price:200}],budgetPrice:200}];
 const plan={rows:[{key:"h",price:90,discount:10},{key:"p",price:160,discount:20}]};
 assert.deepEqual(budgetPricingEntries(rows,plan,0,{home:{priceColumn:"2"},proff:{priceColumn:"4"}}),[{row_key:"h",price_column:2,base_price:100,discount:10,price:90},{row_key:"p",price_column:4,base_price:200,discount:20,price:160}]);
});

test("an explicit price column cannot silently fall back for a row missing it",()=>{
 const row={key:"a",editable:true,blankLabel:"HOME",budgetPrice:100,budgetPrices:[{column:2,price:100}]};
 const [input]=budgetRequestRows([row],new Map(),{home:{priceColumn:"3"}});
 assert.equal(input.base_price,0);
});

test("automatic budget quantities remain adjustable while manual deviations and explicit pins lock",()=>{
 const rows=[{key:"a",editable:true,inserted:10,budgetPrice:100}];
 let edits=new Map([["a",{value:10,comment:""}]]);
 for(const quantity of [12,14]){
  assert.equal(budgetRequestRows(rows,edits)[0].locked,false);
  const patch=budgetPatches({rows:[{key:"a",before:edits.get("a").value,quantity}]},edits)[0];
  assert.equal(patch.next.budgetApplied,true);
  edits=new Map(edits).set("a",patch.next);
 }
 assert.equal(budgetRequestRows(rows,edits)[0].locked,false);
 edits=new Map(edits).set("a",{value:13,budgetApplied:false});
 assert.equal(budgetRequestRows(rows,edits)[0].locked,true);
 assert.equal(budgetRequestRows(rows,new Map([["a",{value:10}]]),"",new Set(["a"]))[0].locked,true);
});

test("undo budget patches restore the previous automatic-edit marker",()=>{
 const edits=new Map([["a",{value:12,comment:"",budgetApplied:true}]]);
 const [patch]=budgetPatches({rows:[{key:"a",before:12,quantity:14}]},edits);
 assert.equal(patch.previous.budgetApplied,true);
});

test("an explicitly cleared order stays zero in the budget request",()=>{const rows=[{key:"a",editable:true,inserted:10,rounded:10,recommended:10,budgetPrice:100}];const [row]=budgetRequestRows(rows,new Map([["a",{value:"",comment:"убрать"}]]));assert.equal(row.quantity,0)});
