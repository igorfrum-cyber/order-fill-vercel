import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

const { updateCompanyOrderProfile } = vi.hoisted(() => ({
  updateCompanyOrderProfile: vi.fn(async (_companyId, profile) => profile),
}));

vi.mock("../../api/auth.js", () => ({
  clearCompanyLogo: vi.fn(),
  getCompanyOrderProfile: async () => ({
    legal_name: "ООО Тест",
    contact_phone: "+7 900 000-00-00",
    brand_terms: [{ brand: "angiopharm", discount_set: true, discount_basis_points: 3500 }],
  }),
  setCompanyLogo: vi.fn(),
  updateCompany: vi.fn(),
  updateCompanyOrderProfile,
}));

import { CompanyScreen } from "./CompanyScreen.jsx";

const me = { company_id: "company-1", company_name: "Тест", login_slug: "test", has_logo: false };

test("company order profile loads saved requisites and brand discount", async () => {
  render(<CompanyScreen me={me} />);
  expect(await screen.findByDisplayValue("ООО Тест")).toBeInTheDocument();
  expect(screen.getByDisplayValue("35")).toBeInTheDocument();
  expect(screen.getByText(/при следующем расчёте/i)).toBeInTheDocument();
});

test("company order profile blocks invalid values and saves a valid discount", async () => {
  const user = userEvent.setup();
  render(<CompanyScreen me={me} />);
  const phone = await screen.findByLabelText("Телефон получателя");
  await user.clear(phone);
  await user.type(phone, "wrong");
  await user.click(screen.getByRole("button", { name: "Сохранить данные для бланков" }));
  expect(screen.getByRole("alert")).toHaveTextContent(/проверьте телефон/i);
  expect(updateCompanyOrderProfile).not.toHaveBeenCalled();

  await user.clear(phone);
  await user.type(phone, "+7 912 345-67-89");
  const discount = screen.getByLabelText("Скидка ANGIOPHARM, %");
  await user.clear(discount);
  await user.type(discount, "30,25");
  await user.click(screen.getByRole("button", { name: "Сохранить данные для бланков" }));
  expect(updateCompanyOrderProfile).toHaveBeenCalledWith("company-1", expect.objectContaining({
    contact_phone: "+7 912 345-67-89",
    brand_terms: expect.arrayContaining([expect.objectContaining({ brand: "angiopharm", discount_basis_points: 3025, discount_set: true })]),
  }));
});
