import { useState } from "react";
import { acceptInvite } from "../../api/auth.js";
import { isPasswordReady, passwordIssues } from "../../features/auth/password.js";
import { PasswordField, PrimaryButton } from "../widgets.jsx";
import { AuthCard, PasswordHints } from "./AuthShared.jsx";

export function InviteScreen({ token, onDone }) {
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const ready = isPasswordReady(password, repeat);

  async function submit(event) {
    event.preventDefault();
    if (!ready) {
      setError(passwordIssues(password, repeat)[0] || "Проверьте пароль.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const user = await acceptInvite(token, password);
      window.history.replaceState({}, "", "/");
      onDone(user);
    } catch {
      setError("Ссылка недействительна или пароль не принят.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthCard title="Приглашение">
      <p className="mb-5 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
        Задайте свой пароль по одноразовой ссылке. Его можно сгенерировать и сразу скопировать.
      </p>
      <form className="animate-enter space-y-4" onSubmit={submit}>
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
        <PrimaryButton type="submit" disabled={busy || !ready}>
          Сохранить пароль
        </PrimaryButton>
      </form>
    </AuthCard>
  );
}
