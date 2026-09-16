import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

const mockGetInboundCompany = vi.fn();
const mockGetInboundMessages = vi.fn();
const mockGetInboundSettings = vi.fn();
const mockGetInboundDeliveries = vi.fn();
const mockUpdateInboundCompany = vi.fn();
const mockUpdateInboundSettings = vi.fn();
const mockDownloadInboundFile = vi.fn();

vi.mock("../../api/inbound.js", () => ({
  getInboundCompany: (...args) => mockGetInboundCompany(...args),
  getInboundMessages: (...args) => mockGetInboundMessages(...args),
  getInboundSettings: (...args) => mockGetInboundSettings(...args),
  getInboundDeliveries: (...args) => mockGetInboundDeliveries(...args),
  updateInboundCompany: (...args) => mockUpdateInboundCompany(...args),
  updateInboundSettings: (...args) => mockUpdateInboundSettings(...args),
  downloadInboundFile: (...args) => mockDownloadInboundFile(...args),
}));

vi.mock("../../features/help/errors.js", () => ({
  userFacingError: (err) => err?.message || "Ошибка",
}));

import { InboundScreen } from "./InboundScreen.jsx";

const companyMe = { role: "company_owner", company_id: "c1" };

test("company owner sees sender_email even when settings endpoint returns 404", async () => {
  mockGetInboundCompany.mockResolvedValue({
    company_id: "c1",
    receive_address: "7e1432246b724f3bcd6c@cloudmailin.net",
    sender_email: "1c@testcompany.ru",
    enabled: true,
  });
  mockGetInboundMessages.mockResolvedValue({ messages: [] });
  mockGetInboundSettings.mockRejectedValue({ status: 404 });

  render(<InboundScreen me={companyMe} companyId="c1" />);

  const senderCell = await screen.findByText("1c@testcompany.ru");
  expect(senderCell).toBeInTheDocument();
});

test("company owner sees empty sender_email placeholder when not configured", async () => {
  mockGetInboundCompany.mockRejectedValue({ status: 404 });
  mockGetInboundMessages.mockResolvedValue({ messages: [] });
  mockGetInboundSettings.mockRejectedValue({ status: 404 });

  render(<InboundScreen me={companyMe} companyId="c1" />);

  const placeholders = await screen.findAllByText("не задан");
  expect(placeholders.length).toBeGreaterThanOrEqual(1);
});

test("platform admin sees address panel with save button", async () => {
  const platformMe = { role: "platform_admin" };
  mockGetInboundSettings.mockResolvedValue({ enabled: true, receive_address: "7e1432246b724f3bcd6c@cloudmailin.net", webhook_count: 5, error_count: 0 });
  mockGetInboundDeliveries.mockResolvedValue({ deliveries: [] });

  render(<InboundScreen me={platformMe} />);

  const heading = await screen.findByText("Адрес приёма");
  expect(heading).toBeInTheDocument();
  const saveButton = screen.getByRole("button", { name: "Сохранить" });
  expect(saveButton).toBeInTheDocument();
});

test("company panel shows platform address from settings", async () => {
  mockGetInboundCompany.mockResolvedValue({
    company_id: "c1",
    receive_address: "7e1432246b724f3bcd6c@cloudmailin.net",
    sender_email: "1c@testco.ru",
    enabled: true,
  });
  mockGetInboundMessages.mockResolvedValue({ messages: [] });
  mockGetInboundSettings.mockResolvedValue({ receive_address: "7e1432246b724f3bcd6c@cloudmailin.net" });

  render(<InboundScreen me={companyMe} companyId="c1" />);

  const address = await screen.findByText("7e1432246b724f3bcd6c@cloudmailin.net");
  expect(address).toBeInTheDocument();
});
