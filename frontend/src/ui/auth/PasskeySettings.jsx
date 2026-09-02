import { useEffect, useState } from "react";
import { deletePasskey, listPasskeys } from "../../api/auth.js";
import {
  passkeyAddButton,
  passkeyDeleteButton,
  passkeySettingsHint,
  passkeySettingsTitle,
} from "../../features/help/copy.js";
import { passkeySupported } from "../../features/auth/passkey.js";
import { createPasskey } from "../../features/auth/passkeyFlow.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

export function PasskeySettings() {
  const [items, setItems] = useState([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const supported = passkeySupported();

  async function refresh() {
    const result = await listPasskeys();
    setItems(result.passkeys || []);
  }

  useEffect(() => {
    refresh().catch(() => setError("Не удалось загрузить ключи доступа."));
  }, []);

  async function add() {
    setBusy(true);
    setError("");
    try {
      await createPasskey();
      await refresh();
    } catch (err) {
      if (err?.name === "NotAllowedError") {
        setError("Добавление отменено.");
      } else {
        setError("Не удалось добавить ключ доступа. Попробуйте другое приложение или ключ.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function remove(id) {
    setBusy(true);
    setError("");
    try {
      await deletePasskey(id);
      await refresh();
    } catch {
      setError("Не удалось удалить ключ доступа.");
    } finally {
      setBusy(false);
    }
  }

  if (!supported) {
    return (
      <div className="space-y-3 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
        <h2 className="text-[16px] font-semibold">{passkeySettingsTitle}</h2>
        <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
          Этот браузер не умеет входить по ключу доступа. Откройте Chrome, Safari или Edge.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
      <h2 className="text-[16px] font-semibold">{passkeySettingsTitle}</h2>
      <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{passkeySettingsHint}</p>
      {items.length ? (
        <ul className="space-y-2">
          {items.map((item) => (
            <li key={item.id} className="flex items-center justify-between gap-3 rounded-xl bg-[var(--color-ground)] px-3 py-2">
              <div>
                <p className="text-[15px] font-medium">{item.name}</p>
                <p className="text-[13px] text-[var(--color-ink-faint)]">{formatPasskeyDate(item.created_at)}</p>
              </div>
              <GhostButton type="button" onClick={() => remove(item.id)} disabled={busy}>
                {passkeyDeleteButton}
              </GhostButton>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-[14px] text-[var(--color-ink-soft)]">Ключей доступа пока нет.</p>
      )}
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <PrimaryButton type="button" onClick={add} disabled={busy}>
        {passkeyAddButton}
      </PrimaryButton>
    </div>
  );
}

function formatPasskeyDate(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString("ru-RU");
}
