import { useEffect, useRef, useState } from "react";
import { createUser, disableUser, listCompanies, listUsers, resetUser } from "../../api/auth.js";
import {
  canManageListedUser,
  inviteRoleHint,
  inviteRoleOptions,
  needsUsersCompanyPicker,
  pickDefaultCompanyId,
  roleLabel,
  usersCompanyPrompt,
} from "../../features/auth/accessPresentation.js";
import { lastSeenLabel, userInitial, usersByHierarchy } from "../../features/auth/userHierarchy.js";
import { userFacingError } from "../../features/help/errors.js";
import { IconCopy } from "../icons.jsx";
import { GhostButton, Modal, PrimaryButton } from "../widgets.jsx";

export function UsersScreen({ companyId, actorRole, onCompany }) {
  const [users, setUsers] = useState([]);
  const [companies, setCompanies] = useState([]);
  const [login, setLogin] = useState("");
  const roles = inviteRoleOptions(actorRole);
  const [role, setRole] = useState(roles[0] || "purchaser");
  const [invite, setInvite] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  const [resetTarget, setResetTarget] = useState(null);
  const picker = needsUsersCompanyPicker(actorRole);
  const activeCompanies = companies.filter((company) => !company.disabled_at);
  const prompt = usersCompanyPrompt(companyId, companies);

  function reload() {
    if (!companyId) return;
    listUsers(companyId).then((payload) => setUsers(payload.users || [])).catch(() => setUsers([]));
  }

  useEffect(reload, [companyId]);

  useEffect(() => {
    if (!picker) return undefined;
    listCompanies()
      .then((payload) => setCompanies(payload.companies || []))
      .catch(() => setCompanies([]));
    return undefined;
  }, [picker]);

  useEffect(() => {
    if (!picker || companyId || !companies.length) return;
    const next = pickDefaultCompanyId("", companies);
    if (next) onCompany?.(next);
  }, [picker, companyId, companies, onCompany]);

  async function showInvite(urlPath) {
    const url = `${window.location.origin}${urlPath}`;
    setInvite(url);
    setCopied(await copyText(url));
  }

  const bands = usersByHierarchy(users);

  return (
    <section className="animate-enter mx-auto max-w-5xl space-y-5 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-[22px] font-semibold">Пользователи</h1>
          <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">
            Новый человек входит только по ссылке-приглашению. Пароль ему не задаёте — он сам его поставит.
          </p>
        </div>
        {picker ? (
          <select
            className="input max-w-xs"
            value={companyId}
            onChange={(event) => onCompany?.(event.target.value)}
            aria-label="Компания"
            data-tour="company-select"
          >
            <option value="" disabled>
              {activeCompanies.length ? "Выберите компанию" : "Нет компаний"}
            </option>
            {activeCompanies.map((company) => (
              <option key={company.id} value={company.id}>
                {company.name}
              </option>
            ))}
          </select>
        ) : null}
      </div>
      {!companyId ? (
        <p className="text-[14px] text-[var(--color-ink-faint)]">{prompt}</p>
      ) : (
        <>
          <form
            data-tour="invite"
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
                setError(userFacingError(err, "Не удалось пригласить пользователя."));
              }
            }}
          >
            <input className="input flex-1" value={login} onChange={(event) => setLogin(event.target.value)} placeholder="Логин" />
            {roles.length ? (
              <select className="input" value={role} onChange={(event) => setRole(event.target.value)} aria-label="Роль">
                {roles.map((value) => (
                  <option key={value} value={value}>
                    {roleLabel(value)}
                  </option>
                ))}
              </select>
            ) : null}
            <PrimaryButton type="submit" disabled={!login.trim()}>
              Пригласить
            </PrimaryButton>
          </form>
          {roles.length ? <p className="text-[13px] text-[var(--color-ink-faint)]">{inviteRoleHint}</p> : null}
          {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
          {invite ? <InviteBanner url={invite} copied={copied} onCopy={async () => setCopied(await copyText(invite))} /> : null}
          <div data-tour="users-list" className="space-y-5">
            {bands.map((band) => (
              <section key={band.key}>
                <h2 className="mb-2 text-[13px] font-medium uppercase tracking-wide text-[var(--color-ink-faint)]">{band.title}</h2>
                {band.users.length ? (
                  <ul className="grid gap-3 sm:grid-cols-2">
                    {band.users.map((user) => (
                      <li key={user.id}>
                        <UserCard
                          user={user}
                          canManage={canManageListedUser(actorRole, user.role)}
                          onReset={() => setResetTarget(user)}
                          onDisable={async () => {
                            setError("");
                            try {
                              await disableUser(user.id);
                              reload();
                            } catch (err) {
                              setError(userFacingError(err, "Не удалось выключить пользователя."));
                            }
                          }}
                        />
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="rounded-[10px] border border-dashed border-[var(--color-line)] px-4 py-5 text-[14px] text-[var(--color-ink-faint)]">
                    Пока никого
                  </p>
                )}
              </section>
            ))}
          </div>
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
                  setError(userFacingError(err, "Не удалось сбросить доступ."));
                  setResetTarget(null);
                }
              }}
            >
              Текущий пароль перестанет работать. Нужна новая ссылка-приглашение — скопируйте её и передайте человеку.
            </Modal>
          ) : null}
        </>
      )}
    </section>
  );
}

