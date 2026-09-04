import { useEffect, useState } from "react";
import { createCompany, disableCompany, listCompanies } from "../../api/auth.js";
import { companyLoginURL, loginSlugIssue, normalizeLoginSlug } from "../../features/auth/accessPresentation.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";
import { userFacingError } from "../../features/help/errors.js";

export function CompaniesScreen({ selectedId, onSelect }) {
  const [companies, setCompanies] = useState([]);
  const [name, setName] = useState("");
  const [loginSlug, setLoginSlug] = useState("");
  const [error, setError] = useState("");

  function reload() {
    listCompanies()
      .then((payload) => setCompanies(payload.companies || []))
      .catch(() => setCompanies([]));
  }

  useEffect(reload, []);

  const createIssue = loginSlugIssue(loginSlug);

  return (
    <section className="animate-enter mx-auto max-w-3xl space-y-4 p-6">
      <h1 className="text-[22px] font-semibold">Компании</h1>
      <form
        data-tour="companies-create"
        className="space-y-3"
        onSubmit={async (event) => {
          event.preventDefault();
          setError("");
          try {
            await createCompany(name, normalizeLoginSlug(loginSlug));
            setName("");
            setLoginSlug("");
            reload();
          } catch (err) {
            setError(userFacingError(err, "Не удалось создать компанию."));
          }
        }}
      >
        <div className="flex flex-col gap-2 sm:flex-row">
          <input className="input flex-1" value={name} onChange={(event) => setName(event.target.value)} placeholder="Название" />
          <input
            className="input flex-1 font-mono"
            value={loginSlug}
            onChange={(event) => setLoginSlug(event.target.value)}
            placeholder="Адрес входа, латиницей"
            autoComplete="off"
            spellCheck={false}
          />
          <PrimaryButton type="submit" disabled={!name.trim() || Boolean(createIssue)}>
            Создать
          </PrimaryButton>
        </div>
        {loginSlug ? (
          <p className="text-[13px] text-[var(--color-ink-soft)]">
            {createIssue || (
              <>
                Ссылка входа: <span className="font-mono">{companyLoginURL(loginSlug)}</span>
              </>
            )}
          </p>
        ) : (
          <p className="text-[13px] text-[var(--color-ink-faint)]">Первый адрес задаёте вы. Потом его меняет администратор компании.</p>
        )}
      </form>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <ul data-tour="companies-list" className="divide-y divide-[var(--color-line)] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        {companies.map((company) => (
          <li key={company.id} className="space-y-2 px-4 py-3">
            <div className="flex items-center justify-between gap-3">
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
                    setError(userFacingError(err, "Не удалось выключить компанию."));
                  }
                }}
              >
                Выключить
              </GhostButton>
            </div>
            {company.login_slug ? (
              <a className="block font-mono text-[13px] text-[var(--color-brand)]" href={companyLoginURL(company.login_slug)}>
                {companyLoginURL(company.login_slug)}
              </a>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  );
}
