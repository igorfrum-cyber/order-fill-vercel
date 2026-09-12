import { homeScreen, navItemsForRole } from "../auth/accessPresentation.js";

export function parseAppPath(pathname) {
  const path = String(pathname || "/").replace(/\/+$/, "") || "/";
  if (path.startsWith("/invite/") || path.startsWith("/c/")) {
    return { screen: "", jobId: "", unknown: false };
  }
  if (path === "/") return { screen: "", jobId: "", unknown: false };
  if (path === "/overview") return { screen: "overview", jobId: "", unknown: false };
  if (path === "/queue") return { screen: "queue", jobId: "", unknown: false };
  if (path === "/jobs") return { screen: "history", jobId: "", unknown: false };
  if (path === "/jobs/new") return { screen: "order", jobId: "", unknown: false };
  if (path === "/company") return { screen: "company", jobId: "", unknown: false };
  if (path === "/users") return { screen: "users", jobId: "", unknown: false };
  if (path === "/account") return { screen: "account", jobId: "", unknown: false };
  if (path === "/companies") return { screen: "companies", jobId: "", unknown: false };
  if (path === "/north") return { screen: "north", jobId: "", unknown: false };
  const job = path.match(/^\/jobs\/([^/]+)$/);
  if (job) return { screen: "order", jobId: decodeURIComponent(job[1]), unknown: false };
  return { screen: "", jobId: "", unknown: true };
}

export function pathForScreen(screen, jobId = "") {
  if (jobId) return `/jobs/${encodeURIComponent(jobId)}`;
  if (screen === "overview") return "/overview";
  if (screen === "queue") return "/queue";
  if (screen === "history") return "/jobs";
  if (screen === "order") return "/jobs/new";
  if (screen === "company") return "/company";
  if (screen === "users") return "/users";
  if (screen === "account") return "/account";
  if (screen === "companies") return "/companies";
  if (screen === "north") return "/north";
  return "/";
}

export function resolveOrderNavJobId(next, requestedJobId = "", { screen, openJobId } = {}) {
  if (next !== "order") return requestedJobId || "";
  if (requestedJobId) return requestedJobId;
  if (openJobId && (screen === "order" || screen === "account")) return openJobId;
  return "";
}

export function screenAllowed(role, screen, { jobId = "" } = {}) {
  if (!screen) return true;
  if (screen === "account") return true;
  if (screen === "order" && jobId) return true;
  if (screen === "order") return role !== "platform_admin";
  if (screen === "north") return role !== "platform_admin";
  return navItemsForRole(role).some((item) => item.id === screen);
}

export function companyIdFromSearch(search) {
  return new URLSearchParams(String(search || "").replace(/^\?/, "")).get("company") || "";
}

export function withCompanyQuery(path, companyId) {
  if (!companyId) return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}company=${encodeURIComponent(companyId)}`;
}

export function scopedPath(screen, jobId = "", companyId = "") {
  return withCompanyQuery(pathForScreen(screen, jobId), companyId);
}

export function navItemHref(item, { screen, openJobId, companyId } = {}) {
  if (item?.id === "order") {
    const jobId = resolveOrderNavJobId("order", "", { screen, openJobId });
    return scopedPath("order", jobId, companyId);
  }
  return withCompanyQuery(item?.path || "/", companyId);
}

export function homePath(role, companyId = "") {
  return withCompanyQuery(pathForScreen(homeScreen(role)), companyId);
}
