import { useState } from "react";
import { changePassword } from "../../api/auth.js";
import { isPasswordReady } from "../../features/auth/password.js";
import {
  accessSummaryForRole,
  accountPasswordHint,
  profileFields,
} from "../../features/help/copy.js";
import { PasswordField, PrimaryButton } from "../widgets.jsx";
import { PasswordHints } from "./AuthShared.jsx";
import { PasskeySettings } from "./PasskeySettings.jsx";
import { SessionSettings } from "./SessionSettings.jsx";
import { TwoFactorSetup } from "./TwoFactorSetup.jsx";

export function AccountScreen({ me, onBack, onSignedOut, onMe }) {
  const [current, setCurrent] = useState("");
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const ready = Boolean(current) && isPasswordReady(password, repeat);
  const fields = profileFields(me);

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
        <h1 className="text-[22px] font-semibold">Мой профиль</h1>
      </div>
      <dl className="space-y-3 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
        {fields.map((field) => (
          <div key={field.label}>
            <dt className="text-[13px] font-medium text-[var(--color-ink-faint)]">{field.label}</dt>
            <dd className="mt-1 text-[16px] font-medium">{field.value}</dd>
          </div>
        ))}
        <p className="pt-1 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{accessSummaryForRole(me.role)}</p>
      </dl>
      <PasskeySettings onChanged={(hasPasskey) => onMe?.((current) => ({ ...current, has_passkey: hasPasskey }))} />
      <SessionSettings onSignedOut={onSignedOut} />
      <TwoFactorSetup
        enabled={Boolean(me.two_factor_enabled)}
        hasPasskey={Boolean(me.has_passkey)}
        onChanged={(enabled) => onMe?.((current) => ({ ...current, two_factor_enabled: enabled }))}
      />
      <form className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6" onSubmit={submit}>
        <h2 className="text-[16px] font-semibold">Пароль</h2>
        <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{accountPasswordHint}</p>
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
