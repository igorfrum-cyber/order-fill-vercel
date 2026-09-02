import { useEffect } from "react";
import { helpSections } from "../../features/help/copy.js";
import { IconX } from "../icons.jsx";

export function HelpDrawer({ onClose, onReplay }) {
  useEffect(() => {
    function onKey(event) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      className="help-backdrop fixed inset-0 z-40 flex justify-end bg-slate-900/45"
      role="dialog"
      aria-modal="true"
      aria-labelledby="help-title"
      onClick={onClose}
    >
      <aside
        className="help-drawer flex h-full w-full max-w-md flex-col overflow-hidden bg-[var(--color-surface)] shadow-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-[var(--color-line)] px-5 py-4">
          <h2 id="help-title" className="text-[18px] font-semibold tracking-tight">
            Справка
          </h2>
          <button
            type="button"
            aria-label="Закрыть"
            className="grid h-9 w-9 place-items-center rounded-lg text-[var(--color-ink-faint)] hover:bg-[var(--color-line-soft)] hover:text-[var(--color-ink)]"
            onClick={onClose}
          >
            <IconX className="h-4 w-4" />
          </button>
        </div>
        <div className="flex-1 overflow-auto px-5 py-4">
          {helpSections.map((section) => (
            <section key={section.title} className="mb-5 last:mb-0">
              <h3 className="text-[15px] font-semibold">{section.title}</h3>
              <p className="mt-1 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{section.body}</p>
            </section>
          ))}
          {onReplay ? (
            <button
              type="button"
              className="mt-2 text-[14px] font-medium text-[var(--color-brand)]"
              onClick={onReplay}
            >
              Показать подсказки по экрану
            </button>
          ) : null}
        </div>
      </aside>
    </div>
  );
}
