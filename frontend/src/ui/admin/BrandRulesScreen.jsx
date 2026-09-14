import { useEffect, useState } from "react";
import { listBrandRules, updateBrandRule } from "../../api/brands.js";
import { userFacingError } from "../../features/help/errors.js";
import { Field, GhostButton, Modal } from "../widgets.jsx";

const adjustments = [
  ["none", "Без округления"],
  ["box", "До коробки из бланка"],
  ["multiple", "Вверх до кратности"],
  ["nearestMultiple", "Ближайшая кратность"],
  ["minimum", "Минимальная партия"],
];
const adjustmentLabels = Object.fromEntries(adjustments);

export function BrandRulesScreen() {
  const [rules, setRules] = useState([]);
  const [draft, setDraft] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    listBrandRules()
      .then((payload) => setRules(payload.rules || []))
      .catch((err) => setError(userFacingError(err, "Не удалось загрузить правила брендов.")))
      .finally(() => setLoading(false));
  }, []);

  async function save() {
    const { error: validationError, rule } = validatedRule(draft);
    if (validationError) {
      setError(validationError);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const updated = await updateBrandRule(rule.brand, rule);
      setRules((current) => current.map((item) => (item.brand === updated.brand ? updated : item)));
      setDraft(null);
    } catch (err) {
      setError(userFacingError(err, "Не удалось сохранить правило бренда."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <section className="animate-enter mx-auto max-w-6xl space-y-4 p-6">
        <div>
          <h1 className="text-[22px] font-semibold">Правила брендов</h1>
          <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Параметры применяются к новым обработкам. После изменения проверьте результат на реальном бланке.</p>
        </div>
        {error && !draft ? <p role="alert" className="rounded-xl bg-[var(--color-danger-soft)] p-4 text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        {loading ? <div className="h-24 animate-pulse rounded-xl bg-[var(--color-line-soft)]" /> : null}
        {!loading && (!error || draft) ? <div className="grid gap-3 lg:grid-cols-2">{rules.map((rule) => <RuleCard key={rule.brand} rule={rule} onEdit={() => { setError(""); setDraft(toDraft(rule)); }} />)}</div> : null}
      </section>
      {draft ? (
        <Modal title={`Правила ${draft.label || draft.brand}`} wide cancelLabel="Отмена" confirmLabel={saving ? "Сохраняю…" : "Сохранить"} confirmDisabled={saving} onCancel={() => { setDraft(null); setError(""); }} onConfirm={save}>
          {error ? <p role="alert" className="mb-4 rounded-lg bg-[var(--color-danger-soft)] p-3 text-[var(--color-danger)]">{error}</p> : null}
          <RuleForm rule={draft} onChange={setDraft} />
        </Modal>
      ) : null}
    </>
  );
}

function RuleCard({ rule, onEdit }) {
  return (
    <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
      <div className="flex items-start justify-between gap-3">
        <div><h2 className="text-[17px] font-semibold">{rule.label || rule.brand}</h2><p className="font-mono text-[12px] text-[var(--color-ink-faint)]">{rule.brand}</p></div>
        <GhostButton onClick={onEdit}>Редактировать</GhostButton>
      </div>
      <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 text-[13px]">
        <RuleValue label="Расчёт" value={adjustmentLabels[rule.adjustment] || rule.adjustment} />
        <RuleValue label="Вариант" value={rule.variant || "основной"} mono />
        <RuleValue label="Кратность" value={rule.quantity_multiple ? `× ${rule.quantity_multiple}` : "—"} mono />
        <RuleValue label="Минимум" value={rule.min_quantity || "—"} mono />
        <RuleValue label="Подпись расчёта" value={rule.adjustment_label || "—"} />
        <RuleValue label="Комментарий" value={rule.adjustment_comment || "—"} />
        <RuleValue label="Макет бланка" value={rule.blank_layout || "обычный"} />
        <RuleValue label="Колонка количества" value={rule.blank_quantity_header || "авто"} mono />
        <RuleValue label="Колонка коробки" value={rule.blank_box_header || "авто"} mono />
        <RuleValue label="Префиксы артикулов" value={rule.prefix_aliases?.join(", ") || "—"} mono />
        <RuleValue label="Дефис в артикуле" value={rule.preserve_hyphen ? "сохранять" : "нормализовать"} />
        <RuleValue label="Малый заказ" value={rule.allow_small_positive_order ? "разрешён" : "обычные правила"} />
        <RuleValue label="Единица измерения" value={rule.require_unit == null ? "авто" : rule.require_unit ? "обязательна" : "не обязательна"} />
      </dl>
    </article>
  );
}

function RuleValue({ label, value, mono = false }) {
  return <div><dt className="text-[var(--color-ink-faint)]">{label}</dt><dd className={`mt-0.5 text-[var(--color-ink-soft)] ${mono ? "font-mono" : ""}`}>{String(value)}</dd></div>;
}

function RuleForm({ rule, onChange }) {
  const set = (key, value) => onChange({ ...rule, [key]: value });
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Field label="Название"><input className="input w-full" maxLength={80} value={rule.label} onChange={(e) => set("label", e.target.value)} /></Field>
      <Field label="Алгоритм расчёта"><select className="input w-full" value={rule.adjustment} onChange={(e) => set("adjustment", e.target.value)}>{adjustments.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></Field>
      <NumberField label="Кратность" value={rule.quantity_multiple} max={10000} onChange={(value) => set("quantity_multiple", value)} />
      <NumberField label="Минимальное количество" value={rule.min_quantity} max={1000000} onChange={(value) => set("min_quantity", value)} />
      <TextField label="Подпись расчёта" value={rule.adjustment_label} onChange={(value) => set("adjustment_label", value)} />
      <TextField label="Комментарий к округлению" value={rule.adjustment_comment} onChange={(value) => set("adjustment_comment", value)} />
      <TextField label="Ключ колонки количества" value={rule.blank_quantity_header} onChange={(value) => set("blank_quantity_header", value)} />
      <TextField label="Ключ колонки коробки" value={rule.blank_box_header} onChange={(value) => set("blank_box_header", value)} />
      <TextField label="Макет бланка" value={rule.blank_layout} onChange={(value) => set("blank_layout", value)} />
      <Field label="Префиксы артикулов через запятую"><input className="input w-full" maxLength={659} value={rule.prefix_aliases_text} onChange={(e) => set("prefix_aliases_text", e.target.value)} placeholder="MT, ABC" /></Field>
      <Check label="Сохранять дефис в артикулах" checked={rule.preserve_hyphen} onChange={(value) => set("preserve_hyphen", value)} />
      <Check label="Разрешать малый положительный заказ" checked={rule.allow_small_positive_order} onChange={(value) => set("allow_small_positive_order", value)} />
      <Field label="Требовать единицу измерения"><select className="input w-full" value={rule.require_unit == null ? "auto" : String(rule.require_unit)} onChange={(e) => set("require_unit", e.target.value === "auto" ? null : e.target.value === "true")}><option value="auto">Автоматически</option><option value="true">Да</option><option value="false">Нет</option></select></Field>
    </div>
  );
}

function TextField({ label, value, onChange }) {
  return <Field label={label}><input className="input w-full" maxLength={120} value={value} onChange={(e) => onChange(e.target.value)} /></Field>;
}

function NumberField({ label, value, max, onChange }) {
  return <Field label={label}><input className="input w-full" type="number" inputMode="numeric" min="0" max={max} step="1" value={value} onChange={(e) => onChange(Math.min(max, Math.max(0, Number.parseInt(e.target.value || "0", 10))))} /></Field>;
}

function Check({ label, checked, onChange }) {
  return <label className="flex items-center gap-3 rounded-control border border-[var(--color-line)] px-4 py-3 text-[14px]"><input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />{label}</label>;
}

function toDraft(rule) {
  return { ...rule, label: rule.label || "", adjustment_label: rule.adjustment_label || "", adjustment_comment: rule.adjustment_comment || "", blank_quantity_header: rule.blank_quantity_header || "", blank_box_header: rule.blank_box_header || "", blank_layout: rule.blank_layout || "", quantity_multiple: rule.quantity_multiple || 0, min_quantity: rule.min_quantity || 0, prefix_aliases_text: (rule.prefix_aliases || []).join(", ") };
}

function validatedRule(rule) {
  if (!rule.label.trim()) return { error: "Укажите название бренда." };
  if (["multiple", "nearestMultiple"].includes(rule.adjustment) && rule.quantity_multiple < 1) return { error: "Для этого алгоритма укажите кратность больше нуля." };
  if (rule.adjustment === "minimum" && rule.min_quantity < 1) return { error: "Для минимальной партии укажите количество больше нуля." };
  const aliases = rule.prefix_aliases_text.split(",").map((value) => value.trim()).filter(Boolean);
  if (aliases.length > 20 || aliases.some((value) => value.length > 32)) return { error: "Не больше 20 префиксов, каждый — до 32 символов." };
  const clean = { ...rule };
  delete clean.prefix_aliases_text;
  return { error: "", rule: { ...clean, prefix_aliases: [...new Set(aliases)] } };
}
