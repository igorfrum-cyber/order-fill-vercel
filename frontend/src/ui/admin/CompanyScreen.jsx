import { useEffect, useState } from "react";
import { apiClient } from "../../api/client.js";
import { clearCompanyLogo, getCompanyOrderProfile, setCompanyLogo, updateCompany, updateCompanyOrderProfile } from "../../api/auth.js";
import { companyLoginLogoURL, companyLoginURL, loginSlugIssue, normalizeLoginSlug } from "../../features/auth/accessPresentation.js";
import {
  discountText,
  EMPTY_ORDER_PROFILE,
  normalizeOrderProfile,
  ORDER_PROFILE_BRANDS,
  orderProfileIssue,
  profileForSave,
  setTerms,
  termsFor,
} from "../../features/company/orderProfile.js";
import { Field, GhostButton, PrimaryButton } from "../widgets.jsx";
import { userFacingError } from "../../features/help/errors.js";

export function CompanyScreen({ me, onSaved }) {
  const [name, setName] = useState(me.company_name || "");
  const [loginSlug, setLoginSlug] = useState(me.login_slug || "");
  const [logoFile, setLogoFile] = useState(null);
  const [logoPreview, setLogoPreview] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  const [orderProfile, setOrderProfile] = useState(EMPTY_ORDER_PROFILE);
  const [discounts, setDiscounts] = useState({ angiopharm: "", skin_synergy: "", klapp: "" });
  const [orderLoading, setOrderLoading] = useState(true);
  const [orderBusy, setOrderBusy] = useState(false);
  const [orderError, setOrderError] = useState("");
  const [orderDone, setOrderDone] = useState(false);

  useEffect(() => {
    setName(me.company_name || "");
    setLoginSlug(me.login_slug || "");
  }, [me.company_name, me.login_slug]);

  useEffect(() => {
    let cancelled = false;
    setOrderLoading(true);
    getCompanyOrderProfile(me.company_id)
      .then((payload) => {
        if (cancelled) return;
        const profile = normalizeOrderProfile(payload);
        setOrderProfile(profile);
        setDiscounts(Object.fromEntries(ORDER_PROFILE_BRANDS.map((brand) => [brand.id, discountText(termsFor(profile, brand.id))])));
        setOrderError("");
      })
      .catch((err) => {
        if (!cancelled) setOrderError(userFacingError(err, "Не удалось загрузить реквизиты для бланков."));
      })
      .finally(() => {
        if (!cancelled) setOrderLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [me.company_id]);

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
    <section className="animate-enter mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-[22px] font-semibold">Компания</h1>
        <p className="mt-1 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">
          Эти данные видят сотрудники на экране входа.
        </p>
      </div>
      <form
        data-tour="company-profile"
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
            setError(userFacingError(err, "Не удалось сохранить данные компании."));
          } finally {
            setBusy(false);
          }
        }}
      >
        <div data-tour="company-logo">
        <Field label="Логотип" as="div">
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
                      setError(userFacingError(err, "Не удалось убрать логотип."));
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
        </div>
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
          <p className="text-[13px] text-[var(--color-ink-faint)]">Латиницей. Сотрудники входят по ссылке ниже.</p>
        )}
        {error ? <p className="text-[14px] text-[var(--color-danger)]">{error}</p> : null}
        {done ? <p className="text-[14px] text-[var(--color-ok)]">Данные компании сохранены.</p> : null}
        <PrimaryButton type="submit" disabled={busy || !ready}>
          Сохранить
        </PrimaryButton>
      </form>
      <form
        className="space-y-5 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6"
        aria-busy={orderLoading || orderBusy}
        onSubmit={async (event) => {
          event.preventDefault();
          const issue = orderProfileIssue(orderProfile, discounts);
          if (issue || orderBusy || orderLoading) {
            setOrderError(issue);
            return;
          }
          setOrderBusy(true);
          setOrderError("");
          setOrderDone(false);
          try {
            const saved = normalizeOrderProfile(await updateCompanyOrderProfile(me.company_id, profileForSave(orderProfile, discounts)));
            setOrderProfile(saved);
            setDiscounts(Object.fromEntries(ORDER_PROFILE_BRANDS.map((brand) => [brand.id, discountText(termsFor(saved, brand.id))])));
            setOrderDone(true);
          } catch (err) {
            setOrderError(userFacingError(err, "Не удалось сохранить реквизиты для бланков."));
          } finally {
            setOrderBusy(false);
          }
        }}
      >
        <div>
          <h2 className="text-[18px] font-semibold">Данные для бланков заказа</h2>
          <p className="mt-1 text-[13px] leading-relaxed text-[var(--color-ink-soft)]">
            Заполните один раз — при следующем расчёте подходящие поля появятся в бланке поставщика автоматически.
          </p>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <ProfileField label="Юридическое лицо" field="legal_name" profile={orderProfile} onChange={setOrderProfile} autoComplete="organization" />
          <ProfileField label="Грузополучатель" field="consignee" profile={orderProfile} onChange={setOrderProfile} />
          <ProfileField label="Адрес доставки" field="address" profile={orderProfile} onChange={setOrderProfile} autoComplete="street-address" />
          <ProfileField label="Контактное лицо" field="contact_name" profile={orderProfile} onChange={setOrderProfile} autoComplete="name" />
          <ProfileField label="Телефон получателя" field="contact_phone" profile={orderProfile} onChange={setOrderProfile} autoComplete="tel" inputMode="tel" />
          <ProfileField label="Транспортная компания" field="carrier" profile={orderProfile} onChange={setOrderProfile} />
          <ProfileField label="Кто оплачивает доставку" field="delivery_payer" profile={orderProfile} onChange={setOrderProfile} />
          <ProfileField label="До адреса или терминала" field="delivery_destination" profile={orderProfile} onChange={setOrderProfile} />
        </div>

        <div className="space-y-3 border-t border-[var(--color-line-soft)] pt-5">
          <div>
            <h3 className="text-[15px] font-semibold">Условия по брендам</h3>
            <p className="mt-1 text-[13px] text-[var(--color-ink-faint)]">Пустая скидка сохранит значение из исходного бланка. Ноль — установит скидку 0%.</p>
          </div>
          {ORDER_PROFILE_BRANDS.map((brand) => (
            <BrandTermsCard
              key={brand.id}
              brand={brand}
              terms={termsFor(orderProfile, brand.id)}
              discount={discounts[brand.id]}
              onDiscount={(value) => setDiscounts((current) => ({ ...current, [brand.id]: value }))}
              onTerms={(patch) => setOrderProfile((current) => setTerms(current, brand.id, patch))}
            />
          ))}
        </div>

        {orderLoading ? <p className="text-[13px] text-[var(--color-ink-faint)]">Загружаю реквизиты…</p> : null}
        {orderError ? <p role="alert" className="text-[14px] text-[var(--color-danger)]">{orderError}</p> : null}
        {orderDone ? <p className="text-[14px] text-[var(--color-ok)]">Реквизиты сохранены и будут применены к новым заказам.</p> : null}
        <PrimaryButton type="submit" disabled={orderLoading || orderBusy}>
          Сохранить данные для бланков
        </PrimaryButton>
      </form>
    </section>
  );
}

