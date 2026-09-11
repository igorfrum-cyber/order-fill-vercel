import { useState } from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";

import { Modal, PasswordField, Stepper } from "./widgets.jsx";

function PasswordHarness() {
  const [value, setValue] = useState("");
  return <PasswordField label="Пароль" value={value} onChange={setValue} autoComplete="current-password" />;
}

function StepperHarness() {
  const [value, setValue] = useState("");
  return <Stepper value={value} onChange={setValue} step={1} />;
}

test("user can type a password and reveal it", async () => {
  const user = userEvent.setup();
  render(<PasswordHarness />);

  const input = screen.getByLabelText("Пароль");
  await user.type(input, "secret-pass");

  expect(input).toHaveValue("secret-pass");
  expect(input).toHaveAttribute("type", "password");

  await user.click(screen.getByRole("button", { name: "Показать пароль" }));
  expect(input).toHaveAttribute("type", "text");
});

test("password reveal does not steal the field name from the input", () => {
  render(<PasswordHarness />);
  expect(screen.getByLabelText("Пароль")).toHaveAttribute("type", "password");
});

test("user can type a decimal quantity", async () => {
  const user = userEvent.setup();
  render(<StepperHarness />);

  const input = screen.getByLabelText("Количество");
  await user.type(input, "12,5");

  expect(input).toHaveValue("12.5");
});

test("stepper plus and minus have names", () => {
  render(<StepperHarness />);
  expect(screen.getByRole("button", { name: "Меньше" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Больше" })).toBeEnabled();
});

test("modal closes on Escape and backdrop click", async () => {
  const user = userEvent.setup();
  const onCancel = vi.fn();
  render(
    <Modal title="Проверьте спорные строки" onCancel={onCancel} onConfirm={() => {}}>
      текст
    </Modal>,
  );

  await user.keyboard("{Escape}");
  expect(onCancel).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("dialog"));
  expect(onCancel).toHaveBeenCalledTimes(2);
});
