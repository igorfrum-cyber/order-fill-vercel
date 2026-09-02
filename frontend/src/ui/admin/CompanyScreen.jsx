import { useEffect, useState } from "react";
import { apiClient } from "../../api/client.js";
import { clearCompanyLogo, setCompanyLogo, updateCompany } from "../../api/auth.js";
import { companyLoginLogoURL, companyLoginURL, loginSlugIssue, normalizeLoginSlug } from "../../features/auth/accessPresentation.js";
import { Field, GhostButton, PrimaryButton } from "../widgets.jsx";

export function CompanyScreen({ me, onSaved }) {
  const [name, setName] = useState(me.company_name || "");
  const [loginSlug, setLoginSlug] = useState(me.login_slug || "");
  const [logoFile, setLogoFile] = useState(null);
  const [logoPreview, setLogoPreview] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    setName(me.company_name || "");
    setLoginSlug(me.login_slug || "");
  }, [me.company_name, me.login_slug]);

  useEffect(() => {
    if (!logoFile) {
      setLogoPreview("");
      return undefined;
    }
    const url = URL.createObjectURL(logoFile);
    setLogoPreview(url);
    return () => URL.revokeObjectURL(url);
  }, [logoFile]);

  const slugIssue = loginSlugIssue(loginSlug);
  const nameReady = Boolean(name.trim());
  const profileChanged =
    name.trim() !== (me.company_name || "").trim() || normalizeLoginSlug(loginSlug) !== (me.login_slug || "");
  const ready = nameReady && !slugIssue && (profileChanged || Boolean(logoFile));
  const savedLogoSrc =
    me.has_logo && me.login_slug ? `${apiClient.absoluteUrl(companyLoginLogoURL(me.login_slug))}?v=${me.login_slug}` : "";

  return (
    <section className="animate-enter mx-auto max-w-lg space-y-5 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Компания</h1>
        <p className="mt-1 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
          Эти данные видят сотрудники на экране входа.
        </p>
      </div>
      <form
        className="space-y-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6"
        onSubmit={async (event) => {
          event.preventDefault();
          if (!ready || busy) return;
          setBusy(true);
          setError("");
          setDone(false);
          try {
            let company = {
              name: name.trim(),
              login_slug: normalizeLoginSlug(loginSlug),
              has_logo: Boolean(me.has_logo),
            };
            if (profileChanged) {
              company = await updateCompany(me.company_id, company.name, company.login_slug);
            }
            if (logoFile) {
              company = { ...company, ...(await setCompanyLogo(me.company_id, logoFile)) };
              setLogoFile(null);
            }
            onSaved?.(company);
            setDone(true);
          } catch (err) {
            setError(err.message || "Не удалось сохранить данные компании.");
          } finally {
            setBusy(false);
          }
        }}
      >
        <Field label="Логотип">
          <div className="flex items-center gap-3">
            {logoPreview || savedLogoSrc ? (
              <img src={logoPreview || savedLogoSrc} alt="" className="h-16 w-16 rounded-xl object-contain bg-[var(--color-ground)]" />
            ) : (
              <div className="grid h-16 w-16 place-items-center rounded-xl bg-[var(--color-ground)] text-[12px] text-[var(--color-ink-faint)]">
                нет
              </div>
            )}
            <div className="min-w-0 space-y-2">
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="block w-full text-[13px] text-[var(--color-ink-soft)] file:mr-3 file:rounded-lg file:border-0 file:bg-[var(--color-brand-soft)] file:px-3 file:py-1.5 file:text-[13px] file:font-medium file:text-[var(--color-brand-strong)]"
                onChange={(event) => setLogoFile(event.target.files?.[0] || null)}
              />
              {me.has_logo && !logoFile ? (
                <GhostButton
                  type="button"
                  onClick={async () => {
                    setBusy(true);
                    setError("");
                    try {
                      const company = await clearCompanyLogo(me.company_id);
                      onSaved?.(company);
                    } catch (err) {
                      setError(err.message || "Не удалось убрать логотип.");
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  Убрать
                </GhostButton>
              ) : null}
            </div>
          </div>
        </Field>
        <p className="text-[13px] text-[var(--color-ink-faint)]">PNG, JPEG или WebP, до 512 КБ.</p>
        <Field label="Название">
          <input className="input" value={name} onChange={(event) => setName(event.target.value)} autoComplete="organization" />
        </Field>
        <Field label="Адрес входа">
          <input
            className="input font-mono"
            value={loginSlug}
            onChange={(event) => setLoginSlug(event.target.value)}
            autoComplete="off"
            spellCheck={false}
          />
        </Field>
        {slugIssue && normalizeLoginSlug(loginSlug) !== (me.login_slug || "") ? (
          <p className="text-[13px] text-[var(--color-danger)]">{slugIssue}</p>
        ) : loginSlug ? (
          <p className="text-[13px] text-[var(--color-ink-soft)]">
            Ссылка входа:{" "}
            <a className="font-mono text-[var(--color-brand)]" href={companyLoginURL(loginSlug)}>
              {companyLoginURL(loginSlug)}
            </a>
          </p>
        ) : (
          <p className="text-[13px] text-[var(--color-ink-faint)]">Латиницей. Откроется как kristail.localhost.</p>
        )}
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        {done ? <p className="text-[14px] text-[var(--color-ok)]">Данные компании сохранены.</p> : null}
        <PrimaryButton type="submit" disabled={busy || !ready}>
          Сохранить
        </PrimaryButton>
      </form>
    </section>
  );
}
