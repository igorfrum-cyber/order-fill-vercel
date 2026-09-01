import { useEffect, useState } from "react";
import { createCompany, createUser, disableCompany, disableUser, listCompanies, listJobs, listUsers, resetUser } from "../../api/auth.js";
import { inviteRoleOptions, roleLabel } from "../../features/auth/accessPresentation.js";
import { brandLabel } from "../../features/brands/brandPresentation.js";
import { jobStatusLabel } from "../../features/report/reportModel.js";
import { IconCopy } from "../icons.jsx";
import { GhostButton, Modal, PrimaryButton } from "../widgets.jsx";

export function JobHistory({ me, companyId, onCompany, onOpen, onNew }) {
  const [jobs, setJobs] = useState([]);
  const [companies, setCompanies] = useState([]);
  const [error, setError] = useState("");
  const platform = me.role === "platform_admin";
  const canCreate = !platform;

  useEffect(() => {
    listJobs(platform ? companyId : "")
      .then((payload) => setJobs(payload.jobs || []))
      .catch((err) => setError(err.message || "Не удалось загрузить историю."));
  }, [me, companyId, platform]);

  useEffect(() => {
    if (!platform) return undefined;
    listCompanies()
      .then((payload) => setCompanies(payload.companies || []))
      .catch(() => setCompanies([]));
    return undefined;
  }, [platform]);

  return (
    <section className="animate-enter mx-auto max-w-5xl p-6">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-[22px] font-semibold">Выгрузки</h1>
        {platform ? (
          <select className="input max-w-xs" value={companyId} onChange={(event) => onCompany(event.target.value)}>
            <option value="">Все компании</option>
            {companies
              .filter((company) => !company.disabled_at)
              .map((company) => (
                <option key={company.id} value={company.id}>
                  {company.name}
                </option>
              ))}
          </select>
        ) : null}
      </div>
      {canCreate ? (
        <div className="mb-6 grid gap-3 sm:grid-cols-2">
          <JobTypeCard
            title="Бланк закупки"
            hint="Таблица заказа и текущий бланк поставщика"
            onClick={() => onNew("order")}
          />
          <JobTypeCard
            title="Север"
            hint="Объединение городских бланков и таблицы Тюмени"
            onClick={() => onNew("north")}
          />
        </div>
      ) : (
        <p className="mb-4 text-[14px] text-[var(--color-ink-soft)]">
          Здесь только просмотр: новую выгрузку создаёт закупщик или админ компании.
        </p>
      )}
      {error ? <p className="text-[var(--color-danger)]">{error}</p> : null}
      <div className="overflow-hidden rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <table className="w-full text-left text-[14px]">
          <thead className="bg-[var(--color-ground)] text-[var(--color-ink-faint)]">
            <tr>
              <th className="px-4 py-3 font-medium">Дата</th>
              <th className="px-4 py-3 font-medium">Тип</th>
              <th className="px-4 py-3 font-medium">Бренд</th>
              <th className="px-4 py-3 font-medium">Статус</th>
              <th className="px-4 py-3 font-medium">Автор</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.id} className="cursor-pointer border-t border-[var(--color-line)] hover:bg-[var(--color-line-soft)]" onClick={() => onOpen(job)}>
                <td className="px-4 py-3">{formatJobWhen(job.created_at)}</td>
                <td className="px-4 py-3">{job.type === "north_merge" ? "Север" : "Бланк"}</td>
                <td className="px-4 py-3">{brandLabel(job.brand)}</td>
                <td className="px-4 py-3">
                  <JobStatus status={job.status} />
                </td>
                <td className="px-4 py-3">{job.created_by_login || "—"}</td>
              </tr>
            ))}
            {!jobs.length ? (
              <tr>
                <td className="px-4 py-8 text-[var(--color-ink-faint)]" colSpan={5}>
                  Пока нет выгрузок
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function formatJobWhen(iso) {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("ru-RU", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function JobStatus({ status }) {
  const tone = {
    completed: "bg-[var(--color-ok-soft)] text-[var(--color-ok)]",
    needs_review: "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]",
    failed: "bg-[var(--color-danger-soft)] text-[var(--color-danger)]",
    processing: "bg-[var(--color-neutral-soft)] text-[var(--color-ink-soft)]",
    queued: "bg-[var(--color-neutral-soft)] text-[var(--color-ink-soft)]",
    finalizing: "bg-[var(--color-neutral-soft)] text-[var(--color-ink-soft)]",
  }[status] || "bg-[var(--color-neutral-soft)] text-[var(--color-ink-soft)]";
  return <span className={`inline-flex rounded-full px-2.5 py-1 text-[13px] font-medium ${tone}`}>{jobStatusLabel(status)}</span>;
}

function JobTypeCard({ title, hint, onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5 text-left transition hover:border-[var(--color-brand)] hover:shadow-sm"
    >
      <div className="text-[16px] font-semibold">{title}</div>
      <p className="mt-1 text-[13px] leading-relaxed text-[var(--color-ink-soft)]">{hint}</p>
    </button>
  );
}

export function CompaniesScreen({ selectedId, onSelect }) {
  const [companies, setCompanies] = useState([]);
  const [name, setName] = useState("");
  const [error, setError] = useState("");

  function reload() {
    listCompanies().then((payload) => setCompanies(payload.companies || [])).catch(() => setCompanies([]));
  }

  useEffect(reload, []);

  return (
    <section className="animate-enter mx-auto max-w-3xl space-y-4 p-6">
      <h1 className="text-[22px] font-semibold">Компании</h1>
      <form
        className="flex gap-2"
        onSubmit={async (event) => {
          event.preventDefault();
          setError("");
          try {
            await createCompany(name);
            setName("");
            reload();
          } catch (err) {
            setError(err.message || "Не удалось создать компанию.");
          }
        }}
      >
        <input className="input flex-1" value={name} onChange={(event) => setName(event.target.value)} placeholder="Название" />
        <PrimaryButton type="submit" disabled={!name.trim()}>Создать</PrimaryButton>
      </form>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <ul className="divide-y divide-[var(--color-line)] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        {companies.map((company) => (
          <li key={company.id} className="flex items-center justify-between px-4 py-3">
            <button
              type="button"
              className={`text-left font-medium ${company.id === selectedId ? "text-[var(--color-brand-strong)]" : ""}`}
              onClick={() => onSelect(company.id)}
            >
              {company.name}
              {company.disabled_at ? <span className="ml-2 text-[13px] font-normal text-[var(--color-ink-faint)]">выключена</span> : null}
            </button>
            <GhostButton
              onClick={async () => {
                try {
                  await disableCompany(company.id);
                  reload();
                } catch (err) {
                  setError(err.message || "Не удалось выключить компанию.");
                }
              }}
            >
              Выключить
            </GhostButton>
          </li>
        ))}
      </ul>
    </section>
  );
}

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
