import { useState } from "react";
import { acceptInvite, login } from "../../api/auth.js";
import { Field, PrimaryButton } from "../widgets.jsx";

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
      setError("Неверный логин или пароль.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthCard title="Вход">
      <form className="space-y-4" onSubmit={submit}>
        <Field label="Логин">
          <input value={name} onChange={(event) => setName(event.target.value)} className="input" autoComplete="username" />
        </Field>
        <Field label="Пароль">
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} className="input" autoComplete="current-password" />
        </Field>
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
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

  async function submit(event) {
    event.preventDefault();
    if (password !== repeat) {
      setError("Пароли не совпадают.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const user = await acceptInvite(token, password);
      window.history.replaceState({}, "", "/");
      onDone(user);
    } catch {
      setError("Ссылка недействительна или пароль слишком короткий.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthCard title="Приглашение">
      <form className="space-y-4" onSubmit={submit}>
        <Field label="Новый пароль">
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} className="input" autoComplete="new-password" />
        </Field>
        <Field label="Ещё раз">
          <input type="password" value={repeat} onChange={(event) => setRepeat(event.target.value)} className="input" autoComplete="new-password" />
        </Field>
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        <PrimaryButton type="submit" disabled={busy || password.length < 10}>
          Сохранить пароль
        </PrimaryButton>
      </form>
    </AuthCard>
  );
}

function AuthCard({ title, children }) {
  return (
    <div className="grid min-h-full place-items-center bg-[var(--color-ground)] p-6">
      <div className="w-full max-w-md rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-8 shadow-sm">
        <h1 className="mb-6 text-[22px] font-semibold">{title}</h1>
        {children}
      </div>
    </div>
  );
}
