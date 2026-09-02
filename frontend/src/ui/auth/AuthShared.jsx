import { passwordIssues } from "../../features/auth/password.js";

export function PasswordHints({ password, repeat }) {
  const issues = passwordIssues(password, repeat);
  if (!password && !repeat) return null;
  if (!issues.length) {
    return <p className="text-[13px] text-[var(--color-ok)]">Пароль подходит.</p>;
  }
  return (
    <ul className="space-y-1 text-[13px] text-[var(--color-danger)]">
      {issues.map((issue) => (
        <li key={issue}>{issue}</li>
      ))}
    </ul>
  );
}

export function AuthCard({ title, lead, logoSrc, children }) {
  return (
    <div className="grid min-h-full place-items-center bg-[var(--color-ground)] p-6">
      <div className="animate-enter w-full max-w-md rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-8 shadow-sm">
        {logoSrc ? <img src={logoSrc} alt="" className="mx-auto mb-5 h-16 w-auto max-w-[220px] object-contain" /> : null}
        <h1 className={`text-[22px] font-semibold ${lead ? "mb-2" : "mb-6"}`}>{title}</h1>
        {lead ? <p className="mb-6 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{lead}</p> : null}
        {children}
      </div>
    </div>
  );
}
