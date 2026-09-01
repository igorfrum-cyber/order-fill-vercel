import { useState } from "react";
import { acceptInvite, changePassword, login } from "../../api/auth.js";
import { loginAccessHint, loginFailedMessage } from "../../features/help/copy.js";
import { isPasswordReady, passwordIssues } from "../../features/auth/password.js";
import { Field, PasswordField, PrimaryButton } from "../widgets.jsx";

export function LoginScreen({ onDone }) {
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

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
    <AuthCard title="Вход">
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
          {me.login}. Смена пароля только для этой учётки. Других пользователей сбрасывайте ссылкой-приглашением.
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

function AuthCard({ title, children }) {
  return (
    <div className="grid min-h-full place-items-center bg-[var(--color-ground)] p-6">
      <div className="animate-enter w-full max-w-md rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-8 shadow-sm">
        <h1 className="mb-6 text-[22px] font-semibold">{title}</h1>
        {children}
      </div>
    </div>
  );
}
