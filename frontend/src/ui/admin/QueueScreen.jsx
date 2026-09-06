import { useEffect, useState } from "react";
import { listJobs, listUsers } from "../../api/auth.js";
import { brandLabel } from "../../features/brands/brandPresentation.js";
import { formatSessionWhen } from "../../features/auth/session.js";
import { historyDeskLine, jobNextAction, jobStatusHint, jobStatusLabel, queueJobs } from "../../features/report/reportModel.js";
import { userFacingError } from "../../features/help/errors.js";

const POLL_MS = 8000;

export function QueueScreen({ me, onOpen, onPeople, onCompany }) {
  const [jobs, setJobs] = useState(null);
  const [users, setUsers] = useState(null);
  const [error, setError] = useState("");
  const stuck = queueJobs(jobs || []);
  const desk = jobs ? historyDeskLine(jobs) : "";
  const purchasers = (users || []).filter((user) => user.role === "purchaser" && !user.disabled_at);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const jobsPayload = await listJobs("");
        const usersPayload = me.company_id ? await listUsers(me.company_id).catch(() => ({ users: [] })) : { users: [] };
        if (cancelled) return;
        setJobs(jobsPayload.jobs || []);
        setUsers(usersPayload.users || []);
        setError("");
      } catch (err) {
        if (!cancelled) setError(userFacingError(err, "Не удалось загрузить очередь."));
      }
    }
    load();
    const timer = window.setInterval(load, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [me.company_id]);

  return (
    <section className="animate-enter mx-auto max-w-5xl space-y-6 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Очередь</h1>
        <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">
          {jobs === null ? "Смотрю, что не доделано." : desk || "Сейчас ничего не ждёт проверки."}
        </p>
      </div>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <div className="grid gap-4 lg:grid-cols-2">
        <article className="rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)]">
          <header className="border-b border-[var(--color-line)] px-4 py-3">
            <h2 className="text-[16px] font-semibold">Застрявшие выгрузки</h2>
            <p className="mt-0.5 text-[13px] text-[var(--color-ink-faint)]">На проверке, в работе и со сбоем</p>
          </header>
          <ul data-tour="jobs" className="divide-y divide-[var(--color-line)]">
            {jobs === null ? (
              <li className="px-4 py-6">
                <div className="h-4 w-2/3 animate-pulse rounded bg-[var(--color-line-soft)]" />
              </li>
            ) : null}
            {stuck.map((job) => (
              <li key={job.id}>
                <button
                  type="button"
                  className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-[var(--color-line-soft)]"
                  onClick={() => onOpen?.(job)}
                >
                  <span className="min-w-0">
                    <span className="block truncate text-[14px] font-medium">
                      {job.type === "north_merge" ? "Север" : "Бланк"} · {brandLabel(job.brand)}
                    </span>
                    <span className="mt-0.5 block truncate text-[13px] text-[var(--color-ink-faint)]">
                      {job.created_by_login || "—"}
                      {job.created_at ? ` · ${formatSessionWhen(job.created_at)}` : ""}
                    </span>
                  </span>
                  <span className="flex shrink-0 items-center gap-3">
                    <JobStatus status={job.status} />
                    {jobNextAction(job) ? (
                      <span className="text-[13px] font-medium text-[var(--color-brand)]">{jobNextAction(job)}</span>
                    ) : null}
                  </span>
                </button>
              </li>
            ))}
            {jobs && !stuck.length ? (
              <li className="px-4 py-8 text-[14px] text-[var(--color-ink-faint)]">Сейчас никто не ждёт вас.</li>
            ) : null}
          </ul>
        </article>
        <div className="space-y-4">
          <article className="rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)] p-4">
            <h2 className="text-[16px] font-semibold">Люди</h2>
            {users === null ? (
              <div className="mt-3 h-4 w-1/2 animate-pulse rounded bg-[var(--color-line-soft)]" />
            ) : purchasers.length ? (
              <p className="mt-2 text-[14px] text-[var(--color-ink-soft)]">
                {purchasers.length === 1 ? "Есть закупщик, который заполняет бланки." : `Закупщиков: ${purchasers.length}.`}
              </p>
            ) : (
              <p className="mt-2 text-[14px] text-[var(--color-ink-soft)]">
                Бланки заполняет закупщик. Пока никого нет — пригласите по ссылке.
              </p>
            )}
            <button type="button" className="mt-3 text-[14px] font-medium text-[var(--color-brand)]" onClick={onPeople}>
              {purchasers.length ? "К людям" : "Пригласить закупщика"}
            </button>
          </article>
          <article className="rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)] p-4">
            <h2 className="text-[16px] font-semibold">Компания</h2>
            <p className="mt-2 text-[14px] text-[var(--color-ink-soft)]">{me.company_name || "Название и адрес входа"}</p>
            <button type="button" className="mt-3 text-[14px] font-medium text-[var(--color-brand)]" onClick={onCompany}>
              Профиль компании
            </button>
          </article>
        </div>
      </div>
    </section>
  );
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
    <span className={`inline-flex shrink-0 rounded-full px-2.5 py-1 text-[13px] font-medium ${tone}`} title={jobStatusHint(status)}>
      {jobStatusLabel(status)}
    </span>
  );
}
