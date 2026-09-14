import { useEffect, useState } from "react";
import { listBrandRules } from "../../api/brands.js";
import { userFacingError } from "../../features/help/errors.js";

const adjustmentLabels = {
  box: "До коробки",
  multiple: "До кратности",
  nearestMultiple: "Ближайшая кратность",
  minimum: "Минимальная партия",
  none: "Без округления",
};

export function BrandRulesScreen() {
  const [rules, setRules] = useState([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listBrandRules()
      .then((payload) => setRules(payload.rules || []))
      .catch((err) => setError(userFacingError(err, "Не удалось загрузить правила брендов.")))
      .finally(() => setLoading(false));
  }, []);

  return (
    <section className="animate-enter mx-auto max-w-6xl space-y-4 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Правила брендов</h1>
        <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Действующие правила расчёта и распознавания. Изменения вносятся вместе с новой версией системы.</p>
      </div>
      {error ? <p className="rounded-xl bg-[var(--color-danger-soft)] p-4 text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      {loading ? <div className="h-24 animate-pulse rounded-xl bg-[var(--color-line-soft)]" /> : null}
      {!loading && !error ? (
        <div className="overflow-x-auto rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
          <table className="w-full min-w-[760px] text-left text-[13px]">
            <thead className="bg-[var(--color-ground)] text-[var(--color-ink-faint)]">
              <tr>
                <th className="px-4 py-3 font-medium">Бренд</th>
                <th className="px-4 py-3 font-medium">Заказ поставщику</th>
                <th className="px-4 py-3 font-medium">Кратность / минимум</th>
                <th className="px-4 py-3 font-medium">Структура бланка</th>
                <th className="px-4 py-3 font-medium">Артикулы</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={`${rule.brand}:${rule.variant || "main"}`} className="border-t border-[var(--color-line-soft)] align-top">
                  <td className="px-4 py-3 font-semibold">{rule.label || rule.brand}</td>
                  <td className="px-4 py-3">{adjustmentLabels[rule.adjustment] || rule.adjustment_label || "—"}</td>
                  <td className="px-4 py-3 font-mono">{rule.quantity_multiple > 0 ? `× ${rule.quantity_multiple}` : rule.min_quantity > 0 ? `от ${rule.min_quantity}` : "—"}</td>
                  <td className="px-4 py-3">{rule.blank_layout || "Обычный бланк"}</td>
                  <td className="px-4 py-3">{rule.preserve_hyphen ? "Дефис сохраняется" : "Стандартная нормализация"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
