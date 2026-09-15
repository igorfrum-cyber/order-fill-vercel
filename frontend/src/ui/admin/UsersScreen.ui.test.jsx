import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, test, vi } from "vitest";

const { createPlatformAdmin, enableUser, listPlatformAdmins, listUsers } = vi.hoisted(() => ({
  createPlatformAdmin: vi.fn(),
  enableUser: vi.fn(),
  listPlatformAdmins: vi.fn(),
  listUsers: vi.fn(),
}));

vi.mock("../../api/auth.js", () => ({
  listUsers,
  listPlatformAdmins,
  listCompanies: async () => ({ companies: [{ id: "company", name: "Компания" }] }),
  createUser: vi.fn(),
  createPlatformAdmin,
  disableUser: vi.fn(),
  enableUser,
  resetUser: vi.fn(),
}));

import { UsersScreen } from "./UsersScreen.jsx";

beforeEach(() => {
  createPlatformAdmin.mockReset().mockResolvedValue({ invite_url: "/invite/admin" });
  enableUser.mockReset().mockResolvedValue(undefined);
  listPlatformAdmins.mockReset().mockResolvedValue({
    users: [
      { id: "root", login: "root", role: "platform_admin", is_primary_admin: true, activated: true },
      { id: "ops", login: "ops", role: "platform_admin", is_primary_admin: false, activated: true },
    ],
  });
  listUsers.mockReset();
});

test("primary platform admin can invite a secondary admin and the main account is protected", async () => {
  listUsers.mockResolvedValue({ users: [] });
  const user = userEvent.setup();
  render(<UsersScreen companyId="company" actorRole="platform_admin" actorId="root" actorIsPrimaryAdmin onCompany={() => {}} />);

  expect(await screen.findByText("Администраторы сервиса")).toBeInTheDocument();
  expect(screen.getAllByText(/Главный администратор/)).toHaveLength(2);
  expect(screen.queryAllByRole("button", { name: "Выключить" })).toHaveLength(1);
  await user.type(screen.getByRole("textbox", { name: "Логин администратора" }), "new-ops");
  await user.click(screen.getByRole("button", { name: "Пригласить администратора" }));
  expect(createPlatformAdmin).toHaveBeenCalledWith("new-ops");
});

test("platform admin sees invite activation and can refresh it", async () => {
  listUsers
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: false }] })
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: true }] });
  const user = userEvent.setup();
  render(<UsersScreen companyId="company" actorRole="platform_admin" actorId="admin" onCompany={() => {}} />);
  expect(await screen.findByText("Ждёт активации")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Обновить" }));
  await waitFor(() => expect(screen.queryByText("Ждёт активации")).not.toBeInTheDocument());
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
