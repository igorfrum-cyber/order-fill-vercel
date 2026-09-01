import { useEffect, useState } from "react";
import { getMe, logout } from "./api/auth.js";
import { onAuthRequired } from "./api/client.js";
import { getJob, getJobReport, listJobFiles } from "./api/jobs.js";
import { CompaniesScreen, JobHistory, UsersScreen } from "./ui/admin/AdminScreens.jsx";
import { InviteScreen, LoginScreen } from "./ui/auth/AuthScreens.jsx";
import { NorthApp } from "./ui/north/NorthApp.jsx";
import { OrderFillApp } from "./ui/order/OrderFillApp.jsx";
import { initialEditState } from "./features/order/reviewEdits.js";

export default function App() {
  const inviteToken = inviteTokenFromPath();
  const [me, setMe] = useState(undefined);
  const [screen, setScreen] = useState("history");
  const [mode, setMode] = useState("order");
  const [companyId, setCompanyId] = useState("");
  const [resume, setResume] = useState(null);

  useEffect(() => {
    getMe()
      .then((user) => {
        setMe(user);
        if (user.company_id) setCompanyId(user.company_id);
      })
      .catch(() => setMe(null));
  }, []);

  useEffect(() => onAuthRequired(() => setMe(null)), []);

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
    return (
      <LoginScreen
        onDone={(user) => {
          setMe(user);
          if (user.company_id) setCompanyId(user.company_id);
        }}
      />
    );
  }

  if (screen === "order") {
    return (
      <OrderFillApp
        mode={mode}
        onMode={(next) => {
          setMode(next);
          setScreen(next === "north" ? "north" : "order");
        }}
        companyId={companyId}
        resumeJob={resume}
        onHome={() => {
          setResume(null);
          setScreen("history");
        }}
      />
    );
  }

  if (screen === "north") {
    return (
      <NorthApp
        mode={mode}
        onMode={(next) => {
          setMode(next);
          setScreen(next === "north" ? "north" : "order");
        }}
        companyId={companyId}
        onHome={() => {
          setResume(null);
          setScreen("history");
        }}
      />
    );
  }

  return (
    <div className="flex h-full flex-col bg-[var(--color-ground)]">
      <header className="flex items-center justify-between border-b border-[var(--color-line)] bg-[var(--color-surface)] px-6 py-3">
        <nav className="flex gap-2 text-[14px] font-medium">
          <NavButton active={screen === "history"} onClick={() => setScreen("history")}>
            Выгрузки
          </NavButton>
          {me.role === "platform_admin" ? (
            <NavButton active={screen === "companies"} onClick={() => setScreen("companies")}>
              Компании
            </NavButton>
          ) : null}
          {me.role !== "purchaser" ? (
            <NavButton active={screen === "users"} onClick={() => setScreen("users")}>
              Пользователи
            </NavButton>
          ) : null}
        </nav>
        <div className="flex items-center gap-3 text-[14px] text-[var(--color-ink-soft)]">
          <span>{me.login}</span>
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
      <main className="flex-1 overflow-auto">
        {screen === "history" ? (
          <JobHistory
            me={me}
            companyId={companyId}
            onCompany={setCompanyId}
            onNew={() => {
              if (me.role === "platform_admin" && !companyId) return;
              setResume(null);
              setScreen("order");
            }}
            onOpen={async (job) => {
              if (job.type === "north_merge") {
                setMode("north");
                setScreen("north");
                return;
              }
              const loaded = await loadOrderResume(job.id);
              setResume(loaded);
              setMode("order");
              setScreen("order");
            }}
          />
        ) : null}
        {screen === "companies" ? <CompaniesScreen selectedId={companyId} onSelect={setCompanyId} /> : null}
        {screen === "users" ? <UsersScreen companyId={me.role === "platform_admin" ? companyId : me.company_id} /> : null}
      </main>
    </div>
  );
}

function NavButton({ active, onClick, children }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-lg px-3 py-1.5 ${active ? "bg-[var(--color-brand-soft)] text-[var(--color-brand-strong)]" : "text-[var(--color-ink-faint)]"}`}
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
