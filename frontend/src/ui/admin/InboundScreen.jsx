import { useEffect, useState } from "react";
import { downloadInboundFile, getInboundCompany, getInboundDeliveries, getInboundMessages, getInboundSettings, updateInboundCompany, updateInboundSettings } from "../../api/inbound.js";
import { userFacingError } from "../../features/help/errors.js";
import { Field, GhostButton, PrimaryButton } from "../widgets.jsx";

const statusLabels = {
  received: "Принято",
  processed: "Обработано",
  "error:unknown_address": "Неизвестный адрес",
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

  const refresh = () => {
    Promise.all([getInboundSettings(), getInboundDeliveries()])
      .then(([s, d]) => {
        setSettings(s);
        setDeliveries(d.deliveries || d.messages || []);
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
      const updated = await updateInboundSettings({ enabled: !settings.enabled });
      setSettings(updated);
    } catch (err) {
      setError(userFacingError(err, "Не удалось изменить статус приёма."));
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
  const [messages, setMessages] = useState([]);
  const [draft, setDraft] = useState(null);
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
    Promise.all([getInboundCompany(companyId), getInboundMessages(companyId)])
      .then(([company, payload]) => {
        if (cancelled) return;
        setState(company);
        setMessages(payload.messages || []);
      })
      .catch((err) => {
        if (cancelled) return;
        if (err?.status === 404) return;
        setError(userFacingError(err, "Не удалось загрузить настройки интеграции."));
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

  if (loading) {
    return (
      <>
        <div className="h-40 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
        <div className="h-64 animate-pulse rounded-xl bg-[var(--color-line-soft)]" />
      </>
    );
  }

  const company = draft || state || { receive_address: "", allowed_from: [], enabled: true };

  return (
    <>
      {error ? <p role="alert" className="rounded-xl bg-[var(--color-danger-soft)] p-4 text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-[17px] font-semibold">Почтовый адрес для 1С</h2>
            <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Письма с этого адреса обрабатываются, а файлы попадают в компанию.</p>
          </div>
          <GhostButton onClick={() => setDraft({ receive_address: company.receive_address || "", allowed_from: [...(company.allowed_from || [])], enabled: Boolean(company.enabled) })}>Настроить</GhostButton>
        </div>
        <dl className="mt-4 grid grid-cols-2 gap-4 text-[13px]">
          <Stat label="Адрес" value={company.receive_address ? company.receive_address : "не задан"} mono />
          <Stat label="Приём" value={company.enabled ? "включён" : "остановлен"} />
          <Stat label="Допустимые отправители" value={(company.allowed_from || []).length ? company.allowed_from.join(", ") : "любой отправитель"} mono />
          <Stat label="Писем получено" value={messages.length} />
        </dl>
      </article>
      {draft ? (
        <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
          <h2 className="text-[17px] font-semibold">Настройка приёма</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Field label="Адрес приёма"><input className="input w-full" type="email" value={company.receive_address} onChange={(e) => setDraft({ ...company, receive_address: e.target.value.trim() })} placeholder="zakaz@example.com" /></Field>
            <Field label="Допустимые отправители"><input className="input w-full" value={company.allowed_from.join(", ")} onChange={(e) => setDraft({ ...company, allowed_from: e.target.value.split(",").map((value) => value.trim()).filter(Boolean) })} placeholder="1с@example.com" /></Field>
            <label className="flex items-center gap-3 rounded-control border border-[var(--color-line)] px-4 py-3 text-[14px]">
              <input type="checkbox" checked={company.enabled} onChange={(e) => setDraft({ ...company, enabled: e.target.checked })} />
              Принимать письма от 1С
            </label>
          </div>
          <div className="mt-5 flex gap-2">
            <PrimaryButton onClick={save} disabled={!company.receive_address}>Сохранить</PrimaryButton>
            <GhostButton onClick={() => setDraft(null)}>Отмена</GhostButton>
          </div>
        </article>
      ) : null}
      <article className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5">
        <h2 className="text-[17px] font-semibold">Поступления</h2>
        <MessagesTable
          rows={messages}
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

function MessagesTable({ rows, onDownload }) {
  if (!rows.length) {
    return <p className="mt-4 text-[13px] text-[var(--color-ink-faint)]">Писем от 1С пока не было.</p>;
  }
  return (
    <div className="mt-4 overflow-x-auto">
      <table className="w-full text-left text-[13px]">
        <thead>
          <tr className="border-b border-[var(--color-line)] text-[var(--color-ink-faint)]">
            <th className="py-2 pr-4 font-medium">Время</th>
            <th className="py-2 pr-4 font-medium">Отправитель</th>
            <th className="py-2 pr-4 font-medium">Вложения</th>
            <th className="py-2 pr-4 font-medium">Статус</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="border-b border-[var(--color-line-soft)] last:border-0">
              <td className="py-2 pr-4 text-[var(--color-ink-soft)]">{shortDateTime(row.received_at)}</td>
              <td className="py-2 pr-4">{row.envelope_from || "—"}</td>
              <td className="py-2 pr-4">
                {(row.attachments || []).map((attachment) => (
                  <button key={attachment.id} type="button" className="mr-2 inline-flex items-center gap-1 font-medium text-[var(--color-brand)] hover:underline" onClick={() => onDownload(row.id, attachment.id)}>
                    {attachment.filename || "файл"} <span className="text-[var(--color-ink-faint)]">{formatBytes(attachment.size)}</span>
                  </button>
                ))}
              </td>
              <td className="py-2 pr-4"><StatusPill status={row.status} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
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