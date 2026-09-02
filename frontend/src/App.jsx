import { useEffect, useState } from "react";
import { getCompanyLogin, getMe, logout } from "./api/auth.js";
import { onAuthRequired } from "./api/client.js";
import { getJob, getJobReport, listJobFiles } from "./api/jobs.js";
import { canEditCompanyProfile, companyLoginURL, companySlugFromHost, companySlugFromPath, needsSecurityNudge, resolveUsersCompanyId } from "./features/auth/accessPresentation.js";
import { consumeQuickStart } from "./features/help/firstRun.js";
import { securitySetupLabel, twoFactorRequiredHint } from "./features/help/copy.js";
import { initialEditState } from "./features/order/reviewEdits.js";
import { CompaniesScreen, CompanyScreen, JobHistory, UsersScreen } from "./ui/admin/AdminScreens.jsx";
import { AccountScreen, InviteScreen, LoginScreen } from "./ui/auth/AuthScreens.jsx";
import { HelpButton } from "./ui/chrome.jsx";
import { HelpDrawer } from "./ui/help/HelpDrawer.jsx";
import { QuickStart } from "./ui/help/QuickStart.jsx";
import { NorthApp } from "./ui/north/NorthApp.jsx";
import { OrderFillApp } from "./ui/order/OrderFillApp.jsx";

