import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { downloadInboundFile, getInboundCompany, getInboundDeliveries, getInboundMessage, getInboundMessages, getInboundSettings, updateInboundCompany, updateInboundSettings } from "../../api/inbound.js";
import { userFacingError } from "../../features/help/errors.js";
import { Field, GhostButton, PrimaryButton } from "../widgets.jsx";
import { IconFile, IconX } from "../icons.jsx";

const statusLabels = {
  received: "Принято",
  processed: "Обработано",
  "error:unknown_address": "Неизвестный отправитель",
  "error:mismatch_from": "Отправитель не в списке",
  "error:no_attachments": "Нет вложений",
  "error:too_large": "Вложение слишком большое",
};

export function InboundScreen({ me, companyId }) {
  const isPlatform = me.role === "platform_admin";
  return (
    <section className="animate-enter mx-auto max-w-6xl space-y-6 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Интеграция 1С</h1>
        <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Приём писем с файлами заказов 1С по выделенному почтовому адресу.</p>
      </div>
      {isPlatform ? <PlatformPanel /> : <CompanyPanel companyId={companyId} />}
    </section>
  );
}

function PlatformPanel() {
  const [settings, setSettings] = useState(null);
  const [deliveries, setDeliveries] = useState([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [draftAddress, setDraftAddress] = useState("");

  const refresh = () => {
    Promise.all([getInboundSettings(), getInboundDeliveries()])
      .then(([s, d]) => {
        setSettings(s);
        setDeliveries(d.deliveries || d.messages || []);
        if (draftAddress === "") setDraftAddress(s.receive_address || "");
      })
      .catch((err) => setError(userFacingError(err, "Не удалось загрузить статус интеграции.")));
  };

  useEffect(() => {
    refresh();
    const timer = window.setInterval(refresh, 8000);
    return () => window.clearInterval(timer);
  }, []);

  async function toggleEnabled() {
    setSaving(true);
    setError("");
    try {
      const updated = await updateInboundSettings({ enabled: !settings.enabled, receive_address: draftAddress || settings.receive_address });
      setSettings(updated);
    } catch (err) {
      setError(userFacingError(err, "Не удалось изменить статус приёма."));
    } finally {
      setSaving(false);
    }
  }

  async function saveAddress() {
    setSaving(true);
    setError("");
    try {
      const updated = await updateInboundSettings({ enabled: settings.enabled, receive_address: draftAddress });
      setSettings(updated);
    } catch (err) {
      setError(userFacingError(err, "Не удалось сохранить адрес."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      {error ? <p role="alert" className="rounded-xl bg-[var(--color-danger-soft)] p-4 text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      {!settings ? (
        <div className="space-y-3">
          <div className="h-24 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
          <div className="h-64 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
        </div>
      ) : (
        <>
          <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-[17px] font-semibold">Приём писем</h2>
                <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">{settings.enabled ? "Webhook включён, письма принимаются." : "Приём остановлен, письма не обрабатываются."}</p>
              </div>
              <PrimaryButton onClick={toggleEnabled} disabled={saving}>{saving ? "Сохраняю…" : settings.enabled ? "Остановить" : "Включить"}</PrimaryButton>
            </div>
            <dl className="mt-4 grid grid-cols-3 gap-4 text-[13px]">
              <Stat label="Писем принято" value={settings.webhook_count ?? "—"} />
              <Stat label="Ошибок" value={settings.error_count ?? "—"} />
              <Stat label="Последний webhook" value={settings.last_webhook_at ? shortDateTime(settings.last_webhook_at) : "—"} />
            </dl>
          </article>
          <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
            <h2 className="text-[17px] font-semibold">Адрес приёма</h2>
            <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Единый адрес CloudMailin для всей платформы. Компании настраивают 1С на отправку на этот адрес.</p>
            <div className="mt-4 flex gap-2">
              <Field label="Адрес"><input className="input w-full font-mono" type="email" value={draftAddress} onChange={(e) => setDraftAddress(e.target.value.trim())} placeholder="7e1432246b724f3bcd6c@cloudmailin.net" /></Field>
              <div className="flex items-end"><PrimaryButton onClick={saveAddress} disabled={saving || !draftAddress}>Сохранить</PrimaryButton></div>
            </div>
          </article>
          <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
            <h2 className="text-[17px] font-semibold">Поступления</h2>
            <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Только статусы — содержимое писем доступно компаниям-получателям.</p>
            <DeliveriesTable rows={deliveries} />
          </article>
        </>
      )}
    </>
  );
}

function CompanyPanel({ companyId }) {
  const [state, setState] = useState(null);
  const [settings, setSettings] = useState(null);
  const [messages, setMessages] = useState([]);
  const [draft, setDraft] = useState(null);
  const [selectedMessage, setSelectedMessage] = useState(null);
  const [messageLoading, setMessageLoading] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const refreshMessages = () => {
    if (!companyId) return;
    getInboundMessages(companyId)
      .then((payload) => setMessages(payload.messages || []))
      .catch(() => setMessages([]));
  };

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Promise.allSettled([getInboundCompany(companyId), getInboundMessages(companyId), getInboundSettings()])
      .then(([companyResult, messagesResult, settingsResult]) => {
        if (cancelled) return;
        if (companyResult.status === "fulfilled") {
          setState(companyResult.value);
        } else if (companyResult.reason?.status !== 404) {
          setError(userFacingError(companyResult.reason, "Не удалось загрузить настройки интеграции."));
        }
        if (messagesResult.status === "fulfilled") {
          setMessages(messagesResult.value.messages || []);
        }
        if (settingsResult.status === "fulfilled") {
          setSettings(settingsResult.value);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    const timer = window.setInterval(refreshMessages, 8000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [companyId]);

  async function save() {
    setError("");
    try {
      const updated = await updateInboundCompany(companyId, draft);
      setState(updated);
      setDraft(null);
    } catch (err) {
      setError(userFacingError(err, "Не удалось сохранить настройки."));
    }

  }

  async function openMessage(message) {
    setMessageLoading(true);
    setError("");
    try {
      setSelectedMessage(await getInboundMessage(companyId, message.id));
    } catch (err) {
      setError(userFacingError(err, "Не удалось открыть письмо."));
    } finally {
      setMessageLoading(false);
    }
  }

  if (loading) {
    return (
      <>
        <div className="h-40 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
        <div className="h-64 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
      </>
    );
  }

  const company = draft || state || { receive_address: "", sender_email: "", enabled: true };
  const platformAddress = settings?.receive_address || "";

  return (
    <>
      {error ? <p role="alert" className="rounded-xl bg-[var(--color-danger-soft)] p-4 text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-[17px] font-semibold">Настройка приёма</h2>
            <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Укажите email, с которого 1С отправляет письма, и включите приём.</p>
          </div>
          <GhostButton onClick={() => setDraft({ receive_address: company.receive_address || platformAddress, sender_email: company.sender_email || "", enabled: Boolean(company.enabled) })}>Настроить</GhostButton>
        </div>
        <dl className="mt-4 grid grid-cols-2 gap-4 text-[13px]">
          <Stat label="Адрес для 1С" value={platformAddress || "не задан"} mono />
          <Stat label="Приём" value={company.enabled ? "включён" : "остановлен"} />
          <Stat label="Отправитель (email 1С)" value={company.sender_email || "не задан"} mono />
          <Stat label="Писем получено" value={messages.length} />
        </dl>
      </article>
      {draft ? (
        <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
          <h2 className="text-[17px] font-semibold">Настройка приёма</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="text-[13px] text-[var(--color-ink-faint)]">Адрес для 1С</label>
              <p className="mt-1 font-mono text-[14px] text-[var(--color-ink-soft)]">{platformAddress || "не задан"}</p>
            </div>
            <Field label="Отправитель (email 1С)"><input className="input w-full font-mono" type="email" value={draft.sender_email} onChange={(e) => setDraft({ ...draft, sender_email: e.target.value.trim() })} placeholder="1c@company.ru" /></Field>
            <label className="flex items-center gap-3 rounded-control border border-[var(--color-line)] px-4 py-3 text-[14px]">
              <input type="checkbox" checked={draft.enabled} onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })} />
              Принимать письма от 1С
            </label>
          </div>
          <div className="mt-5 flex gap-2">
            <PrimaryButton onClick={save} disabled={!draft.sender_email}>Сохранить</PrimaryButton>
            <GhostButton onClick={() => setDraft(null)}>Отмена</GhostButton>
          </div>
        </article>
      ) : null}
      <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
        <h2 className="text-[17px] font-semibold">Поступления</h2>
        <MessagesTable
          rows={messages}
          onOpen={openMessage}
          onDownload={async (messageId, attachmentId) => {
            setError("");
            try {
              const { blob, fileName } = await downloadInboundFile(companyId, messageId, attachmentId);
              triggerBlobDownload(blob, fileName);
            } catch (err) {
              setError(userFacingError(err, "Не удалось скачать вложение."));
            }
          }}
        />
      </article>
      {messageLoading ? <p className="text-[13px] text-[var(--color-ink-faint)]">Открываю письмо…</p> : null}
      {selectedMessage ? <MessageViewer message={selectedMessage} onClose={() => setSelectedMessage(null)} /> : null}
    </>
  );
}

function DeliveriesTable({ rows }) {
  if (!rows.length) {
    return <p className="mt-4 text-[13px] text-[var(--color-ink-faint)]">Поступлений пока нет.</p>;
  }
  return (
    <div className="mt-4 overflow-x-auto">
      <table className="w-full text-left text-[13px]">
        <thead>
          <tr className="border-b border-[var(--color-line)] text-[var(--color-ink-faint)]">
            <th className="py-2 pr-4 font-medium">Время</th>
            <th className="py-2 pr-4 font-medium">Компания</th>
            <th className="py-2 pr-4 font-medium">Статус</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="border-b border-[var(--color-line-soft)] last:border-0">
              <td className="py-2 pr-4 text-[var(--color-ink-soft)]">{shortDateTime(row.received_at)}</td>
              <td className="py-2 pr-4">{row.company_id || "—"}</td>
              <td className="py-2 pr-4"><StatusPill status={row.status} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function MessagesTable({ rows, onDownload, onOpen }) {
  if (!rows.length) {
    return <p className="mt-4 text-[13px] text-[var(--color-ink-faint)]">Писем от 1С пока не было.</p>;
  }
  return (
    <div className="mt-4 overflow-x-auto">
      <table className="w-full text-left text-[13px]">
        <thead>
          <tr className="border-b border-[var(--color-line)] text-[var(--color-ink-faint)]">
            <th className="py-2 pr-4 font-medium">Время</th>
            <th className="py-2 pr-4 font-medium">Тема</th>
            <th className="py-2 pr-4 font-medium">Отправитель</th>
            <th className="py-2 pr-4 font-medium">Вложения</th>
            <th className="py-2 pr-4 font-medium">Статус</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="border-b border-[var(--color-line-soft)] last:border-0">
              <td className="py-2 pr-4 text-[var(--color-ink-soft)]">{shortDateTime(row.received_at)}</td>
              <td className="py-2 pr-4">
                <button type="button" className="font-medium text-[var(--color-brand)] hover:underline" onClick={() => onOpen(row)}>
                  {row.subject || "Без темы"}
                </button>
              </td>
              <td className="py-2 pr-4">{row.envelope_from || "—"}</td>
              <td className="py-2 pr-4">
                {(row.attachments || []).map((attachment) => (
                  <button key={attachment.id} type="button" className="mr-2 inline-flex items-center gap-1 font-medium text-[var(--color-brand)] hover:underline" onClick={() => onDownload(row.id, attachment.id)}>
                    {attachment.name || "файл"} <span className="text-[var(--color-ink-faint)]">{formatBytes(attachment.size)}</span>
                  </button>
                ))}
                {(!row.attachments || row.attachments.length === 0) && row.attachment_count > 0 && (
                  <span className="text-[var(--color-ink-faint)]">{row.attachment_count} файл(ов)</span>
                )}
              </td>
              <td className="py-2 pr-4"><StatusPill status={row.status} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function MessageViewer({ message, onClose }) {
  const html = String(message.body_html || "").trim();
  const plain = String(message.body_text || "").trim();
  const attachments = message.attachments || [];

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  return createPortal(
    (
    <div
      className="message-viewer-backdrop fixed inset-0 z-30 flex items-center justify-center p-3 sm:p-6"
      role="dialog"
      aria-modal="true"
      aria-label="Письмо"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <article className="animate-enter flex max-h-[92vh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-white/70 bg-[var(--color-surface)] shadow-[0_24px_80px_rgba(15,23,42,0.28)]">
        <header className="border-b border-[var(--color-line)] bg-gradient-to-b from-white to-slate-50/70 px-5 py-4 sm:px-7 sm:py-5">
          <div className="flex items-start gap-4">
            <div className="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-[var(--color-brand-soft)] text-[var(--color-brand)]">
              <IconFile className="h-5 w-5" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="mb-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-[var(--color-brand)]">Полученное письмо</p>
              <h2 className="truncate text-[19px] font-semibold tracking-tight sm:text-[21px]">{message.subject || "Без темы"}</h2>
              <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[13px] text-[var(--color-ink-faint)]">
                <span><strong className="font-medium text-[var(--color-ink-soft)]">От:</strong> {message.envelope_from || "—"}</span>
                <span><strong className="font-medium text-[var(--color-ink-soft)]">Получено:</strong> {shortDateTime(message.received_at)}</span>
              </div>
            </div>
            <button
              type="button"
              className="grid h-9 w-9 shrink-0 place-items-center rounded-lg text-[var(--color-ink-faint)] transition hover:bg-[var(--color-line-soft)] hover:text-[var(--color-ink)] focus:outline-none focus-visible:ring-4 focus-visible:ring-[var(--color-brand-soft)]"
              onClick={onClose}
              aria-label="Закрыть письмо"
              title="Закрыть (Esc)"
            >
              <IconX className="h-5 w-5" />
            </button>
          </div>
          {attachments.length ? (
            <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-[var(--color-line-soft)] pt-3">
              <span className="text-[12px] font-medium text-[var(--color-ink-faint)]">Вложения:</span>
              {attachments.map((attachment) => (
                <span key={attachment.id} className="inline-flex items-center gap-1.5 rounded-lg bg-[var(--color-neutral-soft)] px-2.5 py-1.5 text-[12px] font-medium text-[var(--color-ink-soft)]">
                  <IconFile className="h-3.5 w-3.5 text-[var(--color-brand)]" />
                  {attachment.name || "файл"}
                  {attachment.size ? <span className="font-normal text-[var(--color-ink-faint)]">{formatBytes(attachment.size)}</span> : null}
                </span>
              ))}
            </div>
          ) : null}
        </header>
        <div className="min-h-0 overflow-auto bg-[linear-gradient(180deg,#f8fafc_0%,#f1f5f9_100%)] p-3 sm:p-6">
          {html ? (
            <div className="mx-auto max-w-4xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_8px_30px_rgba(15,23,42,0.07)]">
              <div className="flex items-center justify-between border-b border-slate-100 bg-white px-4 py-2.5">
                <span className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">Содержимое письма</span>
                <span className="inline-flex items-center gap-1.5 text-[11px] text-slate-400">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  Безопасный просмотр
                </span>
              </div>
              <iframe
                title="Содержимое письма"
                className="block min-h-[52vh] w-full bg-white sm:min-h-[480px]"
                sandbox=""
                referrerPolicy="no-referrer"
                srcDoc={prepareEmailHTML(html)}
              />
            </div>
          ) : plain ? (
            <div className="mx-auto max-w-4xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_8px_30px_rgba(15,23,42,0.07)]">
              <div className="border-b border-slate-100 px-4 py-2.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">Текст письма</div>
              <pre className="min-h-[52vh] whitespace-pre-wrap p-6 text-[14px] leading-7 text-slate-700 sm:min-h-[480px]">{plain}</pre>
            </div>
          ) : (
            <div className="grid min-h-[260px] place-items-center rounded-xl border border-dashed border-[var(--color-line)] bg-white p-8 text-center shadow-sm">
              <div>
                <div className="mx-auto grid h-12 w-12 place-items-center rounded-full bg-[var(--color-neutral-soft)] text-[var(--color-ink-faint)]">
                  <IconFile className="h-5 w-5" />
                </div>
                <p className="mt-3 text-[14px] font-medium text-[var(--color-ink-soft)]">Текст письма отсутствует</p>
                <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">В письме есть только заголовки и вложения.</p>
              </div>
            </div>
          )}
        </div>
      </article>
    </div>
    ),
    document.body,
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

function StatusPill({ status }) {
  const failed = String(status || "").startsWith("error:");
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[12px] font-medium ${failed ? "bg-[var(--color-danger-soft)] text-[var(--color-danger)]" : "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]"}`}>
      {statusLabels[status] || status}
    </span>
  );
}

function Stat({ label, value, mono = false }) {
  return (
    <div>
      <dt className="text-[var(--color-ink-faint)]">{label}</dt>
      <dd className={`mt-0.5 text-[var(--color-ink-soft)] ${mono ? "font-mono" : ""}`}>{String(value)}</dd>
    </div>
  );
}

function shortDateTime(value) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", year: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function formatBytes(value) {
  if (value == null) return "";
  if (value < 1024) return `${value} Б`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} КБ`;
  return `${(value / (1024 * 1024)).toFixed(1)} МБ`;
}

function prepareEmailHTML(html) {
  const baseStyles = `
    <meta name="color-scheme" content="light">
    <style>
      html { background: #ffffff; }
      body { margin: 0; padding: 24px; color: #172033; font-family: Arial, Helvetica, sans-serif; font-size: 15px; line-height: 1.6; }
      img { max-width: 100%; height: auto; }
      table { max-width: 100%; }
      a { color: #4f46e5; }
      h1, h2, h3 { line-height: 1.25; }
    </style>
  `;
  if (/<head[\s>]/i.test(html)) {
    return html.replace(/<head(\s[^>]*)?>/i, (tag) => `${tag}${baseStyles}`);
  }
  if (/<html[\s>]/i.test(html)) {
    return html.replace(/<html(\s[^>]*)?>/i, (tag) => `${tag}<head>${baseStyles}</head>`);
  }
  return `<!doctype html><html><head>${baseStyles}</head><body>${html}</body></html>`;
}
