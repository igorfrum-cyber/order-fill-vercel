import { useEffect, useMemo, useState } from "react";
import { listJobs } from "../../api/auth.js";
import { downloadJobFile, listJobFiles } from "../../api/jobs.js";
import { ORDER_BRANDS, brandLabel } from "../../features/brands/brandPresentation.js";
import {
  filterJobs,
  jobNextAction,
  jobStatusHint,
  jobStatusLabel,
  jobsEmptyMessage,
} from "../../features/report/reportModel.js";
import { userFacingError } from "../../features/help/errors.js";

const POLL_MS = 8000;

export function JobHistory({ me, companyId, onOpen, onNew }) {
  const [jobs, setJobs] = useState(null);
  const [error, setError] = useState("");
  const [brand, setBrand] = useState("");
  const [status, setStatus] = useState("");
  const [month, setMonth] = useState("");
  const [busyId, setBusyId] = useState("");
  const platform = me.role === "platform_admin";
  const canCreate = !platform;
  const visible = useMemo(() => filterJobs(jobs || [], { brand, status, month }), [jobs, brand, status, month]);
  const months = [...new Set((jobs || []).map((job) => String(job.created_at || "").slice(0, 7)).filter(Boolean))];
  const brands = [...new Set([...(jobs || []).map((job) => job.brand).filter(Boolean), ...ORDER_BRANDS.map((item) => item.id)])];

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const payload = await listJobs(platform ? companyId : "");
        if (!cancelled) {
          setJobs(payload.jobs || []);
          setError("");
        }
      } catch (err) {
        if (!cancelled) setError(userFacingError(err, "Не удалось загрузить историю."));
      }
    }
    load();
    const timer = window.setInterval(load, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [me, companyId, platform]);

  async function download(event, job) {
    event.preventDefault();
    event.stopPropagation();
    setBusyId(job.id);
    try {
      const payload = await listJobFiles(job.id);
      for (const file of payload.files || []) {
        const item = await downloadJobFile(job.id, file.id);
        triggerBlobDownload(item.blob, item.fileName || file.name);
      }
    } catch (err) {
      setError(userFacingError(err, "Не удалось скачать файлы."));
    } finally {
      setBusyId("");
    }
  }

  return (
    <section className="animate-enter mx-auto max-w-5xl p-6">
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-[22px] font-semibold">{me.role === "platform_admin" ? "Выгрузки" : "Файлы"}</h1>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-2">
          <select className="input py-2" value={brand} onChange={(event) => setBrand(event.target.value)} aria-label="Бренд">
            <option value="">Все бренды</option>
            {brands.map((id) => (
              <option key={id} value={id}>
                {brandLabel(id)}
              </option>
            ))}
          </select>
          <select className="input py-2" value={status} onChange={(event) => setStatus(event.target.value)} aria-label="Статус">
            <option value="">Все статусы</option>
            <option value="live">В работе</option>
            <option value="needs_review">На проверке</option>
            <option value="completed">Готово</option>
            <option value="failed">Сбой</option>
          </select>
          <select className="input py-2" value={month} onChange={(event) => setMonth(event.target.value)} aria-label="Месяц">
            <option value="">Все месяцы</option>
            {months.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
          {canCreate ? (
            <>
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
            </>
          ) : null}
        </div>
      </div>
      {!canCreate ? (
        <p className="mb-4 text-[14px] text-[var(--color-ink-soft)]">
          Здесь только просмотр: новую выгрузку создаёт закупщик, владелец или администратор компании.
        </p>
      ) : null}
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
              <th className="px-4 py-3 font-medium"> </th>
            </tr>
          </thead>
          <tbody>
            {jobs === null ? (
              <tr>
                <td className="px-4 py-6" colSpan={6}>
                  <div className="h-4 w-2/3 animate-pulse rounded bg-[var(--color-line-soft)]" />
                </td>
              </tr>
            ) : null}
            {visible.map((job) => {
              const action = jobNextAction(job);
              return (
                <tr key={job.id} className="border-t border-[var(--color-line)] hover:bg-[var(--color-line-soft)]">
                  <td className="px-4 py-3">
                    <a
                      href={`/jobs/${encodeURIComponent(job.id)}`}
                      className="font-medium text-[var(--color-ink)]"
                      onClick={(event) => {
                        event.preventDefault();
                        onOpen(job);
                      }}
                    >
                      {formatJobWhen(job.created_at)}
                    </a>
                  </td>
                  <td className="px-4 py-3">{job.type === "north_merge" ? "Север" : "Бланк"}</td>
                  <td className="px-4 py-3">{brandLabel(job.brand)}</td>
                  <td className="px-4 py-3">
                    <JobStatus status={job.status} />
                  </td>
                  <td className="px-4 py-3">{job.created_by_login || "—"}</td>
                  <td className="px-4 py-3">
                    {action === "Скачать" ? (
                      <button
                        type="button"
                        className="text-[13px] font-medium text-[var(--color-brand)]"
                        disabled={busyId === job.id}
                        onClick={(event) => download(event, job)}
                      >
                        {busyId === job.id ? "…" : "Скачать"}
                      </button>
                    ) : action ? (
                      <a
                        href={`/jobs/${encodeURIComponent(job.id)}`}
                        className="text-[13px] font-medium text-[var(--color-brand)]"
                        onClick={(event) => {
                          event.preventDefault();
                          onOpen(job);
                        }}
                      >
                        {action}
                      </a>
                    ) : null}
                  </td>
                </tr>
              );
            })}
            {jobs && !visible.length ? (
              <tr>
                <td className="px-4 py-8 text-[var(--color-ink-faint)]" colSpan={6}>
                  {jobsEmptyMessage(me.role, jobs, visible, companyId)}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function triggerBlobDownload(blob, fileName) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName || "файл";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
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
    <span className={`inline-flex rounded-full px-2.5 py-1 text-[13px] font-medium ${tone}`} title={jobStatusHint(status)}>
      {jobStatusLabel(status)}
    </span>
  );
}
