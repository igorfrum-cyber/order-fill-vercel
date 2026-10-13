import { useEffect, useMemo, useState } from "react";
import { planOrderBudget } from "../../api/budget.js";
import { budgetTargetValue, discountValue } from "../../features/order/budgetInput.js";
import { budgetPatches, budgetRequestRows, budgetPricingEntries, budgetGroup } from "../../features/order/budgetWorkflow.js";
import { GhostButton, Modal } from "../widgets.jsx";
import { ChristinaCompare } from "./ChristinaCompare.jsx";

const money = (value) => Number(value || 0).toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const targetInput = (value) => /^\d{0,12}(?:[.,]\d{0,2})?$/.test(String(value).replace(/\s/g, ""));
const discountInput = (value) => /^\d{0,2}(?:[.,]\d{0,2})?%?$/.test(String(value).trim());

export function BudgetPanel({ sourceCity = "", brand, deliveryWeeks, rows, edits, onEdit, pricing = [], onPricingChange = () => {} }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState("");
  const groups = useMemo(() => [...new Set(rows.filter((row) => row.editable !== false).map(budgetGroup))], [rows]);
  const [appliedSettings, setAppliedSettings] = useState(() => initialSettings(rows, pricing, brand));
  const [settings, setSettings] = useState(() => initialSettings(rows, pricing, brand));
  const [separate, setSeparate] = useState(false);
  const [groupTargets, setGroupTargets] = useState({});
  const [plan, setPlan] = useState(null);
  const [error, setError] = useState("");
  const [inputError, setInputError] = useState("");
  const [undo, setUndo] = useState(null);
  const [allowOverSix, setAllowOverSix] = useState(false);
  const [allowBelowOne, setAllowBelowOne] = useState(false);
  const [calculating, setCalculating] = useState(false);
  const [quote, setQuote] = useState(null);
  const [lockedKeys,setLockedKeys]=useState(()=>new Set());
  const input = useMemo(() => budgetRequestRows(rows, edits, open ? settings : appliedSettings, lockedKeys), [rows, edits, open, settings, appliedSettings, lockedKeys]);
  const quoteKey = JSON.stringify({brand, deliveryWeeks, rows:input});
  useEffect(() => {
    let active=true;
    const request=JSON.parse(quoteKey);
    if (!request.rows.length) return;
    planOrderBudget({brand:request.brand,delivery_weeks:request.deliveryWeeks || 0,rows:request.rows,discount:0,target:0,quote_only:true})
      .then((result) => {if(active)setQuote({key:quoteKey,result});})
      .catch((err) => {if(active)setQuote({key:quoteKey,error:err.message || "Не удалось получить сумму заказа."});});
    return () => {active=false;};
  },[quoteKey]);
  if (!input.length) return null;
  const total = quote?.key === quoteKey ? quote.result?.total : null;
  const quoteError = quote?.key === quoteKey ? quote.error : "";
  const totalLabel = total == null ? "Считаю сумму…" : `${money(total)} ₽`;
  const budgetAllowed = !sourceCity || /тюмен|tyumen/i.test(sourceCity) || brand === "angiopharm";
  const changed = plan?.rows.filter((row) => row.quantity !== row.before) || [];

  function resetPreview() {setPlan(null);setError("");}
  function changeSettings(group,patch) {
    setSettings((current) => ({...current,[group]:{...groupSettings(current,group),...patch}}));
    setAllowOverSix(false);setAllowBelowOne(false);setInputError("");resetPreview();
  }

  async function calculate() {
    setCalculating(true);
    try {
      for(const group of groups) {const config=groupSettings(settings,group);if(config.pricingMode==="gross")discountValue(config.discount);}
      const requestRows=budgetRequestRows(rows,edits,settings,lockedKeys);
      const base={brand,discount:0,delivery_weeks:deliveryWeeks || 0,allow_over_six:allowOverSix,allow_below_one:allowBelowOne};
      const requests=separate && groups.length>1 ? groups.map((group) => ({...base,target:budgetTargetValue(groupTargets[group] || ""),rows:requestRows.filter((row)=>row.group===group)})) : [{...base,target:budgetTargetValue(target),rows:requestRows}];
      const plans=await Promise.all(requests.map((request)=>planOrderBudget(request)));
      const next=combinePlans(plans, separate ? groups : []);
      setPlan(next);setError(next.reason ? budgetReason(next.reason) : "");setInputError("");
    } catch(err) {setPlan(null);setError(err.message || "Не удалось пересчитать заказ.");}
    finally {setCalculating(false);}
  }

  function apply() {
    if (!plan.complete) return;
    const patches=budgetPatches(plan,edits);
    setUndo({patches,settings:appliedSettings,pricing});
    for(const patch of patches)onEdit(patch.key,patch.next);
    onPricingChange(budgetPricingEntries(rows,plan,0,settings));
    setAppliedSettings(structuredClone(settings));setOpen(false);resetPreview();
  }

  function undoLast() {
    for(const patch of undo?.patches || [])onEdit(patch.key,patch.previous);
    setAppliedSettings(undo?.settings || {});onPricingChange(undo?.pricing || []);setUndo(null);
  }

  function openDialog() {
    setTarget(total == null ? "" : String(Math.round(total)));
    setSettings(structuredClone(appliedSettings));setAllowOverSix(false);setAllowBelowOne(false);
    setInputError("");resetPreview();setOpen(true);
  }

  const canCalculate = !calculating && groups.every((group) => {
    const config=groupSettings(settings,group);
    const value=separate && groups.length>1 ? groupTargets[group] || "" : target;
    return validBudgetInputs(value,config.pricingMode,config.discount);
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="text-[13px] text-[var(--color-ink-soft)]">
        Заказ: <strong className="font-mono text-[var(--color-ink)]">{totalLabel}</strong>
      </span>
      <GhostButton onClick={openDialog} disabled={!budgetAllowed}>Заказ до суммы</GhostButton>
      {!budgetAllowed ? <span className="text-[13px] text-[var(--color-ink-soft)]">Расчёт до суммы доступен для Тюмени. Северные заказы — в «Соединить бланки».</span> : null}
      {undo ? <button type="button" onClick={undoLast} className="cursor-pointer text-[13px] text-[var(--color-brand-strong)] hover:underline">Отменить перерасчёт</button> : null}
      {open ? (
        <Modal
          title="Заказ до суммы"
          cancelLabel="Закрыть"
          confirmLabel={plan ? "Применить" : "Рассчитать"}
          confirmDisabled={calculating || (plan ? !plan.complete : !canCalculate)}
          onCancel={() => setOpen(false)}
          onConfirm={plan ? apply : calculate}
          wide
        >
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3">
              <span className="block text-[13px]">Текущий заказ</span>
              <strong className="mt-1 block font-mono text-[20px] text-[var(--color-ink)]">{totalLabel}</strong>
            </div>
            {(!separate || groups.length<2) ? <TargetInput label="Целевая сумма, ₽" value={target} onChange={(value)=>{setTarget(value);setAllowOverSix(false);setAllowBelowOne(false);resetPreview();}} onInvalid={setInputError}/> : null}
          </div>
          {brand === "christina" && groups.length>1 ? <label className="mt-4 flex items-center gap-2 font-medium"><input type="checkbox" checked={separate} onChange={(event)=>{setSeparate(event.target.checked);setAllowOverSix(false);setAllowBelowOne(false);resetPreview();}}/>Отдельные суммы HOME и PROFF</label> : null}
          {separate && groups.length>1 ? <div className="mt-3 grid gap-3 sm:grid-cols-2">{groups.map((group)=><TargetInput key={group} label={`Целевая сумма ${group.toUpperCase()}, ₽`} value={groupTargets[group] || ""} onChange={(value)=>{setGroupTargets((current)=>({...current,[group]:value}));setAllowOverSix(false);setAllowBelowOne(false);resetPreview();}} onInvalid={setInputError}/>)}</div> : null}
          <div className="mt-4 grid gap-3 sm:grid-cols-2">{groups.map((group)=>{
            const config=groupSettings(settings,group);
            const prices=brand === "sothys" ? [] : [...new Map(rows.filter((row)=>budgetGroup(row)===group).flatMap((row)=>row.budgetPrices || []).map((price)=>[String(price.column),price])).values()];
            const suffix=groups.length>1 ? ` ${group.toUpperCase()}` : "";
            return <fieldset key={group} className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3">
              <legend className="font-medium">{groups.length>1 ? group.toUpperCase() : "Как указана цена в бланке"}</legend>
              {prices.length>1 ? <label className="block font-medium">Колонка цены{suffix}<select className="input mt-2" value={config.priceColumn} onChange={(event)=>changeSettings(group,{priceColumn:event.target.value})}><option value="">Распознанная цена</option>{prices.map((price)=><option key={price.column} value={price.column}>{price.label}</option>)}</select></label> : null}
              <div className="mt-2 space-y-2">{[["net","Цена уже со скидкой"],["gross","Вычесть скидку"]].map(([value,label])=><label key={value} className="flex items-center gap-2"><input type="radio" name={`budget-price-mode-${group}`} value={value} checked={config.pricingMode===value} onChange={()=>changeSettings(group,{pricingMode:value})}/>{label}{suffix}</label>)}</div>
              {config.pricingMode==="gross" ? <label className="mt-3 block font-medium">Скидка{suffix}, %<input className="input mt-2 font-mono" inputMode="decimal" value={config.discount} onChange={(event)=>{const value=event.target.value;if(!discountInput(value)){setInputError("Скидка: число от 0 до 99,99 с двумя знаками после запятой.");return;}changeSettings(group,{discount:value});}}/></label> : null}
            </fieldset>;
          })}</div>
          <p className="mt-2 text-[13px]">Согласованная цена попадёт в скачиваемый бланк. CHRISTINA сохраняет цены поставщика. Ручные правки остаются закреплёнными.</p>
          <details className="mt-3 rounded-xl border border-[var(--color-line)] px-4 py-3"><summary className="cursor-pointer font-medium">Закрепить позиции</summary><div className="mt-2 space-y-2">{input.map((row)=>{
            const manual=row.locked && !lockedKeys.has(row.key);
            return <label key={row.key} className="flex items-center gap-2 text-[13px]"><input type="checkbox" aria-label={`Закрепить ${row.name}`} checked={row.locked} disabled={manual} onChange={(event)=>{const checked=event.target.checked;setLockedKeys((current)=>{const next=new Set(current);if(checked)next.add(row.key);else next.delete(row.key);return next;});setAllowOverSix(false);setAllowBelowOne(false);resetPreview();}}/>{row.name}{manual ? " · ручное количество закреплено" : ""}</label>;
          })}</div></details>
          {quoteError ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{quoteError}</p> : null}
          {inputError ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{inputError}</p> : null}
          {error ? <p role="alert" className="mt-3 text-[var(--color-danger)]">{error}</p> : null}

          {plan?.reason === "overSix" ? (
            <label className="mt-3 flex cursor-pointer items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowOverSix} onChange={(event) => { setAllowOverSix(event.target.checked); resetPreview(); }} />
              Разрешить запас больше 6 месяцев и рассчитать снова
            </label>
          ) : null}
          {plan?.reason === "belowOne" ? (
            <label className="mt-3 flex cursor-pointer items-start gap-2 text-[var(--color-ink)]">
              <input type="checkbox" checked={allowBelowOne} onChange={(event) => { setAllowBelowOne(event.target.checked); resetPreview(); }} />
              Разрешить запас меньше 1 месяца и рассчитать снова
            </label>
          ) : null}

          {plan ? (
            <div className="mt-4" aria-label="Предпросмотр бюджета" aria-live="polite">
              <div className="flex flex-wrap items-end justify-between gap-2 rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3">
                <div>
                  <span className="block text-[13px]">После расчёта</span>
                  <strong className="font-mono text-[20px] text-[var(--color-ink)]">{money(plan.before)} ₽ → {money(plan.total)} ₽</strong>
                </div>
                <span className="text-[13px]">Изменено позиций: {changed.length}</span>
              </div>
              {plan.groupPlans?.length>1 ? <div className="mt-2 grid gap-2 sm:grid-cols-2">{plan.groupPlans.map((groupPlan)=><div key={groupPlan.group} className="rounded-lg border border-[var(--color-line-soft)] px-3 py-2"><strong>{groupPlan.group.toUpperCase()}</strong><span className="ml-2 font-mono">{money(groupPlan.before)} ₽ → {money(groupPlan.total)} ₽</span></div>)}</div> : null}
              <ChristinaCompare plan={plan} />
              {plan.lineSteps?.length ? (
                <details className="mt-2 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3">
                  <summary className="cursor-pointer font-medium text-[var(--color-ink)]">
                    PROFF: комплекты и дополнительная скидка
                  </summary>
                  <div className="mt-2 overflow-x-auto">
                    <table className="w-full text-[13px]">
                      <thead>
                        <tr className="text-left text-[var(--color-ink-soft)]">
                          <th className="pb-1 pr-4 font-medium">Линия</th>
                          <th className="pb-1 pr-4 font-medium">Комплектов</th>
                          <th className="pb-1 pr-4 font-medium">Доп. скидка, ₽</th>
                          <th className="pb-1 font-medium">Добавлено, ₽</th>
                        </tr>
                      </thead>
                      <tbody>
                        {plan.lineSteps.map((step) => (
                          <tr key={step.id} className="border-t border-[var(--color-line-soft)]">
                            <td className="py-1 pr-4 text-[var(--color-ink)]">{step.name}</td>
                            <td className="py-1 pr-4 font-mono">{step.sets}</td>
                            <td className="py-1 pr-4 font-mono">{money(step.saved)}</td>
                            <td className="py-1 font-mono">{money(step.added)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </details>
              ) : null}
              {(() => {
                const lines = plan.lineGroups || [];
                if (!lines.length) return null;
                return (
                  <details className="mt-2 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3">
                    <summary className="cursor-pointer font-medium text-[var(--color-ink)]">
                      PROFF: состояние линеек
                    </summary>
                    <div className="mt-2 overflow-x-auto">
                      <table className="w-full text-[13px]">
                        <thead>
                          <tr className="text-left text-[var(--color-ink-soft)]">
                            <th className="pb-1 pr-4 font-medium">Линия</th>
                            <th className="pb-1 pr-4 font-medium">Полных комплектов</th>
                            <th className="pb-1 pr-4 font-medium">Доп. скидка, ₽</th>
                            <th className="pb-1 font-medium">Закупка, ₽</th>
                          </tr>
                        </thead>
                        <tbody>
                          {lines.map((l) => (
                            <tr key={l.id} className="border-t border-[var(--color-line-soft)]">
                              <td className="py-1 pr-4 text-[var(--color-ink)]">{l.name}</td>
                              <td className="py-1 pr-4 font-mono">{l.valid ? l.sets : "Состав не подтверждён"}</td>
                              <td className="py-1 pr-4 font-mono">{money(l.saving)}</td>
                              <td className="py-1 font-mono">{money(l.net)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </details>
                );
              })()}
              {changed.length ? (
                <div className="mt-2 space-y-1">
                  {changed.map((row) => (
                    <div key={row.key} className="grid gap-1 rounded-lg border border-[var(--color-line-soft)] bg-[var(--color-surface)] px-3 py-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-4">
                      <span className="truncate text-[var(--color-ink)]">{row.name} · {row.category}</span>
                      <span className="font-mono text-[var(--color-ink)]">{row.before} → {row.quantity}</span>
                    </div>
                  ))}
                </div>
              ) : <p className="mt-2">Заказ уже соответствует указанной сумме.</p>}
            </div>
          ) : null}
        </Modal>
      ) : null}
    </div>
  );
}

function budgetReason(reason) {
  if (reason === "overSix") return "Расчёт остановлен у запаса в 6 месяцев.";
  if (reason === "belowOne") return "Расчёт остановлен у защитного запаса в 1 месяц.";
  return reason;
}

function validBudgetInputs(target, pricingMode, discount) {
  try {
    budgetTargetValue(target);
    if (pricingMode === "gross") discountValue(discount);
    return true;
  } catch {
    return false;
  }
}

function groupSettings(settings,group) {return {priceColumn:"",pricingMode:"net",discount:"30",...settings[group]};}

function initialSettings(rows,pricing,brand) {
 const result={};const byKey=new Map(rows.map((row)=>[row.key,row]));
 for(const entry of pricing){const row=byKey.get(entry.row_key);if(row)result[budgetGroup(row)]={priceColumn:brand === "sothys" ? "" : String(entry.price_column),pricingMode:entry.discount>0?"gross":"net",discount:String(entry.discount || 30)};}
 return result;
}

function combinePlans(plans,groups) {
 const result={...(plans.find((plan)=>plan.lineGroups?.length || plan.lineSteps?.length) || plans[0])};
 return {...result,before:plans.reduce((sum,plan)=>sum+plan.before,0),total:plans.reduce((sum,plan)=>sum+plan.total,0),target:plans.reduce((sum,plan)=>sum+(plan.target || 0),0),complete:plans.every((plan)=>plan.complete),reason:plans.find((plan)=>!plan.complete)?.reason || "",rows:plans.flatMap((plan)=>plan.rows),lineSteps:plans.flatMap((plan)=>plan.lineSteps || []),lineGroups:plans.flatMap((plan)=>plan.lineGroups || []),groupPlans:groups.map((group,index)=>({group,...plans[index]}))};
}

function TargetInput({label,value,onChange,onInvalid}) {
 return <label className="block rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 font-medium text-[var(--color-ink)]">{label}<input inputMode="decimal" autoComplete="off" className="input mt-2 font-mono" value={value} onChange={(event)=>{const next=event.target.value;if(!targetInput(next)){onInvalid("Сумма: только цифры и не больше двух знаков после запятой.");return;}onInvalid("");onChange(next);}}/></label>;
}
