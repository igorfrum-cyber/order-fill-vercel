import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

const listJobs = vi.fn(async (companyId) => ({
  jobs: [
    { id: "all", status: "queued", brand: "klapp", created_by_login: "all" },
    ...(companyId === "c-1"
      ? [{ id: "one", status: "queued", brand: "klapp", created_by_login: "one" }]
      : []),
  ].filter((job) => (companyId ? job.id === "one" : job.id === "all")),
}));

vi.mock("../../api/auth.js", () => ({
  listStatus: async () => ({
    components: [
      { id: "api", ok: true },
      { id: "worker", ok: true },
      { id: "postgres", ok: true },
      { id: "queue", ok: true },
      { id: "files", ok: true },
    ],
  }),
  listJobs: (...args) => listJobs(...args),
  listAudit: async () => ({ events: [] }),
}));

import { OverviewScreen } from "./OverviewScreen.jsx";

test("overview live jobs follow the selected company", async () => {
  render(<OverviewScreen companyId="c-1" />);
  expect(await screen.findByText("one")).toBeInTheDocument();
  expect(screen.queryByText("all")).toBeNull();
  expect(listJobs).toHaveBeenCalledWith("c-1");
});
