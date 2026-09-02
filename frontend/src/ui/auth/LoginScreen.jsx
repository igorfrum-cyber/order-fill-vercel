import { useState } from "react";
import { login } from "../../api/auth.js";
import { apiClient } from "../../api/client.js";
import { companyLoginCopy, companyLoginLogoURL } from "../../features/auth/accessPresentation.js";
import { loginAccessHint, loginFailedMessage } from "../../features/help/copy.js";
import { Field, PasswordField, PrimaryButton } from "../widgets.jsx";
import { AuthCard } from "./AuthShared.jsx";

export function LoginScreen({ onDone, company }) {
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const copy = companyLoginCopy(company);
  const logoSrc =
    company?.has_logo && company.login_slug ? apiClient.absoluteUrl(companyLoginLogoURL(company.login_slug)) : "";

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      onDone(await login(name, password));
    } catch {
      setError(loginFailedMessage);
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthCard title={copy.title} lead={copy.lead} logoSrc={logoSrc}>
      <form className="animate-enter space-y-4" onSubmit={submit}>
        <Field label="Логин">
          <input value={name} onChange={(event) => setName(event.target.value)} className="input" autoComplete="username" />
        </Field>
        <PasswordField label="Пароль" value={password} onChange={setPassword} autoComplete="current-password" />
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        <p className="text-[13px] leading-relaxed text-[var(--color-ink-soft)]">{loginAccessHint}</p>
        <PrimaryButton type="submit" disabled={busy || !name || password.length < 10}>
          Войти
        </PrimaryButton>
      </form>
    </AuthCard>
  );
}
