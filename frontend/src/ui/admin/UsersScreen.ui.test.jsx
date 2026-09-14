import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

const { listUsers } = vi.hoisted(() => ({
  listUsers: vi.fn()
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: false }] })
    .mockResolvedValueOnce({ users: [{ id: "pending", login: "new-buyer", role: "purchaser", activated: true }] }),
}));

vi.mock("../../api/auth.js", () => ({
  listUsers,
  listCompanies: async () => ({ companies: [{ id: "company", name: "Компания" }] }),
  createUser: vi.fn(),
  disableUser: vi.fn(),
  resetUser: vi.fn(),
}));

import { UsersScreen } from "./UsersScreen.jsx";

test("platform admin sees invite activation and can refresh it", async () => {
  const user = userEvent.setup();
  render(<UsersScreen companyId="company" actorRole="platform_admin" actorId="admin" onCompany={() => {}} />);
  expect(await screen.findByText("Ждёт активации по ссылке")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Обновить" }));
  expect(await screen.findByText(/Аккаунт активирован/)).toBeInTheDocument();
  expect(listUsers).toHaveBeenCalledTimes(2);
});
