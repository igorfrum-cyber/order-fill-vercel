import { useState } from "react";
import { disableTwoFactor, enableTwoFactor, startTwoFactorSetup } from "../../api/auth.js";
import { twoFactorOpenAppHref } from "../../features/auth/twoFactor.js";
import {
  twoFactorCodeLabel,
  twoFactorDisableLabel,
  twoFactorEnableLabel,
  twoFactorManualKeyLabel,
  twoFactorOpenAppLabel,
  twoFactorRecoveryHint,
  twoFactorSetupHint,
  twoFactorSetupSteps,
} from "../../features/help/copy.js";
import { Field, GhostButton, PasswordField, PrimaryButton } from "../widgets.jsx";

export function TwoFactorSetup({ enabled, hasPasskey = false, onChanged }) {
  const [phase, setPhase] = useState(enabled ? "enabled" : "idle");
  const [setup, setSetup] = useState(null);
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState([]);
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function startSetup() {
    setBusy(true);
    setError("");
    try {
      const next = await startTwoFactorSetup();
      setSetup(next);
      setPhase("confirm");
    } catch {
      setError("Не удалось начать настройку. Попробуйте ещё раз.");
    } finally {
      setBusy(false);
    }
  }

  async function confirm(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await enableTwoFactor(code);
      setRecoveryCodes(result.recovery_codes || []);
      setPhase("recovery");
      onChanged?.(true);
    } catch {
      setError("Неверный код. Проверьте приложение и попробуйте снова.");
    } finally {
      setBusy(false);
    }
  }

  async function disable(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await disableTwoFactor(password);
      setPassword("");
      setSetup(null);
      setCode("");
      setRecoveryCodes([]);
      setPhase("idle");
      onChanged?.(false);
    } catch {
      setError("Не удалось отключить защиту. Проверьте пароль.");
    } finally {
      setBusy(false);
    }
  }

  if (phase === "enabled") {
    return (
      <div className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
        <h2 className="text-[16px] font-semibold">Вход по коду</h2>
        <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
          {hasPasskey
            ? "Код включён как запасной вход. На работе можно входить по Face ID или Touch ID."
            : "Вход по коду включён."}
        </p>
        <form className="space-y-3" onSubmit={disable}>
          <PasswordField label="Текущий пароль" value={password} onChange={setPassword} autoComplete="current-password" />
          {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
          <GhostButton type="submit" disabled={busy || !password}>
            {twoFactorDisableLabel}
          </GhostButton>
        </form>
      </div>
    );
  }

  if (phase === "recovery") {
    return (
      <div className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
        <h2 className="text-[16px] font-semibold">Запасные коды</h2>
        <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{twoFactorRecoveryHint}</p>
        <ul className="grid gap-2 font-mono text-[15px]">
          {recoveryCodes.map((item) => (
            <li key={item} className="rounded-lg bg-[var(--color-ground)] px-3 py-2">
              {item}
            </li>
          ))}
        </ul>
        <PrimaryButton type="button" onClick={() => setPhase("enabled")}>
          Готово
        </PrimaryButton>
      </div>
    );
  }

  if (phase === "confirm" && setup) {
    const openHref = twoFactorOpenAppHref(setup.otpauth_url);
    return (
      <form className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6" onSubmit={confirm}>
        <h2 className="text-[16px] font-semibold">{twoFactorEnableLabel}</h2>
        <ol className="list-decimal space-y-2 pl-5 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
          {twoFactorSetupSteps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
        {openHref ? (
          <a
            href={openHref}
            className="flex items-center justify-center gap-2 rounded-xl bg-[var(--color-brand)] px-5 py-3 text-center text-[15px] font-semibold text-white transition hover:bg-[var(--color-brand-strong)] focus:outline-none focus:ring-4 focus:ring-[var(--color-brand-soft)]"
          >
            {twoFactorOpenAppLabel}
          </a>
        ) : null}
        {setup.qr_png_base64 ? (
          <img
            src={`data:image/png;base64,${setup.qr_png_base64}`}
            alt="QR-код для приложения"
            className="mx-auto h-48 w-48 rounded-xl border border-[var(--color-line)] bg-white p-2"
          />
        ) : null}
        <div>
          <p className="mb-1 text-[13px] font-medium text-[var(--color-ink-faint)]">{twoFactorManualKeyLabel}</p>
          <p className="break-all font-mono text-[15px]">{setup.secret}</p>
        </div>
        <Field label={twoFactorCodeLabel}>
          <input
            value={code}
            onChange={(event) => setCode(event.target.value)}
            className="input"
            autoComplete="one-time-code"
            inputMode="numeric"
          />
        </Field>
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        <PrimaryButton type="submit" disabled={busy || !code.trim()}>
          Подтвердить
        </PrimaryButton>
      </form>
    );
  }

  return (
    <div className="space-y-3 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
      <h2 className="text-[16px] font-semibold">Запасной вход по коду</h2>
      <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{twoFactorSetupHint}</p>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <GhostButton type="button" onClick={startSetup} disabled={busy}>
        {twoFactorEnableLabel}
      </GhostButton>
    </div>
  );
}
