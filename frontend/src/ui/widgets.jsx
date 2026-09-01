import { useState } from "react";
import { generatePassword } from "../features/auth/password.js";
import { IconChevron, IconEye, IconEyeOff, IconPlus, IconX } from "./icons.jsx";

export function Field({ label, children }) {
  return (
    <label className="block">
      <span className="mb-2 block text-[14px] font-medium text-[var(--color-ink-soft)]">{label}</span>
      {children}
    </label>
  );
}

export function PasswordField({ label, value, onChange, autoComplete, generate = false, onGenerated }) {
  const [visible, setVisible] = useState(false);
  return (
    <Field label={label}>
      <div className="relative">
        <input
          type={visible ? "text" : "password"}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          className={`input ${generate ? "pr-44" : "pr-24"}`}
          autoComplete={autoComplete}
        />
        <div className="absolute inset-y-0 right-2 flex items-center gap-1">
          {generate ? (
            <button
              type="button"
              className="rounded-lg px-2 py-1 text-[12px] font-medium text-[var(--color-brand)] hover:bg-[var(--color-brand-soft)]"
              onClick={() => {
                const next = generatePassword();
                onChange(next);
                onGenerated?.(next);
              }}
            >
              Сгенерировать
            </button>
          ) : null}
          <button
            type="button"
            className="grid h-8 w-8 place-items-center rounded-lg text-[var(--color-ink-faint)] hover:bg-[var(--color-line-soft)] hover:text-[var(--color-ink)]"
            onClick={() => setVisible((current) => !current)}
            aria-label={visible ? "Скрыть пароль" : "Показать пароль"}
          >
            {visible ? <IconEyeOff className="h-4 w-4" /> : <IconEye className="h-4 w-4" />}
          </button>
        </div>
      </div>
    </Field>
  );
}

export function Select({ value, onChange, options }) {
  return (
    <div className="relative">
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full appearance-none rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 text-[16px] font-medium outline-none transition focus:border-[var(--color-brand)] focus:ring-4 focus:ring-[var(--color-brand-soft)]"
      >
        {options.map((option) => (
          <option key={option.value ?? option} value={option.value ?? option} disabled={option.disabled}>
            {option.label ?? option}
          </option>
        ))}
      </select>
      <IconChevron className="pointer-events-none absolute right-3.5 top-1/2 h-5 w-5 -translate-y-1/2 text-[var(--color-ink-faint)]" />
    </div>
  );
}

export function PrimaryButton({ children, onClick, disabled, type = "button" }) {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className="flex items-center gap-2 rounded-xl bg-[var(--color-brand)] px-5 py-3 text-[15px] font-semibold text-white transition hover:bg-[var(--color-brand-strong)] focus:outline-none focus:ring-4 focus:ring-[var(--color-brand-soft)] disabled:cursor-not-allowed disabled:opacity-40"
    >
      {children}
    </button>
  );
}

export function GhostButton({ children, onClick, disabled, type = "button" }) {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className="flex items-center gap-1.5 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-2.5 text-[14px] font-medium text-[var(--color-ink-soft)] transition hover:border-[var(--color-ink-faint)] hover:text-[var(--color-ink)] disabled:cursor-not-allowed disabled:opacity-40"
    >
      {children}
    </button>
  );
}

export function StageHeading({ index, kicker, title, children }) {
  return (
    <>
      <div className="mb-2 flex items-center gap-2 text-[13px] font-medium text-[var(--color-brand)]">
        <span className="font-mono text-[12px]">{index}</span>
        <span className="h-px w-6 bg-[var(--color-brand)]/40" />
        {kicker}
      </div>
      <h1 className="text-[36px] font-semibold tracking-tight">{title}</h1>
      {children}
    </>
  );
}

export function ProgressBar({ value, label }) {
  const pct = Math.max(0, Math.min(100, Math.round(Number(value || 0) * 100)));
  return (
    <div className="mt-5">
      <div className="mb-2 flex items-center justify-between gap-3 text-[14px] text-[var(--color-ink-soft)]">
        <span>{label || "Обработка..."}</span>
        <span className="font-mono tabular-nums text-[13px] text-[var(--color-ink-faint)]">{pct}%</span>
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-[var(--color-line-soft)]">
        <div
          className="h-full rounded-full bg-[var(--color-brand)] transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export function Ring({ value }) {
  const size = 68;
  const width = 7;
  const radius = (size - width) / 2;
  const circumference = 2 * Math.PI * radius;
  const pct = Math.round(value * 100);
  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="var(--color-line)" strokeWidth={width} />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="var(--color-ok)"
          strokeWidth={width}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={circumference * (1 - value)}
          className="transition-all duration-500"
        />
      </svg>
      <span className="absolute inset-0 grid place-items-center font-mono text-[15px] font-semibold tabular-nums">
        {pct}%
      </span>
    </div>
  );
}

export function Stepper({ value, disabled, onChange, step }) {
  if (disabled) {
    return <span className="px-2 font-mono text-[14px] tabular-nums text-[var(--color-ink-faint)]">—</span>;
  }
  const numeric = value === "" || value == null ? 0 : Number(value);
  const btn =
    "grid h-9 w-8 place-items-center text-[var(--color-ink-soft)] transition hover:bg-[var(--color-line-soft)] hover:text-[var(--color-ink)] disabled:opacity-30";
  return (
    <div
      className={`flex items-center overflow-hidden rounded-lg border transition focus-within:border-[var(--color-brand)] focus-within:ring-4 focus-within:ring-[var(--color-brand-soft)] ${
        numeric > 0 ? "border-[var(--color-ok)]" : "border-[var(--color-line)]"
      }`}
    >
      <button type="button" className={btn} disabled={numeric <= 0} onClick={() => onChange(Math.max(0, numeric - step))} tabIndex={-1}>
        <IconX className="h-2.5 w-2.5 rotate-45" />
      </button>
      <input
        type="text"
        inputMode="numeric"
        value={value ?? ""}
        placeholder="0"
        onChange={(event) => {
          const raw = event.target.value.replace(/\D/g, "");
          onChange(raw === "" ? "" : Number(raw));
        }}
        className="w-16 bg-transparent text-center font-mono text-[16px] tabular-nums outline-none placeholder:text-[var(--color-ink-faint)]"
      />
      <button type="button" className={btn} onClick={() => onChange(numeric + step)} tabIndex={-1}>
        <IconPlus className="h-3 w-3" />
      </button>
    </div>
  );
}

export function Modal({ title, children, onCancel, onConfirm, cancelLabel = "Назад", confirmLabel = "Продолжить", confirmDisabled }) {
  return (
    <div className="fixed inset-0 z-20 grid place-items-center bg-slate-900/45 p-5" role="dialog" aria-modal="true">
      <div className="w-full max-w-lg rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6 shadow-xl">
        <h2 className="text-[18px] font-semibold tracking-tight">{title}</h2>
        <div className="mt-3 max-h-64 overflow-auto text-[14px] leading-relaxed text-[var(--color-ink-soft)] whitespace-pre-line">
          {children}
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <GhostButton onClick={onCancel}>{cancelLabel}</GhostButton>
          <PrimaryButton onClick={onConfirm} disabled={confirmDisabled}>{confirmLabel}</PrimaryButton>
        </div>
      </div>
    </div>
  );
}
