import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

const mockListJobs = vi.fn();
const mockListUsers = vi.fn();

vi.mock("../../api/auth.js", () => ({
  listJobs: (...args) => mockListJobs(...args),
  listUsers: (...args) => mockListUsers(...args),
}));

import { QueueScreen } from "./QueueScreen.jsx";

test("queue shows stuck jobs, purchaser count, and opens a selected job", async () => {
  const user = userEvent.setup();
  const onOpen = vi.fn();
  mockListJobs.mockResolvedValue({
    jobs: [{ id: "job-1", type: "north_merge", brand: "klapp", status: "needs_review", created_by_login: "buyer" }],
  });
  mockListUsers.mockResolvedValue({
    users: [
      { role: "purchaser", disabled_at: null },
      { role: "purchaser", disabled_at: "2026-01-01" },
      { role: "company_owner", disabled_at: null },
    ],
  });

  render(<QueueScreen me={{ company_id: "company-1", company_name: "Northwind" }} onOpen={onOpen} />);

  const jobButton = await screen.findByRole("button", { name: /Север/ });
  expect(jobButton).toHaveTextContent("buyer");
  expect(screen.getByText("Есть закупщик, который заполняет бланки.")).toBeInTheDocument();
  expect(screen.getByText("Northwind")).toBeInTheDocument();
  await user.click(jobButton);
  expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: "job-1" }));
  expect(mockListUsers).toHaveBeenCalledWith("company-1");
});

test("queue shows a friendly message when jobs fail to load", async () => {
  mockListJobs.mockRejectedValue(new Error("network unavailable"));
  mockListUsers.mockReset();

  render(<QueueScreen me={{ company_id: "" }} />);

  expect(await screen.findByText("Не удалось загрузить очередь.")).toBeInTheDocument();
  expect(mockListUsers).not.toHaveBeenCalled();
});

test("empty queue invites a purchaser and links to the company profile", async () => {
  const user = userEvent.setup();
  const onPeople = vi.fn();
  const onCompany = vi.fn();
  mockListJobs.mockResolvedValue({ jobs: [] });
  mockListUsers.mockResolvedValue({ users: [] });

  render(<QueueScreen me={{ company_id: "company-2" }} onPeople={onPeople} onCompany={onCompany} />);

  expect(await screen.findByText("Сейчас никто не ждёт вас.")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Пригласить закупщика" }));
  await user.click(screen.getByRole("button", { name: "Профиль компании" }));
  expect(onPeople).toHaveBeenCalledOnce();
  expect(onCompany).toHaveBeenCalledOnce();
});
