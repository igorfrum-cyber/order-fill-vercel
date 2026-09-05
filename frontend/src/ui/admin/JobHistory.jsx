import { useEffect, useState } from "react";
import { listCompanies, listJobs } from "../../api/auth.js";
import { brandLabel } from "../../features/brands/brandPresentation.js";
import { jobStatusHint, jobStatusLabel, jobsEmptyState } from "../../features/report/reportModel.js";
import { userFacingError } from "../../features/help/errors.js";

export function JobHistory({ me, companyId, onCompany, onOpen, onNew }) {
  const [jobs, setJobs] = useState([]);
  const [companies, setCompanies] = useState([]);
  const [error, setError] = useState("");
  const platform = me.role === "platform_admin";
  const canCreate = !platform;

  useEffect(() => {
    listJobs(platform ? companyId : "")
      .then((payload) => setJobs(payload.jobs || []))
      .catch((err) => setError(userFacingError(err, "Не удалось загрузить историю.")));
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
          <select data-tour="company-select" className="input max-w-xs" value={companyId} onChange={(event) => onCompany(event.target.value)}>
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
        <div className="mb-4 flex flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            data-tour="order"
            onClick={() => onNew("order")}
            className="rounded-lg bg-[var(--color-brand)] px-3 py-2 text-[14px] font-medium text-white transition hover:bg-[var(--color-brand-strong)]"
          >
            Заполнить бланк закупки
          </button>
          <button
            type="button"
            data-tour="north"
            onClick={() => onNew("north")}
            className="rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] px-3 py-2 text-[14px] font-medium text-[var(--color-ink-soft)] transition hover:border-[var(--color-brand)] hover:text-[var(--color-ink)]"
          >
            Соединить северные бланки
          </button>
        </div>
      ) : (
        <p className="mb-4 text-[14px] text-[var(--color-ink-soft)]">
          Здесь только просмотр: новую выгрузку создаёт закупщик или администратор компании.
        </p>
      )}
      {error ? <p className="text-[var(--color-danger)]">{error}</p> : null}
      <div data-tour="jobs" className="overflow-hidden rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
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
                  {jobsEmptyState(me.role)}
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
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-[13px] font-medium ${tone}`}
      title={jobStatusHint(status)}
    >
      {jobStatusLabel(status)}
    </span>
  );
}
