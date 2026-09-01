import { useEffect, useState } from "react";
import { createCompany, disableCompany, listCompanies } from "../../api/auth.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

export function CompaniesScreen({ selectedId, onSelect }) {
  const [companies, setCompanies] = useState([]);
  const [name, setName] = useState("");
  const [error, setError] = useState("");

  function reload() {
    listCompanies().then((payload) => setCompanies(payload.companies || [])).catch(() => setCompanies([]));
  }

  useEffect(reload, []);

  return (
    <section className="animate-enter mx-auto max-w-3xl space-y-4 p-6">
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
              {company.disabled_at ? <span className="ml-2 text-[13px] font-normal text-[var(--color-ink-faint)]">выключена</span> : null}
            </button>
            <GhostButton
              onClick={async () => {
                try {
                  await disableCompany(company.id);
                  reload();
                } catch (err) {
                  setError(err.message || "Не удалось выключить компанию.");
                }
              }}
            >
              Выключить
            </GhostButton>
          </li>
        ))}
      </ul>
    </section>
  );
}
