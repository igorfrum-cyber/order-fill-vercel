import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, test, vi } from "vitest";

const { enableUser, listUsers } = vi.hoisted(() => ({
  enableUser: vi.fn(),
  listUsers: vi.fn(),
}));

vi.mock("../../api/auth.js", () => ({
  listUsers,
  listCompanies: async () => ({ companies: [{ id: "company", name: "Компания" }] }),
  createUser: vi.fn(),
  disableUser: vi.fn(),
  enableUser,
  resetUser: vi.fn(),
}));

import { UsersScreen } from "./UsersScreen.jsx";

beforeEach(() => {
  enableUser.mockReset().mockResolvedValue(undefined);
  listUsers.mockReset();
});

test("platform admin sees invite activation and can refresh it", async () => {
  listUsers
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: false }] })
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: true }] });
  const user = userEvent.setup();
  render(<UsersScreen companyId="company" actorRole="platform_admin" actorId="admin" onCompany={() => {}} />);
  expect(await screen.findByText("Ждёт активации")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Обновить" }));
  expect(await screen.findByText(/Активен/)).toBeInTheDocument();
  expect(listUsers).toHaveBeenCalledTimes(2);
});

test("disabled account has a colored status and can be enabled", async () => {
  listUsers.mockResolvedValue({
    users: [{ id: "buyer", login: "buyer", role: "purchaser", activated: true, disabled_at: "2026-09-15T08:00:00Z" }],
  });
  const user = userEvent.setup();
  render(<UsersScreen companyId="company" actorRole="platform_admin" actorId="admin" onCompany={() => {}} />);

  const status = await screen.findByText("Выключен");
  expect(status).toHaveClass("text-[var(--color-danger)]");
  await user.click(screen.getByRole("button", { name: "Включить" }));
  expect(enableUser).toHaveBeenCalledWith("buyer");
});
