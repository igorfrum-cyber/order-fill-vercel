import { useState } from "react";
import { completeTwoFactorLogin } from "../../api/auth.js";
import {
  loginFailedMessage,
  twoFactorCodeLabel,
  twoFactorLoginTitle,
  twoFactorRecoveryCodeLabel,
  twoFactorRecoveryLabel,
} from "../../features/help/copy.js";
import { Field, PrimaryButton } from "../widgets.jsx";
import { AuthCard } from "./AuthShared.jsx";

export function TwoFactorLogin({ challengeId, onDone, onBack }) {
  const [code, setCode] = useState("");
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const label = useRecovery ? twoFactorRecoveryCodeLabel : twoFactorCodeLabel;

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      onDone(await completeTwoFactorLogin(challengeId, code));
    } catch {
      setError(loginFailedMessage);
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthCard title={twoFactorLoginTitle}>
      <form className="animate-enter space-y-4" onSubmit={submit}>
        <Field label={label}>
          <input
            value={code}
            onChange={(event) => setCode(event.target.value)}
            className="input"
            autoComplete="one-time-code"
            inputMode={useRecovery ? "text" : "numeric"}
            autoFocus
          />
        </Field>
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        <PrimaryButton type="submit" disabled={busy || !code.trim()}>
          Войти
        </PrimaryButton>
        <button
          type="button"
          className="text-[14px] text-[var(--color-brand)]"
          onClick={() => {
            setUseRecovery((current) => !current);
            setCode("");
            setError("");
          }}
        >
          {useRecovery ? twoFactorCodeLabel : twoFactorRecoveryLabel}
        </button>
        {onBack ? (
          <button type="button" className="block text-[14px] text-[var(--color-ink-soft)]" onClick={onBack}>
            ← Назад к паролю
          </button>
        ) : null}
      </form>
    </AuthCard>
  );
}