export default function App() {
  const inviteToken = inviteTokenFromPath();
  const hostSlug = companySlugFromHost(window.location.hostname);
  const pathSlug = companySlugFromPath(window.location.pathname);
  const companySlug = hostSlug || pathSlug;
  const [me, setMe] = useState(undefined);
  const [companyLogin, setCompanyLogin] = useState(companySlug ? undefined : null);
  const [screen, setScreen] = useState("history");
  const [companyId, setCompanyId] = useState("");
  const [resume, setResume] = useState(null);
  const [quickStartOpen, setQuickStartOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);

  useEffect(() => {
    getMe()
      .then((user) => {
        setMe(user);
        if (user.company_id) setCompanyId(user.company_id);
      })
      .catch(() => setMe(null));
  }, []);

  useEffect(() => onAuthRequired(() => setMe(null)), []);

  useEffect(() => {
    if (pathSlug && !hostSlug) {
      const target = companyLoginURL(pathSlug);
      if (target && new URL(target).host !== window.location.host) {
        window.location.replace(target);
      }
      return;
    }
    if (hostSlug && pathSlug) {
      window.history.replaceState(null, "", "/");
    }
  }, [hostSlug, pathSlug]);

  useEffect(() => {
    if (!companySlug) {
      setCompanyLogin(null);
      return;
    }
    getCompanyLogin(companySlug)
      .then(setCompanyLogin)
      .catch(() => setCompanyLogin(null));
  }, [companySlug]);

  const userId = me?.id;
  useEffect(() => {
    if (!userId) {
      setQuickStartOpen(false);
      setHelpOpen(false);
      return;
    }
    setQuickStartOpen(consumeQuickStart({ id: userId }));
  }, [userId]);

  if (me === undefined) {
    return <div className="grid min-h-full place-items-center text-[var(--color-ink-faint)]">Загрузка…</div>;
  }

  if (inviteToken && !me) {
    return (
      <InviteScreen
        token={inviteToken}
        onDone={(user) => {
          setMe(user);
          if (user.company_id) setCompanyId(user.company_id);
        }}
      />
    );
  }

  if (!me) {
    if (companySlug && companyLogin === undefined) {
      return <div className="grid min-h-full place-items-center text-[var(--color-ink-faint)]">Загрузка…</div>;
    }
    return (
      <LoginScreen
        company={companyLogin}
        onDone={(user) => {
          setMe(user);
          if (user.company_id) setCompanyId(user.company_id);
        }}
      />
    );
  }

  function goHome() {
    setResume(null);
    setScreen("history");
  }

  return (
    <>
      {screen === "order" ? (
        <OrderFillApp companyId={companyId} resumeJob={resume} onHome={goHome} onHelp={() => setHelpOpen(true)} />
      ) : screen === "north" ? (
        <NorthApp companyId={companyId} onHome={goHome} onHelp={() => setHelpOpen(true)} />
      ) : (
        <div className="flex h-full flex-col bg-[var(--color-ground)]">
          <header className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 sm:px-6">
            <nav className="flex gap-2 text-[14px] font-medium">
              <NavButton active={screen === "history"} onClick={() => setScreen("history")}>
                Выгрузки
              </NavButton>
              {me.role === "platform_admin" ? (
                <NavButton dataTour="companies" active={screen === "companies"} onClick={() => setScreen("companies")}>
                  Компании
                </NavButton>
              ) : null}
              {canEditCompanyProfile(me.role) ? (
                <NavButton dataTour="company" active={screen === "company"} onClick={() => setScreen("company")}>
                  Компания
                </NavButton>
              ) : null}
              {me.role !== "purchaser" ? (
                <NavButton dataTour="users" active={screen === "users"} onClick={() => setScreen("users")}>
                  Пользователи
                </NavButton>
              ) : null}
            </nav>
            <div className="flex items-center gap-3 text-[14px] text-[var(--color-ink-soft)]">
              <HelpButton onClick={() => setHelpOpen(true)} />
              <button
                type="button"
                className={screen === "account" ? "font-medium text-[var(--color-brand-strong)]" : "hover:text-[var(--color-ink)]"}
                onClick={() => setScreen("account")}
              >
                {me.login}
              </button>
              <button
                type="button"
                className="text-[var(--color-brand)]"
                onClick={async () => {
                  await logout();
                  setMe(null);
                }}
              >
                Выйти
              </button>
            </div>
          </header>
          {needsSecurityNudge(me) ? (
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-line)] bg-[var(--color-brand-soft)] px-4 py-3 sm:px-6">
              <p className="text-[14px] leading-relaxed text-[var(--color-ink)]">{twoFactorRequiredHint}</p>
              {screen !== "account" ? (
                <button
                  type="button"
                  className="text-[14px] font-medium text-[var(--color-brand)]"
                  onClick={() => setScreen("account")}
                >
                  {securitySetupLabel}
                </button>
              ) : null}
            </div>
          ) : null}
          <main className="flex-1 overflow-auto">
            {screen === "history" ? (
              <JobHistory
                me={me}
                companyId={companyId}
                onCompany={setCompanyId}
                onNew={(kind) => {
                  if (me.role === "platform_admin") return;
                  setResume(null);
                  setScreen(kind === "north" ? "north" : "order");
                }}
                onOpen={async (job) => {
                  if (job.type === "north_merge") {
                    setScreen("north");
                    return;
                  }
                  const loaded = await loadOrderResume(job.id);
                  setResume(loaded);
                  setScreen("order");
                }}
              />
            ) : null}
            {screen === "companies" ? <CompaniesScreen selectedId={companyId} onSelect={setCompanyId} /> : null}
            {screen === "company" ? (
              <CompanyScreen
                me={me}
                onSaved={(company) =>
                  setMe((current) => ({
                    ...current,
                    company_name: company.name,
                    login_slug: company.login_slug,
                    has_logo: Boolean(company.has_logo),
                  }))
                }
              />
            ) : null}
            {screen === "users" ? (
              <UsersScreen
                actorRole={me.role}
                companyId={resolveUsersCompanyId(me.role, companyId, me.company_id)}
                onCompany={setCompanyId}
              />
            ) : null}
            {screen === "account" ? (
              <AccountScreen
                me={me}
                onBack={() => setScreen("history")}
                onSignedOut={() => setMe(null)}
                onMe={setMe}
              />
            ) : null}
          </main>
          {quickStartOpen ? (
            <QuickStart
              me={me}
              onLater={() => setQuickStartOpen(false)}
              onDismiss={() => setQuickStartOpen(false)}
            />
          ) : null}
        </div>
      )}
      {helpOpen ? (
        <HelpDrawer
          onClose={() => setHelpOpen(false)}
          onReplay={() => {
            setHelpOpen(false);
            setScreen("history");
            setQuickStartOpen(true);
          }}
        />
      ) : null}
    </>
  );
}

function NavButton({ active, onClick, children, dataTour }) {
  return (
    <button
      type="button"
      data-tour={dataTour}
      onClick={onClick}
      className={`rounded-lg px-3 py-1.5 transition-colors duration-200 ${active ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"}`}
    >
      {children}
    </button>
  );
}

function inviteTokenFromPath() {
  const match = window.location.pathname.match(/^\/invite\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

async function loadOrderResume(jobId) {
  const job = await getJob(jobId);
  const report = await getJobReport(jobId).catch(() => null);
  const files = job.status === "completed" ? await listJobFiles(jobId).catch(() => ({ files: [] })) : { files: [] };
  const rows = report?.rows || [];
  return {
    jobId: job.id,
    brand: job.brand,
    month: job.order_month,
    status: job.status,
    rows,
    results: report ? [{ summary: report.summary, reportRows: rows }] : [],
    edits: initialEditState(rows),
    outputFiles: files.files || [],
    finalized: job.status === "completed",
  };
}
