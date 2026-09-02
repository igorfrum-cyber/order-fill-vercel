export function twoFactorOpenAppHref(authURL) {
  const value = String(authURL || "").trim();
  if (!value.toLowerCase().startsWith("otpauth://")) return "";
  return value;
}
