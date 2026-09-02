import { quickStartForRole } from "../../features/help/copy.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

export function QuickStart({ me, onDismiss, onLater }) {
  const steps = quickStartForRole(me.role);
  const subtitle = steps.length >= 4 ? "Четыре шага до готовых файлов." : "Коротко о том, с чего начать.";

  return (
    <div className="fixed inset-0 z-30 grid place-items-center bg-slate-900/45 p-5" role="dialog" aria-modal="true" aria-labelledby="quick-start-title">
      <div className="w-full max-w-lg rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6 shadow-xl">
        <h2 id="quick-start-title" className="text-[18px] font-semibold tracking-tight">
          Быстрый старт
        </h2>
        <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">{subtitle}</p>
        <ol className="mt-4 list-decimal space-y-2 pl-5 text-[14px] leading-relaxed text-[var(--color-ink)]">
          {steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
        <div className="mt-5 flex justify-end gap-2">
          <GhostButton onClick={onLater}>Показать позже</GhostButton>
          <PrimaryButton onClick={onDismiss}>Понятно</PrimaryButton>
        </div>
      </div>
    </div>
  );
}