function ProfileField({ label, field, profile, onChange, ...inputProps }) {
  return (
    <Field label={label}>
      <input
        className="input"
        value={profile[field] || ""}
        maxLength={300}
        onChange={(event) => onChange((current) => ({ ...current, [field]: event.target.value }))}
        {...inputProps}
      />
    </Field>
  );
}

const TERM_LABELS = {
  dealer_name: "Название дилера",
  payment_method: "Способ оплаты",
  payment_control: "Контроль оплаты",
  customer_type: "Тип покупателя",
};

function BrandTermsCard({ brand, terms, discount, onDiscount, onTerms }) {
  return (
    <details className="rounded-xl border border-[var(--color-line)] bg-[var(--color-ground)] px-4 py-3" open={brand.id === "angiopharm"}>
      <summary className="cursor-pointer select-none text-[14px] font-semibold text-[var(--color-ink)]">{brand.label}</summary>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        {brand.fields.includes("discount") ? (
          <Field label={`Скидка ${brand.label}, %`}>
            <input
              className="input tabular-nums"
              value={discount}
              onChange={(event) => onDiscount(event.target.value)}
              inputMode="decimal"
              placeholder="например, 30"
              maxLength={6}
            />
          </Field>
        ) : null}
        {brand.fields.filter((field) => field !== "discount").map((field) => (
          <Field key={field} label={TERM_LABELS[field]}>
            <input className="input" value={terms[field] || ""} maxLength={300} onChange={(event) => onTerms({ [field]: event.target.value })} />
          </Field>
        ))}
      </div>
    </details>
  );
}
