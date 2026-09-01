export const MIN_PASSWORD_LENGTH = 10;
export const MAX_PASSWORD_LENGTH = 1024;

const PASSWORD_ALPHABET = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%";
const PASSWORD_LETTERS = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ";
const PASSWORD_DIGITS = "23456789";

export function generatePassword(length = 16) {
  const size = Math.max(MIN_PASSWORD_LENGTH, Math.min(length, 64));
  const bytes = new Uint32Array(size + 2);
  globalThis.crypto.getRandomValues(bytes);
  const chars = Array.from({ length: size }, (_, index) => PASSWORD_ALPHABET[bytes[index] % PASSWORD_ALPHABET.length]);
  chars[0] = PASSWORD_LETTERS[bytes[size] % PASSWORD_LETTERS.length];
  chars[1] = PASSWORD_DIGITS[bytes[size + 1] % PASSWORD_DIGITS.length];
  for (let index = chars.length - 1; index > 0; index -= 1) {
    const swap = bytes[index] % (index + 1);
    [chars[index], chars[swap]] = [chars[swap], chars[index]];
  }
  return chars.join("");
}

export function passwordIssues(password, repeat) {
  const value = String(password || "");
  const issues = [];
  if (value.length < MIN_PASSWORD_LENGTH) issues.push(`минимум ${MIN_PASSWORD_LENGTH} символов`);
  if (value.length > MAX_PASSWORD_LENGTH) issues.push("слишком длинный");
  if (/\s/.test(value)) issues.push("без пробелов");
  if (value && !/[A-Za-zА-Яа-яЁё]/.test(value)) issues.push("хотя бы одна буква");
  if (value && !/\d/.test(value)) issues.push("хотя бы одна цифра");
  if (repeat != null && value !== repeat) issues.push("пароли не совпадают");
  return issues;
}

export function isPasswordReady(password, repeat) {
  return passwordIssues(password, repeat).length === 0;
}