function UserCard({ user, canManage, onReset, onDisable }) {
  return (
    <article
      className={`rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)] p-4 ${
        user.disabled_at ? "opacity-55" : ""
      }`}
    >
      <div className="flex items-start gap-3">
        <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-[var(--color-neutral-soft)] text-[14px] font-semibold text-[var(--color-ink-soft)]">
          {userInitial(user.login)}
        </span>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[15px] font-semibold">{user.login}</div>
          <div className="mt-0.5 text-[13px] text-[var(--color-ink-soft)]">
            {roleLabel(user.role)}
            {user.disabled_at ? " · выключен" : ""}
          </div>
          <div className="mt-1 text-[13px] text-[var(--color-ink-faint)]">{lastSeenLabel(user.last_seen_at)}</div>
        </div>
      </div>
      {canManage ? (
        <div className="mt-3 flex flex-wrap gap-2">
          <GhostButton onClick={onReset}>Сброс доступа</GhostButton>
          <GhostButton onClick={onDisable}>Выключить</GhostButton>
        </div>
      ) : null}
    </article>
  );
}

function InviteBanner({ url, copied, onCopy }) {
  const inputRef = useRef(null);

  async function copyAll() {
    inputRef.current?.focus();
    inputRef.current?.select();
    await onCopy();
    inputRef.current?.select();
  }

  return (
    <div className="space-y-2 rounded-xl border border-[var(--color-brand)] bg-[var(--color-brand-soft)] p-3">
      <p className="text-[14px] text-[var(--color-ink-soft)]">
        {copied
          ? "Ссылка скопирована. Передайте её человеку — пароль он поставит сам."
          : "Ссылка одноразовая. Скопируйте её целиком и передайте человеку."}
      </p>
      <div className="flex flex-wrap gap-2">
        <input
          ref={inputRef}
          className="input min-w-0 flex-1 font-mono text-[13px]"
          value={url}
          readOnly
          onFocus={(event) => event.target.select()}
          aria-label="Ссылка-приглашение"
        />
        <PrimaryButton type="button" onClick={copyAll}>
          <IconCopy className="h-4 w-4" />
          Скопировать всё
        </PrimaryButton>
      </div>
    </div>
  );
}

async function copyText(value) {
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    try {
      const field = document.createElement("textarea");
      field.value = value;
      field.setAttribute("readonly", "");
      field.style.position = "fixed";
      field.style.left = "-9999px";
      document.body.appendChild(field);
      field.select();
      const ok = document.execCommand("copy");
      field.remove();
      return ok;
    } catch {
      return false;
    }
  }
}
