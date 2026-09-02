import { useEffect, useState } from "react";
import { login } from "../../api/auth.js";
import { apiClient } from "../../api/client.js";
import { companyLoginCopy, companyLoginLogoURL } from "../../features/auth/accessPresentation.js";
import { conditionalMediationAvailable, passkeySupported } from "../../features/auth/passkey.js";
import { authenticatePasskey } from "../../features/auth/passkeyFlow.js";
import {
  loginAccessHint,
  loginFailedMessage,
  passkeyLoginButton,
  passkeyLoginHint,
  passkeyLoginTitle,
} from "../../features/help/copy.js";
import { Field, PasswordField, PrimaryButton } from "../widgets.jsx";
import { AuthCard } from "./AuthShared.jsx";
import { TwoFactorLogin } from "./TwoFactorLogin.jsx";

export function LoginScreen({ onDone, company }) {
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [challengeId, setChallengeId] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const copy = companyLoginCopy(company);
  const logoSrc =
    company?.has_logo && company.login_slug ? apiClient.absoluteUrl(companyLoginLogoURL(company.login_slug)) : "";
  const canUsePasskey = passkeySupported();

  useEffect(() => {
    if (!canUsePasskey) return undefined;
    const abort = new AbortController();
    (async () => {
      if (!(await conditionalMediationAvailable())) return;
      try {
        const result = await authenticatePasskey("", { mediation: "conditional", signal: abort.signal });
        if (result) onDone(result);
      } catch {
        // User ignored the suggestion or the browser cancelled it.
      }
    })();
    return () => abort.abort();
  }, [canUsePasskey, onDone]);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await login(name, password);
      if (result.two_factor_required) {
        setChallengeId(result.challenge_id);
        return;
      }
      onDone(result);
    } catch {
      setError(loginFailedMessage);
    } finally {
      setBusy(false);
    }
  }

  async function submitPasskey() {
    setBusy(true);
    setError("");
    try {
      const result = await authenticatePasskey(name);
      if (result) onDone(result);
    } catch {
      setError("Не получилось войти с ключом доступа. Можно войти паролем.");
    } finally {
      setBusy(false);
    }
  }

  if (challengeId) {
    return <TwoFactorLogin challengeId={challengeId} onDone={onDone} onBack={() => setChallengeId("")} />;
  }

  return (
    <AuthCard title={copy.title} lead={copy.lead} logoSrc={logoSrc}>
      <form className="animate-enter space-y-4" onSubmit={submit}>
        <Field label="Логин">
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="input"
            autoComplete="username webauthn"
          />
        </Field>
        <PasswordField label="Пароль" value={password} onChange={setPassword} autoComplete="current-password" />
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        <p className="text-[13px] leading-relaxed text-[var(--color-ink-soft)]">{loginAccessHint}</p>
        <PrimaryButton type="submit" disabled={busy || !name || password.length < 10}>
          Войти
        </PrimaryButton>
      </form>
      {canUsePasskey ? (
        <div className="mt-5 rounded-2xl border border-[var(--color-line)] bg-[var(--color-ground)] p-4">
          <p className="text-[15px] font-semibold">{passkeyLoginTitle}</p>
          <p className="mt-1 text-[13px] leading-relaxed text-[var(--color-ink-soft)]">{passkeyLoginHint}</p>
          <button
            type="button"
            className="mt-3 w-full rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 text-[15px] font-semibold transition hover:border-[var(--color-brand)] hover:text-[var(--color-brand)] disabled:cursor-not-allowed disabled:opacity-40"
            onClick={submitPasskey}
            disabled={busy}
          >
            {passkeyLoginButton}
          </button>
        </div>
      ) : null}
    </AuthCard>
  );
}
