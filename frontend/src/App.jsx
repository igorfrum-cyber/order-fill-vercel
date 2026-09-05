import { useEffect, useState } from "react";
import { getCompanyLogin, getMe, logout } from "./api/auth.js";
import { onAuthRequired } from "./api/client.js";
import { getJob, getJobReport, listJobFiles } from "./api/jobs.js";
import { canEditCompanyProfile, companyLoginURL, companySlugFromHost, companySlugFromPath, homeScreen, needsSecurityNudge, resolveUsersCompanyId } from "./features/auth/accessPresentation.js";
import { shouldAutoStartTour, tourSceneForView } from "./features/help/firstRun.js";
import { headerContext, roleLabel, securitySetupLabel, twoFactorRequiredHint } from "./features/help/copy.js";
import { initialEditState } from "./features/order/reviewEdits.js";
import { CompaniesScreen, CompanyScreen, JobHistory, OverviewScreen, UsersScreen } from "./ui/admin/AdminScreens.jsx";
import { AccountScreen, InviteScreen, LoginScreen } from "./ui/auth/AuthScreens.jsx";
import { HelpButton, ProfileMenu } from "./ui/chrome.jsx";
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
  const [tourFollowUp, setTourFollowUp] = useState(false);
  const [seenTourScenes, setSeenTourScenes] = useState(() => new Set());
  const [orderStage, setOrderStage] = useState("upload");

  useEffect(() => {
    getMe()
      .then((user) => {
        setMe(user);
        if (user.company_id) setCompanyId(user.company_id);
        setScreen(homeScreen(user.role));
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
  const tourScene = tourSceneForView({
    screen,
    stage: orderStage,
    seenHome: seenTourScenes.has("home") || !tourFollowUp,
  });

  useEffect(() => {
    if (!userId) {
      setQuickStartOpen(false);
      setHelpOpen(false);
      setTourFollowUp(false);
      setSeenTourScenes(new Set());
      return;
    }
  }, [userId]);

  useEffect(() => {
    if (!tourFollowUp || !userId) return;
    if (seenTourScenes.has(tourScene)) return;
    setQuickStartOpen(true);
  }, [tourFollowUp, tourScene, userId, seenTourScenes]);

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
          setScreen(homeScreen(user.role));
          if (shouldAutoStartTour("invite")) {
            setTourFollowUp(true);
            setQuickStartOpen(true);
          }
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
          setScreen(homeScreen(user.role));
        }}
      />
    );
  }

  function goHome() {
    setResume(null);
    setOrderStage("upload");
    setScreen(homeScreen(me.role));
  }

  function openHistory() {
    setResume(null);
    setOrderStage("upload");
    setScreen("history");
  }

  function closeTour() {
    setSeenTourScenes((prev) => new Set(prev).add(tourScene));
    setQuickStartOpen(false);
  }

  const shell = headerContext(me);

  return (
    <>
      {screen === "order" ? (
        <OrderFillApp
          companyId={companyId}
          resumeJob={resume}
          onHome={openHistory}
          onHelp={() => setHelpOpen(true)}
          onStage={setOrderStage}
        />
      ) : screen === "north" ? (
        <NorthApp companyId={companyId} onHome={goHome} onHelp={() => setHelpOpen(true)} />
      ) : (
        <div className="flex h-full flex-col bg-[var(--color-ground)]">
          <header className="app-header flex flex-wrap items-center gap-2 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 sm:px-6">
            <nav className="flex min-w-0 flex-wrap gap-1 text-[14px] font-medium sm:gap-2">
              {me.role === "platform_admin" ? (
                <NavButton dataTour="overview" active={screen === "overview"} onClick={() => setScreen("overview")}>
                  Обзор
                </NavButton>
              ) : null}
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
            <div className="ml-auto flex min-w-0 items-center justify-end gap-2 sm:gap-3">
              <div className="app-header-context min-w-0 max-w-[7.5rem] leading-tight sm:max-w-[16rem]">
                <div className="truncate text-[13px] font-medium text-[var(--color-ink)]">{shell.companyLine}</div>
                <div className="truncate text-[12px] text-[var(--color-ink-faint)]">{shell.roleLine}</div>
              </div>
              <HelpButton onClick={() => setHelpOpen(true)} />
              <ProfileMenu
                login={me.login}
                roleLabel={roleLabel(me.role)}
                active={screen === "account"}
                onProfile={() => setScreen("account")}
                onLogout={async () => {
                  await logout();
                  setMe(null);
                }}
              />
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
            {screen === "overview" ? (
              <OverviewScreen
                onOpen={async (job) => {
                  if (job.type === "north_merge") {
                    setScreen("north");
                    return;
                  }
                  const loaded = await loadOrderResume(job.id);
                  setResume(loaded);
                  setOrderStage(loaded.finalized ? "preview" : "fill");
                  setScreen("order");
                }}
              />
            ) : null}
            {screen === "history" ? (
              <JobHistory
                me={me}
                companyId={companyId}
                onCompany={setCompanyId}
                onNew={(kind) => {
                  if (me.role === "platform_admin") return;
                  setResume(null);
                  setOrderStage("upload");
                  setScreen(kind === "north" ? "north" : "order");
                }}
                onOpen={async (job) => {
                  if (job.type === "north_merge") {
                    setScreen("north");
                    return;
                  }
                  const loaded = await loadOrderResume(job.id);
                  setResume(loaded);
                  setOrderStage(loaded.finalized ? "preview" : "fill");
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
                onBack={() => setScreen(homeScreen(me.role))}
                onSignedOut={() => setMe(null)}
                onMe={setMe}
              />
            ) : null}
          </main>
        </div>
      )}
      {quickStartOpen ? (
        <QuickStart me={me} scene={tourScene} onLater={closeTour} onDismiss={closeTour} />
      ) : null}
      {helpOpen ? (
        <HelpDrawer
          onClose={() => setHelpOpen(false)}
          onReplay={() => {
            setHelpOpen(false);
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
      className={`rounded-lg px-3 py-1.5 transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-brand)] ${active ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"}`}
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
