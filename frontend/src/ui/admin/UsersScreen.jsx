import { useEffect, useState } from "react";
import { createUser, disableUser, listUsers, resetUser } from "../../api/auth.js";
import { inviteRoleOptions, roleLabel } from "../../features/auth/accessPresentation.js";
import { IconCopy } from "../icons.jsx";
import { GhostButton, Modal, PrimaryButton } from "../widgets.jsx";

export function UsersScreen({ companyId, actorRole }) {
  const [users, setUsers] = useState([]);
  const [login, setLogin] = useState("");
  const roles = inviteRoleOptions(actorRole);
  const [role, setRole] = useState(roles[0] || "purchaser");
  const [invite, setInvite] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  const [resetTarget, setResetTarget] = useState(null);

  function reload() {
    if (!companyId) return;
    listUsers(companyId).then((payload) => setUsers(payload.users || [])).catch(() => setUsers([]));
  }

  useEffect(reload, [companyId]);

  async function showInvite(urlPath) {
    const url = `${window.location.origin}${urlPath}`;
    setInvite(url);
    setCopied(await copyText(url));
  }

  if (!companyId) {
    return <p className="p-6 text-[var(--color-ink-faint)]">Сначала выберите компанию.</p>;
  }

  return (
    <section className="animate-enter mx-auto max-w-3xl space-y-4 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Пользователи</h1>
        <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">
          Новый человек входит только по ссылке-приглашению. Пароль ему не задаёте — он сам его поставит.
        </p>
      </div>
      <form
        className="flex flex-wrap gap-2"
        onSubmit={async (event) => {
          event.preventDefault();
          setError("");
          try {
            const created = await createUser(companyId, login, role);
            await showInvite(created.invite_url);
            setLogin("");
            reload();
          } catch (err) {
            setError(err.message || "Не удалось пригласить пользователя.");
          }
        }}
      >
        <input className="input flex-1" value={login} onChange={(event) => setLogin(event.target.value)} placeholder="Логин" />
        <select className="input" value={role} onChange={(event) => setRole(event.target.value)}>
          {roles.map((value) => (
            <option key={value} value={value}>
              {roleLabel(value)}
            </option>
          ))}
        </select>
        <PrimaryButton type="submit" disabled={!login.trim()}>Пригласить</PrimaryButton>
      </form>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      {invite ? <InviteBanner url={invite} copied={copied} onCopy={async () => setCopied(await copyText(invite))} /> : null}
      <ul className="divide-y divide-[var(--color-line)] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        {users.map((user) => (
          <li key={user.id} className="flex items-center justify-between gap-2 px-4 py-3">
            <span>
              {user.login}
              <span className="text-[var(--color-ink-faint)]"> · {roleLabel(user.role)}</span>
              {user.disabled_at ? <span className="ml-2 text-[13px] text-[var(--color-ink-faint)]">выключен</span> : null}
            </span>
            <span className="flex gap-2">
              <GhostButton onClick={() => setResetTarget(user)}>Сброс доступа</GhostButton>
              <GhostButton
                onClick={async () => {
                  setError("");
                  try {
                    await disableUser(user.id);
                    reload();
                  } catch (err) {
                    setError(err.message || "Не удалось выключить пользователя.");
                  }
                }}
              >
                Выключить
              </GhostButton>
            </span>
          </li>
        ))}
      </ul>
      {resetTarget ? (
        <Modal
          title={`Сбросить доступ ${resetTarget.login}?`}
          cancelLabel="Отмена"
          confirmLabel="Сбросить"
          onCancel={() => setResetTarget(null)}
          onConfirm={async () => {
            setError("");
            try {
              const payload = await resetUser(resetTarget.id);
              await showInvite(payload.invite_url);
              setResetTarget(null);
              reload();
            } catch (err) {
              setError(err.message || "Не удалось сбросить доступ.");
              setResetTarget(null);
            }
          }}
        >
          Текущий пароль перестанет работать. Нужна новая ссылка-приглашение — скопируйте её и передайте человеку.
        </Modal>
      ) : null}
    </section>
  );
}

function InviteBanner({ url, copied, onCopy }) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-xl bg-[var(--color-brand-soft)] p-3 text-[14px]">
      <p className="min-w-0 break-all">
        {copied ? "Ссылка скопирована. " : "Скопируйте ссылку сейчас: "}
        {url}
      </p>
      <GhostButton onClick={onCopy}>
        <IconCopy className="h-4 w-4" />
        Копировать
      </GhostButton>
    </div>
  );
}

async function copyText(value) {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    return false;
  }
}
