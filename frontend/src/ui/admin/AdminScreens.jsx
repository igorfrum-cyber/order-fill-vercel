import { useEffect, useState } from "react";
import { createCompany, createUser, disableCompany, disableUser, listCompanies, listJobs, listUsers, resetUser } from "../../api/auth.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

export function JobHistory({ me, companyId, onCompany, onOpen, onNew }) {
  const [jobs, setJobs] = useState([]);
  const [companies, setCompanies] = useState([]);
  const [error, setError] = useState("");
  const platform = me.role === "platform_admin";
  const canCreate = !platform || Boolean(companyId);

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
    <section className="mx-auto max-w-5xl p-6">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-[22px] font-semibold">Выгрузки</h1>
        <div className="flex flex-wrap items-center gap-2">
          {platform ? (
            <select className="input" value={companyId} onChange={(event) => onCompany(event.target.value)}>
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
          <PrimaryButton onClick={onNew} disabled={!canCreate}>
            Новая выгрузка
          </PrimaryButton>
        </div>
      </div>
      {platform && !companyId ? (
        <p className="mb-3 text-[14px] text-[var(--color-ink-faint)]">Чтобы создать выгрузку, выберите компанию.</p>
      ) : null}
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
                <td className="px-4 py-3">{job.created_at?.slice(0, 16).replace("T", " ")}</td>
                <td className="px-4 py-3">{job.type === "north_merge" ? "Север" : "Бланк"}</td>
                <td className="px-4 py-3">{job.brand}</td>
                <td className="px-4 py-3">{job.status}</td>
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

export function CompaniesScreen({ selectedId, onSelect }) {
  const [companies, setCompanies] = useState([]);
  const [name, setName] = useState("");
  const [error, setError] = useState("");

  function reload() {
    listCompanies().then((payload) => setCompanies(payload.companies || [])).catch(() => setCompanies([]));
  }

  useEffect(reload, []);

  return (
    <section className="mx-auto max-w-3xl space-y-4 p-6">
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
            </button>
            <GhostButton onClick={async () => { await disableCompany(company.id); reload(); }}>Выключить</GhostButton>
          </li>
        ))}
      </ul>
    </section>
  );
}

export function UsersScreen({ companyId }) {
  const [users, setUsers] = useState([]);
  const [login, setLogin] = useState("");
  const [role, setRole] = useState("purchaser");
  const [invite, setInvite] = useState("");

  function reload() {
    if (!companyId) return;
    listUsers(companyId).then((payload) => setUsers(payload.users || [])).catch(() => setUsers([]));
  }

  useEffect(reload, [companyId]);

  if (!companyId) {
    return <p className="p-6 text-[var(--color-ink-faint)]">Сначала выберите компанию.</p>;
  }

  return (
    <section className="mx-auto max-w-3xl space-y-4 p-6">
      <h1 className="text-[22px] font-semibold">Пользователи</h1>
      <form
        className="flex flex-wrap gap-2"
        onSubmit={async (event) => {
          event.preventDefault();
          const created = await createUser(companyId, login, role);
          setInvite(`${window.location.origin}${created.invite_url}`);
          setLogin("");
          reload();
        }}
      >
        <input className="input flex-1" value={login} onChange={(event) => setLogin(event.target.value)} placeholder="Логин" />
        <select className="input" value={role} onChange={(event) => setRole(event.target.value)}>
          <option value="purchaser">Закупщик</option>
          <option value="company_admin">Админ компании</option>
        </select>
        <PrimaryButton type="submit" disabled={!login.trim()}>Пригласить</PrimaryButton>
      </form>
      {invite ? (
        <p className="rounded-lg bg-[var(--color-brand-soft)] p-3 text-[14px] break-all">
          Ссылка-приглашение (скопируйте сейчас): {invite}
        </p>
      ) : null}
      <ul className="divide-y divide-[var(--color-line)] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        {users.map((user) => (
          <li key={user.id} className="flex items-center justify-between gap-2 px-4 py-3">
            <span>
              {user.login} · {user.role}
            </span>
            <span className="flex gap-2">
              <GhostButton
                onClick={async () => {
                  const payload = await resetUser(user.id);
                  setInvite(`${window.location.origin}${payload.invite_url}`);
                }}
              >
                Сброс
              </GhostButton>
              <GhostButton onClick={async () => { await disableUser(user.id); reload(); }}>Выключить</GhostButton>
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
