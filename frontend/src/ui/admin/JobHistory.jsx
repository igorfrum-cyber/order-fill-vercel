import { useEffect, useState } from "react";
import { listCompanies, listJobs } from "../../api/auth.js";
import { brandLabel } from "../../features/brands/brandPresentation.js";
import { jobStatusLabel } from "../../features/report/reportModel.js";

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
