import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

const mockGetInboundCompany = vi.fn();
const mockGetInboundMessages = vi.fn();
const mockGetInboundMessage = vi.fn();
const mockGetInboundSettings = vi.fn();
const mockGetInboundDeliveries = vi.fn();
const mockUpdateInboundCompany = vi.fn();
const mockUpdateInboundSettings = vi.fn();
const mockDownloadInboundFile = vi.fn();

vi.mock("../../api/inbound.js", () => ({
  getInboundCompany: (...args) => mockGetInboundCompany(...args),
  getInboundMessages: (...args) => mockGetInboundMessages(...args),
  getInboundMessage: (...args) => mockGetInboundMessage(...args),
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

test("platform admin delivery table shows status without mail headers", async () => {
  const platformMe = { role: "platform_admin" };
  mockGetInboundSettings.mockResolvedValue({ enabled: true, receive_address: "in@cloudmailin.net", webhook_count: 1, error_count: 0 });
  mockGetInboundDeliveries.mockResolvedValue({
    deliveries: [{ id: "d1", received_at: "2026-09-15T10:00:00Z", company_id: "c1", status: "received", subject: "секрет", envelope_from: "1c@secret.ru" }],
  });

  render(<InboundScreen me={platformMe} />);

  expect(await screen.findByText("Принято")).toBeInTheDocument();
  expect(screen.queryByText("секрет")).not.toBeInTheDocument();
  expect(screen.queryByText("1c@secret.ru")).not.toBeInTheDocument();
  expect(screen.queryByText("Тема")).not.toBeInTheDocument();
  expect(screen.queryByText("Отправитель")).not.toBeInTheDocument();
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

test("company owner opens formatted inbound message", async () => {
  mockGetInboundCompany.mockResolvedValue({ company_id: "c1", sender_email: "1c@testco.ru", enabled: true });
  mockGetInboundMessages.mockResolvedValue({
    messages: [{ id: "m1", subject: "Заказ", received_at: "2026-09-15T10:00:00Z", envelope_from: "1c@testco.ru", attachments: [] }],
  });
  mockGetInboundSettings.mockResolvedValue({ receive_address: "inbound@example.com" });
  mockGetInboundMessage.mockResolvedValue({
    id: "m1",
    subject: "Заказ",
    received_at: "2026-09-15T10:00:00Z",
    envelope_from: "1c@testco.ru",
    body_html: "<p><strong>Итого</strong></p>",
    body_text: "Итого",
  });

  render(<InboundScreen me={companyMe} companyId="c1" />);

  await screen.findByRole("button", { name: "Заказ" });
  await screen.getByRole("button", { name: "Заказ" }).click();

  expect(await screen.findByTitle("Содержимое письма")).toHaveAttribute("sandbox", "");
  expect(mockGetInboundMessage).toHaveBeenCalledWith("c1", "m1");
});
