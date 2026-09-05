import { useEffect, useState } from "react";
import { createCompany, disableCompany, listCompanies, updateCompany } from "../../api/auth.js";
import { companyLoginURL, loginSlugIssue, matchingModeOptions, normalizeLoginSlug, normalizeMatchingMode } from "../../features/auth/accessPresentation.js";
import { GhostButton, PrimaryButton } from "../widgets.jsx";
import { IconCheck } from "../icons.jsx";
import { userFacingError } from "../../features/help/errors.js";

export function CompaniesScreen({ selectedId, onSelect }) {
  const [companies, setCompanies] = useState([]);
  const [name, setName] = useState("");
  const [loginSlug, setLoginSlug] = useState("");
  const [matchingMode, setMatchingMode] = useState("standard");
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
            await createCompany(name, normalizeLoginSlug(loginSlug), matchingMode);
            setName("");
            setLoginSlug("");
            setMatchingMode("standard");
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
        <MatchingModePicker name="create-matching-mode" value={matchingMode} onChange={setMatchingMode} />
      </form>
      {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
      <ul data-tour="companies-list" className="divide-y divide-[var(--color-line)] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        {companies.map((company) => (
          <li key={company.id} className="space-y-3 px-4 py-3">
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
            <MatchingModePicker
              name={`matching-mode-${company.id}`}
              value={normalizeMatchingMode(company.matching_mode)}
              onChange={async (mode) => {
                if (mode === normalizeMatchingMode(company.matching_mode)) return;
                setError("");
                try {
                  const updated = await updateCompany(company.id, company.name, company.login_slug, mode);
                  setCompanies((items) => items.map((item) => (item.id === company.id ? { ...item, ...updated } : item)));
                } catch (err) {
                  setError(userFacingError(err, "Не удалось сменить сопоставление."));
                }
              }}
            />
          </li>
        ))}
      </ul>
    </section>
  );
}

function MatchingModePicker({ name, value, onChange }) {
  const selected = normalizeMatchingMode(value);
  return (
    <fieldset className="space-y-2">
      <legend className="text-[13px] font-medium text-[var(--color-ink-faint)]">Сопоставление</legend>
      <div className="grid gap-2 sm:grid-cols-2">
        {matchingModeOptions().map((option) => {
          const on = selected === option.value;
          return (
            <label
              key={option.value}
              className={`flex cursor-pointer items-start gap-3 rounded-xl border p-3 ${
                on ? "border-[var(--color-brand)] bg-[var(--color-brand-soft)]" : "border-[var(--color-line)]"
              }`}
            >
              <input
                type="radio"
                className="sr-only"
                name={name}
                value={option.value}
                checked={on}
                onChange={() => onChange(option.value)}
              />
              <span
                className={`mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-md border ${
                  on ? "border-[var(--color-brand)] bg-[var(--color-brand)] text-white" : "border-[var(--color-line)] bg-[var(--color-surface)]"
                }`}
              >
                {on ? <IconCheck className="h-3.5 w-3.5" /> : null}
              </span>
              <span>
                <span className="block text-[14px] font-medium">{option.label}</span>
                <span className="block text-[12px] leading-snug text-[var(--color-ink-faint)]">{option.hint}</span>
              </span>
            </label>
          );
        })}
      </div>
    </fieldset>
  );
}
