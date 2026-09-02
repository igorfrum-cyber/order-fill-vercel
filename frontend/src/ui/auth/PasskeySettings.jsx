import { useEffect, useState } from "react";
import { deletePasskey, listPasskeys } from "../../api/auth.js";
import {
  passkeyAddButton,
  passkeyDeleteButton,
  passkeyInsecureOriginHint,
  passkeySettingsHint,
  passkeySettingsTitle,
} from "../../features/help/copy.js";
import { defaultPasskeyName, passkeyErrorMessage, passkeyOriginIssue, passkeyUsable } from "../../features/auth/passkey.js";
import { passkeyWhen } from "../../features/auth/session.js";
import { createPasskey } from "../../features/auth/passkeyFlow.js";
import { IconComputer } from "../icons.jsx";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

export function PasskeySettings({ onChanged }) {
  const [items, setItems] = useState([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const usable = passkeyUsable();
  const originIssue = passkeyOriginIssue();

  async function refresh() {
    const result = await listPasskeys();
    const next = result.passkeys || [];
    setItems(next);
    onChanged?.(next.length > 0);
  }

  useEffect(() => {
    refresh().catch(() => setError("Не удалось загрузить устройства для входа."));
  }, []);

  async function add() {
    setBusy(true);
    setError("");
    try {
      await createPasskey(defaultPasskeyName(items.length));
      await refresh();
    } catch (err) {
      setError(passkeyErrorMessage(err, "add"));
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
      setError("Не удалось удалить устройство.");
    } finally {
      setBusy(false);
    }
  }

  const hint = originIssue
    ? passkeyInsecureOriginHint
    : usable
      ? passkeySettingsHint
      : "Этот браузер не умеет Face ID, Touch ID или Windows Hello. Откройте Chrome, Safari или Edge — либо включите вход по коду ниже.";

  return (
    <div className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
      <h2 className="text-[16px] font-semibold">{passkeySettingsTitle}</h2>
      <p className="text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{hint}</p>
      {items.length ? (
        <ul className="space-y-2">
          {items.map((item) => (
            <li key={item.id} className="flex items-center gap-3 rounded-xl bg-[var(--color-ground)] px-3 py-3">
              <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-[var(--color-surface)] text-[var(--color-brand)]">
                <IconComputer className="h-5 w-5" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-[15px] font-medium">{item.name}</p>
                <p className="text-[13px] text-[var(--color-ink-faint)]">{passkeyWhen(item)}</p>
              </div>
              <GhostButton type="button" onClick={() => remove(item.id)} disabled={busy}>
                {passkeyDeleteButton}
              </GhostButton>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-[14px] text-[var(--color-ink-soft)]">Устройств для быстрого входа пока нет.</p>
      )}
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      {usable ? (
        <PrimaryButton type="button" onClick={add} disabled={busy}>
          {passkeyAddButton}
        </PrimaryButton>
      ) : null}
    </div>
  );
}
