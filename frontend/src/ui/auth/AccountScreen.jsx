import { useState } from "react";
import { changePassword } from "../../api/auth.js";
import { accessSummary } from "../../features/auth/accessPresentation.js";
import { isPasswordReady } from "../../features/auth/password.js";
import { PasswordField, PrimaryButton } from "../widgets.jsx";
import { PasswordHints } from "./AuthShared.jsx";

export function AccountScreen({ me, onBack }) {
  const [current, setCurrent] = useState("");
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const ready = Boolean(current) && isPasswordReady(password, repeat);

  async function submit(event) {
    event.preventDefault();
    if (!ready) return;
    setBusy(true);
    setError("");
    setDone(false);
    try {
      await changePassword(current, password);
      setCurrent("");
      setPassword("");
      setRepeat("");
      setDone(true);
    } catch {
      setError("Не удалось сменить пароль. Проверьте текущий.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="animate-enter mx-auto max-w-lg space-y-5 p-6">
      <button type="button" className="text-[14px] text-[var(--color-brand)]" onClick={onBack}>
        ← Назад
      </button>
      <div>
        <h1 className="text-[22px] font-semibold">Пароль</h1>
        <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">
          {me.login}. {accessSummary(me.role)} Смена пароля только для этой учётки. Других пользователей сбрасывайте ссылкой-приглашением.
        </p>
      </div>
      <form className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6" onSubmit={submit}>
        <PasswordField label="Текущий пароль" value={current} onChange={setCurrent} autoComplete="current-password" />
        <PasswordField
          label="Новый пароль"
          value={password}
          onChange={setPassword}
          autoComplete="new-password"
          generate
          onGenerated={setRepeat}
        />
        <PasswordField label="Ещё раз" value={repeat} onChange={setRepeat} autoComplete="new-password" />
        <PasswordHints password={password} repeat={repeat} />
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        {done ? <p className="text-[14px] text-[var(--color-ok)]">Пароль обновлён.</p> : null}
        <PrimaryButton type="submit" disabled={busy || !ready}>
          Сохранить
        </PrimaryButton>
      </form>
    </section>
  );
}
