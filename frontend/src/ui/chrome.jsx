import { IconCheck, IconList } from "./icons.jsx";

export function TopBar({ brandLabel, monthLabel, stage, mode, onMode }) {
  return (
    <header className="flex items-center justify-between border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
      <div className="flex items-center gap-2.5">
        <div className="grid h-8 w-8 place-items-center rounded-md bg-[var(--color-brand)] text-white">
          <IconList className="h-4 w-4" />
        </div>
        <div className="leading-tight">
          <div className="text-[13.5px] font-semibold tracking-tight">Бланки закупки</div>
          <div className="text-[11px] text-[var(--color-ink-faint)]">автозаполнение</div>
        </div>
        <div className="ml-4 inline-flex gap-1 rounded-lg border border-[var(--color-line)] bg-[var(--color-ground)] p-1">
          <ModeButton active={mode === "order"} onClick={() => onMode("order")}>Бланк</ModeButton>
          <ModeButton active={mode === "north"} onClick={() => onMode("north")}>Север</ModeButton>
        </div>
      </div>
      {stage === "fill" && (
        <div className="flex items-center gap-2 font-mono text-[11px] text-[var(--color-ink-soft)]">
          <span className="rounded-full bg-[var(--color-brand-soft)] px-2.5 py-1 text-[var(--color-brand-strong)]">
            {brandLabel}
          </span>
          <span className="text-[var(--color-ink-faint)]">·</span>
          <span>{monthLabel}</span>
        </div>
      )}
    </header>
  );
}

function ModeButton({ active, onClick, children }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`rounded-md px-3 py-1.5 text-[12.5px] font-semibold transition ${
        active ? "bg-[var(--color-surface)] text-[var(--color-brand-strong)] shadow-sm" : "text-[var(--color-ink-faint)] hover:text-[var(--color-ink)]"
      }`}
    >
      {children}
    </button>
  );
}

export function StageRail({ stage, brandLabel, monthLabel, filesReady, onGoto }) {
  const steps = [
    { key: "setup", n: 1, title: "Настройка", summary: `${brandLabel} · ${monthLabel}` },
    { key: "upload", n: 2, title: "Загрузка файлов", summary: filesReady ? "файлы загружены" : undefined },
    { key: "fill", n: 3, title: "Заполнение" },
  ];
  const order = ["setup", "upload", "fill"];
  const currentIdx = Math.max(0, order.indexOf(stage === "processing" ? "upload" : stage));

  return (
    <nav className="flex items-center gap-1 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-2.5">
      {steps.map((step, index) => {
        const idx = order.indexOf(step.key);
        const state = idx < currentIdx ? "done" : idx === currentIdx ? "active" : "todo";
        const clickable = idx < currentIdx || (idx === 1 && stage === "fill");
        return (
          <div key={step.key} className="flex items-center">
            <button
              type="button"
              disabled={!clickable}
              onClick={() => clickable && onGoto(step.key)}
              className={`group flex items-center gap-2.5 rounded-lg px-3 py-1.5 text-left transition ${
                clickable ? "hover:bg-[var(--color-line-soft)]" : "cursor-default"
              }`}
            >
              <span
                className={`grid h-6 w-6 shrink-0 place-items-center rounded-full font-mono text-[11px] font-semibold transition ${
                  state === "active"
                    ? "bg-[var(--color-brand)] text-white"
                    : state === "done"
                      ? "bg-[var(--color-ok-soft)] text-[var(--color-ok)]"
                      : "bg-[var(--color-neutral-soft)] text-[var(--color-ink-faint)]"
                }`}
              >
                {state === "done" ? <IconCheck className="h-3.5 w-3.5" /> : step.n}
              </span>
              <span className="leading-tight">
                <span className={`block text-[12.5px] font-medium ${state === "todo" ? "text-[var(--color-ink-faint)]" : "text-[var(--color-ink)]"}`}>
                  {step.title}
                </span>
                {state === "done" && step.summary && (
                  <span className="block font-mono text-[10.5px] text-[var(--color-ink-faint)]">{step.summary}</span>
                )}
              </span>
            </button>
            {index < steps.length - 1 && <span className="mx-1 h-px w-8 bg-[var(--color-line)]" aria-hidden />}
          </div>
        );
      })}
    </nav>
  );
}
