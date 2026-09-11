import { useEffect, useState } from "react";
import { getCompanyLogin, getMe, listCompanies, listJobs, logout } from "./api/auth.js";
import { onAuthRequired } from "./api/client.js";
import { getJob, getJobReport, listJobFiles } from "./api/jobs.js";
import { companyIdFromSearch, parseAppPath, pathForScreen, resolveOrderNavJobId, screenAllowed, withCompanyQuery } from "./features/app/routes.js";
import { companyLoginURL, companySlugFromHost, companySlugFromPath, homeScreen, navItemsForRole, needsSecurityNudge, resolveUsersCompanyId } from "./features/auth/accessPresentation.js";
import { shouldAutoStartTour, tourSceneForView } from "./features/help/firstRun.js";
import { headerContext, roleLabel, securitySetupLabel, twoFactorRequiredHint } from "./features/help/copy.js";
import { initialEditState } from "./features/order/reviewEdits.js";
import { CompaniesScreen, CompanyScreen, JobHistory, OverviewScreen, QueueScreen, UsersScreen } from "./ui/admin/AdminScreens.jsx";
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
  const [companies, setCompanies] = useState([]);
  const [resume, setResume] = useState(null);
  const [quickStartOpen, setQuickStartOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [tourFollowUp, setTourFollowUp] = useState(false);
  const [seenTourScenes, setSeenTourScenes] = useState(() => new Set());
  const [orderStage, setOrderStage] = useState("upload");
  const [completedJob, setCompletedJob] = useState(false);
  const [liveJobId, setLiveJobId] = useState("");

  useEffect(() => {
    getMe()
      .then((user) => {
        setMe(user);
        applyLocation(user, setScreen, setCompanyId, setResume, setOrderStage);
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

  useEffect(() => {
    if (me?.role !== "purchaser") {
      setCompletedJob(false);
      return undefined;
    }
    let cancelled = false;
    listJobs()
      .then((payload) => {
        if (!cancelled && (payload.jobs || []).some((job) => job.status === "completed")) {
          setCompletedJob(true);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [me?.id, me?.role]);

  useEffect(() => {
    if (me?.role !== "platform_admin") return undefined;
    listCompanies()
      .then((payload) => setCompanies(payload.companies || []))
      .catch(() => setCompanies([]));
    return undefined;
  }, [me?.role]);

  useEffect(() => {
    if (!me) return undefined;
    function onPop() {
      applyLocation(me, setScreen, setCompanyId, setResume, setOrderStage);
    }
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, [me]);

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
    return (
      <div className="grid min-h-full place-items-center">
        <div className="h-8 w-48 animate-pulse rounded-lg bg-[var(--color-line-soft)]" />
      </div>
    );
  }

  if (inviteToken && !me) {
    return (
      <InviteScreen
        token={inviteToken}
        onDone={(user) => {
          setMe(user);
          enter(user, homeScreen(user.role), setScreen, setCompanyId, setResume, setOrderStage);
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
      return (
        <div className="grid min-h-full place-items-center">
          <div className="h-8 w-48 animate-pulse rounded-lg bg-[var(--color-line-soft)]" />
        </div>
      );
    }
    return (
      <LoginScreen
        company={companyLogin}
        onDone={(user) => {
          setMe(user);
          enter(user, homeScreen(user.role), setScreen, setCompanyId, setResume, setOrderStage);
        }}
      />
    );
  }

  const platformCompany = me.role === "platform_admin" ? companyId : "";
  const openJobId = resume?.jobId || liveJobId;

  function markJobCompletedIfNeeded() {
    if (screen === "order" && orderStage === "preview") setCompletedJob(true);
  }

  function go(next, jobId = "") {
    const nextJobId = resolveOrderNavJobId(next, jobId, { screen, openJobId });
    if (next !== "north") {
      window.history.pushState(null, "", withCompanyQuery(pathForScreen(next, nextJobId), platformCompany));
    }
    if (next === "order" && !nextJobId) {
      setResume(null);
      setLiveJobId("");
      setOrderStage("upload");
    } else if (next !== "order") {
      markJobCompletedIfNeeded();
      setResume(null);
      setLiveJobId("");
    } else {
      setLiveJobId(nextJobId);
    }
    setScreen(next);
  }

  function goHome() {
    setResume(null);
    setOrderStage("upload");
    go(homeScreen(me.role));
  }

  function leaveJob() {
    markJobCompletedIfNeeded();
    setResume(null);
    setLiveJobId("");
    setOrderStage("upload");
    go(homeScreen(me.role) === "order" ? "history" : homeScreen(me.role));
  }

  function closeTour() {
    setSeenTourScenes((prev) => new Set(prev).add(tourScene));
    setQuickStartOpen(false);
  }

  async function openJob(job) {
    if (job.type === "north_merge") {
      setScreen("north");
      return;
    }
    const loaded = await loadOrderResume(job.id);
    setResume(loaded);
    setOrderStage(loaded.finalized ? "preview" : "fill");
    go("order", loaded.jobId);
  }

  const shell = headerContext(me);
  const nav = navItemsForRole(me.role);

  return (
    <>
      {screen === "north" ? (
        <NorthApp companyId={companyId} onHome={goHome} onHelp={() => setHelpOpen(true)} />
      ) : (
        <div className="flex h-full flex-col bg-[var(--color-ground)]">
          <header className="app-header flex flex-wrap items-center gap-2 border-b border-[var(--color-line)] bg-[var(--color-surface)] px-4 py-3 sm:px-6">
            <nav className="flex min-w-0 flex-wrap gap-1 text-[14px] font-medium sm:gap-2">
              {nav.map((item) => (
                <NavButton
                  key={item.id}
                  href={withCompanyQuery(item.path, platformCompany)}
                  dataTour={item.id}
                  active={screen === item.id}
                  onClick={() => go(item.id)}
                >
                  {item.label}
                </NavButton>
              ))}
            </nav>
            <div className="ml-auto flex min-w-0 items-center justify-end gap-2 sm:gap-3">
              {me.role === "platform_admin" ? (
                <select
                  data-tour="company-select"
                  className="input max-w-[10rem] py-1.5 sm:max-w-xs"
                  value={companyId}
                  aria-label="Компания"
                  onChange={(event) => {
                    const next = event.target.value;
                    setCompanyId(next);
                    window.history.replaceState(null, "", withCompanyQuery(pathForScreen(screen), next));
                  }}
                >
                  <option value="">Все компании</option>
                  {companies
                    .filter((company) => !company.disabled_at)
                    .map((company) => (
                      <option key={company.id} value={company.id}>
                        {company.name}
                      </option>
                    ))}
                </select>
              ) : (
                <div className="app-header-context min-w-0 max-w-[7.5rem] leading-tight sm:max-w-[16rem]">
                  <div className="truncate text-[13px] font-medium text-[var(--color-ink)]">{shell.companyLine}</div>
                  {shell.roleLine ? <div className="truncate text-[12px] text-[var(--color-ink-faint)]">{shell.roleLine}</div> : null}
                </div>
              )}
              <HelpButton onClick={() => setHelpOpen(true)} />
              <ProfileMenu
                login={me.login}
                roleLabel={roleLabel(me.role)}
                active={screen === "account"}
                onProfile={() => go("account")}
                onLogout={async () => {
                  await logout();
                  setMe(null);
                }}
              />
            </div>
          </header>
          {needsSecurityNudge(me, { completedJob }) ? (
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-line)] bg-[var(--color-brand-soft)] px-4 py-3 sm:px-6">
              <p className="text-[14px] leading-relaxed text-[var(--color-ink)]">{twoFactorRequiredHint}</p>
              {screen !== "account" ? (
                <button type="button" className="text-[14px] font-medium text-[var(--color-brand)]" onClick={() => go("account")}>
                  {securitySetupLabel}
                </button>
              ) : null}
            </div>
          ) : null}
          <main className={`flex-1 ${screen === "order" ? "min-h-0 overflow-hidden" : "overflow-auto"}`}>
            {screen === "order" ? (
              <OrderFillApp
                key={resume?.jobId || "new"}
                companyId={companyId}
                canSelectCompany={me.role === "platform_admin"}
                resumeJob={resume}
                onHome={leaveJob}
                onHelp={() => setHelpOpen(true)}
                onStage={setOrderStage}
                onJobReady={(id) => go("order", id)}
                embedded
              />
            ) : null}
            {screen === "overview" ? <OverviewScreen onOpen={openJob} /> : null}
            {screen === "queue" ? (
              <QueueScreen me={me} onOpen={openJob} onPeople={() => go("users")} onCompany={() => go("company")} />
            ) : null}
            {screen === "history" ? (
              <JobHistory
                me={me}
                companyId={companyId}
                onNew={(kind) => {
                  if (me.role === "platform_admin") return;
                  setResume(null);
                  setLiveJobId("");
                  setOrderStage("upload");
                  if (kind === "north") {
                    setScreen("north");
                    return;
                  }
                  go("order");
                }}
                onOpen={openJob}
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
              <UsersScreen actorId={me.id} actorRole={me.role} companyId={resolveUsersCompanyId(me.role, companyId, me.company_id)} onCompany={setCompanyId} />
            ) : null}
            {screen === "account" ? (
              <AccountScreen me={me} onBack={goHome} onSignedOut={() => setMe(null)} onMe={setMe} />
            ) : null}
          </main>
        </div>
      )}
      {quickStartOpen ? <QuickStart me={me} scene={tourScene} onLater={closeTour} onDismiss={closeTour} /> : null}
      {helpOpen ? (
        <HelpDrawer
          role={me.role}
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

function NavButton({ active, onClick, children, dataTour, href }) {
  return (
    <a
      href={href}
      data-tour={dataTour}
      onClick={(event) => {
        event.preventDefault();
        onClick();
      }}
      className={`rounded-lg px-3 py-1.5 transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-brand)] ${active ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"}`}
    >
      {children}
    </a>
  );
}

function inviteTokenFromPath() {
  const match = window.location.pathname.match(/^\/invite\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

function enter(user, screen, setScreen, setCompanyId, setResume, setOrderStage) {
  const company = user.role === "platform_admin" ? "" : user.company_id || "";
  setCompanyId(company);
  setResume(null);
  setOrderStage("upload");
  setScreen(screen);
  window.history.replaceState(null, "", withCompanyQuery(pathForScreen(screen), user.role === "platform_admin" ? "" : ""));
}

function applyLocation(user, setScreen, setCompanyId, setResume, setOrderStage) {
  const parsed = parseAppPath(window.location.pathname);
  const qCompany = companyIdFromSearch(window.location.search);
  if (user.role === "platform_admin") setCompanyId(qCompany);
  else if (user.company_id) setCompanyId(user.company_id);

  const wanted = parsed.screen || homeScreen(user.role);
  const ok = !parsed.unknown && screenAllowed(user.role, wanted, { jobId: parsed.jobId });
  const nextScreen = ok ? wanted : homeScreen(user.role);
  const nextJob = ok ? parsed.jobId : "";
  const cid = user.role === "platform_admin" ? qCompany : "";
  const nextPath = withCompanyQuery(pathForScreen(nextScreen, nextJob), cid);
  if (`${window.location.pathname}${window.location.search}` !== nextPath) {
    window.history.replaceState(null, "", nextPath);
  }
  setScreen(nextScreen);
  if (!nextJob) {
    setResume(null);
    return;
  }
  loadOrderResume(nextJob)
    .then((loaded) => {
      setResume(loaded);
      setOrderStage(loaded.finalized ? "preview" : "fill");
    })
    .catch(() => {
      setResume(null);
      const home = homeScreen(user.role);
      setScreen(home);
      window.history.replaceState(null, "", withCompanyQuery(pathForScreen(home), cid));
    });
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
