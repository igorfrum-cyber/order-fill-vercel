import { useEffect, useState } from "react";
import { listAudit, listJobs, listStatus } from "../../api/auth.js";
import { brandLabel } from "../../features/brands/brandPresentation.js";
import { formatSessionWhen } from "../../features/auth/session.js";
import { accessAuditEvents } from "../../features/ops/auditPresentation.js";
import { presentStatus, statusHeadline } from "../../features/ops/statusPresentation.js";
import { jobStatusHint, jobStatusLabel, liveJobs } from "../../features/report/reportModel.js";
import { IconDatabase, IconFiles, IconQueue, IconServer, IconWorker } from "../icons.jsx";

const STATUS_ICONS = {
  api: IconServer,
  worker: IconWorker,
  postgres: IconDatabase,
  queue: IconQueue,
  files: IconFiles,
};

const POLL_MS = 8000;

export function OverviewScreen({ onOpen }) {
  const [components, setComponents] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [events, setEvents] = useState([]);
  const [error, setError] = useState("");
  const tiles = presentStatus(components);
  const live = liveJobs(jobs);
  const feed = accessAuditEvents(events);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const [statusPayload, jobsPayload, auditPayload] = await Promise.all([listStatus(), listJobs(""), listAudit()]);
        if (cancelled) return;
        setComponents(statusPayload.components || []);
        setJobs(jobsPayload.jobs || []);
        setEvents(auditPayload.events || []);
        setError("");
      } catch (err) {
        if (!cancelled) setError(err.message || "Не удалось загрузить обзор.");
      }
    }
    load();
    const timer = window.setInterval(load, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  return (
    <section className="animate-enter mx-auto max-w-6xl space-y-6 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Обзор</h1>
        <p className="mt-1 text-[14px] text-[var(--color-ink-soft)]">{statusHeadline(tiles)}</p>
      </div>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        {tiles.map((tile) => (
          <StatusTile key={tile.id} tile={tile} icon={STATUS_ICONS[tile.id] || IconServer} />
        ))}
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <article className="rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)]">
          <header className="border-b border-[var(--color-line)] px-4 py-3">
            <h2 className="text-[16px] font-semibold">Сейчас в работе</h2>
            <p className="mt-0.5 text-[13px] text-[var(--color-ink-faint)]">Выгрузки, которые ещё не готовы</p>
          </header>
          <ul data-tour="jobs" className="divide-y divide-[var(--color-line)]">
            {live.map((job) => (
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
                  <JobStatus status={job.status} />
                </button>
              </li>
            ))}
            {!live.length ? (
              <li className="px-4 py-8 text-[14px] text-[var(--color-ink-faint)]">Сейчас никто не считает бланки.</li>
            ) : null}
          </ul>
        </article>
        <article className="rounded-[10px] border border-[var(--color-line)] bg-[var(--color-surface)]">
          <header className="border-b border-[var(--color-line)] px-4 py-3">
            <h2 className="text-[16px] font-semibold">Кто менял доступ</h2>
            <p className="mt-0.5 text-[13px] text-[var(--color-ink-faint)]">Приглашения, сброс, отключения и смена пароля</p>
          </header>
          <ul className="divide-y divide-[var(--color-line)]">
            {feed.map((event) => (
              <li key={event.id} className="px-4 py-3">
                <p className="text-[14px] leading-snug">{event.line}</p>
                <p className="mt-0.5 text-[13px] text-[var(--color-ink-faint)]">{formatSessionWhen(event.at)}</p>
              </li>
            ))}
            {!feed.length ? (
              <li className="px-4 py-8 text-[14px] text-[var(--color-ink-faint)]">Пока никто не менял доступ.</li>
            ) : null}
          </ul>
        </article>
      </div>
    </section>
  );
}

function StatusTile({ tile, icon: Icon }) {
  const tone = !tile.known
    ? "border-[var(--color-line)] bg-[var(--color-surface)]"
    : tile.ok
      ? "border-[color-mix(in_srgb,var(--color-ok)_22%,var(--color-line))] bg-[var(--color-ok-soft)]"
      : "border-[color-mix(in_srgb,var(--color-danger)_22%,var(--color-line))] bg-[var(--color-danger-soft)]";
  const hint = !tile.known
    ? "text-[var(--color-ink-faint)]"
    : tile.ok
      ? "text-[var(--color-ok)]"
      : "text-[var(--color-danger)]";
  return (
    <article className={`rounded-[10px] border p-4 ${tone}`}>
      <Icon className={`h-5 w-5 ${hint}`} />
      <div className="mt-3 text-[15px] font-semibold text-[var(--color-ink)]">{tile.title}</div>
      <div className={`mt-1 text-[13px] ${hint}`}>{tile.hint}</div>
    </article>
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
